// +build ignore

#include "vmlinux.h"
#include "bpf_helpers.h"
#include "bpf_endian.h"

#define ETH_P_IP 0x0800
#define AF_INET 2

#define DROPPED_PACKET 0

#define MAX_MAP_SIZE 1024

// this is just static for now
#define PROXY_PORT 9090
// this maps to the loopback interface
#define PROXY_ADDRESS 16777343
#define PROXY_KEY 0

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

struct __attribute__((__packed__)) socket_tuple {
  __u32 address;
  __u32 port;
};

struct __attribute__((__packed__)) exemption {
  __be32 source;
  __be32 destination;
  __be16 port;
};

struct {
	__uint(type, BPF_MAP_TYPE_ARRAY);
	__type(key, u32);
	__type(value, u64);
	__uint(max_entries, 1);
} packet_counter SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_SOCKHASH);
	__type(key, u32);
	__type(value, int); // socket fd
	__uint(max_entries, 1); // only hold the upstream proxy
} proxy_socket SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__type(key, struct socket_tuple);
	__type(value, u8);
	__uint(max_entries, MAX_MAP_SIZE);
} marked_sockets SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__type(key, struct exemption);
	__type(value, u8);
	__uint(max_entries, MAX_MAP_SIZE);
} exemptions SEC(".maps");

struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__type(key, struct ingress);
	__type(value, u8);
	__uint(max_entries, MAX_MAP_SIZE);
} ingresses SEC(".maps");

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
int ingress_classifier(struct xdp_md *ctx) {
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

static __always_inline bool socket_marked(__u32 address, __u32 port) {
  struct socket_tuple key = {
    .address = address,
    .port = port,
  };
  u8 *marked = bpf_map_lookup_elem(&marked_sockets, &key);
  if (marked) {
    return true;
  }
  return false;
}

SEC("sk_lookup/dispatcher")
int dispatcher(struct bpf_sk_lookup *ctx) {
  if (ctx->family == AF_INET) {      
    if (ingress_is_tracked(ctx->local_ip4, bpf_ntohs(ctx->local_port))) {      
      if (!socket_marked(ctx->remote_ip4, bpf_ntohs(ctx->remote_port))) {
        __u32 key = PROXY_KEY;
        struct bpf_sock *proxy = bpf_map_lookup_elem(&proxy_socket, &key);
        if (proxy) {
          bpf_sk_assign(ctx, proxy, 0);
          bpf_sk_release(proxy);
        }
      }
    }
  }

  return SK_PASS;
}

SEC("sockops/sockmap")
int sockmap(struct bpf_sock_ops *ops) {
  struct bpf_sock *sk = ops->sk;
  if (sk) {
    if (sk->mark == 0xdeadbeef) {
      struct socket_tuple key = {
        .address = ops->local_ip4,
        .port = ops->local_port,
      };
      u8 value = 0;
      bpf_map_update_elem(&marked_sockets, &key, &value, BPF_ANY);
    }
  }

  if (ops->family != AF_INET) {
    return 0;
  }

  if (ops->local_ip4 == PROXY_ADDRESS && ops->local_port == PROXY_PORT) {    
    __u32 key = PROXY_KEY;
    switch (ops->op) {
    case BPF_SOCK_OPS_TCP_LISTEN_CB:
      bpf_sock_hash_update(ops, &proxy_socket, &key, BPF_NOEXIST);
    }
  }

	return 0;
}

char __license[] SEC("license") = "BSD-3-Clause";