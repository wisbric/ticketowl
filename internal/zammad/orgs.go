package zammad

import (
	"context"
	"encoding/json"
	"fmt"
)

// Organisation represents a Zammad organisation.
type Organisation struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// GetOrganisation returns an organisation by ID.
func (c *Client) GetOrganisation(ctx context.Context, id int) (*Organisation, error) {
	path := fmt.Sprintf("/api/v1/organizations/%d", id)

	body, err := c.get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("getting organisation %d: %w", id, err)
	}

	var org Organisation
	if err := json.Unmarshal(body, &org); err != nil {
		return nil, fmt.Errorf("decoding organisation %d: %w", id, err)
	}

	return &org, nil
}
