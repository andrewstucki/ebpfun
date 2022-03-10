# Demo

Spin up the upstreams.
```bash
./upstreams
```

Show ability to route to each service.

```bash
curl --silent localhost:8000 | grep server
curl --silent localhost:8001 | grep server
curl --silent localhost:8002 | grep server
curl --silent localhost:8003 | grep server
socat -t 0.1 - tcp:127.0.0.1:8004
echo | socat -t 0.1 - udp:127.0.0.1:8005
```

Boot up `ebpfun`.

```bash
sudo ./ebpfun -config config.hcl
```

Demonstrate no `iptables` rules.
```bash
sudo iptables -S
```

Test out service 1.
```bash
curl --silent localhost:8000
curl --silent localhost:8000 -H "x-foo: 1" | grep server
curl localhost:8000 -H "x-foo: 1" --head
```

Test out service 2.
```bash
curl --silent localhost:8001
curl --silent localhost:8001 -H "x-foo: 1"
curl --silent localhost:8001 -H "x-bar: 1" | grep server
curl localhost:8001 -H "x-bar: 1" --head
```

Test out service 3.
```bash
curl --silent localhost:8002 | grep server
curl localhost:8002 --head
```