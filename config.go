package main

import (
	"errors"
	"net"

	"github.com/andrewstucki/ebpfun/firewall"
)

type Ingress struct {
	Address string `hcl:"address"`
	Port    int    `hcl:"port"`
}

func parseIP(address string) (net.IP, error) {
	ip := net.ParseIP(address)
	if ip == nil || ip.To4() == nil {
		return nil, errors.New("invalid ip address")
	}
	return ip.To4(), nil
}

func (i Ingress) ToFirewall() (firewall.Ingress, error) {
	ip, err := parseIP(i.Address)
	if err != nil {
		return firewall.Ingress{}, err
	}

	return firewall.Ingress{
		Address: ip,
		Port:    i.Port,
	}, nil
}

type Exemption struct {
	Source      string `hcl:"source"`
	Destination string `hcl:"destination"`
	Port        int    `hcl:"port"`
}

func (e Exemption) ToFirewall() (firewall.Exemption, error) {
	sourceIP, err := parseIP(e.Source)
	if err != nil {
		return firewall.Exemption{}, err
	}
	destinationIP, err := parseIP(e.Destination)
	if err != nil {
		return firewall.Exemption{}, err
	}

	return firewall.Exemption{
		Source:      sourceIP,
		Destination: destinationIP,
		Port:        e.Port,
	}, nil
}

type Configuration struct {
	Ingresses  []Ingress   `hcl:"ingress,block"`
	Exemptions []Exemption `hcl:"exemption,block"`
}

func (c Configuration) ToFirewall() ([]firewall.Ingress, []firewall.Exemption, error) {
	ingresses := make([]firewall.Ingress, len(c.Ingresses))
	for i, ingress := range c.Ingresses {
		firewallIngress, err := ingress.ToFirewall()
		if err != nil {
			return nil, nil, err
		}
		ingresses[i] = firewallIngress
	}

	exemptions := make([]firewall.Exemption, len(c.Exemptions))
	for i, exemption := range c.Exemptions {
		firewallExemption, err := exemption.ToFirewall()
		if err != nil {
			return nil, nil, err
		}
		exemptions[i] = firewallExemption
	}

	return ingresses, exemptions, nil
}
