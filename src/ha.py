"""Home Assistant REST client for weather state and forecasts."""

from __future__ import annotations

import logging
from typing import Any

import httpx

logger = logging.getLogger(__name__)


class HomeAssistantClient:
    def __init__(self, base_url: str, token: str, timeout: float = 30.0) -> None:
        self._base_url = base_url.rstrip("/")
        self._client = httpx.Client(
            base_url=self._base_url,
            headers={
                "Authorization": f"Bearer {token}",
                "Content-Type": "application/json",
            },
            timeout=timeout,
        )

    def close(self) -> None:
        self._client.close()

    def __enter__(self) -> HomeAssistantClient:
        return self

    def __exit__(self, *args: object) -> None:
        self.close()

    def get_state(self, entity_id: str) -> dict[str, Any]:
        response = self._client.get(f"/api/states/{entity_id}")
        response.raise_for_status()
        return response.json()

    def get_forecasts(self, entity_id: str, forecast_type: str) -> list[dict[str, Any]]:
        """Call weather.get_forecasts and return the forecast list."""
        response = self._client.post(
            "/api/services/weather/get_forecasts?return_response",
            json={"entity_id": entity_id, "type": forecast_type},
        )
        response.raise_for_status()
        body = response.json()
        service_response = body.get("service_response", body)
        entity_data = service_response.get(entity_id)
        if not entity_data:
            raise ValueError(
                f"No forecast in HA response for {entity_id} "
                f"(type={forecast_type}): {body!r}"
            )
        forecast = entity_data.get("forecast")
        if forecast is None:
            raise ValueError(
                f"Missing forecast array for {entity_id} "
                f"(type={forecast_type}): {entity_data!r}"
            )
        logger.debug(
            "Fetched %s forecast: %s entries", forecast_type, len(forecast)
        )
        return forecast
