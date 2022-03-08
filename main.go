package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andrewstucki/ebpfun/firewall"

	"github.com/cilium/ebpf/rlimit"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"golang.org/x/sync/errgroup"
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

	group, ctx := errgroup.WithContext(ctx)

	// firewall.Update can be called any time to update the ingresses/exemptions live
	if err := firewall.Update(ingresses, exemptions); err != nil {
		log.Fatalf("error updating firewall configuration: %v", err)
	}
	defer firewall.Cleanup()

	group.Go(func() error {
		for {
			select {
			case stats := <-firewall.Stats:
				log.Println(stats)
			case <-ctx.Done():
				return nil
			}
		}
	})

	group.Go(func() error {
		return firewall.Poll(ctx, 1*time.Second)
	})

	if err := group.Wait(); err != nil {
		log.Fatalf("error: %v", err)
	}
}
