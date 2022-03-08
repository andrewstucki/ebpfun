package firewall

import "fmt"

const (
	droppedPacket uint32 = 0
)

type PacketStats struct {
	Dropped uint64
}

func (s PacketStats) String() string {
	return fmt.Sprintf("Packets dropped: %d", s.Dropped)
}

func readPacketCounter(objs *bpfObjects) (PacketStats, error) {
	stats := PacketStats{}
	if err := objs.bpfMaps.PacketCounter.Lookup(droppedPacket, &stats.Dropped); err != nil {
		return PacketStats{}, err
	}
	return stats, nil
}
