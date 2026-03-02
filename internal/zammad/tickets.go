package zammad

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Ticket represents a Zammad ticket.
type Ticket struct {
	ID             int        `json:"id"`
	Number         string     `json:"number"`
	Title          string     `json:"title"`
	StateID        int        `json:"state_id"`
	State          string     `json:"state"`
	PriorityID     int        `json:"priority_id"`
	Priority       string     `json:"priority"`
	GroupID        int        `json:"group_id"`
	Group          string     `json:"group"`
	OwnerID        int        `json:"owner_id"`
	Owner          string     `json:"owner"`
	CustomerID     int        `json:"customer_id"`
	OrganisationID int        `json:"organization_id"`
	Tags           []string   `json:"tags"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CloseAt        *time.Time `json:"close_at"`
}

// ListTicketsOptions controls ticket list filtering and pagination.
type ListTicketsOptions struct {
	Page     int
	PerPage  int
	StateIDs []int
	GroupIDs []int
	OrgID    *int
}

// TicketCreateRequest is the payload for creating a ticket.
type TicketCreateRequest struct {
	Title      string         `json:"title"`
	GroupID    int            `json:"group_id"`
	CustomerID int            `json:"customer_id,omitempty"`
	StateID    int            `json:"state_id,omitempty"`
	PriorityID int            `json:"priority_id,omitempty"`
	Article    *ArticleCreate `json:"article,omitempty"`
}

// TicketUpdateRequest is the payload for updating a ticket.
type TicketUpdateRequest struct {
	StateID    *int `json:"state_id,omitempty"`
	PriorityID *int `json:"priority_id,omitempty"`
	OwnerID    *int `json:"owner_id,omitempty"`
	GroupID    *int `json:"group_id,omitempty"`
}

// ListTickets returns tickets matching the given options.
func (c *Client) ListTickets(ctx context.Context, opts ListTicketsOptions) ([]Ticket, error) {
	path := "/api/v1/tickets?" + buildListQuery(opts)

	body, err := c.get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("listing tickets: %w", err)
	}

	var tickets []Ticket
	if err := json.Unmarshal(body, &tickets); err != nil {
		return nil, fmt.Errorf("decoding tickets: %w", err)
	}

	return tickets, nil
}

// GetTicket returns a single ticket by ID.
func (c *Client) GetTicket(ctx context.Context, id int) (*Ticket, error) {
	path := fmt.Sprintf("/api/v1/tickets/%d?expand=true", id)

	body, err := c.get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("getting ticket %d: %w", id, err)
	}

	var ticket Ticket
	if err := json.Unmarshal(body, &ticket); err != nil {
		return nil, fmt.Errorf("decoding ticket %d: %w", id, err)
	}

	return &ticket, nil
}

// CreateTicket creates a new ticket in Zammad.
func (c *Client) CreateTicket(ctx context.Context, req TicketCreateRequest) (*Ticket, error) {
	body, err := c.post(ctx, "/api/v1/tickets", req)
	if err != nil {
		return nil, fmt.Errorf("creating ticket: %w", err)
	}

	var ticket Ticket
	if err := json.Unmarshal(body, &ticket); err != nil {
		return nil, fmt.Errorf("decoding created ticket: %w", err)
	}

	return &ticket, nil
}

// UpdateTicket updates an existing ticket.
func (c *Client) UpdateTicket(ctx context.Context, id int, req TicketUpdateRequest) (*Ticket, error) {
	path := fmt.Sprintf("/api/v1/tickets/%d", id)

	body, err := c.put(ctx, path, req)
	if err != nil {
		return nil, fmt.Errorf("updating ticket %d: %w", id, err)
	}

	var ticket Ticket
	if err := json.Unmarshal(body, &ticket); err != nil {
		return nil, fmt.Errorf("decoding updated ticket %d: %w", id, err)
	}

	return &ticket, nil
}

// SearchTickets searches for tickets matching the given query.
func (c *Client) SearchTickets(ctx context.Context, query string, opts ListTicketsOptions) ([]Ticket, error) {
	params := buildListQuery(opts)
	params += "&query=" + url.QueryEscape(query)
	path := "/api/v1/tickets/search?" + params

	body, err := c.get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("searching tickets: %w", err)
	}

	// Zammad search returns { "tickets": [...ids], "assets": { "Ticket": { ... } } }
	// or in expand mode, returns a flat array of tickets.
	var tickets []Ticket
	if err := json.Unmarshal(body, &tickets); err != nil {
		return nil, fmt.Errorf("decoding search results: %w", err)
	}

	return tickets, nil
}

func buildListQuery(opts ListTicketsOptions) string {
	params := url.Values{}
	params.Set("expand", "true")

	if opts.Page > 0 {
		params.Set("page", strconv.Itoa(opts.Page))
	}
	if opts.PerPage > 0 {
		params.Set("per_page", strconv.Itoa(opts.PerPage))
	}
	if len(opts.StateIDs) > 0 {
		ids := make([]string, len(opts.StateIDs))
		for i, id := range opts.StateIDs {
			ids[i] = strconv.Itoa(id)
		}
		params.Set("state_id", strings.Join(ids, ","))
	}
	if len(opts.GroupIDs) > 0 {
		ids := make([]string, len(opts.GroupIDs))
		for i, id := range opts.GroupIDs {
			ids[i] = strconv.Itoa(id)
		}
		params.Set("group_id", strings.Join(ids, ","))
	}
	if opts.OrgID != nil {
		params.Set("organization_id", strconv.Itoa(*opts.OrgID))
	}

	return params.Encode()
}
