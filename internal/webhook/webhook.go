package webhook

import (
	"time"
)

// ZammadEvent wraps a Zammad webhook payload for the Redis stream.
type ZammadEvent struct {
	Event    string `json:"event"` // "ticket.create" | "ticket.update" | "article.create"
	TicketID int    `json:"ticket_id"`
	Number   string `json:"number"`
	StateID  int    `json:"state_id"`
	State    string `json:"state"`
	Priority string `json:"priority"`
}

// NightOwlEvent wraps a NightOwl webhook payload for the Redis stream.
type NightOwlEvent struct {
	Event      string     `json:"event"` // "incident.created" | "incident.resolved"
	IncidentID string     `json:"incident_id"`
	Slug       string     `json:"slug"`
	Summary    string     `json:"summary"`
	Severity   string     `json:"severity"`
	Status     string     `json:"status"`
	Service    string     `json:"service"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

// StreamKey returns the Redis stream key for a tenant.
func StreamKey(tenantSlug string) string {
	return "ticketowl:" + tenantSlug + ":events"
}
