# tgn-watch

`tgn-watch` is a lightweight monitoring agent for Linux/server infrastructure. It runs local checks and sends Telegram alerts through `tgn-relay`.

Current release: **v0.1.1**

## Features

- HTTP checks
- TCP checks
- DNS checks
- systemd service checks
- process checks
- disk usage checks
- memory usage checks
- TLS certificate expiry checks
- file age checks
- command checks with explicit enable flag
- per-check interval
- alert/recovery notifications
- deduplication window
- YAML-style config without external Go dependencies
- systemd unit example

## Architecture

```text
tgn-watch  ->  tgn-relay  ->  Telegram
```

`tgn-watch` does not talk to Telegram directly. It sends messages to `tgn-relay` using `/api/v1/send`.

## Build

```bash
go mod tidy
make build
./bin/tgn-watch -version
```

Release builds use `-trimpath`, so local build-machine paths are not embedded in stack traces.

## Run

```bash
go run ./cmd/tgn-watch -config configs/config.example.yml
```

or:

```bash
./bin/tgn-watch -config configs/config.example.yml
```

## Config

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
  command_checks_enabled: false
```

Each check may override the global interval:

```yaml
checks:
  - name: "OpenBLD TLS certificate"
    type: "tls_cert"
    target: "openbld.net:443"
    warn_days: 14
    critical_days: 3
    interval: "6h"
    timeout: "5s"
    group: "monitoring"
    severity: "warning"
```

## Check examples

### HTTP

```yaml
- name: "tgn-relay local health"
  type: "http"
  url: "http://127.0.0.1:8080/healthz"
  expect_status: 200
  timeout: "3s"
  group: "monitoring"
  severity: "critical"
```

### TCP

```yaml
- name: "Local SSH TCP"
  type: "tcp"
  target: "127.0.0.1:22"
  timeout: "2s"
  group: "monitoring"
  severity: "warning"
```

### DNS

DNS checks use a small native UDP DNS query implementation, without third-party DNS libraries.

```yaml
- name: "DNS resolver check"
  type: "dns"
  server: "127.0.0.1:53"
  qname: "openbld.net."
  qtype: "A"
  expect_rcode: "NOERROR"
  timeout: "2s"
  group: "monitoring"
  severity: "critical"
```

Supported qtypes: `A`, `AAAA`, `NS`, `CNAME`, `SOA`, `PTR`, `MX`, `TXT`, `SRV`.

Optional answer threshold:

```yaml
expect_min_answers: 1
```

### systemd

```yaml
- name: "tgn-relay systemd"
  type: "systemd"
  service: "tgn-relay.service"
  timeout: "3s"
  group: "monitoring"
  severity: "critical"
```

### process

```yaml
- name: "tgn-relay process"
  type: "process"
  process: "tgn-relay"
  timeout: "3s"
  group: "monitoring"
  severity: "critical"
```

The process check scans `/proc`, so it is currently Linux-focused.

### disk

```yaml
- name: "Root disk"
  type: "disk"
  path: "/"
  max_used_percent: 90
  timeout: "3s"
  group: "monitoring"
  severity: "warning"
```

### memory

```yaml
- name: "Memory usage"
  type: "memory"
  max_used_percent: 90
  timeout: "3s"
  group: "monitoring"
  severity: "warning"
```

### TLS certificate expiry

```yaml
- name: "OpenBLD TLS certificate"
  type: "tls_cert"
  target: "openbld.net:443"
  warn_days: 14
  critical_days: 3
  interval: "6h"
  timeout: "5s"
  group: "monitoring"
  severity: "warning"
```

### file age

```yaml
- name: "Blocklist freshness"
  type: "file_age"
  path: "/var/lib/zbld/hosts.txt"
  max_age: "6h"
  interval: "10m"
  timeout: "3s"
  group: "monitoring"
  severity: "warning"
```

### command

Command checks are disabled by default. Enable them explicitly:

```yaml
watcher:
  command_checks_enabled: true
```

Then define a command check:

```yaml
- name: "Custom systemctl check"
  type: "command"
  command: "systemctl is-active tgn-relay.service"
  expect_exit_code: 0
  interval: "1m"
  timeout: "3s"
  group: "monitoring"
  severity: "critical"
```

Command checks execute through `/bin/sh -c`, so treat them as trusted local configuration only.

## Notification logic

```text
UNKNOWN -> FAIL     alert
OK      -> FAIL     alert
FAIL    -> FAIL     repeat only after dedup_window
FAIL    -> OK       recovery, if notify_on_recovery is true
OK      -> OK       no notification
```

## Install with systemd

```bash
sudo useradd --system --home /var/lib/tgn-watch --shell /usr/sbin/nologin tgn-watch || true
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

## Changelog

### v0.1.1

- fix: protect state map with mutex
- build: add `-trimpath`
- feat: DNS check
- feat: process check
- feat: file age check
- feat: command check with explicit enable flag
- feat: per-check interval

### v0.1.0

- initial MVP
- HTTP/TCP/systemd/disk/memory/TLS certificate checks
- alert/recovery notifications through tgn-relay
