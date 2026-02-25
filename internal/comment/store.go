package comment

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/wisbric/ticketowl/internal/db"
)

// Store provides database operations for internal notes.
type Store struct {
	dbtx db.DBTX
}

// NewStore creates a comment Store backed by the given database connection.
func NewStore(dbtx db.DBTX) *Store {
	return &Store{dbtx: dbtx}
}

// ListByTicketMetaID returns all internal notes for a ticket, ordered by created_at ascending.
func (s *Store) ListByTicketMetaID(ctx context.Context, ticketMetaID uuid.UUID) ([]InternalNote, error) {
	rows, err := s.dbtx.Query(ctx,
		`SELECT id, ticket_meta_id, author_id, author_name, body, created_at, updated_at
		 FROM internal_notes
		 WHERE ticket_meta_id = $1
		 ORDER BY created_at ASC`, ticketMetaID)
	if err != nil {
		return nil, fmt.Errorf("querying internal_notes: %w", err)
	}
	defer rows.Close()

	var notes []InternalNote
	for rows.Next() {
		var n InternalNote
		if err := rows.Scan(&n.ID, &n.TicketMetaID, &n.AuthorID, &n.AuthorName, &n.Body, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning internal_note: %w", err)
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// Create inserts a new internal note.
func (s *Store) Create(ctx context.Context, note *InternalNote) error {
	_, err := s.dbtx.Exec(ctx,
		`INSERT INTO internal_notes (id, ticket_meta_id, author_id, author_name, body)
		 VALUES ($1, $2, $3, $4, $5)`,
		note.ID, note.TicketMetaID, note.AuthorID, note.AuthorName, note.Body)
	if err != nil {
		return fmt.Errorf("inserting internal_note: %w", err)
	}
	return nil
}

// GetTicketMetaID resolves a Zammad ticket ID to a TicketOwl ticket_meta UUID.
func (s *Store) GetTicketMetaID(ctx context.Context, zammadID int) (uuid.UUID, error) {
	var metaID uuid.UUID
	err := s.dbtx.QueryRow(ctx,
		`SELECT id FROM ticket_meta WHERE zammad_id = $1`, zammadID).Scan(&metaID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, pgx.ErrNoRows
		}
		return uuid.Nil, fmt.Errorf("looking up ticket_meta by zammad_id %d: %w", zammadID, err)
	}
	return metaID, nil
}
