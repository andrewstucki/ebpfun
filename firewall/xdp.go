package firewall

import (
	"context"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
)

//go:generate bpf2go -strip $BPF_STRIP -cc $BPF_CLANG -cflags $BPF_CFLAGS bpf xdp.c -- -I../headers

var (
	// since we're working with a global eBPF module
	// these variables are used to track the global
	// state of the module
	updateMutex        sync.Mutex
	objects            bpfObjects
	attachedLinks      []link.Link
	attachedIngresses  []Ingress
	attachedExemptions []Exemption

	// this is a global channel where we send stat
	// updates
	Stats = make(chan PacketStats, 1)
)

func init() {
	if err := loadBpfObjects(&objects, nil); err != nil {
		log.Fatalf("error loading bpf program: %v", err)
	}
}

func Poll(ctx context.Context, timeout time.Duration) error {
	ticker := time.NewTicker(timeout)
	current := PacketStats{}

	for {
		select {
		case <-ticker.C:
			updated, err := readPacketCounter(&objects)
			if err != nil {
				return err
			}
			if updated != current {
				select {
				case Stats <- updated:
					current = updated
				default:
					// drop and pick up the stats update next time
				}
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func Update(ingresses []Ingress, exemptions []Exemption) error {
	updateMutex.Lock()
	defer updateMutex.Unlock()

	ingresses = dedupIngresses(ingresses)
	exemptions = dedupExemptions(exemptions)
	interfaces := []net.Interface{}
	for _, ingress := range ingresses {
		ingressInterfaces, err := ingress.Interfaces()
		if err != nil {
			return err
		}
		interfaces = append(interfaces, ingressInterfaces...)
	}
	interfaces = dedupInterfaces(interfaces)

	if err := detachAll(); err != nil {
		return err
	}

	return attach(ingresses, interfaces, exemptions)
}

func Cleanup() error {
	updateMutex.Lock()
	defer updateMutex.Unlock()

	return detachAll()
}

func detachAll() error {
	for _, link := range attachedLinks {
		if err := link.Close(); err != nil {
			return err
		}
	}
	attachedLinks = []link.Link{}

	// remove our watched ingresses
	existingIngresses := [][6]byte{}
	for _, ingress := range attachedIngresses {
		existingIngresses = append(existingIngresses, ingress.key())
	}
	if len(existingIngresses) > 0 {
		_, err := objects.Ingresses.BatchDelete(existingIngresses, &ebpf.BatchOptions{})
		if err != nil {
			return err
		}
	}
	attachedIngresses = []Ingress{}

	// remove our exemptions
	existingExemptions := [][10]byte{}
	for _, exemption := range attachedExemptions {
		existingExemptions = append(existingExemptions, exemption.key())
	}
	if len(existingExemptions) > 0 {
		_, err := objects.Exemptions.BatchDelete(existingExemptions, &ebpf.BatchOptions{})
		if err != nil {
			return err
		}
	}
	attachedExemptions = []Exemption{}

	return nil
}

func attach(ingresses []Ingress, interfaces []net.Interface, exemptions []Exemption) error {
	// add desired ingresses
	ingressKeys := [][6]byte{}
	ingressValues := []uint8{}
	for _, ingress := range ingresses {
		ingressKeys = append(ingressKeys, ingress.key())
		ingressValues = append(ingressValues, uint8(0))
	}
	if len(ingressKeys) > 0 {
		_, err := objects.Ingresses.BatchUpdate(ingressKeys, ingressValues, &ebpf.BatchOptions{})
		if err != nil {
			return err
		}
	}
	attachedIngresses = ingresses

	// add desired exemptions
	exemptionKeys := [][10]byte{}
	exemptionValues := []uint8{}
	for _, exemption := range exemptions {
		exemptionKeys = append(exemptionKeys, exemption.key())
		exemptionValues = append(exemptionValues, uint8(0))
	}
	if len(exemptionKeys) > 0 {
		_, err := objects.Exemptions.BatchUpdate(exemptionKeys, exemptionValues, &ebpf.BatchOptions{})
		if err != nil {
			return err
		}
	}
	attachedExemptions = exemptions

	// attach the bpf program to the desired interfaces
	for _, iface := range interfaces {
		xdp, err := link.AttachXDP(link.XDPOptions{
			Program:   objects.IngressClassifier,
			Interface: iface.Index,
		})
		if err != nil {
			return err
		}
		attachedLinks = append(attachedLinks, xdp)
	}

	// attach our sklookup program to our current network
	// namespace
	netns, err := os.Open("/proc/self/ns/net")
	if err != nil {
		return err
	}
	defer netns.Close()
	program, err := link.AttachNetNs(int(netns.Fd()), objects.Dispatcher)
	if err != nil {
		return err
	}
	attachedLinks = append(attachedLinks, program)

	// attach our sockmap program
	sockmap, err := link.AttachCgroup(link.CgroupOptions{
		Path:    "/sys/fs/cgroup",
		Attach:  ebpf.AttachCGroupSockOps,
		Program: objects.bpfPrograms.Sockmap,
	})
	if err != nil {
		return err
	}
	attachedLinks = append(attachedLinks, sockmap)

	return nil
}
