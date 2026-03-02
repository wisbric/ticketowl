package zammad

import (
	"context"
	"encoding/json"
	"fmt"
)

// TicketState represents a Zammad ticket state.
type TicketState struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// TicketPriority represents a Zammad ticket priority.
type TicketPriority struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ListStates returns all ticket states from Zammad.
func (c *Client) ListStates(ctx context.Context) ([]TicketState, error) {
	body, err := c.get(ctx, "/api/v1/ticket_states")
	if err != nil {
		return nil, fmt.Errorf("listing states: %w", err)
	}

	var states []TicketState
	if err := json.Unmarshal(body, &states); err != nil {
		return nil, fmt.Errorf("decoding states: %w", err)
	}

	return states, nil
}

// ListPriorities returns all ticket priorities from Zammad.
func (c *Client) ListPriorities(ctx context.Context) ([]TicketPriority, error) {
	body, err := c.get(ctx, "/api/v1/ticket_priorities")
	if err != nil {
		return nil, fmt.Errorf("listing priorities: %w", err)
	}

	var priorities []TicketPriority
	if err := json.Unmarshal(body, &priorities); err != nil {
		return nil, fmt.Errorf("decoding priorities: %w", err)
	}

	return priorities, nil
}

// TicketGroup represents a Zammad group (ticket assignment group).
type TicketGroup struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ListGroups returns all groups from Zammad.
func (c *Client) ListGroups(ctx context.Context) ([]TicketGroup, error) {
	body, err := c.get(ctx, "/api/v1/groups")
	if err != nil {
		return nil, fmt.Errorf("listing groups: %w", err)
	}

	var groups []TicketGroup
	if err := json.Unmarshal(body, &groups); err != nil {
		return nil, fmt.Errorf("decoding groups: %w", err)
	}

	return groups, nil
}
