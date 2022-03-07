package firewall

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/cilium/ebpf/link"
)

//go:generate bpf2go -strip $BPF_STRIP -cc $BPF_CLANG -cflags $BPF_CFLAGS bpf xdp.c -- -I./headers

const (
	ipPacket   uint32 = 0
	ipv6Packet uint32 = 1
)

type packetStats struct {
	IP   uint64
	IPv6 uint64
}

func (s packetStats) String() string {
	return fmt.Sprintf("Packets received: IP - %d, IPv6 - %d", s.IP, s.IPv6)
}

func Start(ctx context.Context, interfaces []net.Interface) error {
	objs := bpfObjects{}
	if err := loadBpfObjects(&objs, nil); err != nil {
		return err
	}

	for _, iface := range interfaces {
		xdp, err := link.AttachXDP(link.XDPOptions{
			Program:   objs.Classifier,
			Interface: iface.Index,
		})
		if err != nil {
			return err
		}
		defer xdp.Close()
	}

	ticker := time.NewTicker(1 * time.Second)
	current := packetStats{}

	for {
		select {
		case <-ticker.C:
			stats, err := readPacketCounter(&objs)
			if err != nil {
				return err
			}
			if stats != current {
				log.Println(stats.String())
				current = stats
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func readPacketCounter(objs *bpfObjects) (packetStats, error) {
	stats := packetStats{}
	if err := objs.bpfMaps.PacketCounter.Lookup(ipPacket, &stats.IP); err != nil {
		return packetStats{}, err
	}
	if err := objs.bpfMaps.PacketCounter.Lookup(ipv6Packet, &stats.IPv6); err != nil {
		return packetStats{}, err
	}
	return stats, nil
}
