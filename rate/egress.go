package rate

import (
	"fmt"
	"net"
)

type Egress struct {
	Address net.IP
	Port    int
	Rate    int
}

func (e *Egress) String() string {
	return fmt.Sprintf(
		"%s:%d",
		e.Address, e.Port,
	)
}
