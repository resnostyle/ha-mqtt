# ha-mqtt

Go services that poll Home Assistant and publish retained JSON to Mosquitto, with optional Home Assistant MQTT discovery.

| Service | Binary | Image |
|---------|--------|-------|
| Weather + sun | `cmd/weather` | `ghcr.io/resnostyle/ha-mqtt` |
| Cast / Google Home ping | `cmd/pinger` | `ghcr.io/resnostyle/ha-mqtt/pinger` |

Shared code lives under [`internal/lib`](internal/lib) (HA client, MQTT publisher, env helpers).

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

See [`.env.pinger.example`](.env.pinger.example) for all pinger variables. Use `network_mode: host` in Docker so mDNS and LAN probes work; add `cap_add: [NET_RAW]` for `PING_METHOD=icmp`.

## Quick start

```bash
cp .env.example .env
# Edit HA_URL, HA_TOKEN, MQTT_*, HA_WEATHER_SOURCES

go test ./...
go run ./cmd/weather
```

Pinger:

```bash
cp .env.pinger.example .env.pinger
set -a && source .env.pinger && set +a
go run ./cmd/pinger
```

Create a long-lived access token in Home Assistant (Profile → Security).

## Docker

```bash
docker compose up -d --build
```

Or add sidecars to your Home Assistant host compose:

```yaml
ha-mqtt-weather:
  image: ghcr.io/resnostyle/ha-mqtt:latest
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
```

Pull requests run `go test ./...` and build images without pushing. Pushes to `main` skip tests and build + push both images to GHCR.

| Image | Tags |
|-------|------|
| `ghcr.io/resnostyle/ha-mqtt` | `latest`, commit SHA, branch name |
| `ghcr.io/resnostyle/ha-mqtt/pinger` | `latest`, commit SHA, branch name |

Pull requests build images without pushing. Use **Actions → CI → Run workflow** to trigger manually.

After the first push, open each package under **GitHub → Packages** and set visibility/linking if needed. The workflow uses `GITHUB_TOKEN` with `packages: write`.

## Kubernetes

Helm charts in the k8s-gitops repo deploy both services to the `automation` namespace.

**Weather** runs as a normal pod. Point `MQTT_HOST` at the in-cluster broker (e.g. `emqx.automation.svc.cluster.local`). Secrets (`HA_TOKEN`, optional `MQTT_USERNAME` / `MQTT_PASSWORD`) come from the `weather-mqtt-secrets` Vault-synced secret.

**Pinger** requires `hostNetwork: true` so mDNS can discover Cast devices and TCP probes reach LAN hosts. It reuses the same `weather-mqtt-secrets` secret. For `PING_METHOD=icmp`, add `securityContext.capabilities.add: [NET_RAW]`.

| Setting | Weather | Pinger |
|---------|---------|--------|
| Image | `ghcr.io/resnostyle/ha-mqtt` | `ghcr.io/resnostyle/ha-mqtt/pinger` |
| Network | cluster | `hostNetwork: true` |
| Probes | disabled (no HTTP endpoint) | disabled |

## Layout

```
cmd/weather/          weather + sun poller
cmd/pinger/           Cast ping service
internal/lib/         shared HA, MQTT, env, poll loop
internal/weather/     weather payloads + discovery
internal/pinger/      cast discovery, mDNS, probe, discovery
```
