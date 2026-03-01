package zammad

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Tester tests connectivity to a Zammad instance by calling GET /api/v1/users/me.
type Tester struct{}

// TestConnection verifies that url + apiToken can reach Zammad.
func (Tester) TestConnection(ctx context.Context, url, apiToken string) error {
	c := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"/api/v1/users/me", nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Token token="+apiToken)

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("connecting to Zammad: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("authentication failed (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d from Zammad", resp.StatusCode)
	}

	return nil
}
