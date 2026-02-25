package customer

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/wisbric/ticketowl/internal/link"
	"github.com/wisbric/ticketowl/internal/sla"
	"github.com/wisbric/ticketowl/internal/zammad"
)

// Org represents a customer organisation within a tenant.
type Org struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	OIDCGroup   string    `json:"oidc_group"`
	ZammadOrgID *int      `json:"zammad_org_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PortalTicket is the customer-facing ticket summary.
type PortalTicket struct {
	ID        int       `json:"id"`
	Number    string    `json:"number"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	Priority  string    `json:"priority"`
	SLAState  *string   `json:"sla_state,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PortalTicketDetail is the customer-facing ticket detail view.
type PortalTicketDetail struct {
	PortalTicket
	Comments        []PortalComment `json:"comments"`
	LinkedArticles  []PortalArticle `json:"linked_articles"`
	ResponseDueAt   *time.Time      `json:"response_due_at,omitempty"`
	ResolutionDueAt *time.Time      `json:"resolution_due_at,omitempty"`
}

// PortalComment is a public comment visible to customers.
type PortalComment struct {
	ID        int       `json:"id"`
	Body      string    `json:"body"`
	Sender    string    `json:"sender"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// PortalArticle is a linked BookOwl article visible to customers.
type PortalArticle struct {
	ID    uuid.UUID `json:"id"`
	Slug  string    `json:"slug"`
	Title string    `json:"title"`
}

// ReplyRequest is the payload for a customer reply.
type ReplyRequest struct {
	Body string `json:"body"`
}

// CustomerStore defines the store operations the customer service needs.
type CustomerStore interface {
	GetOrg(ctx context.Context, orgID uuid.UUID) (*Org, error)
	GetTicketMetaID(ctx context.Context, zammadID int) (uuid.UUID, error)
	ListArticleLinks(ctx context.Context, ticketMetaID uuid.UUID) ([]link.ArticleLink, error)
	GetSLAState(ctx context.Context, ticketMetaID uuid.UUID) (*sla.State, error)
}

// ZammadClient defines the Zammad operations the customer service needs.
type ZammadClient interface {
	ListTickets(ctx context.Context, opts zammad.ListTicketsOptions) ([]zammad.Ticket, error)
	GetTicket(ctx context.Context, id int) (*zammad.Ticket, error)
	ListArticles(ctx context.Context, ticketID int) ([]zammad.Article, error)
	CreateArticle(ctx context.Context, req zammad.ArticleCreate) (*zammad.Article, error)
}
