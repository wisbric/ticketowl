package sla

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrPolicyNotFound is returned when a referenced SLA policy does not exist.
var ErrPolicyNotFound = errors.New("sla policy not found")

// StateLabel represents the SLA state of a ticket.
type StateLabel string

const (
	LabelOnTrack  StateLabel = "on_track"
	LabelWarning  StateLabel = "warning"
	LabelBreached StateLabel = "breached"
	LabelMet      StateLabel = "met"
)

// Policy represents an SLA policy from the database.
type Policy struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	Priority          string    `json:"priority"`
	ResponseMinutes   int       `json:"response_minutes"`
	ResolutionMinutes int       `json:"resolution_minutes"`
	WarningThreshold  float64   `json:"warning_threshold"` // e.g. 0.20 = warn at 20% remaining
	IsDefault         bool      `json:"is_default"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// State represents the persisted SLA state for a ticket.
type State struct {
	ID                   uuid.UUID  `json:"id"`
	TicketMetaID         uuid.UUID  `json:"ticket_meta_id"`
	ResponseDueAt        *time.Time `json:"response_due_at,omitempty"`
	ResolutionDueAt      *time.Time `json:"resolution_due_at,omitempty"`
	ResponseMetAt        *time.Time `json:"response_met_at,omitempty"`
	FirstBreachAlertedAt *time.Time `json:"first_breach_alerted_at,omitempty"`
	Label                StateLabel `json:"state"`
	Paused               bool       `json:"paused"`
	PausedAt             *time.Time `json:"paused_at,omitempty"`
	AccumulatedPauseSecs int        `json:"accumulated_pause_secs"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// ComputeInput holds the inputs for the pure SLA computation.
type ComputeInput struct {
	ResponseMinutes   int
	ResolutionMinutes int
	WarningThreshold  float64
	TicketCreatedAt   time.Time
	ResponseMetAt     *time.Time
	AccumulatedPause  int // seconds
	Paused            bool
	Now               time.Time
}

// ComputeResult holds the output of the pure SLA computation.
type ComputeResult struct {
	Label                      StateLabel
	ResponseDueAt              time.Time
	ResolutionDueAt            time.Time
	ResponseSecondsRemaining   int
	ResolutionSecondsRemaining int
	Paused                     bool
}

// CreatePolicyRequest is the payload for creating an SLA policy.
type CreatePolicyRequest struct {
	Name              string  `json:"name"`
	Priority          string  `json:"priority"`
	ResponseMinutes   int     `json:"response_minutes"`
	ResolutionMinutes int     `json:"resolution_minutes"`
	WarningThreshold  float64 `json:"warning_threshold"`
	IsDefault         bool    `json:"is_default"`
}

// UpdatePolicyRequest is the payload for updating an SLA policy.
type UpdatePolicyRequest struct {
	Name              *string  `json:"name,omitempty"`
	ResponseMinutes   *int     `json:"response_minutes,omitempty"`
	ResolutionMinutes *int     `json:"resolution_minutes,omitempty"`
	WarningThreshold  *float64 `json:"warning_threshold,omitempty"`
	IsDefault         *bool    `json:"is_default,omitempty"`
}
