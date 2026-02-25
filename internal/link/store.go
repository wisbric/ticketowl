package link

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/wisbric/ticketowl/internal/db"
)

// Store provides database operations for ticket links.
type Store struct {
	dbtx db.DBTX
}

// NewStore creates a link Store backed by the given database connection.
func NewStore(dbtx db.DBTX) *Store {
	return &Store{dbtx: dbtx}
}

// GetTicketMetaID returns the ticket_meta ID for a given Zammad ticket ID.
func (s *Store) GetTicketMetaID(ctx context.Context, zammadID int) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.dbtx.QueryRow(ctx,
		`SELECT id FROM ticket_meta WHERE zammad_id = $1`, zammadID).Scan(&id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("looking up ticket_meta for zammad_id %d: %w", zammadID, err)
	}
	return id, nil
}

// --- Incident Links ---

// ListIncidentLinks returns all incident links for a ticket.
func (s *Store) ListIncidentLinks(ctx context.Context, ticketMetaID uuid.UUID) ([]IncidentLink, error) {
	rows, err := s.dbtx.Query(ctx,
		`SELECT id, ticket_meta_id, incident_id, incident_slug, linked_by, created_at
		 FROM incident_links WHERE ticket_meta_id = $1 ORDER BY created_at`, ticketMetaID)
	if err != nil {
		return nil, fmt.Errorf("listing incident links: %w", err)
	}
	defer rows.Close()

	var links []IncidentLink
	for rows.Next() {
		var l IncidentLink
		if err := rows.Scan(&l.ID, &l.TicketMetaID, &l.IncidentID, &l.IncidentSlug, &l.LinkedBy, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning incident link: %w", err)
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

// CreateIncidentLink inserts an incident link.
func (s *Store) CreateIncidentLink(ctx context.Context, ticketMetaID, incidentID, linkedBy uuid.UUID, slug string) (*IncidentLink, error) {
	var l IncidentLink
	err := s.dbtx.QueryRow(ctx,
		`INSERT INTO incident_links (ticket_meta_id, incident_id, incident_slug, linked_by)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, ticket_meta_id, incident_id, incident_slug, linked_by, created_at`,
		ticketMetaID, incidentID, slug, linkedBy).
		Scan(&l.ID, &l.TicketMetaID, &l.IncidentID, &l.IncidentSlug, &l.LinkedBy, &l.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating incident link: %w", err)
	}
	return &l, nil
}

// DeleteIncidentLink removes an incident link.
func (s *Store) DeleteIncidentLink(ctx context.Context, ticketMetaID, incidentID uuid.UUID) error {
	tag, err := s.dbtx.Exec(ctx,
		`DELETE FROM incident_links WHERE ticket_meta_id = $1 AND incident_id = $2`,
		ticketMetaID, incidentID)
	if err != nil {
		return fmt.Errorf("deleting incident link: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// --- Article Links ---

// ListArticleLinks returns all article links for a ticket.
func (s *Store) ListArticleLinks(ctx context.Context, ticketMetaID uuid.UUID) ([]ArticleLink, error) {
	rows, err := s.dbtx.Query(ctx,
		`SELECT id, ticket_meta_id, article_id, article_slug, article_title, linked_by, created_at
		 FROM article_links WHERE ticket_meta_id = $1 ORDER BY created_at`, ticketMetaID)
	if err != nil {
		return nil, fmt.Errorf("listing article links: %w", err)
	}
	defer rows.Close()

	var links []ArticleLink
	for rows.Next() {
		var l ArticleLink
		if err := rows.Scan(&l.ID, &l.TicketMetaID, &l.ArticleID, &l.ArticleSlug, &l.ArticleTitle, &l.LinkedBy, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning article link: %w", err)
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

// CreateArticleLink inserts an article link.
func (s *Store) CreateArticleLink(ctx context.Context, ticketMetaID, articleID, linkedBy uuid.UUID, slug, title string) (*ArticleLink, error) {
	var l ArticleLink
	err := s.dbtx.QueryRow(ctx,
		`INSERT INTO article_links (ticket_meta_id, article_id, article_slug, article_title, linked_by)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, ticket_meta_id, article_id, article_slug, article_title, linked_by, created_at`,
		ticketMetaID, articleID, slug, title, linkedBy).
		Scan(&l.ID, &l.TicketMetaID, &l.ArticleID, &l.ArticleSlug, &l.ArticleTitle, &l.LinkedBy, &l.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating article link: %w", err)
	}
	return &l, nil
}

// DeleteArticleLink removes an article link.
func (s *Store) DeleteArticleLink(ctx context.Context, ticketMetaID, articleID uuid.UUID) error {
	tag, err := s.dbtx.Exec(ctx,
		`DELETE FROM article_links WHERE ticket_meta_id = $1 AND article_id = $2`,
		ticketMetaID, articleID)
	if err != nil {
		return fmt.Errorf("deleting article link: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// --- Post-Mortem Links ---

// GetPostMortemLink returns the post-mortem link for a ticket, if any.
func (s *Store) GetPostMortemLink(ctx context.Context, ticketMetaID uuid.UUID) (*PostMortemLink, error) {
	var l PostMortemLink
	err := s.dbtx.QueryRow(ctx,
		`SELECT id, ticket_meta_id, postmortem_id, postmortem_url, created_by, created_at
		 FROM postmortem_links WHERE ticket_meta_id = $1`, ticketMetaID).
		Scan(&l.ID, &l.TicketMetaID, &l.PostMortemID, &l.PostMortemURL, &l.CreatedBy, &l.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// CreatePostMortemLink inserts a post-mortem link.
func (s *Store) CreatePostMortemLink(ctx context.Context, ticketMetaID, postmortemID, createdBy uuid.UUID, url string) (*PostMortemLink, error) {
	var l PostMortemLink
	err := s.dbtx.QueryRow(ctx,
		`INSERT INTO postmortem_links (ticket_meta_id, postmortem_id, postmortem_url, created_by)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, ticket_meta_id, postmortem_id, postmortem_url, created_by, created_at`,
		ticketMetaID, postmortemID, url, createdBy).
		Scan(&l.ID, &l.TicketMetaID, &l.PostMortemID, &l.PostMortemURL, &l.CreatedBy, &l.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating postmortem link: %w", err)
	}
	return &l, nil
}
