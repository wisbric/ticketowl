package bookowl

import (
	"context"
	"encoding/json"
	"fmt"
)

// CreatePostMortemRequest is the payload for creating a post-mortem in BookOwl.
type CreatePostMortemRequest struct {
	Title        string   `json:"title"`
	TicketID     string   `json:"ticket_id"`
	TicketNumber string   `json:"ticket_number"`
	IncidentIDs  []string `json:"incident_ids"`
	Summary      string   `json:"summary"`
	Tags         []string `json:"tags,omitempty"`
}

// PostMortem represents the created post-mortem response from BookOwl.
type PostMortem struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// CreatePostMortem creates a new post-mortem document in BookOwl.
func (c *Client) CreatePostMortem(ctx context.Context, req CreatePostMortemRequest) (*PostMortem, error) {
	body, err := c.post(ctx, "/api/v1/postmortems", req)
	if err != nil {
		return nil, fmt.Errorf("creating post-mortem: %w", err)
	}

	var pm PostMortem
	if err := json.Unmarshal(body, &pm); err != nil {
		return nil, fmt.Errorf("decoding created post-mortem: %w", err)
	}

	return &pm, nil
}
