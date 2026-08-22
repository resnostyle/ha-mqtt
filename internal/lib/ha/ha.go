// Package ha is a Home Assistant REST and WebSocket client.
package ha

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// HA registry list responses can exceed the coder/websocket default (32 KiB).
const wsReadLimit = 16 * 1024 * 1024

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func New(baseURL, token string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) GetState(ctx context.Context, entityID string) (map[string]any, error) {
	var out map[string]any
	if err := c.getJSON(ctx, "/api/states/"+url.PathEscape(entityID), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetForecasts(ctx context.Context, entityID, forecastType string) ([]map[string]any, error) {
	body := map[string]any{
		"entity_id": entityID,
		"type":      forecastType,
	}
	var raw map[string]any
	if err := c.postJSON(ctx, "/api/services/weather/get_forecasts?return_response", body, &raw); err != nil {
		return nil, err
	}
	serviceResponse, _ := raw["service_response"].(map[string]any)
	if serviceResponse == nil {
		serviceResponse = raw
	}
	entityData, _ := serviceResponse[entityID].(map[string]any)
	if entityData == nil {
		return nil, fmt.Errorf("no forecast in HA response for %s (type=%s): %v", entityID, forecastType, raw)
	}
	forecastRaw, ok := entityData["forecast"]
	if !ok || forecastRaw == nil {
		return nil, fmt.Errorf("missing forecast array for %s (type=%s): %v", entityID, forecastType, entityData)
	}
	forecast, err := asMapSlice(forecastRaw)
	if err != nil {
		return nil, err
	}
	slog.Debug("fetched forecast", "type", forecastType, "count", len(forecast))
	return forecast, nil
}

type EntityRegistryEntry struct {
	EntityID     string  `json:"entity_id"`
	Platform     string  `json:"platform"`
	DeviceID     string  `json:"device_id"`
	DisabledBy   *string `json:"disabled_by"`
	HiddenBy     *string `json:"hidden_by"`
	Name         *string `json:"name"`
	OriginalName *string `json:"original_name"`
	AreaID       *string `json:"area_id"`
}

type DeviceRegistryEntry struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	NameByUser   string     `json:"name_by_user"`
	Manufacturer string     `json:"manufacturer"`
	Model        string     `json:"model"`
	Identifiers  [][]string `json:"identifiers"`
	AreaID       *string    `json:"area_id"`
}

func (c *Client) ListEntityRegistry(ctx context.Context) ([]EntityRegistryEntry, error) {
	var entries []EntityRegistryEntry
	if err := c.wsRequest(ctx, "config/entity_registry/list", &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (c *Client) ListDeviceRegistry(ctx context.Context) ([]DeviceRegistryEntry, error) {
	var entries []DeviceRegistryEntry
	if err := c.wsRequest(ctx, "config/device_registry/list", &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (c *Client) getJSON(ctx context.Context, path string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	c.auth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, dest)
}

func (c *Client) postJSON(ctx context.Context, path string, body any, dest any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	c.auth(req)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, dest)
}

func (c *Client) auth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
}

func decodeResponse(resp *http.Response, dest any) error {
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HA HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	if dest == nil {
		return nil
	}
	return json.Unmarshal(b, dest)
}

func asMapSlice(v any) ([]map[string]any, error) {
	switch t := v.(type) {
	case []map[string]any:
		return t, nil
	case []any:
		out := make([]map[string]any, 0, len(t))
		for _, item := range t {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("forecast entry is not an object")
			}
			out = append(out, m)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("forecast is not an array")
	}
}

func (c *Client) wsURL() (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	u.Path = "/api/websocket"
	u.RawQuery = ""
	return u.String(), nil
}

func (c *Client) wsRequest(ctx context.Context, msgType string, dest any) error {
	wsURL, err := c.wsURL()
	if err != nil {
		return err
	}
	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPClient: c.httpClient,
	})
	if err != nil {
		return fmt.Errorf("ha websocket dial: %w", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	conn.SetReadLimit(wsReadLimit)

	var hello map[string]any
	if err := wsjson.Read(ctx, conn, &hello); err != nil {
		return err
	}
	if hello["type"] != "auth_required" {
		return fmt.Errorf("unexpected HA websocket hello: %v", hello)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{
		"type":         "auth",
		"access_token": c.token,
	}); err != nil {
		return err
	}
	var auth map[string]any
	if err := wsjson.Read(ctx, conn, &auth); err != nil {
		return err
	}
	if auth["type"] != "auth_ok" {
		return fmt.Errorf("HA websocket auth failed: %v", auth)
	}
	if err := wsjson.Write(ctx, conn, map[string]any{
		"id":   1,
		"type": msgType,
	}); err != nil {
		return err
	}
	for {
		var message map[string]any
		if err := wsjson.Read(ctx, conn, &message); err != nil {
			return err
		}
		id, _ := asInt(message["id"])
		if id != 1 {
			continue
		}
		if ok, _ := message["success"].(bool); !ok {
			return fmt.Errorf("HA websocket %s failed: %v", msgType, message)
		}
		raw, err := json.Marshal(message["result"])
		if err != nil {
			return err
		}
		return json.Unmarshal(raw, dest)
	}
}

func asInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		return int(i), err == nil
	case int:
		return n, true
	default:
		return 0, false
	}
}
