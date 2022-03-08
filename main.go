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
)

func init() {
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

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

	// firewall.Update can be called any time to update the ingresses/exemptions live
	if err := firewall.Update(ingresses, exemptions); err != nil {
		log.Fatalf("error updating firewall configuration: %v", err)
	}
	defer firewall.Cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	errors := make(chan error, 1)
	go func() {
		errors <- firewall.Poll(ctx, 1*time.Second)
	}()

	<-stop
	cancel()

	if err := <-errors; err != nil {
		log.Fatalf("error linking XDP program: %v", err)
	}
}
