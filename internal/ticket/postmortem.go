package ticket

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/wisbric/ticketowl/internal/bookowl"
	"github.com/wisbric/ticketowl/internal/link"
	"github.com/wisbric/ticketowl/internal/nightowl"
)

// ErrPostMortemExists is returned when a post-mortem already exists for a ticket.
type ErrPostMortemExists struct {
	URL string
}

func (e *ErrPostMortemExists) Error() string {
	return fmt.Sprintf("post-mortem already exists: %s", e.URL)
}

// PostMortemCreator defines the BookOwl operations for post-mortem creation.
type PostMortemCreator interface {
	CreatePostMortem(ctx context.Context, req bookowl.CreatePostMortemRequest) (*bookowl.PostMortem, error)
}

// NightOwlIncidentFetcher defines the NightOwl operations for post-mortem creation.
type NightOwlIncidentFetcher interface {
	GetIncident(ctx context.Context, incidentID string) (*nightowl.Incident, error)
}

// PostMortemStore defines the store operations for post-mortem creation.
type PostMortemStore interface {
	GetTicketMetaID(ctx context.Context, zammadID int) (uuid.UUID, error)
	GetPostMortemLink(ctx context.Context, ticketMetaID uuid.UUID) (*link.PostMortemLink, error)
	CreatePostMortemLink(ctx context.Context, ticketMetaID, postmortemID, createdBy uuid.UUID, url string) (*link.PostMortemLink, error)
	ListIncidentLinks(ctx context.Context, ticketMetaID uuid.UUID) ([]link.IncidentLink, error)
}

// PostMortemResult is the response from creating a post-mortem.
type PostMortemResult struct {
	PostMortemID  string `json:"postmortem_id"`
	PostMortemURL string `json:"postmortem_url"`
}

// CreatePostMortem creates a post-mortem in BookOwl for the given ticket.
// Returns ErrPostMortemExists (with the existing URL) if one already exists.
func CreatePostMortem(
	ctx context.Context,
	zammadID int,
	ticketNumber string,
	ticketTitle string,
	createdBy uuid.UUID,
	store PostMortemStore,
	bookowlClient PostMortemCreator,
	nightowlClient NightOwlIncidentFetcher,
) (*PostMortemResult, error) {
	metaID, err := store.GetTicketMetaID(ctx, zammadID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("ticket not tracked in ticketowl")
		}
		return nil, fmt.Errorf("resolving ticket meta: %w", err)
	}

	// Check if a post-mortem already exists.
	existing, err := store.GetPostMortemLink(ctx, metaID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("checking existing post-mortem: %w", err)
	}
	if existing != nil {
		return nil, &ErrPostMortemExists{URL: existing.PostMortemURL}
	}

	// Load linked incidents.
	incidentLinks, err := store.ListIncidentLinks(ctx, metaID)
	if err != nil {
		return nil, fmt.Errorf("listing incident links: %w", err)
	}

	// Fetch incident details from NightOwl.
	var incidentIDs []string
	var summary string
	for _, il := range incidentLinks {
		incident, err := nightowlClient.GetIncident(ctx, il.IncidentID.String())
		if err != nil {
			continue // Best effort — don't fail if NightOwl is unreachable for one incident.
		}
		incidentIDs = append(incidentIDs, incident.ID)
		if summary == "" {
			summary = incident.Summary
		}
	}

	// Create post-mortem in BookOwl.
	pm, err := bookowlClient.CreatePostMortem(ctx, bookowl.CreatePostMortemRequest{
		Title:        fmt.Sprintf("Post-Mortem: %s", ticketTitle),
		TicketID:     metaID.String(),
		TicketNumber: ticketNumber,
		IncidentIDs:  incidentIDs,
		Summary:      summary,
	})
	if err != nil {
		return nil, fmt.Errorf("creating post-mortem in BookOwl: %w", err)
	}

	// Store the link.
	pmID, err := uuid.Parse(pm.ID)
	if err != nil {
		pmID = uuid.New()
	}
	_, err = store.CreatePostMortemLink(ctx, metaID, pmID, createdBy, pm.URL)
	if err != nil {
		return nil, fmt.Errorf("storing post-mortem link: %w", err)
	}

	return &PostMortemResult{
		PostMortemID:  pm.ID,
		PostMortemURL: pm.URL,
	}, nil
}
