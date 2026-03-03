package link

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/wisbric/ticketowl/internal/bookowl"
	"github.com/wisbric/ticketowl/internal/nightowl"
)

// NightOwlClient defines the NightOwl operations the link service needs.
type NightOwlClient interface {
	GetIncident(ctx context.Context, incidentID string) (*nightowl.Incident, error)
	SearchIncidents(ctx context.Context, query string, limit int) ([]nightowl.Incident, error)
}

// BookOwlClient defines the BookOwl operations the link service needs.
type BookOwlClient interface {
	GetArticle(ctx context.Context, articleID string) (*bookowl.Article, error)
	SearchArticles(ctx context.Context, opts bookowl.SearchOptions) ([]bookowl.ArticleSummary, error)
}

// IncidentLink represents a link between a TicketOwl ticket and a NightOwl incident.
type IncidentLink struct {
	ID           uuid.UUID `json:"id"`
	TicketMetaID uuid.UUID `json:"ticket_meta_id"`
	IncidentID   uuid.UUID `json:"incident_id"`
	IncidentSlug string    `json:"incident_slug"`
	LinkedBy     uuid.UUID `json:"linked_by"`
	CreatedAt    time.Time `json:"created_at"`
}

// ArticleLink represents a link between a TicketOwl ticket and a BookOwl article.
type ArticleLink struct {
	ID           uuid.UUID `json:"id"`
	TicketMetaID uuid.UUID `json:"ticket_meta_id"`
	ArticleID    uuid.UUID `json:"article_id"`
	ArticleSlug  string    `json:"article_slug"`
	ArticleTitle string    `json:"article_title"`
	LinkedBy     uuid.UUID `json:"linked_by"`
	CreatedAt    time.Time `json:"created_at"`
}

// PostMortemLink represents a link between a TicketOwl ticket and a BookOwl post-mortem.
type PostMortemLink struct {
	ID            uuid.UUID `json:"id"`
	TicketMetaID  uuid.UUID `json:"ticket_meta_id"`
	PostMortemID  uuid.UUID `json:"postmortem_id"`
	PostMortemURL string    `json:"postmortem_url"`
	CreatedBy     uuid.UUID `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
}

// TicketLinks aggregates all links for a ticket.
type TicketLinks struct {
	Incidents  []IncidentLink  `json:"incidents"`
	Articles   []ArticleLink   `json:"articles"`
	PostMortem *PostMortemLink `json:"postmortem,omitempty"`
}

// CreateIncidentLinkRequest is the payload for linking an incident.
type CreateIncidentLinkRequest struct {
	IncidentID string `json:"incident_id"`
}

// CreateArticleLinkRequest is the payload for linking an article.
type CreateArticleLinkRequest struct {
	ArticleID string `json:"article_id"`
}

// LinkStore defines the database operations the link service needs.
type LinkStore interface {
	GetTicketMetaID(ctx context.Context, zammadID int) (uuid.UUID, error)
	ListIncidentLinks(ctx context.Context, ticketMetaID uuid.UUID) ([]IncidentLink, error)
	CreateIncidentLink(ctx context.Context, ticketMetaID, incidentID, linkedBy uuid.UUID, slug string) (*IncidentLink, error)
	DeleteIncidentLink(ctx context.Context, ticketMetaID, incidentID uuid.UUID) error
	ListArticleLinks(ctx context.Context, ticketMetaID uuid.UUID) ([]ArticleLink, error)
	CreateArticleLink(ctx context.Context, ticketMetaID, articleID, linkedBy uuid.UUID, slug, title string) (*ArticleLink, error)
	DeleteArticleLink(ctx context.Context, ticketMetaID, articleID uuid.UUID) error
	GetPostMortemLink(ctx context.Context, ticketMetaID uuid.UUID) (*PostMortemLink, error)
	CreatePostMortemLink(ctx context.Context, ticketMetaID, postmortemID, createdBy uuid.UUID, url string) (*PostMortemLink, error)
}
