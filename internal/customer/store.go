package customer

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/wisbric/ticketowl/internal/db"
	"github.com/wisbric/ticketowl/internal/link"
	"github.com/wisbric/ticketowl/internal/sla"
)

// Store provides database operations for the customer portal.
type Store struct {
	dbtx db.DBTX
}

// NewStore creates a customer Store backed by the given database connection.
func NewStore(dbtx db.DBTX) *Store {
	return &Store{dbtx: dbtx}
}

// GetOrg returns a customer organisation by its UUID.
func (s *Store) GetOrg(ctx context.Context, orgID uuid.UUID) (*Org, error) {
	var o Org
	err := s.dbtx.QueryRow(ctx,
		`SELECT id, name, oidc_group, zammad_org_id, created_at, updated_at
		 FROM customer_orgs WHERE id = $1`, orgID).
		Scan(&o.ID, &o.Name, &o.OIDCGroup, &o.ZammadOrgID, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting customer org %s: %w", orgID, err)
	}
	return &o, nil
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

// ListArticleLinks returns all article links for a ticket.
func (s *Store) ListArticleLinks(ctx context.Context, ticketMetaID uuid.UUID) ([]link.ArticleLink, error) {
	rows, err := s.dbtx.Query(ctx,
		`SELECT id, ticket_meta_id, article_id, article_slug, article_title, linked_by, created_at
		 FROM article_links WHERE ticket_meta_id = $1 ORDER BY created_at`, ticketMetaID)
	if err != nil {
		return nil, fmt.Errorf("listing article links: %w", err)
	}
	defer rows.Close()

	var links []link.ArticleLink
	for rows.Next() {
		var l link.ArticleLink
		if err := rows.Scan(&l.ID, &l.TicketMetaID, &l.ArticleID, &l.ArticleSlug, &l.ArticleTitle, &l.LinkedBy, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning article link: %w", err)
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

// GetSLAState returns the SLA state for a ticket.
func (s *Store) GetSLAState(ctx context.Context, ticketMetaID uuid.UUID) (*sla.State, error) {
	var st sla.State
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
