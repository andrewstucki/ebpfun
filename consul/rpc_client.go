package consul

import (
	"context"
	"fmt"
	"log"
	"net"
	"reflect"
	"sort"
	"time"

	"github.com/hashicorp/consul/agent/pool"
	"github.com/hashicorp/consul/agent/structs"
	"github.com/hashicorp/consul/tlsutil"
	"github.com/ryboe/q"
)

type RPCClient struct {
	dc           string
	pool         *pool.ConnPool
	serverIPAddr net.Addr
}

func NewRPCClient(serverAddr, dc string) (*RPCClient, error) {
	tlsCfg, err := tlsutil.NewConfigurator(tlsutil.Config{}, nil)
	if err != nil {
		panic(err)
	}

	addr, err := net.ResolveTCPAddr("tcp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid server address: %w", err)
	}

	return &RPCClient{
		dc:           dc,
		serverIPAddr: addr,
		pool: &pool.ConnPool{
			SrcAddr:         nil,
			MaxTime:         time.Hour,
			MaxStreams:      1000000,
			TLSConfigurator: tlsCfg,
			Logger:          log.Default(),
		},
	}, nil
}

func (c *RPCClient) RPC(method string, args, resp interface{}) error {
	return c.pool.RPC(c.dc, "server1", c.serverIPAddr, method, &args, &resp)
}

type ExemptionSet struct {
	Exemptions []string
	Error      error
}

// WatchExemptionsForService uses the Internal.ServiceTopology RPC call to get
// all the instances that could be downstream of an instance of a given service.
// On initial call the list will be build and delivered to the returned chan.
// Each time the set of addresses changes the new complete set is sent on the
// chan. Blocking will continue until the context is cancelled or an
// unrecoverable error occurs. If an error occurs an ExemptionSet with Error set
// will be delivered and blocking will terminate.
func (c *RPCClient) WatchExemptionsForService(ctx context.Context, service string) (<-chan ExemptionSet, error) {
	ch := make(chan ExemptionSet, 1)
	go c.watchExemptions(ctx, service, ch)
	return ch, nil
}

func (c *RPCClient) watchExemptions(ctx context.Context, service string, ch chan ExemptionSet) {
	args := &structs.ServiceSpecificRequest{
		Datacenter:  c.dc,
		ServiceName: service,
		Connect:     true,
		QueryOptions: structs.QueryOptions{
			// TODO ACL support
			MaxQueryTime: 10 * time.Minute,
			AllowStale:   true,
		},
	}

	var resp structs.IndexedServiceTopology
	var lastSet *ExemptionSet

	for {
		// Check context
		if ctx.Err() != nil {
			return
		}

		if err := c.RPC("Internal.ServiceTopology", args, &resp); err != nil {
			log.Printf("ERROR fetching service toplogy (will retry in 5s): %s", err)
			select {
			case <-time.After(5 * time.Second):
				continue
			case <-ctx.Done():
				return
			}
		}

		q.Q(resp)

		// Got a result! Process it into exemptions
		e := exemptionsFromTopology(service, &resp)

		if lastSet == nil || reflect.DeepEqual(e, *lastSet) {
			select {
			case <-ctx.Done():
				return
			case ch <- e:
			}
		}

		lastSet = &e

		// Set index for next time around
		args.MinQueryIndex = resp.Index

		if args.MinQueryIndex < 1 {
			// Prevent hot loops if the server ever returns 0 which is invalid
			args.MinQueryIndex = 1
		}
	}
}

func exemptionsFromTopology(service string, t *structs.IndexedServiceTopology) ExemptionSet {
	es := ExemptionSet{}

	// Find all downstreams that are allowed.
	for _, csn := range t.ServiceTopology.Downstreams {
		// Check this is allowed
		sn := structs.NewServiceName(csn.Service.Service, &csn.Service.EnterpriseMeta)
		q.Q(sn, sn.String())
		decision, ok := t.ServiceTopology.DownstreamDecisions[sn.String()]
		if !ok {
			// Skip if there is no allow intention
			continue
		}
		if decision.Allowed {
			addr := csn.Service.Address
			if addr == "" {
				addr = csn.Node.Address
			}
			es.Exemptions = append(es.Exemptions, addr)
		}
	}
	es.Exemptions = sort.StringSlice(es.Exemptions)
	return es
}
