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
  consul_server_ip = "192.0.2.254"
  dashboard_server_ip = "192.0.2.10"
  counting_server_ip = "192.0.2.20"

  config.vm.box = "ubuntu/jammy64"
  #config.vm.box_version = "20210622.0.0"

  # Define a global configuration for all clients
  config.vm.cloud_init :user_data, content_type: "text/jinja2", path: "cloud-configs/common.cfg"

  # Define a basic Consul config which sets bind, advertise, and retry join
  # addresses
  config.vm.cloud_init :user_data, content_type: "text/cloud-config",
      inline: CLOUD_CONFIG_MERGE_SNIPPET + <<~EOF
        write_files:
          - path: /etc/consul.d/vagrant-config.hcl
            permissions: '0640'
            content: |
              bind_addr = "{{ GetAllInterfaces | include \\"rfc\\" \\"5735\\" | attr \\"address\\" }}"
              advertise_addr = "{{ GetAllInterfaces | include \\"rfc\\" \\"5735\\" | attr \\"address\\" }}"

              retry_join = ["#{consul_server_ip}"]

      EOF

  # This cloud-init user script performs a last-minute provisoning tasks which
  # are required to fully intialize the Consul agent.
  #
  # It must be run after various dependenices have been satisfied by the
  # cloud-init's provisioning, which is perfect because by convention user
  # scripts are executed after (almost) all previous script types have run.
  config.vm.cloud_init :user_data, content_type: "text/x-shellscript", path: "cloud-configs/finalize-consul-config.sh"

  # Provision a single-node Consul server cluster
  config.vm.define "consul-server" do |machine|
    machine.vm.hostname = "consul"
    machine.vm.network "private_network", ip: consul_server_ip

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

  # Create a client machine which registers a service running on port 80
  config.vm.define "dashboard" do |machine|
    machine.vm.hostname = "dashboard"
    machine.vm.network "private_network", ip: dashboard_server_ip
    machine = configureConsulServiceConfig(vmCfg: machine, port: 80)

    # Include custom configuration to install the dashboard service
    machine.vm.cloud_init :user_data, content_type: "text/cloud-config",
      inline: CLOUD_CONFIG_MERGE_SNIPPET + <<~EOF
        write_files:
          - path: /etc/systemd/system/dashboard.service
            permissions: '0640'
            content: |
              [Unit]
              Description=Dashboard service

              [Service]
              Type=simple
              Environment="COUNTING_SERVICE_URL=http://counting.service.consul"
              ExecStart=/usr/local/bin/dashboard-service
              Restart=on-failure
              RestartSec=5

              [Install]
              WantedBy=multi-user.target
          - path: /tmp/install-dashboard.sh
            permissions: '0750'
            content: |
              #!/usr/bin/env bash
              cd /tmp
              wget https://github.com/hashicorp/demo-consul-101/releases/download/0.0.3.1/dashboard-service_linux_amd64.zip
              unzip dashboard-service_linux_amd64.zip -d /usr/local/bin/

              cd /usr/local/bin
              mv dashboard-service_linux_amd64 dashboard-service
              chmod +x dashboard-service
        runcmd:
          - /tmp/install-dashboard.sh
          - systemctl enable dashboard.service
          - systemctl start dashboard.service
      EOF
  end

  # Create a client machine which registers a service running on port 80
  config.vm.define "counting" do |machine|
    machine.vm.hostname = "counting"
    machine.vm.network "private_network", ip: counting_server_ip
    machine = configureConsulServiceConfig(vmCfg: machine, port: 80)

    # Include custom configuration to install the counting service
    machine.vm.cloud_init :user_data, content_type: "text/cloud-config",
      inline: CLOUD_CONFIG_MERGE_SNIPPET + <<~EOF
        write_files:
          - path: /etc/systemd/system/counting.service
            permissions: '0640'
            content: |
              [Unit]
              Description=Counting service

              [Service]
              Type=simple
              ExecStart=/usr/local/bin/counting-service
              Restart=on-failure
              RestartSec=5

              [Install]
              WantedBy=multi-user.target
          - path: /tmp/install-counting.sh
            permissions: '0750'
            content: |
              #!/usr/bin/env bash
              cd /tmp
              wget https://github.com/hashicorp/demo-consul-101/releases/download/0.0.3.1/counting-service_linux_amd64.zip
              unzip counting-service_linux_amd64.zip -d /usr/local/bin/

              cd /usr/local/bin
              mv counting-service_linux_amd64 counting-service
              chmod +x counting-service
        runcmd:
          - /tmp/install-counting.sh
          - systemctl enable counting.service
          - systemctl start counting.service
        EOF
  end
end

# Configures a cloud-init configuration which creates a file containing basic
# Consul service information that is used by a later script to ultimately create
# the service registration file.
#
def configureConsulServiceConfig(vmCfg:, port:)
    vmCfg.vm.cloud_init :user_data, content_type: "text/cloud-config",
      inline: CLOUD_CONFIG_MERGE_SNIPPET + <<~EOF
        write_files:
          - path: /srv/consul/service-config.json
            permissions: '0640'
            content: |
              {
                "name": "#{vmCfg.vm.hostname}",
                "port": #{port.to_i}
              }
      EOF

      return vmCfg
end
