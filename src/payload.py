"""Shape HA weather data into MQTT JSON payloads."""

from __future__ import annotations

from datetime import datetime, timezone
from typing import Any


CONDITION_LABELS = {
    "sunny": "Sunny",
    "clear-night": "Clear skies tonight",
    "partlycloudy": "Partly cloudy",
    "cloudy": "Cloudy",
    "rainy": "Rain",
    "pouring": "Heavy rain",
    "snowy": "Snow",
    "snowy-rainy": "Snow/rain",
    "fog": "Foggy",
    "hail": "Hail",
    "lightning": "Lightning",
    "lightning-rainy": "Thunderstorms",
    "windy": "Windy",
    "windy-variant": "Windy",
    "exceptional": "Exceptional",
}


def _utc_now_iso() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat()


def build_current(state: dict[str, Any]) -> dict[str, Any]:
    attrs = state.get("attributes") or {}
    return {
        "condition": state.get("state"),
        "temperature": attrs.get("temperature"),
        "apparent_temperature": attrs.get("apparent_temperature"),
        "dew_point": attrs.get("dew_point"),
        "humidity": attrs.get("humidity"),
        "ozone": attrs.get("ozone"),
        "cloud_coverage": attrs.get("cloud_coverage"),
        "pressure": attrs.get("pressure"),
        "pressure_unit": attrs.get("pressure_unit"),
        "wind_bearing": attrs.get("wind_bearing"),
        "wind_speed": attrs.get("wind_speed"),
        "wind_gust_speed": attrs.get("wind_gust_speed"),
        "wind_speed_unit": attrs.get("wind_speed_unit"),
        "visibility": attrs.get("visibility"),
        "visibility_unit": attrs.get("visibility_unit"),
        "temperature_unit": attrs.get("temperature_unit"),
        "precipitation_unit": attrs.get("precipitation_unit"),
        "attribution": attrs.get("attribution"),
        "entity_id": state.get("entity_id"),
        "updated": state.get("last_updated") or _utc_now_iso(),
        "published": _utc_now_iso(),
    }


def build_forecast_payload(
    forecast: list[dict[str, Any]],
    *,
    forecast_type: str,
    entity_id: str,
) -> dict[str, Any]:
    return {
        "type": forecast_type,
        "entity_id": entity_id,
        "count": len(forecast),
        "forecast": forecast,
        "published": _utc_now_iso(),
    }


def trim_hourly(
    forecast: list[dict[str, Any]], hours: int
) -> list[dict[str, Any]]:
    if hours <= 0:
        return forecast
    return forecast[:hours]


def build_5day(
    forecast: list[dict[str, Any]],
    *,
    source: str = "pirateweather",
    max_days: int = 5,
    unit: str = "°F",
) -> dict[str, Any]:
    """Compact payload compatible with the old home/weather/5day automation."""
    days = forecast[:max_days]
    entries: list[dict[str, Any]] = []
    for i in range(max_days):
        if i < len(days):
            day = days[i]
            condition = day.get("condition")
            entries.append(
                {
                    "day_index": i,
                    "datetime": day.get("datetime"),
                    "condition": CONDITION_LABELS.get(condition, condition),
                    "condition_raw": condition,
                    "temperature": day.get("temperature"),
                    "templow": day.get("templow"),
                    "humidity": day.get("humidity"),
                    "wind_speed": day.get("wind_speed"),
                    "precipitation": day.get("precipitation"),
                    "precipitation_probability": day.get(
                        "precipitation_probability"
                    ),
                }
            )
        else:
            entries.append(
                {
                    "day_index": i,
                    "condition": "No data",
                    "temperature": None,
                    "templow": None,
                    "humidity": None,
                    "wind_speed": None,
                }
            )

    return {
        "config": {
            "unit": unit,
            "max_days": max_days,
            "fields": [
                "condition",
                "temperature",
                "templow",
                "humidity",
                "wind_speed",
            ],
            "source": source,
        },
        "forecast": entries,
        "published": _utc_now_iso(),
    }
