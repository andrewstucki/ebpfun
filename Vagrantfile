require 'json'

VAGRANT_CONFIGURATION_VERSION = 2

CLOUD_CONFIG_MERGE_SNIPPET = <<-EOF
#cloud-config

# Necessary to merge multiple entries across cloud config files
merge_how:
  - name: list
    settings: [append]
  - name: dict
    settings: [no_replace, recurse_list]
EOF

Vagrant.configure(VAGRANT_CONFIGURATION_VERSION) do |config|

  # Static IPs assigned to the virtual machines
  machine_ips = {
    "consul-server" => "192.168.56.254",
    "web"           => "192.168.56.10",
    "api1"          => "192.168.56.20",
    "api2"          => "192.168.56.21",
    "api3"          => "192.168.56.22",
  }

  config.vm.box = "blakec/ubuntu-22.04-consul-transparent-proxy"

  # Define a basic Consul config which sets bind, advertise, and retry join
  # addresses
  config.vm.cloud_init :user_data, content_type: "text/cloud-config",
      inline: CLOUD_CONFIG_MERGE_SNIPPET + <<~EOF
        write_files:
          - path: /etc/consul.d/vagrant-config.hcl
            permissions: '0640'
            owner: consul:consul
            content: |
              bind_addr = "{{ GetAllInterfaces | include \\"network\\" \\"192.168.56.0/24\\" | attr \\"address\\" }}"
              advertise_addr = "{{ GetAllInterfaces | include \\"network\\" \\"192.168.56.0/24\\" | attr \\"address\\" }}"
              client_addr = "127.0.0.1"

              retry_join = ["#{machine_ips['consul-server']}"]

      EOF

  # Provision a single-node Consul server cluster
  config.vm.define "consul-server" do |machine|
    machine.vm.hostname = "consul"
    machine.vm.network "private_network", ip: machine_ips["consul-server"]

    # Forward port 8500 to this server
    machine.vm.network "forwarded_port", guest: 8500, host: 8500

    # Restart Consul the machine is up to pick up the latest configuration files
    # installed by cloud-init.
    machine.vm.provision "shell", inline: "systemctl restart consul.service"

    # Include custom configuration to make this agent operate as a server
    machine.vm.cloud_init :user_data, content_type: "text/cloud-config",
      inline: CLOUD_CONFIG_MERGE_SNIPPET + <<~EOF
        write_files:
          - path: /etc/consul.d/server.hcl
            permissions: '0640'
            owner: consul:consul
            content: |
              server = true
              bootstrap = true
              bootstrap_expect = 1
              addresses {
                http = "0.0.0.0"
              }
              ui_config {
                enabled = true
              }
      EOF
  end

  # Create a client machine with has a sidecar proxy
  config.vm.define "web", autostart: false do |machine|
    machine.vm.hostname = "web"
    machine.vm.network "private_network", ip: machine_ips["web"]

    # Forward port 9090 to this server
    machine.vm.network "forwarded_port", guest: 9090, host: 9090

    service_config = {
      annotations: {
        "consul.hashicorp.com/connect-service": machine.vm.hostname,
        "consul.hashicorp.com/transparent-proxy": true,
        "consul.hashicorp.com/transparent-proxy-exclude-inbound-ports": "22,9090",

        "consul.hashicorp.com/service-tags": "web,nonprod",
        "consul.hashicorp.com/service-meta-environment": "test",
      }
    }

    machine = configureConsulServiceConfig(vmCfg: machine, config: service_config)

    machine = install_fake_service(vmCfg: machine)

    # Include custom configuration for fake service
    machine.vm.cloud_init :user_data, content_type: "text/cloud-config",
      inline: CLOUD_CONFIG_MERGE_SNIPPET + <<~EOF
        write_files:
          - path: /srv/consul/fake-service.env
            permissions: '0640'
            content: |
              NAME="Web"
              MESSAGE="I am a web server. *Beep boop bop*"
              UPSTREAM_URIS="http://api.virtual.consul"
        EOF
  end

  # Create a client machine which registers a service running on port 9091
  (1..3).each do |i|
    config.vm.define "api#{i}", autostart: false do |machine|
      machine.vm.hostname = "api#{i}"
      machine.vm.network "private_network", ip: machine_ips["api#{i}"]
      service_config = {
        annotations: {
          "consul.hashicorp.com/connect-service": "api",
          "consul.hashicorp.com/transparent-proxy": true,
          "consul.hashicorp.com/transparent-proxy-exclude-inbound-ports": 22,

          "consul.hashicorp.com/connect-service-port": "9090",
          "consul.hashicorp.com/service-tags": "api#{i},nonprod",
          "consul.hashicorp.com/service-meta-version": "v#{i}",
          "consul.hashicorp.com/service-meta-environment": "test",
        }
      }
      machine = configureConsulServiceConfig(vmCfg: machine, config: service_config)

      machine = install_fake_service(vmCfg: machine)

      # Include custom configuration for fake service
      machine.vm.cloud_init :user_data, content_type: "text/cloud-config",
        inline: CLOUD_CONFIG_MERGE_SNIPPET + <<~EOF
          write_files:
            - path: /srv/consul/fake-service.env
              permissions: '0640'
              content: |
                NAME="API #{i}"
                MESSAGE="Greetings from API server #{i}!"
                ERROR_RATE=#{(i.to_f * 10) / 100 % 0.3}
          EOF
    end
  end
end

# Configures a cloud-init configuration which creates a file containing basic
# Consul service information that is used by a later script to ultimately create
# the service registration file.
#
def configureConsulServiceConfig(vmCfg:, config:)
    json_config = JSON.pretty_generate(config)

    yaml_config = {
      "write_files" => [
        {
            "path" => "/srv/consul/service-config.json",
            "permissions" => "0640",
            "content" => json_config
        }
      ]
    }
    vmCfg.vm.cloud_init :user_data, content_type: "text/cloud-config",
      inline: CLOUD_CONFIG_MERGE_SNIPPET + yaml_config.to_yaml.sub("---", "")

      return vmCfg
end

# Install fake-service on the virtual machine
def install_fake_service(vmCfg:)
    vmCfg.vm.cloud_init :user_data, content_type: "text/cloud-config",
      inline: CLOUD_CONFIG_MERGE_SNIPPET + <<~EOF
        write_files:
          - path: /etc/systemd/system/fake-service.service
            permissions: '0640'
            content: |
              [Unit]
              Description=Fake service

              [Service]
              Type=simple
              EnvironmentFile=/srv/consul/fake-service.env
              ExecStart=/usr/local/bin/fake-service
              Restart=on-failure
              RestartSec=5

              [Install]
              WantedBy=multi-user.target
          - path: /tmp/install-fake-service.sh
            permissions: '0750'
            content: |
              #!/usr/bin/env bash
              cd /tmp
              wget https://github.com/nicholasjackson/fake-service/releases/download/v0.22.7/fake_service_linux_amd64.zip
              unzip fake_service_linux_amd64.zip -d /usr/local/bin/
              chmod +x /usr/local/bin/fake-service
        packages:
          - unzip
        runcmd:
          - /tmp/install-fake-service.sh
          - systemctl enable fake-service.service
          - systemctl start fake-service.service
      EOF

    return vmCfg
end
