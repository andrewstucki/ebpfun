package firewall

import "fmt"

const (
	droppedPacket uint32 = 0
)

type packetStats struct {
	Dropped uint64
}

func (s packetStats) String() string {
	return fmt.Sprintf("Packets dropped: %d", s.Dropped)
}

func readPacketCounter(objs *bpfObjects) (packetStats, error) {
	stats := packetStats{}
	if err := objs.bpfMaps.PacketCounter.Lookup(droppedPacket, &stats.Dropped); err != nil {
		return packetStats{}, err
	}
	return stats, nil
}
