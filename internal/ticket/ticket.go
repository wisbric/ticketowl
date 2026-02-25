package ticket

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/wisbric/ticketowl/internal/zammad"
)

// ZammadClient defines the Zammad operations the ticket service needs.
type ZammadClient interface {
	GetTicket(ctx context.Context, id int) (*zammad.Ticket, error)
	ListTickets(ctx context.Context, opts zammad.ListTicketsOptions) ([]zammad.Ticket, error)
	CreateTicket(ctx context.Context, req zammad.TicketCreateRequest) (*zammad.Ticket, error)
	UpdateTicket(ctx context.Context, id int, req zammad.TicketUpdateRequest) (*zammad.Ticket, error)
	SearchTickets(ctx context.Context, query string, opts zammad.ListTicketsOptions) ([]zammad.Ticket, error)
}

// TicketStore defines the database operations the ticket service needs.
type TicketStore interface {
	GetByZammadID(ctx context.Context, zammadID int) (*Meta, error)
	Upsert(ctx context.Context, zammadID int, zammadNumber string, slaPolicyID *uuid.UUID) (*Meta, error)
	ListByZammadIDs(ctx context.Context, zammadIDs []int) ([]Meta, error)
}

// Meta is the TicketOwl metadata stored alongside a Zammad ticket.
type Meta struct {
	ID           uuid.UUID  `json:"id"`
	ZammadID     int        `json:"zammad_id"`
	ZammadNumber string     `json:"zammad_number"`
	SLAPolicyID  *uuid.UUID `json:"sla_policy_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// EnrichedTicket combines a live Zammad ticket with TicketOwl metadata.
type EnrichedTicket struct {
	// Zammad fields (live).
	ID             int        `json:"id"`
	Number         string     `json:"number"`
	Title          string     `json:"title"`
	State          string     `json:"state"`
	StateID        int        `json:"state_id"`
	Priority       string     `json:"priority"`
	PriorityID     int        `json:"priority_id"`
	Group          string     `json:"group"`
	GroupID        int        `json:"group_id"`
	Owner          string     `json:"owner"`
	OwnerID        int        `json:"owner_id"`
	CustomerID     int        `json:"customer_id"`
	OrganisationID int        `json:"organization_id"`
	Tags           []string   `json:"tags"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CloseAt        *time.Time `json:"close_at,omitempty"`

	// TicketOwl metadata.
	MetaID      *uuid.UUID `json:"meta_id,omitempty"`
	SLAPolicyID *uuid.UUID `json:"sla_policy_id,omitempty"`
}

// ListOptions controls ticket list filtering and pagination.
type ListOptions struct {
	Page     int
	PerPage  int
	StateIDs []int
	GroupIDs []int
	OrgID    *int
	Query    string
}

// CreateRequest is the request payload for creating a ticket.
type CreateRequest struct {
	Title      string `json:"title"`
	GroupID    int    `json:"group_id"`
	CustomerID int    `json:"customer_id"`
	StateID    int    `json:"state_id,omitempty"`
	PriorityID int    `json:"priority_id,omitempty"`
	Body       string `json:"body,omitempty"`
}

// UpdateRequest is the request payload for updating a ticket.
type UpdateRequest struct {
	StateID    *int `json:"state_id,omitempty"`
	PriorityID *int `json:"priority_id,omitempty"`
	OwnerID    *int `json:"owner_id,omitempty"`
	GroupID    *int `json:"group_id,omitempty"`
}

// enrichFromZammad builds an EnrichedTicket from a Zammad ticket and optional metadata.
func enrichFromZammad(t *zammad.Ticket, meta *Meta) EnrichedTicket {
	et := EnrichedTicket{
		ID:             t.ID,
		Number:         t.Number,
		Title:          t.Title,
		State:          t.State,
		StateID:        t.StateID,
		Priority:       t.Priority,
		PriorityID:     t.PriorityID,
		Group:          t.Group,
		GroupID:        t.GroupID,
		Owner:          t.Owner,
		OwnerID:        t.OwnerID,
		CustomerID:     t.CustomerID,
		OrganisationID: t.OrganisationID,
		Tags:           t.Tags,
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
		CloseAt:        t.CloseAt,
	}

	if meta != nil {
		et.MetaID = &meta.ID
		et.SLAPolicyID = meta.SLAPolicyID
	}

	return et
}
