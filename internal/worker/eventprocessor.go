package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/wisbric/ticketowl/internal/webhook"
)

// EventHandler processes events from the Redis stream.
type EventHandler interface {
	HandleZammadEvent(ctx context.Context, evt webhook.ZammadEvent) error
	HandleNightOwlEvent(ctx context.Context, evt webhook.NightOwlEvent) error
}

// EventProcessor reads events from the Redis stream via XREAD and dispatches
// them to the appropriate handler. It blocks on XREAD rather than polling
// on a timer.
type EventProcessor struct {
	rdb       *redis.Client
	streamKey string
	handler   EventHandler
	logger    *slog.Logger
	blockTime time.Duration
}

// NewEventProcessor creates an event processor.
func NewEventProcessor(rdb *redis.Client, streamKey string, handler EventHandler, logger *slog.Logger) *EventProcessor {
	return &EventProcessor{
		rdb:       rdb,
		streamKey: streamKey,
		handler:   handler,
		logger:    logger,
		blockTime: 5 * time.Second,
	}
}

// Run starts the event processing loop. It blocks until the context is cancelled.
// Uses XREAD with a block timeout to wait for new events — no timer-based polling.
func (p *EventProcessor) Run(ctx context.Context) {
	p.logger.Info("event processor started", "stream", p.streamKey)

	lastID := "$" // Start from new messages only.

	for {
		if ctx.Err() != nil {
			p.logger.Info("event processor stopping")
			return
		}

		streams, err := p.rdb.XRead(ctx, &redis.XReadArgs{
			Streams: []string{p.streamKey, lastID},
			Block:   p.blockTime,
			Count:   100,
		}).Result()

		if err != nil {
			if errors.Is(err, redis.Nil) || ctx.Err() != nil {
				continue
			}
			p.logger.Error("XREAD error", "error", err, "stream", p.streamKey)
			// Brief backoff on unexpected errors.
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				lastID = msg.ID
				p.processMessage(ctx, msg)
			}
		}
	}
}

func (p *EventProcessor) processMessage(ctx context.Context, msg redis.XMessage) {
	source, _ := msg.Values["source"].(string)
	data, _ := msg.Values["data"].(string)

	if data == "" {
		p.logger.Warn("empty data in stream message", "id", msg.ID)
		return
	}

	switch source {
	case "zammad":
		var evt webhook.ZammadEvent
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			p.logger.Error("unmarshaling zammad event", "error", err, "id", msg.ID)
			return
		}
		if err := p.handler.HandleZammadEvent(ctx, evt); err != nil {
			p.logger.Error("handling zammad event", "error", err, "event", evt.Event, "id", msg.ID)
		}

	case "nightowl":
		var evt webhook.NightOwlEvent
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			p.logger.Error("unmarshaling nightowl event", "error", err, "id", msg.ID)
			return
		}
		if err := p.handler.HandleNightOwlEvent(ctx, evt); err != nil {
			p.logger.Error("handling nightowl event", "error", err, "event", evt.Event, "id", msg.ID)
		}

	default:
		p.logger.Warn("unknown event source", "source", source, "id", msg.ID)
	}
}
