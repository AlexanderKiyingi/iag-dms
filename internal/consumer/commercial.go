package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/iag/dms/backend/internal/store"
)

type Config struct {
	Brokers []string
	GroupID string
	Topic   string
}

type envelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type Commercial struct {
	reader *kafka.Reader
	repo   *store.Repository
}

func NewCommercial(cfg Config, repo *store.Repository) *Commercial {
	return &Commercial{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  cfg.Brokers,
			GroupID:  cfg.GroupID,
			Topic:    cfg.Topic,
			MinBytes: 1,
			MaxBytes: 10e6,
		}),
		repo: repo,
	}
}

func (c *Commercial) Run(ctx context.Context) error {
	slog.Info("dms commercial consumer started", "topic", c.reader.Config().Topic, "group", c.reader.Config().GroupID)
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			slog.Warn("dms consumer fetch", "err", err)
			time.Sleep(time.Second)
			continue
		}
		if err := c.handle(msg.Value); err != nil {
			slog.Warn("dms consumer handle", "err", err)
		} else if err := c.reader.CommitMessages(ctx, msg); err != nil {
			slog.Warn("dms consumer commit", "err", err)
		}
	}
}

func (c *Commercial) Close() error {
	if c.reader == nil {
		return nil
	}
	return c.reader.Close()
}

func (c *Commercial) handle(raw []byte) error {
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return err
	}
	switch env.Type {
	case "crm.lead.converted", "crm.outlet.synced", "crm.deal.won":
		if c.repo != nil {
			_, _ = c.repo.AppendAudit(context.Background(), "InboundEvent", env.Type, "kafka-consumer")
		}
		slog.Debug("dms consumer event", "type", env.Type)
	}
	return nil
}
