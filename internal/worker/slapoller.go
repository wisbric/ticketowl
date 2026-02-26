package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/wisbric/ticketowl/internal/notification"
	"github.com/wisbric/ticketowl/internal/sla"
	"github.com/wisbric/ticketowl/internal/telemetry"
)

// SLAPollerStore defines the database operations the SLA poller needs.
type SLAPollerStore interface {
	ListBreachedTickets(ctx context.Context, now time.Time) ([]sla.State, error)
	GetPolicyByID(ctx context.Context, id uuid.UUID) (*sla.Policy, error)
	UpsertState(ctx context.Context, state *sla.State) error
	GetTicketMetaByID(ctx context.Context, metaID uuid.UUID) (*TicketMetaInfo, error)
}

// TicketMetaInfo holds the minimal ticket info the poller needs.
type TicketMetaInfo struct {
	ID           uuid.UUID
	ZammadID     int
	ZammadNumber string
	SLAPolicyID  *uuid.UUID
	CreatedAt    time.Time
}

// SLAPoller periodically checks SLA states and fires breach alerts.
type SLAPoller struct {
	store        SLAPollerStore
	notifier     *notification.Service
	logger       *slog.Logger
	pollInterval time.Duration
	pauseStates  []string // Zammad states that pause the SLA clock
}

// NewSLAPoller creates an SLA poller.
func NewSLAPoller(store SLAPollerStore, notifier *notification.Service, logger *slog.Logger, pollInterval time.Duration, pauseStates []string) *SLAPoller {
	return &SLAPoller{
		store:        store,
		notifier:     notifier,
		logger:       logger,
		pollInterval: pollInterval,
		pauseStates:  pauseStates,
	}
}

// Run starts the SLA polling loop. It blocks until the context is cancelled.
func (p *SLAPoller) Run(ctx context.Context) {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	p.logger.Info("SLA poller started", "interval", p.pollInterval)

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("SLA poller stopping")
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

// PollOnce runs a single poll cycle. Exported for testing.
func (p *SLAPoller) PollOnce(ctx context.Context) {
	p.poll(ctx)
}

func (p *SLAPoller) poll(ctx context.Context) {
	start := time.Now()
	defer func() {
		telemetry.WorkerPollDuration.Observe(time.Since(start).Seconds())
	}()

	now := time.Now()
	states, err := p.store.ListBreachedTickets(ctx, now)
	if err != nil {
		p.logger.Error("listing breached SLA states", "error", err)
		return
	}

	for i := range states {
		if ctx.Err() != nil {
			return
		}
		p.processBreachedState(ctx, &states[i], now)
	}
}

func (p *SLAPoller) processBreachedState(ctx context.Context, state *sla.State, now time.Time) {
	if state.FirstBreachAlertedAt != nil {
		// Already alerted — update timestamps for consistency but skip re-alerting.
		state.UpdatedAt = now
		if err := p.store.UpsertState(ctx, state); err != nil {
			p.logger.Error("updating SLA state (already alerted)",
				"error", err,
				"ticket_meta_id", state.TicketMetaID,
			)
		}
		return
	}

	meta, err := p.store.GetTicketMetaByID(ctx, state.TicketMetaID)
	if err != nil {
		p.logger.Error("getting ticket meta for SLA breach",
			"error", err,
			"ticket_meta_id", state.TicketMetaID,
		)
		return
	}

	if meta.SLAPolicyID == nil {
		return
	}

	policy, err := p.store.GetPolicyByID(ctx, *meta.SLAPolicyID)
	if err != nil {
		p.logger.Error("getting SLA policy for breach",
			"error", err,
			"policy_id", meta.SLAPolicyID,
		)
		return
	}

	telemetry.SLABreachesTotal.Inc()

	breachType := "resolution"
	if state.ResponseDueAt != nil && state.ResponseDueAt.Before(now) && state.ResponseMetAt == nil {
		breachType = "response"
	}

	err = p.notifier.AlertSLABreach(ctx, notification.BreachInfo{
		TicketZammadID: meta.ZammadID,
		TicketNumber:   meta.ZammadNumber,
		SLAType:        breachType,
		Priority:       policy.Priority,
	})
	if err != nil {
		p.logger.Error("sending SLA breach alert",
			"error", err,
			"ticket_number", meta.ZammadNumber,
		)
		return
	}

	// Set first_breach_alerted_at to prevent duplicate pages.
	state.Label = sla.LabelBreached
	state.FirstBreachAlertedAt = &now
	state.UpdatedAt = now

	if err := p.store.UpsertState(ctx, state); err != nil {
		p.logger.Error("updating SLA state after breach alert",
			"error", err,
			"ticket_meta_id", state.TicketMetaID,
		)
	}
}
