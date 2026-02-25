package link

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Service encapsulates link business logic.
type Service struct {
	store    LinkStore
	nightowl NightOwlClient
	bookowl  BookOwlClient
	logger   *slog.Logger
}

// NewService creates a link Service.
func NewService(store LinkStore, nightowl NightOwlClient, bookowl BookOwlClient, logger *slog.Logger) *Service {
	return &Service{
		store:    store,
		nightowl: nightowl,
		bookowl:  bookowl,
		logger:   logger,
	}
}

// GetLinks returns all links for a ticket (by Zammad ID).
func (s *Service) GetLinks(ctx context.Context, zammadID int) (*TicketLinks, error) {
	metaID, err := s.store.GetTicketMetaID(ctx, zammadID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &TicketLinks{
				Incidents: []IncidentLink{},
				Articles:  []ArticleLink{},
			}, nil
		}
		return nil, fmt.Errorf("resolving ticket meta: %w", err)
	}

	incidents, err := s.store.ListIncidentLinks(ctx, metaID)
	if err != nil {
		return nil, fmt.Errorf("listing incident links: %w", err)
	}
	if incidents == nil {
		incidents = []IncidentLink{}
	}

	articles, err := s.store.ListArticleLinks(ctx, metaID)
	if err != nil {
		return nil, fmt.Errorf("listing article links: %w", err)
	}
	if articles == nil {
		articles = []ArticleLink{}
	}

	pm, err := s.store.GetPostMortemLink(ctx, metaID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("getting postmortem link: %w", err)
	}

	return &TicketLinks{
		Incidents:  incidents,
		Articles:   articles,
		PostMortem: pm,
	}, nil
}

// LinkIncident validates an incident against NightOwl and creates a link.
func (s *Service) LinkIncident(ctx context.Context, zammadID int, incidentIDStr string, linkedBy uuid.UUID) (*IncidentLink, error) {
	// Validate the incident exists in NightOwl.
	incident, err := s.nightowl.GetIncident(ctx, incidentIDStr)
	if err != nil {
		return nil, fmt.Errorf("validating incident %s: %w", incidentIDStr, err)
	}

	incidentID, err := uuid.Parse(incidentIDStr)
	if err != nil {
		return nil, fmt.Errorf("parsing incident ID %q: %w", incidentIDStr, err)
	}

	metaID, err := s.store.GetTicketMetaID(ctx, zammadID)
	if err != nil {
		return nil, fmt.Errorf("resolving ticket meta for zammad_id %d: %w", zammadID, err)
	}

	link, err := s.store.CreateIncidentLink(ctx, metaID, incidentID, linkedBy, incident.Slug)
	if err != nil {
		return nil, fmt.Errorf("creating incident link: %w", err)
	}

	return link, nil
}

// UnlinkIncident removes an incident link.
func (s *Service) UnlinkIncident(ctx context.Context, zammadID int, incidentIDStr string) error {
	incidentID, err := uuid.Parse(incidentIDStr)
	if err != nil {
		return fmt.Errorf("parsing incident ID %q: %w", incidentIDStr, err)
	}

	metaID, err := s.store.GetTicketMetaID(ctx, zammadID)
	if err != nil {
		return fmt.Errorf("resolving ticket meta for zammad_id %d: %w", zammadID, err)
	}

	return s.store.DeleteIncidentLink(ctx, metaID, incidentID)
}

// LinkArticle validates an article against BookOwl and creates a link.
func (s *Service) LinkArticle(ctx context.Context, zammadID int, articleIDStr string, linkedBy uuid.UUID) (*ArticleLink, error) {
	// Validate the article exists in BookOwl.
	article, err := s.bookowl.GetArticle(ctx, articleIDStr)
	if err != nil {
		return nil, fmt.Errorf("validating article %s: %w", articleIDStr, err)
	}

	articleID, err := uuid.Parse(articleIDStr)
	if err != nil {
		return nil, fmt.Errorf("parsing article ID %q: %w", articleIDStr, err)
	}

	metaID, err := s.store.GetTicketMetaID(ctx, zammadID)
	if err != nil {
		return nil, fmt.Errorf("resolving ticket meta for zammad_id %d: %w", zammadID, err)
	}

	link, err := s.store.CreateArticleLink(ctx, metaID, articleID, linkedBy, article.Slug, article.Title)
	if err != nil {
		return nil, fmt.Errorf("creating article link: %w", err)
	}

	return link, nil
}

// UnlinkArticle removes an article link.
func (s *Service) UnlinkArticle(ctx context.Context, zammadID int, articleIDStr string) error {
	articleID, err := uuid.Parse(articleIDStr)
	if err != nil {
		return fmt.Errorf("parsing article ID %q: %w", articleIDStr, err)
	}

	metaID, err := s.store.GetTicketMetaID(ctx, zammadID)
	if err != nil {
		return fmt.Errorf("resolving ticket meta for zammad_id %d: %w", zammadID, err)
	}

	return s.store.DeleteArticleLink(ctx, metaID, articleID)
}
