package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/wisbric/ticketowl/internal/notification"
)

// Worker orchestrates the SLA poller and event processor goroutines.
// It shuts down cleanly when the context is cancelled.
type Worker struct {
	slaPoller      *SLAPoller
	eventProcessor *EventProcessor
	logger         *slog.Logger
}

// Config holds configuration for the worker.
type Config struct {
	PollIntervalSeconds int
	PauseStates         []string
	TenantSlug          string
}

// New creates a Worker with its sub-components.
func New(
	rdb *redis.Client,
	pollerStore SLAPollerStore,
	notifier *notification.Service,
	eventHandler EventHandler,
	cfg Config,
	logger *slog.Logger,
) *Worker {
	pollInterval := time.Duration(cfg.PollIntervalSeconds) * time.Second
	if pollInterval == 0 {
		pollInterval = 60 * time.Second
	}

	poller := NewSLAPoller(pollerStore, notifier, logger, pollInterval, cfg.PauseStates)

	streamKey := "ticketowl:" + cfg.TenantSlug + ":events"
	processor := NewEventProcessor(rdb, streamKey, eventHandler, logger)

	return &Worker{
		slaPoller:      poller,
		eventProcessor: processor,
		logger:         logger,
	}
}

// Run starts all worker goroutines and blocks until the context is cancelled.
// All goroutines shut down cleanly when the context is done.
func (w *Worker) Run(ctx context.Context) error {
	w.logger.Info("worker starting")

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		w.slaPoller.Run(ctx)
	}()

	go func() {
		defer wg.Done()
		w.eventProcessor.Run(ctx)
	}()

	<-ctx.Done()
	w.logger.Info("worker shutting down, waiting for goroutines")
	wg.Wait()
	w.logger.Info("worker stopped")

	return nil
}
