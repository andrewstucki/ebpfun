module github.com/andrewstucki/ebpfun

go 1.17

require (
	github.com/cilium/ebpf v0.8.1
	github.com/hashicorp/hcl/v2 v2.11.1
)

require (
	github.com/agext/levenshtein v1.2.1 // indirect
	github.com/apparentlymart/go-textseg/v13 v13.0.0 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/mitchellh/go-wordwrap v0.0.0-20150314170334-ad45545899c7 // indirect
	github.com/vishvananda/netlink v1.1.0 // indirect
	github.com/vishvananda/netns v0.0.0-20191106174202-0a2b9b5464df // indirect
	github.com/zclconf/go-cty v1.8.0 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20210906170528-6f6e22806c34 // indirect
	golang.org/x/text v0.3.5 // indirect
)

replace github.com/cilium/ebpf => github.com/joamaki/ebpf v0.8.1-0.20220223162230-d928ec2d207a
