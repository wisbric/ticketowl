package nightowl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// Incident represents a NightOwl incident.
type Incident struct {
	ID         string     `json:"id"`
	Slug       string     `json:"slug"`
	Summary    string     `json:"summary"`
	Severity   string     `json:"severity"`
	Status     string     `json:"status"` // "open" | "acknowledged" | "resolved"
	Service    string     `json:"service"`
	Tags       []string   `json:"tags"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at"`
}

// GetIncident returns an incident by ID.
func (c *Client) GetIncident(ctx context.Context, incidentID string) (*Incident, error) {
	path := fmt.Sprintf("/api/v1/incidents/%s", incidentID)

	body, err := c.get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("getting incident %s: %w", incidentID, err)
	}

	var incident Incident
	if err := json.Unmarshal(body, &incident); err != nil {
		return nil, fmt.Errorf("decoding incident %s: %w", incidentID, err)
	}

	return &incident, nil
}

// ListIncidents returns incidents, optionally filtered by status.
func (c *Client) ListIncidents(ctx context.Context, status string) ([]Incident, error) {
	path := "/api/v1/incidents"
	if status != "" {
		path += "?status=" + status
	}

	body, err := c.get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("listing incidents: %w", err)
	}

	var incidents []Incident
	if err := json.Unmarshal(body, &incidents); err != nil {
		return nil, fmt.Errorf("decoding incidents: %w", err)
	}

	return incidents, nil
}

// SearchIncidents searches NightOwl incidents by query string.
func (c *Client) SearchIncidents(ctx context.Context, query string, limit int) ([]Incident, error) {
	path := fmt.Sprintf("/api/v1/incidents/search?q=%s&limit=%d", url.QueryEscape(query), limit)

	body, err := c.get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("searching incidents: %w", err)
	}

	var incidents []Incident
	if err := json.Unmarshal(body, &incidents); err != nil {
		return nil, fmt.Errorf("decoding incident search results: %w", err)
	}

	return incidents, nil
}
