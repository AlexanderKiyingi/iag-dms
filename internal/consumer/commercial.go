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
	Type   string          `json:"type"`
	Source string          `json:"source"`
	Data   json.RawMessage `json:"data"`
}

// Commercial consumes iag.commercial and applies CRM events to DMS.
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
		if err := c.handle(ctx, msg.Value); err != nil {
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

func (c *Commercial) handle(ctx context.Context, raw []byte) error {
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return err
	}
	if env.Source == "iag.dms" {
		return nil
	}
	switch env.Type {
	case "crm.lead.converted", "crm.outlet.synced", "crm.deal.won":
		if c.repo != nil {
			if err := c.repo.ApplyCommercialEvent(ctx, env.Type, env.Data); err != nil {
				return err
			}
			_, _ = c.repo.AppendAudit(ctx, "InboundEvent", env.Type, "kafka-consumer")
		}
		slog.Debug("dms consumer event applied", "type", env.Type)
	}
	return nil
}
