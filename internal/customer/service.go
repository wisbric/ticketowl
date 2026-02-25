package customer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/wisbric/ticketowl/internal/zammad"
)

// Service encapsulates customer portal business logic.
type Service struct {
	store  CustomerStore
	zammad ZammadClient
	logger *slog.Logger
}

// NewService creates a customer Service.
func NewService(store CustomerStore, zammad ZammadClient, logger *slog.Logger) *Service {
	return &Service{
		store:  store,
		zammad: zammad,
		logger: logger,
	}
}

// ResolveOrg looks up the customer org and returns the zammad_org_id for scoping.
// Returns an error if the org does not exist or has no zammad_org_id configured.
func (s *Service) ResolveOrg(ctx context.Context, orgID uuid.UUID) (*Org, error) {
	org, err := s.store.GetOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("resolving customer org: %w", err)
	}
	if org.ZammadOrgID == nil {
		return nil, fmt.Errorf("customer org %s has no zammad_org_id configured", orgID)
	}
	return org, nil
}

// ListMyTickets returns tickets scoped to the customer's org.
// The orgID filter is always set from the authenticated identity — never from query params.
func (s *Service) ListMyTickets(ctx context.Context, orgID uuid.UUID) ([]PortalTicket, error) {
	org, err := s.ResolveOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	tickets, err := s.zammad.ListTickets(ctx, zammad.ListTicketsOptions{
		OrgID: org.ZammadOrgID,
	})
	if err != nil {
		return nil, fmt.Errorf("listing tickets from Zammad: %w", err)
	}

	result := make([]PortalTicket, len(tickets))
	for i, t := range tickets {
		result[i] = portalTicketFromZammad(&t)
	}
	return result, nil
}

// GetMyTicket returns a single ticket visible to the customer's org.
// Enforces org scoping: the ticket must belong to the customer's zammad_org_id.
func (s *Service) GetMyTicket(ctx context.Context, orgID uuid.UUID, zammadID int) (*PortalTicketDetail, error) {
	org, err := s.ResolveOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	t, err := s.zammad.GetTicket(ctx, zammadID)
	if err != nil {
		return nil, fmt.Errorf("getting ticket from Zammad: %w", err)
	}

	// Enforce org scoping: customer can only see their own org's tickets.
	if t.OrganisationID != *org.ZammadOrgID {
		return nil, fmt.Errorf("ticket does not belong to customer org: %w", ErrForbidden)
	}

	detail := &PortalTicketDetail{
		PortalTicket: portalTicketFromZammad(t),
	}

	// Load public comments (non-internal Zammad articles).
	articles, err := s.zammad.ListArticles(ctx, zammadID)
	if err != nil {
		s.logger.Error("listing articles for portal ticket", "error", err, "zammad_id", zammadID)
	} else {
		for _, a := range articles {
			if !a.Internal {
				detail.Comments = append(detail.Comments, PortalComment{
					ID:        a.ID,
					Body:      a.Body,
					Sender:    a.Sender,
					CreatedBy: a.CreatedBy,
					CreatedAt: a.CreatedAt,
				})
			}
		}
	}
	if detail.Comments == nil {
		detail.Comments = []PortalComment{}
	}

	// Load linked articles and SLA from TicketOwl metadata.
	metaID, err := s.store.GetTicketMetaID(ctx, zammadID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.logger.Error("looking up ticket meta for portal", "error", err, "zammad_id", zammadID)
	}

	if err == nil {
		s.enrichDetail(ctx, detail, metaID)
	}

	if detail.LinkedArticles == nil {
		detail.LinkedArticles = []PortalArticle{}
	}

	return detail, nil
}

// enrichDetail adds linked articles and SLA due times to the portal detail.
func (s *Service) enrichDetail(ctx context.Context, detail *PortalTicketDetail, metaID uuid.UUID) {
	articleLinks, err := s.store.ListArticleLinks(ctx, metaID)
	if err != nil {
		s.logger.Error("listing article links for portal", "error", err)
	} else {
		for _, al := range articleLinks {
			detail.LinkedArticles = append(detail.LinkedArticles, PortalArticle{
				ID:    al.ArticleID,
				Slug:  al.ArticleSlug,
				Title: al.ArticleTitle,
			})
		}
	}

	slaState, err := s.store.GetSLAState(ctx, metaID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.logger.Error("getting SLA state for portal", "error", err)
	}
	if slaState != nil {
		label := string(slaState.Label)
		detail.SLAState = &label
		detail.ResponseDueAt = slaState.ResponseDueAt
		detail.ResolutionDueAt = slaState.ResolutionDueAt
	}
}

// GetLinkedArticles returns the BookOwl articles linked to a customer's ticket.
func (s *Service) GetLinkedArticles(ctx context.Context, orgID uuid.UUID, zammadID int) ([]PortalArticle, error) {
	org, err := s.ResolveOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Verify ticket belongs to the org.
	t, err := s.zammad.GetTicket(ctx, zammadID)
	if err != nil {
		return nil, fmt.Errorf("getting ticket from Zammad: %w", err)
	}
	if t.OrganisationID != *org.ZammadOrgID {
		return nil, fmt.Errorf("ticket does not belong to customer org: %w", ErrForbidden)
	}

	metaID, err := s.store.GetTicketMetaID(ctx, zammadID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []PortalArticle{}, nil
		}
		return nil, fmt.Errorf("looking up ticket meta: %w", err)
	}

	articleLinks, err := s.store.ListArticleLinks(ctx, metaID)
	if err != nil {
		return nil, fmt.Errorf("listing article links: %w", err)
	}

	articles := make([]PortalArticle, len(articleLinks))
	for i, al := range articleLinks {
		articles[i] = PortalArticle{
			ID:    al.ArticleID,
			Slug:  al.ArticleSlug,
			Title: al.ArticleTitle,
		}
	}
	return articles, nil
}

// AddReply adds a public reply to a ticket on behalf of the customer.
func (s *Service) AddReply(ctx context.Context, orgID uuid.UUID, zammadID int, body string) error {
	org, err := s.ResolveOrg(ctx, orgID)
	if err != nil {
		return err
	}

	// Verify ticket belongs to the org.
	t, err := s.zammad.GetTicket(ctx, zammadID)
	if err != nil {
		return fmt.Errorf("getting ticket from Zammad: %w", err)
	}
	if t.OrganisationID != *org.ZammadOrgID {
		return fmt.Errorf("ticket does not belong to customer org: %w", ErrForbidden)
	}

	_, err = s.zammad.CreateArticle(ctx, zammad.ArticleCreate{
		TicketID:    zammadID,
		Body:        body,
		ContentType: "text/plain",
		Internal:    false,
	})
	if err != nil {
		return fmt.Errorf("creating reply in Zammad: %w", err)
	}

	return nil
}

// ErrForbidden is returned when a customer tries to access a ticket outside their org.
var ErrForbidden = fmt.Errorf("forbidden")

// portalTicketFromZammad converts a Zammad ticket to a portal ticket summary.
func portalTicketFromZammad(t *zammad.Ticket) PortalTicket {
	return PortalTicket{
		ID:        t.ID,
		Number:    t.Number,
		Title:     t.Title,
		Status:    t.State,
		Priority:  t.Priority,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}
