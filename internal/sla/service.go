package sla

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// ComputeState is a pure function that computes the current SLA state.
// It has no side effects and does not access the database or any external service.
func ComputeState(in ComputeInput) ComputeResult {
	responseDue := in.TicketCreatedAt.Add(time.Duration(in.ResponseMinutes) * time.Minute)
	resolutionDue := in.TicketCreatedAt.Add(time.Duration(in.ResolutionMinutes) * time.Minute)

	result := ComputeResult{
		ResponseDueAt:   responseDue,
		ResolutionDueAt: resolutionDue,
		Paused:          in.Paused,
	}

	// If paused, the clock is stopped — report current remaining without advancing.
	if in.Paused {
		result.Label = LabelOnTrack
		respRemaining := remainingSeconds(responseDue, in.TicketCreatedAt, in.AccumulatedPause, in.Now)
		resolRemaining := remainingSeconds(resolutionDue, in.TicketCreatedAt, in.AccumulatedPause, in.Now)
		result.ResponseSecondsRemaining = respRemaining
		result.ResolutionSecondsRemaining = resolRemaining
		return result
	}

	// Effective elapsed = (now - created_at) - accumulated_pause_secs
	totalElapsed := in.Now.Sub(in.TicketCreatedAt).Seconds()
	effectiveElapsed := totalElapsed - float64(in.AccumulatedPause)
	if effectiveElapsed < 0 {
		effectiveElapsed = 0
	}

	responseTotalSecs := float64(in.ResponseMinutes) * 60
	resolutionTotalSecs := float64(in.ResolutionMinutes) * 60

	responseRemaining := responseTotalSecs - effectiveElapsed
	resolutionRemaining := resolutionTotalSecs - effectiveElapsed

	result.ResponseSecondsRemaining = max(0, int(responseRemaining))
	result.ResolutionSecondsRemaining = max(0, int(resolutionRemaining))

	// Check if both response and resolution are met.
	// "met" = response_met_at is set, response was on-time, and resolution deadline
	// has not been exceeded. The caller signals resolution completion by ensuring
	// the current time is within the resolution window.
	responseMet := in.ResponseMetAt != nil
	resolutionMet := false

	if responseMet && resolutionRemaining > 0 {
		respEffective := in.ResponseMetAt.Sub(in.TicketCreatedAt).Seconds() - float64(in.AccumulatedPause)
		if respEffective <= responseTotalSecs {
			resolutionMet = true
		}
	}

	if responseMet && resolutionMet {
		result.Label = LabelMet
		return result
	}

	// Check breach: either response or resolution deadline exceeded.
	responseBreach := !responseMet && responseRemaining <= 0
	resolutionBreach := resolutionRemaining <= 0

	if responseBreach || resolutionBreach {
		result.Label = LabelBreached
		return result
	}

	// Check warning: remaining time < warning_threshold * total_time.
	// Use the most urgent (smallest remaining ratio).
	if !responseMet {
		warningThreshold := in.WarningThreshold * responseTotalSecs
		if responseRemaining < warningThreshold {
			result.Label = LabelWarning
			return result
		}
	}

	resolutionWarningThreshold := in.WarningThreshold * resolutionTotalSecs
	if resolutionRemaining < resolutionWarningThreshold {
		result.Label = LabelWarning
		return result
	}

	result.Label = LabelOnTrack
	return result
}

func remainingSeconds(due, created time.Time, pauseSecs int, now time.Time) int {
	totalSecs := due.Sub(created).Seconds()
	effectiveElapsed := now.Sub(created).Seconds() - float64(pauseSecs)
	if effectiveElapsed < 0 {
		effectiveElapsed = 0
	}
	remaining := totalSecs - effectiveElapsed
	if remaining < 0 {
		return 0
	}
	return int(remaining)
}

// Service encapsulates SLA business logic.
type Service struct {
	store  SLAStore
	logger *slog.Logger
}

// SLAStore defines the database operations the SLA service needs.
type SLAStore interface {
	ListPolicies(ctx context.Context) ([]Policy, error)
	GetPolicy(ctx context.Context, id uuid.UUID) (*Policy, error)
	GetDefaultPolicyByPriority(ctx context.Context, priority string) (*Policy, error)
	CreatePolicy(ctx context.Context, req CreatePolicyRequest) (*Policy, error)
	UpdatePolicy(ctx context.Context, id uuid.UUID, req UpdatePolicyRequest) (*Policy, error)
	DeletePolicy(ctx context.Context, id uuid.UUID) error
	GetStateByTicketMetaID(ctx context.Context, ticketMetaID uuid.UUID) (*State, error)
	GetStateByZammadID(ctx context.Context, zammadID int) (*State, error)
	UpsertState(ctx context.Context, state *State) error
}

// NewService creates an SLA Service.
func NewService(store SLAStore, logger *slog.Logger) *Service {
	return &Service{store: store, logger: logger}
}

// ListPolicies returns all SLA policies.
func (s *Service) ListPolicies(ctx context.Context) ([]Policy, error) {
	policies, err := s.store.ListPolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing SLA policies: %w", err)
	}
	return policies, nil
}

// GetPolicy returns a single SLA policy by ID.
func (s *Service) GetPolicy(ctx context.Context, id uuid.UUID) (*Policy, error) {
	policy, err := s.store.GetPolicy(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting SLA policy: %w", err)
	}
	return policy, nil
}

// CreatePolicy creates a new SLA policy.
func (s *Service) CreatePolicy(ctx context.Context, req CreatePolicyRequest) (*Policy, error) {
	policy, err := s.store.CreatePolicy(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("creating SLA policy: %w", err)
	}
	return policy, nil
}

// UpdatePolicy updates an SLA policy.
func (s *Service) UpdatePolicy(ctx context.Context, id uuid.UUID, req UpdatePolicyRequest) (*Policy, error) {
	policy, err := s.store.UpdatePolicy(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("updating SLA policy: %w", err)
	}
	return policy, nil
}

// DeletePolicy deletes an SLA policy.
func (s *Service) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	if err := s.store.DeletePolicy(ctx, id); err != nil {
		return fmt.Errorf("deleting SLA policy: %w", err)
	}
	return nil
}

// GetTicketSLA returns the SLA state for a ticket by meta ID.
func (s *Service) GetTicketSLA(ctx context.Context, ticketMetaID uuid.UUID) (*State, error) {
	state, err := s.store.GetStateByTicketMetaID(ctx, ticketMetaID)
	if err != nil {
		return nil, fmt.Errorf("getting SLA state: %w", err)
	}
	return state, nil
}

// GetTicketSLAByZammadID returns the SLA state for a ticket by Zammad ID.
func (s *Service) GetTicketSLAByZammadID(ctx context.Context, zammadID int) (*State, error) {
	state, err := s.store.GetStateByZammadID(ctx, zammadID)
	if err != nil {
		return nil, fmt.Errorf("getting SLA state by zammad ID: %w", err)
	}
	return state, nil
}

// ApplyPolicy initialises SLA tracking for a ticket with the given policy.
func (s *Service) ApplyPolicy(ctx context.Context, ticketMetaID uuid.UUID, policy *Policy, ticketCreatedAt time.Time) error {
	now := time.Now()
	in := ComputeInput{
		ResponseMinutes:   policy.ResponseMinutes,
		ResolutionMinutes: policy.ResolutionMinutes,
		WarningThreshold:  policy.WarningThreshold,
		TicketCreatedAt:   ticketCreatedAt,
		Now:               now,
	}
	result := ComputeState(in)

	state := &State{
		ID:                   uuid.New(),
		TicketMetaID:         ticketMetaID,
		ResponseDueAt:        &result.ResponseDueAt,
		ResolutionDueAt:      &result.ResolutionDueAt,
		Label:                result.Label,
		AccumulatedPauseSecs: 0,
		UpdatedAt:            now,
	}

	if err := s.store.UpsertState(ctx, state); err != nil {
		return fmt.Errorf("applying SLA policy: %w", err)
	}
	return nil
}

// CheckBreach evaluates the current SLA state and returns true if the ticket
// has newly breached (label is breached and no prior alert was sent).
func CheckBreach(state *State) bool {
	return state.Label == LabelBreached && state.FirstBreachAlertedAt == nil
}
