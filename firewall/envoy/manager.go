package envoy

import (
	"bytes"
	"context"
	"log"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

var (
	bootstrapTemplate   = template.New("bootstrap")
	filterChainTemplate = template.New("chain")
)

func init() {
	_, err := bootstrapTemplate.Parse(bootstrapJSONTemplate)
	if err != nil {
		log.Fatal(err)
	}
	_, err = filterChainTemplate.Parse(filterChainJSONTemplate)
	if err != nil {
		log.Fatal(err)
	}
}

type Manager struct {
	BootstrapFilePath string
}

func NewManager() (*Manager, error) {
	file, err := os.CreateTemp("", "envoy.json")
	if err != nil {
		return nil, err
	}
	path := file.Name()
	file.Close()

	return &Manager{
		BootstrapFilePath: path,
	}, nil
}

func (m *Manager) Cleanup() {
	os.Remove(m.BootstrapFilePath)
}

func (m *Manager) Run(ctx context.Context, rules []EnvoyRule) error {
	if len(rules) == 0 {
		return nil
	}

	if err := m.render(rules); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/envoy", "-c", m.BootstrapFilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	err := cmd.Run()

	// Turns out that when a command spawned with CommandContext
	// has its context canceled, rather than returnng an error about
	// canceled context, it sends an error about the spawned process
	// being killed. This select allows us to effectively check whether
	// or not the process stopped because the context was canceled even
	// if the cancellation error isn't propagated up to us.
	select {
	case <-ctx.Done():
		return nil
	default:
		return err
	}
}

type EnvoyRule struct {
	Address string
	Port    int
	Header  string
}

func (m *Manager) render(rules []EnvoyRule) error {
	chains := []string{}
	for _, rule := range rules {
		var buffer bytes.Buffer
		if err := filterChainTemplate.Execute(&buffer, rule); err != nil {
			return err
		}
		chains = append(chains, buffer.String())
	}
	var buffer bytes.Buffer
	if err := bootstrapTemplate.Execute(&buffer, strings.Join(chains, ",")); err != nil {
		return err
	}
	return os.WriteFile(m.BootstrapFilePath, buffer.Bytes(), 0600)
}

const filterChainJSONTemplate = `
{
	"filter_chain_match": {
		"prefixRanges":  [
			{
				"addressPrefix": "{{ .Address }}",
				"prefixLen": 32
			}
		],
		"destination_port": {{ .Port }}
	},
	"filters": [
		 {
				"name": "envoy.filters.network.http_connection_manager",
				"typed_config": {
					 "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
					 "stat_prefix": "ingress_http",
					 "codec_type": "AUTO",
					 "route_config": {
							"name": "default_route",
							"virtual_hosts": [
								 {
										"name": "local_service",
										"domains": [
											 "*"
										],
										"routes": [
											 {
													"match": {
														 "prefix": "/"
													},
													"route": {
														 "cluster": "original_destination"
													}
											 }
										]
								 }
							]
					 },
					 "http_filters": [
							{
								 "name": "envoy.filters.http.original_src",
								 "typed_config": {
										"@type": "type.googleapis.com/envoy.extensions.filters.http.original_src.v3.OriginalSrc",
										"mark": 3735928559
								 }
							},
							{
								"name": "envoy.filters.http.rbac",
								"typedConfig": {
									"@type": "type.googleapis.com/envoy.extensions.filters.http.rbac.v3.RBAC",
									"rules": {
										"policies": {
											"l7": {
												"permissions": [
													{
														"header": {
															"name": "{{ .Header }}",
															"present_match": true
														}
													}
												],
												"principals": {
													"any": true
												}
											}
										}
									}
								}
							},
							{
								 "name": "envoy.filters.http.router"
							}
					 ]
				}
		 }
	]
}`

const bootstrapJSONTemplate = `
{
	"static_resources": {
		 "listeners": [
				{
					 "name": "ingress",
					 "address": {
							"socket_address": {
								 "address": "127.0.0.1",
								 "port_value": 9090
							}
					 },
					 "listener_filters": [
							{
									"name": "envoy.filters.listener.original_dst",
									"typed_config": {}
							}
					 ],
					 "filter_chains": [{{ . }}]
				}
		 ],
		 "clusters": [
				{
					 "name": "original_destination",
					 "type": "ORIGINAL_DST",
					 "connect_timeout": "6s",
					 "lb_policy": "CLUSTER_PROVIDED"
				}
		 ]
	}
}
`
