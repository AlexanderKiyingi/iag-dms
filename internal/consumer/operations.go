package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/iag/dms/backend/internal/store"
)

type OperationsConfig struct {
	Brokers []string
	GroupID string
	Topic   string
}

type envelopeOps struct {
	Type   string          `json:"type"`
	Source string          `json:"source"`
	Data   json.RawMessage `json:"data"`
}

// Operations consumes iag.operations warehouse events for DMS fulfillment.
type Operations struct {
	reader *kafka.Reader
	repo   *store.Repository
}

func NewOperations(cfg OperationsConfig, repo *store.Repository) *Operations {
	return &Operations{
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

func (c *Operations) Run(ctx context.Context) error {
	slog.Info("dms operations consumer started", "topic", c.reader.Config().Topic, "group", c.reader.Config().GroupID)
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			slog.Warn("dms operations consumer fetch", "err", err)
			time.Sleep(time.Second)
			continue
		}
		if err := c.handle(ctx, msg.Value); err != nil {
			slog.Warn("dms operations consumer handle", "err", err)
		} else if err := c.reader.CommitMessages(ctx, msg); err != nil {
			slog.Warn("dms operations consumer commit", "err", err)
		}
	}
}

func (c *Operations) Close() error {
	if c.reader == nil {
		return nil
	}
	return c.reader.Close()
}

func (c *Operations) handle(ctx context.Context, raw []byte) error {
	var env envelopeOps
	if err := json.Unmarshal(raw, &env); err != nil {
		return err
	}
	if env.Source == "iag.dms" {
		return nil
	}
	switch env.Type {
	case "warehouse.pick.confirmed":
		if c.repo != nil {
			if err := c.repo.ApplyOperationsEvent(ctx, env.Type, env.Data); err != nil {
				return err
			}
			_, _ = c.repo.AppendAudit(ctx, "InboundEvent", env.Type, "kafka-operations-consumer")
		}
		slog.Debug("dms operations event applied", "type", env.Type)
	}
	return nil
}
