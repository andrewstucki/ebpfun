data_dir = "/tmp/consul/client"

server         = false
advertise_addr = "{{ GetInterfaceIP `enp0s8` }}"
bind_addr      = "0.0.0.0"
client_addr    = "0.0.0.0"
retry_join     = ["$CONSUL_HTTP_ADDR"]

datacenter = "dc1"

ports {
  grpc = 8502
}

telemetry {
  prometheus_retention_time = "24h"
  disable_hostname          = true
}
