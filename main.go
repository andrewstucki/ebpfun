package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/andrewstucki/ebpfun/consul"
	"github.com/andrewstucki/ebpfun/firewall"
	"github.com/andrewstucki/ebpfun/rate"
	"github.com/cilium/ebpf/rlimit"
)

func init() {
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	//var configFile string
	var drop bool
	var limit bool
	var serverAddr string
	var ingressAddr string
	var egressAddr string
	var egressRate int
	var service string
	var dc string

	//flag.StringVar(&configFile, "config", "", "source of configuration")
	flag.BoolVar(&drop, "drop", true, "drop non-exempt flows")
	flag.BoolVar(&limit, "limit", false, "drop non-exempt flows")
	flag.StringVar(&serverAddr, "serverAddr", "localhost:8300", "Consul server address")
	flag.StringVar(&ingressAddr, "ingressAddr", "localhost:8080", "local ingress port")
	flag.StringVar(&egressAddr, "egressAddr", "localhost:8080", "remote egress destination")
	flag.IntVar(&egressRate, "egressRate", 0, "egress rate limit")
	flag.StringVar(&service, "service", "foo", "local service name")
	flag.StringVar(&dc, "dc", "dc1", "local DC name")

	flag.Parse()

	addr, err := net.ResolveTCPAddr("tcp", ingressAddr)
	if err != nil {
		log.Fatalln(err)
	}

	ingresses := []firewall.Ingress{{
		Address: addr.IP,
		Port:    addr.Port,
	}}

	c, err := consul.NewRPCClient(serverAddr, dc)
	if err != nil {
		log.Fatalf("failed to create client: %s", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	ch, err := c.WatchExemptionsForService(ctx, service)
	if err != nil {
		log.Fatalf("failed to watch: %s", err)
	}

	if drop {
		for {
			select {
			case <-ctx.Done():
				return
			case es := <-ch:
				log.Printf("Exemptions updated: %v\n", es.Exemptions)

				exemptions := make([]firewall.Exemption, 0, len(es.Exemptions))
				for _, e := range es.Exemptions {
					ip := net.ParseIP(e)
					if ip == nil {
						log.Printf("WARN invalid IP: %s\n", e)
						continue
					}
					exemptions = append(exemptions, firewall.Exemption{
						Source:      ip,
						Destination: addr.IP,
						Port:        addr.Port,
					})
				}

				if err := firewall.Update(ingresses, exemptions); err != nil {
					log.Fatalf("error updating firewall configuration: %v", err)
				}
				log.Printf("updated firewall")
			}
		}
	}

	if limit {
		if egressRate <= 0 {
			log.Fatalln("invalid rate limit provided")
		}

		egressAddr, err := net.ResolveTCPAddr("tcp", egressAddr)
		if err != nil {
			log.Fatalln(err)
		}

		egress := &rate.Egress{
			Address: egressAddr.IP,
			Port:    egressAddr.Port,
			Rate:    egressRate,
		}

		if err := rate.UpdateRateLimit(egress); err != nil {
			log.Fatalf("error updating firewall configuration: %v", err)
		}
		log.Printf("set rate limit for %v to %v per second", egress.String(), egressRate)
	}

	// // firewall.Update can be called any time to update the ingresses/exemptions live
	// if err := firewall.Update(ingresses, exemptions); err != nil {
	// 	log.Fatalf("error updating firewall configuration: %v", err)
	// }
	// defer firewall.Cleanup()

	// var wg sync.WaitGroup
	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	for {
	// 		select {
	// 		case stats := <-firewall.Stats:
	// 			log.Println(stats)
	// 		case <-ctx.Done():
	// 			return
	// 		}
	// 	}
	// }()

	// if err := firewall.Poll(ctx, 1*time.Second); err != nil {
	// 	log.Fatalf("error linking XDP program: %v", err)
	// }

	// wg.Wait()
}
