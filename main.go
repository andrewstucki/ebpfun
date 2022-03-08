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
	"github.com/cilium/ebpf/rlimit"
)

func init() {
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	var configFile string
	var serverAddr string
	var ingressAddr string
	var service string
	var dc string

	flag.StringVar(&configFile, "config", "", "source of configuration")
	flag.StringVar(&serverAddr, "serverAddr", "localhost:8300", "Consul server address")
	flag.StringVar(&ingressAddr, "ingressAddr", "localhost:8080", "local ingress port")
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
		}
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
