package sla

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/wisbric/ticketowl/internal/db"
)

// Store provides database operations for SLA policies and states.
type Store struct {
	dbtx db.DBTX
}

// NewStore creates an SLA Store backed by the given database connection.
func NewStore(dbtx db.DBTX) *Store {
	return &Store{dbtx: dbtx}
}

// --- SLA Policies ---

// ListPolicies returns all SLA policies ordered by priority.
func (s *Store) ListPolicies(ctx context.Context) ([]Policy, error) {
	rows, err := s.dbtx.Query(ctx,
		`SELECT id, name, priority, response_minutes, resolution_minutes,
		        warning_threshold, is_default, created_at, updated_at
		 FROM sla_policies ORDER BY priority, name`)
	if err != nil {
		return nil, fmt.Errorf("querying sla_policies: %w", err)
	}
	defer rows.Close()

	var policies []Policy
	for rows.Next() {
		p, err := scanPolicy(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning sla_policy: %w", err)
		}
		policies = append(policies, *p)
	}
	return policies, rows.Err()
}

// GetPolicy returns a single SLA policy by ID.
func (s *Store) GetPolicy(ctx context.Context, id uuid.UUID) (*Policy, error) {
	row := s.dbtx.QueryRow(ctx,
		`SELECT id, name, priority, response_minutes, resolution_minutes,
		        warning_threshold, is_default, created_at, updated_at
		 FROM sla_policies WHERE id = $1`, id)
	return scanPolicyRow(row)
}

// GetDefaultPolicyByPriority returns the default policy for a given priority.
func (s *Store) GetDefaultPolicyByPriority(ctx context.Context, priority string) (*Policy, error) {
	row := s.dbtx.QueryRow(ctx,
		`SELECT id, name, priority, response_minutes, resolution_minutes,
		        warning_threshold, is_default, created_at, updated_at
		 FROM sla_policies WHERE priority = $1 AND is_default = true`, priority)
	return scanPolicyRow(row)
}

// CreatePolicy inserts a new SLA policy.
func (s *Store) CreatePolicy(ctx context.Context, req CreatePolicyRequest) (*Policy, error) {
	row := s.dbtx.QueryRow(ctx,
		`INSERT INTO sla_policies (name, priority, response_minutes, resolution_minutes, warning_threshold, is_default)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, name, priority, response_minutes, resolution_minutes,
		           warning_threshold, is_default, created_at, updated_at`,
		req.Name, req.Priority, req.ResponseMinutes, req.ResolutionMinutes, req.WarningThreshold, req.IsDefault)
	return scanPolicyRow(row)
}

// UpdatePolicy updates an existing SLA policy.
func (s *Store) UpdatePolicy(ctx context.Context, id uuid.UUID, req UpdatePolicyRequest) (*Policy, error) {
	row := s.dbtx.QueryRow(ctx,
		`UPDATE sla_policies SET
		   name = COALESCE($2, name),
		   response_minutes = COALESCE($3, response_minutes),
		   resolution_minutes = COALESCE($4, resolution_minutes),
		   warning_threshold = COALESCE($5, warning_threshold),
		   is_default = COALESCE($6, is_default),
		   updated_at = now()
		 WHERE id = $1
		 RETURNING id, name, priority, response_minutes, resolution_minutes,
		           warning_threshold, is_default, created_at, updated_at`,
		id, req.Name, req.ResponseMinutes, req.ResolutionMinutes, req.WarningThreshold, req.IsDefault)
	return scanPolicyRow(row)
}

// DeletePolicy deletes an SLA policy.
func (s *Store) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	tag, err := s.dbtx.Exec(ctx, `DELETE FROM sla_policies WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting sla_policy: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func scanPolicy(rows pgx.Rows) (*Policy, error) {
	var p Policy
	err := rows.Scan(&p.ID, &p.Name, &p.Priority, &p.ResponseMinutes, &p.ResolutionMinutes,
		&p.WarningThreshold, &p.IsDefault, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func scanPolicyRow(row pgx.Row) (*Policy, error) {
	var p Policy
	err := row.Scan(&p.ID, &p.Name, &p.Priority, &p.ResponseMinutes, &p.ResolutionMinutes,
		&p.WarningThreshold, &p.IsDefault, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// --- SLA States ---

// GetStateByZammadID returns the SLA state for a ticket identified by its Zammad ID.
func (s *Store) GetStateByZammadID(ctx context.Context, zammadID int) (*State, error) {
	var st State
	err := s.dbtx.QueryRow(ctx,
		`SELECT ss.id, ss.ticket_meta_id, ss.response_due_at, ss.resolution_due_at,
		        ss.response_met_at, ss.first_breach_alerted_at, ss.state, ss.paused,
		        ss.paused_at, ss.accumulated_pause_secs, ss.updated_at
		 FROM sla_states ss
		 JOIN ticket_meta tm ON tm.id = ss.ticket_meta_id
		 WHERE tm.zammad_id = $1`, zammadID).
		Scan(&st.ID, &st.TicketMetaID, &st.ResponseDueAt, &st.ResolutionDueAt,
			&st.ResponseMetAt, &st.FirstBreachAlertedAt, &st.Label,
			&st.Paused, &st.PausedAt, &st.AccumulatedPauseSecs, &st.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// GetStateByTicketMetaID returns the SLA state for a ticket.
func (s *Store) GetStateByTicketMetaID(ctx context.Context, ticketMetaID uuid.UUID) (*State, error) {
	var st State
	err := s.dbtx.QueryRow(ctx,
		`SELECT id, ticket_meta_id, response_due_at, resolution_due_at,
		        response_met_at, first_breach_alerted_at, state, paused,
		        paused_at, accumulated_pause_secs, updated_at
		 FROM sla_states WHERE ticket_meta_id = $1`, ticketMetaID).
		Scan(&st.ID, &st.TicketMetaID, &st.ResponseDueAt, &st.ResolutionDueAt,
			&st.ResponseMetAt, &st.FirstBreachAlertedAt, &st.Label,
			&st.Paused, &st.PausedAt, &st.AccumulatedPauseSecs, &st.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// UpsertState inserts or updates the SLA state for a ticket.
func (s *Store) UpsertState(ctx context.Context, state *State) error {
	_, err := s.dbtx.Exec(ctx,
		`INSERT INTO sla_states (id, ticket_meta_id, response_due_at, resolution_due_at,
		   response_met_at, first_breach_alerted_at, state, paused, paused_at,
		   accumulated_pause_secs, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (ticket_meta_id) DO UPDATE SET
		   response_due_at = EXCLUDED.response_due_at,
		   resolution_due_at = EXCLUDED.resolution_due_at,
		   response_met_at = EXCLUDED.response_met_at,
		   first_breach_alerted_at = EXCLUDED.first_breach_alerted_at,
		   state = EXCLUDED.state,
		   paused = EXCLUDED.paused,
		   paused_at = EXCLUDED.paused_at,
		   accumulated_pause_secs = EXCLUDED.accumulated_pause_secs,
		   updated_at = EXCLUDED.updated_at`,
		state.ID, state.TicketMetaID, state.ResponseDueAt, state.ResolutionDueAt,
		state.ResponseMetAt, state.FirstBreachAlertedAt, state.Label,
		state.Paused, state.PausedAt, state.AccumulatedPauseSecs, state.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upserting sla_state: %w", err)
	}
	return nil
}
