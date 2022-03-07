package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/andrewstucki/ebpfun/firewall"

	"github.com/cilium/ebpf/rlimit"
	"github.com/hashicorp/hcl/v2/hclsimple"
)

func init() {
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatal(err)
	}
}

type Configuration struct {
	Interfaces []string `hcl:"interfaces"`
}

func (c *Configuration) FilteredInterfaces() ([]net.Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	filtered := []net.Interface{}
	for _, iface := range interfaces {
		for _, configured := range c.Interfaces {
			if iface.Name == configured {
				filtered = append(filtered, iface)
				break
			}
		}
	}
	return filtered, nil
}

func main() {
	stop := make(chan os.Signal)
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

	interfaces, err := config.FilteredInterfaces()
	if err != nil {
		log.Fatalf("unable to configure XDP interfaces: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	errors := make(chan error, 1)
	go func() {
		errors <- firewall.Start(ctx, interfaces)
	}()

	<-stop
	cancel()

	if err := <-errors; err != nil {
		log.Fatalf("error linking XDP program: %v", err)
	}
}
