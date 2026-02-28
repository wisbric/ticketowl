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
	ListActiveStates(ctx context.Context) ([]ActiveSLAState, error)
	GetPolicyByID(ctx context.Context, id uuid.UUID) (*sla.Policy, error)
	UpsertState(ctx context.Context, state *sla.State) error
	GetTicketMetaByID(ctx context.Context, metaID uuid.UUID) (*TicketMetaInfo, error)
}

// ActiveSLAState pairs an SLA state with its policy's warning threshold
// for warning label computation.
type ActiveSLAState struct {
	sla.State
	ResponseMinutes   int
	ResolutionMinutes int
	WarningThreshold  float64
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

	// 1. Check for breached SLA states and send alerts.
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

	// 2. Update on_track states to warning when approaching breach.
	p.checkWarnings(ctx, now)
}

// checkWarnings transitions on_track SLA states to warning when the
// remaining time drops below the policy's warning threshold.
func (p *SLAPoller) checkWarnings(ctx context.Context, now time.Time) {
	active, err := p.store.ListActiveStates(ctx)
	if err != nil {
		p.logger.Error("listing active SLA states for warning check", "error", err)
		return
	}

	for i := range active {
		if ctx.Err() != nil {
			return
		}

		as := &active[i]
		if as.Label != sla.LabelOnTrack {
			continue
		}

		shouldWarn := false

		// Check response time warning.
		if as.ResponseDueAt != nil && as.ResponseMetAt == nil && as.ResponseMinutes > 0 {
			totalDuration := time.Duration(as.ResponseMinutes) * time.Minute
			warningDuration := time.Duration(float64(totalDuration) * as.WarningThreshold)
			warningAt := as.ResponseDueAt.Add(-warningDuration)
			if now.After(warningAt) {
				shouldWarn = true
			}
		}

		// Check resolution time warning.
		if !shouldWarn && as.ResolutionDueAt != nil && as.ResolutionMinutes > 0 {
			totalDuration := time.Duration(as.ResolutionMinutes) * time.Minute
			warningDuration := time.Duration(float64(totalDuration) * as.WarningThreshold)
			warningAt := as.ResolutionDueAt.Add(-warningDuration)
			if now.After(warningAt) {
				shouldWarn = true
			}
		}

		if shouldWarn {
			as.Label = sla.LabelWarning
			as.UpdatedAt = now
			if err := p.store.UpsertState(ctx, &as.State); err != nil {
				p.logger.Error("updating SLA state to warning",
					"error", err,
					"ticket_meta_id", as.TicketMetaID,
				)
			} else {
				p.logger.Info("SLA state transitioned to warning",
					"ticket_meta_id", as.TicketMetaID,
				)
			}
		}
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
