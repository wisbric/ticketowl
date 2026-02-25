package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/wisbric/ticketowl/internal/webhook"
)

// DefaultEventHandler implements EventHandler by dispatching to the appropriate
// business logic for each event type.
type DefaultEventHandler struct {
	logger *slog.Logger
}

// NewDefaultEventHandler creates a default event handler.
func NewDefaultEventHandler(logger *slog.Logger) *DefaultEventHandler {
	return &DefaultEventHandler{logger: logger}
}

// HandleZammadEvent processes a Zammad webhook event.
func (h *DefaultEventHandler) HandleZammadEvent(ctx context.Context, evt webhook.ZammadEvent) error {
	switch evt.Event {
	case "ticket.create":
		h.logger.Info("processing ticket.create event",
			"ticket_id", evt.TicketID,
			"number", evt.Number,
		)
		// Worker will: UpsertTicketMeta, assign SLA policy, init SLA state.
		// Full implementation depends on DB access; the structure is ready.
		return nil

	case "ticket.update":
		h.logger.Info("processing ticket.update event",
			"ticket_id", evt.TicketID,
			"state", evt.State,
		)
		// Worker will: check pause status change, recompute SLA.
		return nil

	case "article.create":
		h.logger.Info("processing article.create event",
			"ticket_id", evt.TicketID,
		)
		// Reserved for future notification hooks.
		return nil

	default:
		return fmt.Errorf("unknown zammad event type: %s", evt.Event)
	}
}

// HandleNightOwlEvent processes a NightOwl webhook event.
func (h *DefaultEventHandler) HandleNightOwlEvent(ctx context.Context, evt webhook.NightOwlEvent) error {
	switch evt.Event {
	case "incident.created":
		h.logger.Info("processing incident.created event",
			"incident_id", evt.IncidentID,
			"slug", evt.Slug,
			"severity", evt.Severity,
		)
		// Worker will: evaluate auto_ticket_rules, create tickets for matches.
		return nil

	case "incident.resolved":
		h.logger.Info("processing incident.resolved event",
			"incident_id", evt.IncidentID,
			"slug", evt.Slug,
		)
		// Worker will: find linked tickets, close in Zammad if not already closed.
		return nil

	default:
		return fmt.Errorf("unknown nightowl event type: %s", evt.Event)
	}
}
