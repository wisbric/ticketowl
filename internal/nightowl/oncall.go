package nightowl

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// OnCallInfo represents the current on-call person for a service.
type OnCallInfo struct {
	UserName  string    `json:"user_name"`
	UserEmail string    `json:"user_email"`
	ShiftEnd  time.Time `json:"shift_end"`
}

// GetOnCallForService returns the current on-call for a given service.
func (c *Client) GetOnCallForService(ctx context.Context, service string) (*OnCallInfo, error) {
	path := fmt.Sprintf("/api/v1/oncall/%s", service)

	body, err := c.get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("getting oncall for service %s: %w", service, err)
	}

	var info OnCallInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("decoding oncall for service %s: %w", service, err)
	}

	return &info, nil
}
