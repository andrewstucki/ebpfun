# XDP Skeleton

## Dependencies

**These instructions are intended for a Mac**. Some additional notes for Ubuntu along the way.  You'll need a recent LLVM installation that can target bpfeb and bpfel. By default the `Makefile` attempts to use an LLVM installation installed by Homebrew. If you don't have an installation at `/usr/local/opt/llvm/bin` you can run:

On Mac:

```bash
brew install llvm
```

On Ubuntu:

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

## Vagrant quick start

Start the Consul server with the following `vagrant up` command.

```bash
VAGRANT_EXPERIMENTAL="cloud_init,disks" vagrant up
```

After the instance has booted, you may safely start the remaining nodes in the
Vagrantfile using the aforementioned `vagrant up` command. The remaining node
names are:

* web
* api[1-3]

For example, in order to start the web and api1 service, run the following command.

```shell-session
VAGRANT_EXPERIMENTAL="cloud_init,disks" vagrant up web api1
```

Each of the nodes are configured to run
[nicholasjackson/fake-service](https://github.com/nicholasjackson/fake-service>).

Vagrant is configured to expose fake-service from the `web` node on
`localhost:9090`. The web service is configured to use the API services as an
upstream service. That is to say, any HTTP requests sent to the web service will
in turn trigger an upstream request to the API service. `web` will then assemble
and return the response bodies for *both* the API service, and web's own HTTP
server.

For more information on this Vagrant stack, see <https://github.com/blake/vagrant-consul-tproxy/tree/ubuntu22-update/examples/vagrant/prebuilt-image>.
