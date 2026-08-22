package ha

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestGetStateAndForecasts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, "auth", http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/states/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"entity_id": "weather.pirateweather",
				"state":     "rainy",
				"attributes": map[string]any{
					"temperature": 81,
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/services/weather/get_forecasts":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"service_response": map[string]any{
					"weather.pirateweather": map[string]any{
						"forecast": []map[string]any{
							{"condition": "rainy", "temperature": 85},
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "tok", 5*time.Second)
	ctx := context.Background()
	state, err := c.GetState(ctx, "weather.pirateweather")
	if err != nil {
		t.Fatal(err)
	}
	if state["state"] != "rainy" {
		t.Fatalf("state %v", state["state"])
	}
	forecast, err := c.GetForecasts(ctx, "weather.pirateweather", "daily")
	if err != nil {
		t.Fatal(err)
	}
	if len(forecast) != 1 {
		t.Fatalf("len %d", len(forecast))
	}
}

func TestWSRegistry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/websocket" {
			http.NotFound(w, r)
			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		ctx := r.Context()
		_ = wsjson.Write(ctx, conn, map[string]any{"type": "auth_required"})
		var auth map[string]any
		if err := wsjson.Read(ctx, conn, &auth); err != nil {
			return
		}
		if auth["access_token"] != "tok" {
			_ = wsjson.Write(ctx, conn, map[string]any{"type": "auth_invalid"})
			return
		}
		_ = wsjson.Write(ctx, conn, map[string]any{"type": "auth_ok"})
		var req map[string]any
		if err := wsjson.Read(ctx, conn, &req); err != nil {
			return
		}
		result := []map[string]any{}
		if req["type"] == "config/entity_registry/list" {
			result = []map[string]any{{"entity_id": "media_player.example_speaker", "platform": "cast"}}
		}
		if req["type"] == "config/device_registry/list" {
			result = []map[string]any{{"id": "dev1", "manufacturer": "Google Inc."}}
		}
		_ = wsjson.Write(ctx, conn, map[string]any{
			"id":      req["id"],
			"success": true,
			"result":  result,
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "tok", 5*time.Second)
	ctx := context.Background()
	entities, err := c.ListEntityRegistry(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(entities) != 1 || entities[0].EntityID != "media_player.example_speaker" {
		t.Fatalf("%v", entities)
	}
	devices, err := c.ListDeviceRegistry(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 1 || devices[0].ID != "dev1" {
		t.Fatalf("%v", devices)
	}
}
