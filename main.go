package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/andrewstucki/ebpfun/firewall"

	"github.com/cilium/ebpf/rlimit"
	"github.com/hashicorp/hcl/v2/hclsimple"
)

func init() {
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	var configFile string

	flag.StringVar(&configFile, "config", "", "source of configuration")

	flag.Parse()

	if configFile == "" {
		log.Fatal("-config flag must be specified")
	}

	config := &Configuration{}
	err := hclsimple.DecodeFile(configFile, nil, config)
	if err != nil {
		log.Fatalf("error reading configuration file: %v", err)
	}
	ingresses, exemptions, err := config.ToFirewall()
	if err != nil {
		log.Fatalf("error parsing configuration: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// firewall.Update can be called any time to update the ingresses/exemptions live
	if err := firewall.Update(ingresses, exemptions); err != nil {
		log.Fatalf("error updating firewall configuration: %v", err)
	}
	defer firewall.Cleanup()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case stats := <-firewall.Stats:
				log.Println(stats)
			case <-ctx.Done():
				return
			}
		}
	}()

	if err := firewall.Poll(ctx, 1*time.Second); err != nil {
		log.Fatalf("error linking XDP program: %v", err)
	}

	wg.Wait()
}
