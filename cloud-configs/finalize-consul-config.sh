#!/usr/bin/env bash

# Exit upon receiving any errors
set -o errexit

# Create service-registration.json from template if service config is present
cd /srv/consul
if [[ -f service-config.json ]]; then
  # Merge the user-provided service registration information with the default
  # Connect service stanza
  SERVICE_REGISTRATION=$(jq --slurpfile s service-config.json --monochrome-output '$s[0] + .service | {service: .}' service-template.json)

  if [[ -n "${SERVICE_REGISTRATION}" ]]; then
    echo "${SERVICE_REGISTRATION}" > /etc/consul.d/service-registration.json

    # Enable Envoy proxy for registered service
    PROXY_ID=$(jq --raw-output .service.name /etc/consul.d/service-registration.json)
    systemctl enable envoy@${PROXY_ID}
  fi
fi

# Send .consul DNS queries directly to Consul agent if systemd version is
# 246 or greater.
#
# Otherwise, redirection will happen via iptables rules which
# are installed in /usr/local/bin/consul-redirect-traffic.
SYSTEMD_VERSION=$(systemd --version | awk 'NR==1 { print $2 }')
if [[ ${SYSTEMD_VERSION} -ge 246 ]]; then
  pushd /etc/systemd/resolved.conf.d/
  sed --in-place 's/^\(DNS=127.0.0.1\)$/\1:8600/' consul.conf
  popd
fi

# Restart systemd-resolved to pick up the addition of the new resolved config
systemctl restart systemd-resolved

# Ensure Consul user owns files in config directory
chown --recursive consul:consul /etc/consul.d/

# Start Consul
systemctl start --no-block consul.service

if [[ -n $PROXY_ID ]]; then
  # Start the Envoy proxy
  systemctl start --no-block envoy@${PROXY_ID}
fi
