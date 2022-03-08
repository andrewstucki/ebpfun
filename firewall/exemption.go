package firewall

import (
	"encoding/binary"
	"fmt"
	"net"
)

type Exemption struct {
	Destination net.IP
	Port        int
	Source      net.IP
}

func (s *Exemption) String() string {
	return fmt.Sprintf(
		"%s --> %s:%d",
		s.Source, s.Destination, s.Port,
	)
}

func (r *Exemption) key() [10]byte {
	key := [10]byte{}
	destinationIP := r.Destination.To4()
	sourceIP := r.Source.To4()
	binary.BigEndian.PutUint32(key[0:4], binary.BigEndian.Uint32(sourceIP))
	binary.BigEndian.PutUint32(key[4:8], binary.BigEndian.Uint32(destinationIP))
	binary.BigEndian.PutUint16(key[8:10], uint16(r.Port))
	return key
}
