# eBPFun

## Dependencies

**These instructions are for a Mac**. You'll need a recent LLVM installation that can target bpfeb and bpfel. By default the `Makefile` attempts to use an LLVM installation installed by Homebrew. If you don't have an installation at `/usr/local/opt/llvm/bin` you can run:

```bash
brew install llvm
```

Aside from LLVM, you'll need at least go 1.17 and your PATH variable including the default location that `go install` writes to.

After installing go, install Cilium's bpf2go utility:

```bash
go install github.com/cilium/ebpf/cmd/bpf2go@latest
```

## Building

Once all dependencies are installed, run `make`. You can test the output binary on any of the Vagrant machines by running `sudo ./ebpfun -config config.hcl` in the `/vagrant` directory.

## Output

While running `yes | nc -u 127.0.0.1 81` with the configuration file at the top level of this repo you should see something like the following output:

```bash
vagrant@ubuntu-jammy:/vagrant$ sudo ./ebpfun -config config.hcl
2022/03/08 03:32:04 Packets dropped: 46323
2022/03/08 03:32:05 Packets dropped: 142811
2022/03/08 03:32:06 Packets dropped: 227929
2022/03/08 03:32:07 Packets dropped: 321104
2022/03/08 03:32:08 Packets dropped: 409536
2022/03/08 03:32:09 Packets dropped: 505245
2022/03/08 03:32:10 Packets dropped: 595287
2022/03/08 03:32:11 Packets dropped: 636769
```