package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wisbric/ticketowl/internal/sla"
)

// PollerStore implements SLAPollerStore using a connection pool.
// Each method acquires a connection, sets the tenant search_path,
// executes the query, and releases the connection.
type PollerStore struct {
	pool   *pgxpool.Pool
	schema string
}

// NewPollerStore creates a PollerStore for the given tenant schema.
func NewPollerStore(pool *pgxpool.Pool, schema string) *PollerStore {
	return &PollerStore{pool: pool, schema: schema}
}

// ListBreachedTickets returns SLA states that have breached but not yet been alerted.
func (s *PollerStore) ListBreachedTickets(ctx context.Context, now time.Time) ([]sla.State, error) {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", s.schema)); err != nil {
		return nil, fmt.Errorf("setting search_path: %w", err)
	}

	rows, err := conn.Query(ctx,
		`SELECT id, ticket_meta_id, response_due_at, resolution_due_at,
		        response_met_at, first_breach_alerted_at, state, paused,
		        paused_at, accumulated_pause_secs, updated_at
		 FROM sla_states
		 WHERE first_breach_alerted_at IS NULL
		   AND paused = false
		   AND (
		     (response_due_at IS NOT NULL AND response_due_at < $1 AND response_met_at IS NULL)
		     OR
		     (resolution_due_at IS NOT NULL AND resolution_due_at < $1)
		   )`, now)
	if err != nil {
		return nil, fmt.Errorf("querying breached sla_states: %w", err)
	}
	defer rows.Close()

	var states []sla.State
	for rows.Next() {
		var st sla.State
		if err := rows.Scan(
			&st.ID, &st.TicketMetaID, &st.ResponseDueAt, &st.ResolutionDueAt,
			&st.ResponseMetAt, &st.FirstBreachAlertedAt, &st.Label,
			&st.Paused, &st.PausedAt, &st.AccumulatedPauseSecs, &st.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning sla_state: %w", err)
		}
		states = append(states, st)
	}
	return states, rows.Err()
}

// GetPolicyByID returns an SLA policy by its ID.
func (s *PollerStore) GetPolicyByID(ctx context.Context, id uuid.UUID) (*sla.Policy, error) {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", s.schema)); err != nil {
		return nil, fmt.Errorf("setting search_path: %w", err)
	}

	var p sla.Policy
	err = conn.QueryRow(ctx,
		`SELECT id, name, priority, response_minutes, resolution_minutes,
		        warning_threshold, is_default, created_at, updated_at
		 FROM sla_policies WHERE id = $1`, id).
		Scan(&p.ID, &p.Name, &p.Priority, &p.ResponseMinutes, &p.ResolutionMinutes,
			&p.WarningThreshold, &p.IsDefault, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("querying sla_policy: %w", err)
	}
	return &p, nil
}

// UpsertState inserts or updates an SLA state.
func (s *PollerStore) UpsertState(ctx context.Context, state *sla.State) error {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", s.schema)); err != nil {
		return fmt.Errorf("setting search_path: %w", err)
	}

	_, err = conn.Exec(ctx,
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

// GetTicketMetaByID returns minimal ticket metadata for the SLA poller.
func (s *PollerStore) GetTicketMetaByID(ctx context.Context, metaID uuid.UUID) (*TicketMetaInfo, error) {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", s.schema)); err != nil {
		return nil, fmt.Errorf("setting search_path: %w", err)
	}

	var m TicketMetaInfo
	err = conn.QueryRow(ctx,
		`SELECT id, zammad_id, zammad_number, sla_policy_id, created_at
		 FROM ticket_meta WHERE id = $1`, metaID).
		Scan(&m.ID, &m.ZammadID, &m.ZammadNumber, &m.SLAPolicyID, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("querying ticket_meta: %w", err)
	}
	return &m, nil
}
