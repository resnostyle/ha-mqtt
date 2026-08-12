"""Configuration loaded from environment variables."""

from __future__ import annotations

import os
import re
from dataclasses import dataclass


_SOURCE_RE = re.compile(r"^[a-z][a-z0-9_]*$")


def _require(name: str) -> str:
    value = os.environ.get(name, "").strip()
    if not value:
        raise SystemExit(f"Missing required environment variable: {name}")
    return value


@dataclass(frozen=True)
class WeatherSource:
    """One HA weather entity published under a topic/discovery slug."""

    name: str
    entity_id: str
    label: str

    @property
    def topic_suffix(self) -> str:
        return self.name


@dataclass(frozen=True)
class Settings:
    ha_url: str
    ha_token: str
    weather_sources: tuple[WeatherSource, ...]
    mqtt_host: str
    mqtt_port: int
    mqtt_username: str | None
    mqtt_password: str | None
    mqtt_topic_prefix: str
    mqtt_client_id: str
    mqtt_discovery_enabled: bool
    mqtt_discovery_prefix: str
    temperature_unit: str
    pressure_unit: str
    wind_speed_unit: str
    visibility_unit: str
    poll_interval_seconds: int
    hourly_forecast_hours: int
    log_level: str

    @classmethod
    def from_env(cls) -> Settings:
        username = os.environ.get("MQTT_USERNAME", "").strip() or None
        password = os.environ.get("MQTT_PASSWORD", "").strip() or None
        discovery_raw = os.environ.get("MQTT_DISCOVERY_ENABLED", "true").strip().lower()
        return cls(
            ha_url=_require("HA_URL").rstrip("/"),
            ha_token=_require("HA_TOKEN"),
            weather_sources=parse_weather_sources(),
            mqtt_host=os.environ.get("MQTT_HOST", "127.0.0.1").strip(),
            mqtt_port=int(os.environ.get("MQTT_PORT", "1883")),
            mqtt_username=username,
            mqtt_password=password,
            mqtt_topic_prefix=os.environ.get(
                "MQTT_TOPIC_PREFIX", "home/weather"
            ).strip().rstrip("/"),
            mqtt_client_id=os.environ.get("MQTT_CLIENT_ID", "weather-mqtt").strip(),
            mqtt_discovery_enabled=discovery_raw in ("1", "true", "yes", "on"),
            mqtt_discovery_prefix=os.environ.get(
                "MQTT_DISCOVERY_PREFIX", "homeassistant"
            ).strip().rstrip("/"),
            temperature_unit=os.environ.get("TEMPERATURE_UNIT", "°F").strip(),
            pressure_unit=os.environ.get("PRESSURE_UNIT", "inHg").strip(),
            wind_speed_unit=os.environ.get("WIND_SPEED_UNIT", "mph").strip(),
            visibility_unit=os.environ.get("VISIBILITY_UNIT", "mi").strip(),
            poll_interval_seconds=int(os.environ.get("POLL_INTERVAL_SECONDS", "900")),
            hourly_forecast_hours=int(os.environ.get("HOURLY_FORECAST_HOURS", "48")),
            log_level=os.environ.get("LOG_LEVEL", "INFO").strip().upper(),
        )


def parse_weather_sources(
    raw: str | None = None,
    *,
    legacy_entity: str | None = None,
) -> tuple[WeatherSource, ...]:
    """
    Parse HA_WEATHER_SOURCES like:
      pirate:weather.pirateweather,metno:weather.forecast_home

    Falls back to HA_WEATHER_ENTITY as a single source named \"pirate\".
    """
    if raw is None:
        raw = os.environ.get("HA_WEATHER_SOURCES", "").strip()
    if legacy_entity is None:
        legacy_entity = os.environ.get("HA_WEATHER_ENTITY", "").strip()

    if not raw:
        entity = legacy_entity or "weather.pirateweather"
        return (
            WeatherSource(
                name="pirate",
                entity_id=entity,
                label=_default_label("pirate"),
            ),
        )

    sources: list[WeatherSource] = []
    seen: set[str] = set()
    for part in raw.split(","):
        part = part.strip()
        if not part:
            continue
        if ":" not in part:
            raise SystemExit(
                f"Invalid HA_WEATHER_SOURCES entry {part!r}; "
                "expected name:entity_id (e.g. metno:weather.forecast_home)"
            )
        name, entity_id = part.split(":", 1)
        name = name.strip().lower()
        entity_id = entity_id.strip()
        if not _SOURCE_RE.match(name):
            raise SystemExit(
                f"Invalid weather source name {name!r}; use lowercase letters, "
                "numbers, underscores"
            )
        if not entity_id.startswith("weather."):
            raise SystemExit(
                f"Invalid weather entity {entity_id!r}; expected weather.*"
            )
        if name in seen:
            raise SystemExit(f"Duplicate weather source name: {name}")
        seen.add(name)
        sources.append(
            WeatherSource(
                name=name,
                entity_id=entity_id,
                label=_default_label(name),
            )
        )

    if not sources:
        raise SystemExit("HA_WEATHER_SOURCES is empty")
    return tuple(sources)


def _default_label(name: str) -> str:
    labels = {
        "pirate": "Pirate Weather",
        "metno": "Met.no",
        "met_no": "Met.no",
        "forecast_home": "Met.no",
    }
    if name in labels:
        return labels[name]
    return name.replace("_", " ").title()
