// +build ignore

#include "vmlinux.h"
#include "bpf_helpers.h"
#include "bpf_endian.h"

#define ETH_P_IP 0x0800
#define ETH_P_IPV6 0x86DD

#define IP_PACKET 0
#define IPV6_PACKET 1

#define ensure_size(packet, value)                                             \
  ({                                                                           \
    if ((void *)((void *)packet->current + value) > (void *)packet->end)       \
      return -1;                                                               \
  })

struct bpf_map_def SEC("maps") packet_counter = {
  .type = BPF_MAP_TYPE_ARRAY,
  .key_size = sizeof(u32),
  .value_size = sizeof(u64),
  .max_entries = 2,
};

struct packet_parser {
  void *current;
  void *end;
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

static __always_inline int parse_ip6(struct packet_parser *packet, struct ipv6hdr **ip) {
  ensure_size(packet, sizeof(struct ipv6hdr));

  *ip = packet->current;
  packet->current += sizeof(struct ipv6hdr);
  return 0;
}

static __always_inline int classify_ip(struct packet_parser *packet) {
  struct iphdr *ip;

  if (!parse_ip(packet, &ip)) {
    count(IP_PACKET);

    // TODO: fill in
    return XDP_PASS;
  }
  return XDP_PASS;
}

static __always_inline int classify_ip6(struct packet_parser *packet) {
  struct ipv6hdr *ip;
  
  if (!parse_ip6(packet, &ip)) {
    count(IPV6_PACKET);

    // TODO: fill in
    return XDP_PASS;
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
    switch (eth->h_proto) {
    case bpf_htons(ETH_P_IP):
      return classify_ip(&packet);
    case bpf_htons(ETH_P_IPV6):
      return classify_ip6(&packet);
    }
  }

  return XDP_PASS;
}

char __license[] SEC("license") = "BSD-3-Clause";