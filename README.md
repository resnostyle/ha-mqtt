# weather-mqtt

Poll Home Assistant weather entities (Pirate Weather, Met.no, …) and publish
retained JSON to Mosquitto, plus Home Assistant MQTT discovery so sensors appear
automatically under one device per source.

## Sources

Configure with `HA_WEATHER_SOURCES`:

```bash
HA_WEATHER_SOURCES=pirate:weather.pirateweather,metno:weather.forecast_home
```

Each `name:entity_id` pair gets its own topic tree and discovery device.

## Topics

For each source name (e.g. `pirate`, `metno`):

| Topic | Contents |
|-------|----------|
| `home/weather/<name>/current` | Current condition + attributes |
| `home/weather/<name>/daily` | Full daily forecast from `weather.get_forecasts` |
| `home/weather/<name>/hourly` | Hourly forecast (default next 48 hours; skipped if unsupported) |
| `home/weather/<name>/5day` | Compact 5-day summary |

All messages are retained JSON.

> **Note:** Earlier builds used unscoped `home/weather/current`. Topics are now
> namespaced per source (`…/pirate/…`, `…/metno/…`).

## MQTT discovery (Home Assistant)

On startup the bridge publishes retained configs to
`homeassistant/sensor/<id>/config`.

Example entities:

- `sensor.weather_mqtt_pirate_temperature`
- `sensor.weather_mqtt_pirate_daily`
- `sensor.weather_mqtt_metno_temperature`
- `sensor.weather_mqtt_metno_5day`

Devices:

- **Weather MQTT (Pirate Weather)**
- **Weather MQTT (Met.no)**

Set `MQTT_DISCOVERY_ENABLED=false` to skip discovery and only publish data topics.

## Quick start

```bash
cp .env.example .env
# Edit HA_URL, HA_TOKEN, MQTT_*, HA_WEATHER_SOURCES

python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
python -m src.main
```

Create a long-lived access token in Home Assistant (Profile → Security).

## Docker

```bash
docker compose -f docker-compose.example.yml up -d --build
```

Or add to the Home Assistant host compose as an image-only sidecar:

```yaml
weather-mqtt:
  image: ghcr.io/<you>/weather-mqtt:latest
  container_name: weather-mqtt
  restart: unless-stopped
  network_mode: host
  env_file: .env
```

## Environment

See [`.env.example`](.env.example). Required: `HA_URL`, `HA_TOKEN`.
