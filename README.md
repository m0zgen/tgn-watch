# tgn-watch

`tgn-watch` is a lightweight monitoring agent for Linux/server infrastructure. It runs local checks and sends Telegram alerts through `tgn-relay`.

Current release: **v0.1.3**

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
- auto-recovery actions with explicit enable flag
- action retries, timeout, delay and cooldown
- per-check interval
- alert/recovery notifications
- deduplication window
- local observability HTTP server
- `/healthz`, `/status`, `/metrics`
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
  actions_enabled: false

server:
  enabled: true
  listen: "127.0.0.1:34351"
```

## Observability endpoints

When `server.enabled: true`, tgn-watch exposes local HTTP endpoints:

```bash
curl -s http://127.0.0.1:34351/healthz
curl -s http://127.0.0.1:34351/status
curl -s http://127.0.0.1:34351/metrics
```

`/healthz` returns a minimal health response:

```json
{"status":"ok"}
```

`/status` returns runtime state, version, uptime, check status, last messages and counters.

`/metrics` returns Prometheus-compatible text format without external dependencies.

Example metrics:

```text
tgn_watch_up 1
tgn_watch_uptime_seconds 3600
tgn_watch_checks_total 8
tgn_watch_checks_ok 7
tgn_watch_checks_fail 1
tgn_watch_check_status{group="monitoring",name="tgn-relay local health",severity="critical",type="http"} 1
```

## Per-check interval

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

```yaml
- name: "Custom command example"
  type: "command"
  command: "systemctl is-active tgn-relay.service"
  expect_exit_code: 0
  interval: "1m"
  timeout: "3s"
  group: "monitoring"
  severity: "critical"
```

## Auto-recovery actions

Auto actions are disabled globally by default:

```yaml
watcher:
  actions_enabled: false
```

Enable the global guard first:

```yaml
watcher:
  actions_enabled: true
```

Then enable action on specific checks only:

```yaml
- name: "tgn-relay TCP"
  type: "tcp"
  target: "127.0.0.1:8080"
  timeout: "2s"
  interval: "30s"
  group: "monitoring"
  severity: "critical"

  action_enabled: true
  action_command: "sudo /usr/local/sbin/tgn-watch-restart-tgn-relay"
  action_retries: 2
  action_timeout: "10s"
  action_delay: "2s"
  action_cooldown: "5m"
```

If `watcher.actions_enabled: false`, checks still run and notifications still work, but action commands are skipped.

## systemd

Example unit:

```ini
[Unit]
Description=tgn-watch lightweight monitoring agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=tgn-watch
Group=tgn-watch
ExecStart=/usr/local/bin/tgn-watch -config /etc/tgn-watch/config.yml
Restart=on-failure
RestartSec=3s

[Install]
WantedBy=multi-user.target
```

## Notes

- Keep the observability server bound to `127.0.0.1` unless you intentionally expose it.
- Do not enable command checks globally unless you trust the config source.
- Do not enable auto actions globally unless sudoers/wrapper scripts are restricted.
- Prefer root-owned wrapper scripts for restart actions instead of broad sudo permissions.
