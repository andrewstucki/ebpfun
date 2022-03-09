package rate

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	tc "github.com/florianl/go-tc"
	"github.com/florianl/go-tc/core"
	"github.com/florianl/go-tc/internal/unix"
)

// UpdateRateLimit uses tc to set a rate limit policy on a specific host
func UpdateRateLimit(egress *Egress) error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}
	fmt.Println(interfaces)

	// https://www.badunetworks.com/traffic-shaping-with-tc/
	// I'm sure this is not smart but I'm just adding this to every device for now
	// because I don't know necessarily know which interface we're sending on.
	for _, iface := range interfaces {
		tcnl, err := tc.Open(&tc.Config{})
		if err != nil {
			return err
		}
		defer func() {
			if err := tcnl.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "could not close tfnetlink socket: %v\n", err)
			}
		}()

		linklayerEthernet := uint8(1)
		// These should be getting read as bps?
		// https://github.com/florianl/go-tc/blob/main/example_tbf_test.go#L78
		egressRate, err := strconv.ParseUint(fmt.Sprintf("%#x", egress.Rate), 16, 32)
		if err != nil {
			return err
		}
		egressRate32 := uint32(egressRate)
		burst, err := strconv.ParseUint(fmt.Sprintf("%#x", egress.Rate*2), 16, 32)
		if err != nil {
			return err
		}
		burst32 := uint32(burst)
		// There must be a better way.  It's late and I'm tired.

		log.Printf("egressRate32: %v\n", egressRate32)
		log.Printf("burst32: %v\n", burst32)

		tcMsg := tc.Msg{
			Family:  unix.AF_UNSPEC,
			Ifindex: uint32(iface.Index),
			Handle:  core.BuildHandle(tc.HandleRoot, 0x0),
			Parent:  tc.HandleRoot,
			Info:    0,
		}

		// https://github.com/florianl/go-tc/blob/main/example_tbf_test.go#L52-L76
		qdisc := tc.Object{
			tcMsg,
			tc.Attribute{
				Kind: "tbf",
				Tbf: &tc.Tbf{
					Parms: &tc.TbfQopt{
						Mtu:   1514,
						Limit: 0x5000,
						Rate: tc.RateSpec{
							Rate:      egressRate32,
							Linklayer: linklayerEthernet,
							CellLog:   0x3,
						},
					},
					Burst: &burst32,
				},
			},
		}

		if err := tcnl.Qdisc().Add(&qdisc); err != nil {
			return err
		}

		qdiscs, err := tcnl.Qdisc().Get()
		if err != nil {
			return err
		}

		fmt.Println("## qdiscs:")
		for _, qdisc := range qdiscs {

			iface, err := net.InterfaceByIndex(int(qdisc.Ifindex))
			if err != nil {
				return err
			}
			fmt.Printf("%20s\t%-11s\n", iface.Name, qdisc.Kind)
		}

		filter := tc.Object{
			tcMsg,
			tc.Attribute{
				// https://github.com/florianl/go-tc/blob/master/filter_test.go
				// TODO: All of it.
			}
		}
	}

	return nil
}
