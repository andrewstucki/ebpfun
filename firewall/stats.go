package firewall

import "fmt"

const (
	droppedPacket uint32 = 0
)

type Stats struct {
	Dropped uint64
}

func (s Stats) String() string {
	return fmt.Sprintf("Packets dropped: %d", s.Dropped)
}

func readPacketCounter(objs *bpfObjects) (Stats, error) {
	stats := Stats{}
	if err := objs.bpfMaps.PacketCounter.Lookup(droppedPacket, &stats.Dropped); err != nil {
		return Stats{}, err
	}
	return stats, nil
}
