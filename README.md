# ha-mqtt

Go services that poll Home Assistant, probe hosts, and run speed tests, then publish retained JSON to Mosquitto, with optional Home Assistant MQTT discovery.

| Service | Binary | Image |
|---------|--------|-------|
| Weather + sun | `cmd/weather` | `ghcr.io/resnostyle/ha-mqtt/weather` |
| Cast / Google Home ping | `cmd/pinger` | `ghcr.io/resnostyle/ha-mqtt/pinger` |
| Host monitor | `cmd/monitor` | `ghcr.io/resnostyle/ha-mqtt/monitor` |
| Speedtest | `cmd/speedtest` | `ghcr.io/resnostyle/ha-mqtt/speedtest` |

Shared code lives under [`internal/lib`](internal/lib) (HA client, MQTT publisher, env helpers, probe).

## Weather service

Poll Home Assistant weather entities (Pirate Weather, Met.no, …) and sun day context, then publish retained JSON to Mosquitto. Optional MQTT discovery creates sensors under one device per source.

Configure sources with `HA_WEATHER_SOURCES`:

```bash
HA_WEATHER_SOURCES=pirate:weather.pirateweather,metno:weather.forecast_home
```

### Topics

For each source name (e.g. `pirate`, `metno`):

| Topic | Contents |
|-------|----------|
| `home/weather/<name>/current` | Current condition + attributes |
| `home/weather/<name>/daily` | Full daily forecast |
| `home/weather/<name>/hourly` | Hourly forecast (skipped if unsupported) |
| `home/weather/<name>/5day` | Compact 5-day summary |

When `SUN_ENABLED=true` (default), also publishes `home/sun/current`.

See [`.env.example`](.env.example) for all weather variables.

## Pinger service

Discover Google Cast / Google Home devices from Home Assistant, probe LAN connectivity, and publish retained JSON metrics to Mosquitto. Optional MQTT discovery creates latency and reachability sensors per device.

1. **Discovery** — HA entity/device registry over WebSocket; filters Cast `media_player` entities (default manufacturer `Google Inc.`, excludes Cast groups).
2. **IP resolution** — browses `_googlecast._tcp.local.` via mDNS and matches Cast UUIDs from HA to LAN IPs.
3. **Probing** — TCP connect to port 8008 (default) or optional ICMP ping.
4. **MQTT** — per-device metrics plus a summary topic.

### Topics

| Topic | Contents |
|-------|----------|
| `home/ping/<slug>/current` | Latest probe + rolling stats for one device |
| `home/ping/summary` | Snapshot of all devices |

See [`.env.example`](.env.example) for shared variables (including optional `PING_*`). For a dedicated pinger-only env file (e.g. Docker/K8s), see [`.env.pinger.example`](.env.pinger.example). Use `network_mode: host` in Docker so mDNS and LAN probes work; add `cap_add: [NET_RAW]` for `PING_METHOD=icmp`.

## Monitor service

Probe a static list of hosts (no Home Assistant required) and publish retained JSON metrics to Mosquitto. Optional MQTT discovery creates latency and reachability sensors per host.

Configure hosts with `MONITOR_HOSTS`:

```bash
MONITOR_HOSTS=router:192.168.1.1,nas:192.168.1.50,printer:10.0.0.12
```

Default probe method is ICMP (`MONITOR_METHOD=icmp`). Use `MONITOR_METHOD=tcp` with `MONITOR_TCP_PORT` for TCP connect checks.

### Topics

| Topic | Contents |
|-------|----------|
| `home/monitor/<slug>/current` | Latest probe + rolling stats for one host |
| `home/monitor/summary` | Snapshot of all hosts |

See [`.env.monitor.example`](.env.monitor.example). Use `network_mode: host` in Docker so LAN probes work; add `cap_add: [NET_RAW]` for ICMP (the default).

## Speedtest service

Run periodic internet speed tests via pure Go ([`showwin/speedtest-go`](https://github.com/showwin/speedtest-go)) and publish retained JSON to Mosquitto. No Home Assistant credentials required. Optional MQTT discovery creates download, upload, ping, and jitter sensors.

```bash
SPEEDTEST_INTERVAL_SECONDS=3600
# Optional: pin a speedtest.net server ID (empty = auto nearest)
SPEEDTEST_SERVER_ID=
```

### Topics

| Topic | Contents |
|-------|----------|
| `home/speedtest/current` | Latest download/upload Mbps, ping/jitter, server info |

See [`.env.speedtest.example`](.env.speedtest.example). Needs internet egress; `hostNetwork` is only needed when MQTT is on localhost via host networking (same as other sidecars).

## Quick start

```bash
cp .env.example .env
# Edit HA_URL, HA_TOKEN, MQTT_*, HA_WEATHER_SOURCES (and optional PING_* / MONITOR_* / SPEEDTEST_* below)

go test ./...
mise run weather    # loads .env
mise run pinger     # loads .env; overrides MQTT topic/client for ping
mise run monitor    # loads .env; overrides MQTT topic/client for monitor
mise run speedtest  # loads .env; overrides MQTT topic/client for speedtest
```

All tasks can share `.env`. Weather uses `MQTT_TOPIC_PREFIX` / `MQTT_CLIENT_ID` from the file; mise overrides force distinct prefixes/clients for pinger (`home/ping`), monitor (`home/monitor`), and speedtest (`home/speedtest`). For Docker/K8s, monitor and speedtest can use dedicated env files (no HA credentials required).

Create a long-lived access token in Home Assistant (Profile → Security) for weather and pinger.

## Docker

```bash
docker compose up -d --build
```

Or add sidecars to your Home Assistant host compose:

```yaml
ha-mqtt-weather:
  image: ghcr.io/resnostyle/ha-mqtt/weather:latest
  container_name: ha-mqtt-weather
  restart: unless-stopped
  network_mode: host
  env_file: .env

ha-mqtt-pinger:
  image: ghcr.io/resnostyle/ha-mqtt/pinger:latest
  container_name: ha-mqtt-pinger
  restart: unless-stopped
  network_mode: host
  env_file: .env.pinger

ha-mqtt-monitor:
  image: ghcr.io/resnostyle/ha-mqtt/monitor:latest
  container_name: ha-mqtt-monitor
  restart: unless-stopped
  network_mode: host
  env_file: .env.monitor
  # Required for MONITOR_METHOD=icmp (default):
  # cap_add:
  #   - NET_RAW

ha-mqtt-speedtest:
  image: ghcr.io/resnostyle/ha-mqtt/speedtest:latest
  container_name: ha-mqtt-speedtest
  restart: unless-stopped
  network_mode: host
  env_file: .env.speedtest
```

Pull requests run `go test ./...` and build images without pushing. Pushes to `main` skip tests and build + push all images to GHCR.

| Image | Tags |
|-------|------|
| `ghcr.io/resnostyle/ha-mqtt/weather` | `latest`, commit SHA, branch name |
| `ghcr.io/resnostyle/ha-mqtt/pinger` | `latest`, commit SHA, branch name |
| `ghcr.io/resnostyle/ha-mqtt/monitor` | `latest`, commit SHA, branch name |
| `ghcr.io/resnostyle/ha-mqtt/speedtest` | `latest`, commit SHA, branch name |

Pull requests build images without pushing. Use **Actions → CI → Run workflow** to trigger manually.

After the first push, open each package under **GitHub → Packages** and set visibility/linking if needed. The workflow uses `GITHUB_TOKEN` with `packages: write`.

## Kubernetes

Helm charts in the k8s-gitops repo deploy services to the `automation` namespace.

**Weather** runs as a normal pod. Point `MQTT_HOST` at the in-cluster broker (e.g. `emqx.automation.svc.cluster.local`). Secrets (`HA_TOKEN`, optional `MQTT_USERNAME` / `MQTT_PASSWORD`) come from the `weather-mqtt-secrets` Vault-synced secret.

**Pinger** requires `hostNetwork: true` so mDNS can discover Cast devices and TCP probes reach LAN hosts. It reuses the same `weather-mqtt-secrets` secret. For `PING_METHOD=icmp`, add `securityContext.capabilities.add: [NET_RAW]`.

**Monitor** also needs `hostNetwork: true` so probes reach LAN hosts. It only needs MQTT credentials (no HA token). For `MONITOR_METHOD=icmp` (default), add `securityContext.capabilities.add: [NET_RAW]`.

**Speedtest** only needs MQTT credentials and internet egress. Cluster networking is fine when `MQTT_HOST` points at an in-cluster broker; use `hostNetwork` only if matching the other localhost-MQTT sidecars.

| Setting | Weather | Pinger | Monitor | Speedtest |
|---------|---------|--------|---------|-----------|
| Image | `ghcr.io/resnostyle/ha-mqtt/weather` | `ghcr.io/resnostyle/ha-mqtt/pinger` | `ghcr.io/resnostyle/ha-mqtt/monitor` | `ghcr.io/resnostyle/ha-mqtt/speedtest` |
| Network | cluster | `hostNetwork: true` | `hostNetwork: true` | cluster (or host for localhost MQTT) |
| Probes | disabled (no HTTP endpoint) | disabled | disabled | disabled |

## Layout

```
cmd/weather/          weather + sun poller
cmd/pinger/           Cast ping service
cmd/monitor/          static host monitor
cmd/speedtest/        internet speedtest
internal/lib/         shared HA, MQTT, env, poll, probe
internal/weather/     weather payloads + discovery
internal/pinger/      cast discovery, mDNS, probe, discovery
internal/monitor/     host list config, probe, discovery
internal/speedtest/   speedtest runner, payload, discovery
```
