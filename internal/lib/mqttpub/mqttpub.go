// Package mqttpub publishes retained JSON to MQTT and Home Assistant discovery topics.
package mqttpub

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Config struct {
	ObjectID  string
	Component string
	Payload   map[string]any
}

type Publisher struct {
	client      mqtt.Client
	topicPrefix string
}

func New(host string, port int, clientID, username, password, topicPrefix string) (*Publisher, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", host, port))
	opts.SetClientID(clientID)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetAutoReconnect(true)
	if username != "" {
		opts.SetUsername(username)
		opts.SetPassword(password)
	}
	client := mqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(15 * time.Second) {
		return nil, fmt.Errorf("mqtt connect timeout %s:%d", host, port)
	}
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("mqtt connect %s:%d: %w", host, port, err)
	}
	prefix := strings.TrimRight(topicPrefix, "/")
	slog.Info("connected to mqtt", "host", host, "port", port, "prefix", prefix)
	return &Publisher{client: client, topicPrefix: prefix}, nil
}

func (p *Publisher) Close() {
	if p == nil || p.client == nil {
		return
	}
	p.client.Disconnect(250)
}

func (p *Publisher) TopicPrefix() string {
	return p.topicPrefix
}

func Marshal(payload any) ([]byte, error) {
	return json.Marshal(payload)
}

func Join(prefix, suffix string) string {
	prefix = strings.TrimRight(prefix, "/")
	suffix = strings.TrimLeft(suffix, "/")
	if prefix == "" {
		return suffix
	}
	if suffix == "" {
		return prefix
	}
	return prefix + "/" + suffix
}

func DiscoveryTopic(discoveryPrefix, component, objectID string) string {
	return Join(discoveryPrefix, component) + "/" + objectID + "/config"
}

func (p *Publisher) PublishRaw(topic string, payload any, retain bool) error {
	body, err := Marshal(payload)
	if err != nil {
		return err
	}
	token := p.client.Publish(topic, 1, retain, body)
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("mqtt publish timeout for %s", topic)
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("mqtt publish failed for %s: %w", topic, err)
	}
	slog.Info("published", "topic", topic, "bytes", len(body), "retain", retain)
	return nil
}

func (p *Publisher) Publish(suffix string, payload any, retain bool) error {
	return p.PublishRaw(Join(p.topicPrefix, suffix), payload, retain)
}

func (p *Publisher) PublishDiscovery(configs []Config, discoveryPrefix string) error {
	prefix := strings.TrimRight(discoveryPrefix, "/")
	for _, cfg := range configs {
		component := cfg.Component
		if component == "" {
			component = "sensor"
		}
		topic := DiscoveryTopic(prefix, component, cfg.ObjectID)
		if err := p.PublishRaw(topic, cfg.Payload, true); err != nil {
			return err
		}
	}
	return nil
}

// Sink is used by services in tests without a broker.
type Sink interface {
	Publish(suffix string, payload any, retain bool) error
	PublishRaw(topic string, payload any, retain bool) error
	PublishDiscovery(configs []Config, discoveryPrefix string) error
}

var _ Sink = (*Publisher)(nil)
