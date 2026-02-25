package nightowl

import (
	"context"
	"encoding/json"
	"fmt"
)

// CreateAlertRequest is the payload for creating an alert in NightOwl.
type CreateAlertRequest struct {
	Name        string            `json:"name"`
	Summary     string            `json:"summary"`
	Severity    string            `json:"severity"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Alert represents the created alert response from NightOwl.
type Alert struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Severity string `json:"severity"`
	Status   string `json:"status"`
}

// CreateAlert creates a new alert in NightOwl (used for SLA breach escalation).
func (c *Client) CreateAlert(ctx context.Context, req CreateAlertRequest) (*Alert, error) {
	body, err := c.post(ctx, "/api/v1/alerts", req)
	if err != nil {
		return nil, fmt.Errorf("creating alert: %w", err)
	}

	var alert Alert
	if err := json.Unmarshal(body, &alert); err != nil {
		return nil, fmt.Errorf("decoding created alert: %w", err)
	}

	return &alert, nil
}
