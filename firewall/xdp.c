// +build ignore

#include "vmlinux.h"
#include "bpf_helpers.h"
#include "bpf_endian.h"

#define ETH_P_IP 0x0800

#define DROPPED_PACKET 0

#define MAX_MAP_SIZE 1024

#define ensure_size(packet, value)                                             \
  ({                                                                           \
    if ((void *)((void *)packet->current + value) > (void *)packet->end)       \
      return -1;                                                               \
  })

struct packet_parser {
  void *current;
  void *end;
};

struct __attribute__((__packed__)) ingress {
  __be32 address;
  __be16 port;
};

struct __attribute__((__packed__)) exemption {
  __be32 source;
  __be32 destination;
  __be16 port;
};

struct bpf_map_def SEC("maps") packet_counter = {
  .type = BPF_MAP_TYPE_ARRAY,
  .key_size = sizeof(u32),
  .value_size = sizeof(u64),
  .max_entries = 1,
};

struct bpf_map_def SEC("maps") exemptions = {
  .type = BPF_MAP_TYPE_HASH,
  .key_size = sizeof(struct exemption),
  .value_size = sizeof(u8),
  .max_entries = MAX_MAP_SIZE,
};

struct bpf_map_def SEC("maps") ingresses = {
  .type = BPF_MAP_TYPE_HASH,
  .key_size = sizeof(struct ingress),
  .value_size = sizeof(u8),
  .max_entries = MAX_MAP_SIZE,
};

static __always_inline void count(u32 version) {
  u64 default_value = 1;
  u32 version_key = version;
  u64 *value = bpf_map_lookup_elem(&packet_counter, &version_key);
  if (value) {
    __sync_fetch_and_add(value, 1);
  } else {
    bpf_map_update_elem(&packet_counter, &version_key, &default_value, BPF_ANY);
  }
}

static __always_inline bool ingress_is_tracked(__be32 address, __be16 port) {
  struct ingress key = {
    .address = address,
    .port = port,
  };
  if (bpf_map_lookup_elem(&ingresses, &key)) {
    return true;
  }
  return false;
}

static __always_inline bool has_exemption(__be32 source, __be32 destination, __be16 port) {
  struct exemption key = {
    .source = source,
    .destination = destination,
    .port = port,
  };

  if (bpf_map_lookup_elem(&exemptions, &key)) {
    return true;
  }
  return false;
}

static __always_inline int maybe_drop(__be32 source, __be32 destination, __be16 port) {  
  // are we actively tracking this destination?
  if (ingress_is_tracked(destination, port)) {
    // if we are, check if we have an exemption
    if (!has_exemption(source, destination, port)) {
      // if we don't have an exemption, drop the packet
      count(DROPPED_PACKET);
      return XDP_DROP;
    }
  }
  return XDP_PASS;
}

static __always_inline int parse_ethernet(struct packet_parser *packet, struct ethhdr **eth) {
  ensure_size(packet, sizeof(struct ethhdr));

  *eth = packet->current;
  packet->current += sizeof(struct ethhdr);
  return 0;
}

static __always_inline int parse_ip(struct packet_parser *packet, struct iphdr **ip) {
  ensure_size(packet, sizeof(struct iphdr));

  struct iphdr *header = (struct iphdr *)packet->current;
  u32 header_size = header->ihl << 2;
  if (header_size < sizeof(struct iphdr)) {
    return -1;
  }

  ensure_size(packet, header_size);
  packet->current += header_size;
  *ip = header;
  return 0;
}

static __always_inline int parse_tcp(struct packet_parser *packet, struct tcphdr **tcp) {
  ensure_size(packet, sizeof(struct tcphdr));

  struct tcphdr *header = (struct tcphdr *)packet->current;
  u32 offset = header->doff << 2;
  if (offset < sizeof(struct tcphdr)) {
    return -1;
  }

  ensure_size(packet, offset);
  packet->current += offset;
  *tcp = header;
  return 0;
}

static __always_inline int parse_udp(struct packet_parser *packet, struct udphdr **udp) {
  ensure_size(packet, sizeof(struct udphdr));

  *udp = (struct udphdr *)packet->current;
  packet->current += sizeof(struct udphdr);
  return 0;
}

static __always_inline int classify_ip(struct packet_parser *packet) {
  struct iphdr *ip;
  struct tcphdr *tcp;
  struct udphdr *udp;
  
  if (!parse_ip(packet, &ip)) {
    switch (ip->protocol) {
    case IPPROTO_UDP:
      if (!parse_udp(packet, &udp)) {
        return maybe_drop(ip->saddr, ip->daddr, udp->dest);
      }
      break;
    case IPPROTO_TCP:
      if (!parse_tcp(packet, &tcp)) {
        return maybe_drop(ip->saddr, ip->daddr, tcp->dest);
      }
      break;
    }
  }
  return XDP_PASS;
}

SEC("xdp_classifier")
int classifier(struct xdp_md *ctx) {
  struct packet_parser packet = {
      .current = (void *)(long)ctx->data,
      .end = (void *)(long)ctx->data_end,
  };

  struct ethhdr *eth;
  if (!parse_ethernet(&packet, &eth)) {
    if (eth->h_proto == bpf_htons(ETH_P_IP)) {
      return classify_ip(&packet);
    }
  }

  return XDP_PASS;
}

char __license[] SEC("license") = "BSD-3-Clause";