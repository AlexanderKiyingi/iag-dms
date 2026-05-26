package events

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

const (
	SpecVersion = "1.0"
	Source      = "iag.dms"
	TopicOps    = "iag.operations"
)

type Bus struct {
	writer  *kafka.Writer
	enabled bool
}

type Config struct {
	Brokers []string
	Enabled bool
}

func NewFromEnv() *Bus {
	return New(Config{
		Brokers: ParseBrokers(os.Getenv("KAFKA_BROKERS")),
		Enabled: strings.EqualFold(os.Getenv("EVENT_BUS_ENABLED"), "true"),
	})
}

func New(cfg Config) *Bus {
	if !cfg.Enabled || len(cfg.Brokers) == 0 {
		return &Bus{}
	}
	return &Bus{
		enabled: true,
		writer: &kafka.Writer{
			Addr:     kafka.TCP(cfg.Brokers...),
			Topic:    TopicOps,
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (b *Bus) Enabled() bool { return b.enabled }

func (b *Bus) Close() error {
	if b.writer == nil {
		return nil
	}
	return b.writer.Close()
}

func ParseBrokers(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

type Envelope struct {
	SpecVersion string          `json:"specversion"`
	ID          string          `json:"id"`
	Source      string          `json:"source"`
	Type        string          `json:"type"`
	Data        json.RawMessage `json:"data"`
}

func (b *Bus) Publish(ctx context.Context, eventType string, data any) error {
	if !b.enabled {
		return nil
	}
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	env := Envelope{SpecVersion: SpecVersion, ID: uuid.NewString(), Source: Source, Type: eventType, Data: body}
	raw, err := json.Marshal(env)
	if err != nil {
		return err
	}
	if err := b.writer.WriteMessages(ctx, kafka.Message{Value: raw}); err != nil {
		slog.Warn("kafka publish", "type", eventType, "err", err)
		return err
	}
	return nil
}
