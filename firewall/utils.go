package firewall

import "net"

var globalListener = net.IPv4(0, 0, 0, 0)

func dedupExemptions(exemptions []Exemption) []Exemption {
	exemptionSet := make(map[[10]byte]Exemption)
	for _, exemption := range exemptions {
		exemptionSet[exemption.key()] = exemption
	}

	deduped := []Exemption{}
	for _, exemption := range exemptionSet {
		deduped = append(deduped, exemption)
	}
	return deduped
}

func dedupIngresses(ingresses []Ingress) []Ingress {
	ingressSet := make(map[[6]byte]Ingress)
	for _, ingress := range ingresses {
		ingressSet[ingress.key()] = ingress
	}

	deduped := []Ingress{}
	for _, ingress := range ingressSet {
		deduped = append(deduped, ingress)
	}
	return deduped
}

func dedupInterfaces(interfaces []net.Interface) []net.Interface {
	interfaceSet := make(map[int]net.Interface)
	for _, iface := range interfaces {
		interfaceSet[iface.Index] = iface
	}

	deduped := []net.Interface{}
	for _, iface := range interfaceSet {
		deduped = append(deduped, iface)
	}
	return deduped
}

func interfacesForIPv4(ip net.IP) ([]net.Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	if ip.Equal(globalListener) {
		return interfaces, nil
	}
	filtered := []net.Interface{}
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			if address, ok := addr.(*net.IPNet); ok {
				if ip.Equal(address.IP.To4()) {
					filtered = append(filtered, iface)
					break
				}
			}
		}
	}
	return filtered, nil
}
