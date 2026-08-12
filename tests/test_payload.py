"""Unit tests for MQTT payload shaping (no HA/MQTT required)."""

from src.payload import build_5day, build_current, trim_hourly


def test_build_current():
    state = {
        "entity_id": "weather.pirateweather",
        "state": "rainy",
        "last_updated": "2026-08-11T23:17:19+00:00",
        "attributes": {
            "temperature": 81,
            "apparent_temperature": 82,
            "humidity": 64,
            "temperature_unit": "°F",
        },
    }
    payload = build_current(state)
    assert payload["condition"] == "rainy"
    assert payload["temperature"] == 81
    assert payload["entity_id"] == "weather.pirateweather"
    assert "published" in payload


def test_trim_hourly():
    hours = [{"datetime": str(i)} for i in range(60)]
    assert len(trim_hourly(hours, 48)) == 48
    assert len(trim_hourly(hours, 0)) == 60


def test_build_5day():
    forecast = [
        {
            "datetime": "2026-08-12T00:00:00+00:00",
            "condition": "rainy",
            "temperature": 85,
            "templow": 70,
            "humidity": 60,
            "wind_speed": 10,
        }
    ]
    payload = build_5day(forecast, source="metno")
    assert payload["config"]["source"] == "metno"
    assert len(payload["forecast"]) == 5
    assert payload["forecast"][0]["condition"] == "Rain"
    assert payload["forecast"][1]["condition"] == "No data"
