LLVM_PATH ?= /usr/local/opt/llvm/bin

CLANG ?= $(LLVM_PATH)/clang
STRIP ?= $(LLVM_PATH)/llvm-strip
CFLAGS := -O2 -g -Wall -Werror $(CFLAGS)
GOOS := linux
GOLDFLAGS := -s -w

firewall/bpf_bpfel.go: export BPF_STRIP := $(STRIP)
firewall/bpf_bpfel.go: export BPF_CLANG := $(CLANG)
firewall/bpf_bpfel.go: export BPF_CFLAGS := $(CFLAGS)
firewall/bpf_bpfel.go: firewall/xdp.c
	go generate ./...

.PHONY: generate
generate: firewall/bpf_bpfel.go

ebpfun: export GOOS := $(GOOS)
ebpfun: generate
	go build -ldflags "$(GOLDFLAGS)"

clean:
	@rm -f firewall/bpf_* ebpfun

.DEFAULT_GOAL := ebpfun