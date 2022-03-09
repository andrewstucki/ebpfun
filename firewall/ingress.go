package firewall

import (
	"encoding/binary"
	"fmt"
	"net"
)

type Ingress struct {
	Address net.IP
	Port    int
	HTTP    bool
}

func (i *Ingress) String() string {
	return fmt.Sprintf(
		"%s:%d",
		i.Address, i.Port,
	)
}

func (i *Ingress) Interfaces() ([]net.Interface, error) {
	return interfacesForIPv4(i.Address)
}

func (i *Ingress) key() [6]byte {
	key := [6]byte{}
	binary.BigEndian.PutUint32(key[0:4], binary.BigEndian.Uint32(i.Address.To4()))
	binary.BigEndian.PutUint16(key[4:6], uint16(i.Port))
	return key
}
