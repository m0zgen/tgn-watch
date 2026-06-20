# tgn-watch

`tgn-watch` is a lightweight YAML-based monitoring agent for Linux/server infrastructure.
It checks HTTP endpoints, TCP ports, systemd services, disk usage, memory usage, and TLS certificate expiration, then sends alerts and recovery notifications through `tgn-relay`.

It is designed as a small alternative/complement to Monit or Uptime Kuma for simple production nodes and DNS PoPs.

## Features

- HTTP checks
- TCP checks
- systemd service checks
- disk usage checks
- Linux memory checks via `/proc/meminfo`
- TLS certificate expiration checks
- alert on failure
- recovery notification
- dedup window to avoid Telegram spam
- per-check severity and group
- safe logs without secrets
- systemd unit example

## Quick start

Start `tgn-relay` first, then run:

```bash
go mod tidy
go run ./cmd/tgn-watch -config configs/config.example.yml
```

Build binary:

```bash
make build
./bin/tgn-watch -config configs/config.example.yml
```

## Test notification flow

Example check config points to local `tgn-relay`:

```yaml
relay:
  endpoint: "http://127.0.0.1:8080/api/v1/send"
  key: "change-me-super-secret"
  default_group: "monitoring"
  timeout: "3s"
```

The group must exist in `tgn-relay` config.

## Config example

```yaml
relay:
  endpoint: "http://127.0.0.1:8080/api/v1/send"
  key: "change-me-super-secret"
  default_group: "monitoring"
  timeout: "3s"

watcher:
  interval: "30s"
  notify_on_recovery: true
  dedup_window: "10m"
  hostname: ""

checks:
  - name: "tgn-relay local health"
    type: "http"
    url: "http://127.0.0.1:8080/healthz"
    expect_status: 200
    timeout: "3s"
    group: "monitoring"
    severity: "critical"
```

## Check types

### HTTP

```yaml
- name: "Website"
  type: "http"
  url: "https://openbld.net/"
  expect_status: 200
  timeout: "5s"
```

Optional fields:

```yaml
expect_status_min: 200
expect_status_max: 399
expect_contains: "OpenBLD"
```

### TCP

```yaml
- name: "DNS TCP"
  type: "tcp"
  target: "127.0.0.1:53"
  timeout: "2s"
```

### systemd

```yaml
- name: "zBLD service"
  type: "systemd"
  service: "zbld.service"
  timeout: "3s"
```

### Disk

```yaml
- name: "Root disk"
  type: "disk"
  path: "/"
  max_used_percent: 90
```

### Memory

```yaml
- name: "Memory usage"
  type: "memory"
  max_used_percent: 90
```

### TLS certificate

```yaml
- name: "OpenBLD TLS certificate"
  type: "tls_cert"
  target: "openbld.net:443"
  warn_days: 14
  critical_days: 3
  timeout: "5s"
```

## State logic

`tgn-watch` avoids notification spam:

```text
UNKNOWN -> OK       no notification
UNKNOWN -> FAIL     alert
OK      -> FAIL     alert
FAIL    -> FAIL     silent until dedup_window expires
FAIL    -> OK       recovery, if notify_on_recovery is true
OK      -> OK       silent
```

## Install as systemd service

```bash
sudo useradd --system --home /var/lib/tgn-watch --shell /usr/sbin/nologin tgn-watch
sudo mkdir -p /etc/tgn-watch /var/lib/tgn-watch
sudo cp bin/tgn-watch /usr/local/bin/tgn-watch
sudo cp configs/config.example.yml /etc/tgn-watch/config.yml
sudo cp deploy/systemd/tgn-watch.service /etc/systemd/system/tgn-watch.service
sudo systemctl daemon-reload
sudo systemctl enable --now tgn-watch
```

Logs:

```bash
journalctl -u tgn-watch -f
```

## Roadmap

- DNS checks
- process checks
- command checks
- file age checks
- Prometheus metrics
- SIGHUP config reload
- structured `/api/v1/event` support after `tgn-relay v0.2.0`
