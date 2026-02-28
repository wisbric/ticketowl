package zammad

import (
	"context"
	"encoding/json"
	"fmt"
)

// User represents a Zammad user.
type User struct {
	ID           int    `json:"id"`
	Login        string `json:"login"`
	Firstname    string `json:"firstname"`
	Lastname     string `json:"lastname"`
	Email        string `json:"email"`
	Active       bool   `json:"active"`
	Organization string `json:"organization"`
}

// GetUser returns a user by ID.
func (c *Client) GetUser(ctx context.Context, id int) (*User, error) {
	path := fmt.Sprintf("/api/v1/users/%d?expand=true", id)

	body, err := c.get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("getting user %d: %w", id, err)
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("decoding user %d: %w", id, err)
	}

	return &user, nil
}

// GetCurrentUser returns the user associated with the API token.
func (c *Client) GetCurrentUser(ctx context.Context) (*User, error) {
	body, err := c.get(ctx, "/api/v1/users/me")
	if err != nil {
		return nil, fmt.Errorf("getting current user: %w", err)
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("decoding current user: %w", err)
	}

	return &user, nil
}

// SearchUsersByEmail searches for a Zammad user by email address.
// Returns nil if no user is found.
func (c *Client) SearchUsersByEmail(ctx context.Context, email string) (*User, error) {
	path := "/api/v1/users/search?query=" + email + "&limit=1&expand=true"

	body, err := c.get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("searching users by email %s: %w", email, err)
	}

	var users []User
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("decoding user search results: %w", err)
	}

	if len(users) == 0 {
		return nil, nil
	}

	return &users[0], nil
}
