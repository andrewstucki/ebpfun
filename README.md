# XDP Skeleton

## Dependencies

**These instructions are intended for a Mac**. Some additional notes for Ubuntu along the way.  You'll need a recent LLVM installation that can target bpfeb and bpfel. By default the `Makefile` attempts to use an LLVM installation installed by Homebrew. If you don't have an installation at `/usr/local/opt/llvm/bin` you can run:

```bash
brew install llvm
```

```bash
sudo apt install llvm-10 clang-10
```

Aside from LLVM, you'll need at least go 1.17 and your PATH variable including the default location that `go install` writes to.

After installing go, install Cilium's bpf2go utility:

```bash
go install github.com/cilium/ebpf/cmd/bpf2go@latest
```

## Building

Once all dependencies are installed, run `make`. You can test the output binary on any of the Vagrant machines by running `sudo ./ebpfun -config config.hcl` in the `/vagrant` directory.

## Output

Since this program is just a dead-simple packet counter you can test the program output by running the default configuration and pinging your loopback interface. While running `sudo ping -f localhost` and `sudo ping -f ::1` you should see something like the following output:

```bash
vagrant@ubuntu-jammy:/vagrant$ sudo ./ebpfun -config config.hcl
2022/03/07 19:11:32 Packets received: IP - 26650, IPv6 - 0
2022/03/07 19:11:33 Packets received: IP - 100540, IPv6 - 0
2022/03/07 19:11:36 Packets received: IP - 100540, IPv6 - 101651
2022/03/07 19:11:37 Packets received: IP - 100540, IPv6 - 182532
```
