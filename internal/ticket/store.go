package ticket

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/wisbric/ticketowl/internal/db"
)

// Store provides database operations for ticket metadata.
type Store struct {
	dbtx db.DBTX
}

// NewStore creates a ticket Store backed by the given database connection.
func NewStore(dbtx db.DBTX) *Store {
	return &Store{dbtx: dbtx}
}

// GetByZammadID returns the metadata for a ticket by its Zammad ID.
func (s *Store) GetByZammadID(ctx context.Context, zammadID int) (*Meta, error) {
	row := s.dbtx.QueryRow(ctx,
		`SELECT id, zammad_id, zammad_number, sla_policy_id, created_at, updated_at
		 FROM ticket_meta WHERE zammad_id = $1`, zammadID)
	return scanMeta(row)
}

// Upsert inserts or updates ticket metadata. Returns the resulting row.
func (s *Store) Upsert(ctx context.Context, zammadID int, zammadNumber string, slaPolicyID *uuid.UUID) (*Meta, error) {
	row := s.dbtx.QueryRow(ctx,
		`INSERT INTO ticket_meta (zammad_id, zammad_number, sla_policy_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (zammad_id) DO UPDATE SET
		   zammad_number = EXCLUDED.zammad_number,
		   sla_policy_id = COALESCE(EXCLUDED.sla_policy_id, ticket_meta.sla_policy_id),
		   updated_at = now()
		 RETURNING id, zammad_id, zammad_number, sla_policy_id, created_at, updated_at`,
		zammadID, zammadNumber, slaPolicyID)
	return scanMeta(row)
}

// GetByID returns the metadata for a ticket by its internal UUID.
func (s *Store) GetByID(ctx context.Context, id uuid.UUID) (*Meta, error) {
	row := s.dbtx.QueryRow(ctx,
		`SELECT id, zammad_id, zammad_number, sla_policy_id, created_at, updated_at
		 FROM ticket_meta WHERE id = $1`, id)
	return scanMeta(row)
}

// ListByZammadIDs returns metadata for multiple Zammad tickets.
func (s *Store) ListByZammadIDs(ctx context.Context, zammadIDs []int) ([]Meta, error) {
	rows, err := s.dbtx.Query(ctx,
		`SELECT id, zammad_id, zammad_number, sla_policy_id, created_at, updated_at
		 FROM ticket_meta WHERE zammad_id = ANY($1)`, zammadIDs)
	if err != nil {
		return nil, fmt.Errorf("querying ticket_meta: %w", err)
	}
	defer rows.Close()

	var metas []Meta
	for rows.Next() {
		m, err := scanMetaRows(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning ticket_meta: %w", err)
		}
		metas = append(metas, *m)
	}
	return metas, rows.Err()
}

func scanMeta(row pgx.Row) (*Meta, error) {
	var m Meta
	var zammadID int32
	err := row.Scan(&m.ID, &zammadID, &m.ZammadNumber, &m.SLAPolicyID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	m.ZammadID = int(zammadID)
	return &m, nil
}

func scanMetaRows(rows pgx.Rows) (*Meta, error) {
	var m Meta
	var zammadID int32
	err := rows.Scan(&m.ID, &zammadID, &m.ZammadNumber, &m.SLAPolicyID, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	m.ZammadID = int(zammadID)
	return &m, nil
}

// MetaRow is used by tests to build mocks.
type MetaRow struct {
	ID           uuid.UUID
	ZammadID     int32
	ZammadNumber string
	SLAPolicyID  *uuid.UUID
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
