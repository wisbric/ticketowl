package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// APIKeyAuthenticator validates API keys against the database.
type APIKeyAuthenticator struct {
	Store Storage
}

// APIKeyResult holds the resolved identity data from an API key lookup.
type APIKeyResult struct {
	APIKeyID  uuid.UUID
	TenantID  uuid.UUID
	KeyPrefix string
	Role      string
	Scopes    []string
	ExpiresAt *time.Time
}

// Authenticate hashes the raw key, looks it up in public.api_keys, and
// validates expiration.
func (a *APIKeyAuthenticator) Authenticate(ctx context.Context, rawKey string) (*APIKeyResult, error) {
	if rawKey == "" {
		return nil, fmt.Errorf("empty API key")
	}

	hash := HashAPIKey(rawKey)

	key, err := a.Store.GetAPIKeyByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("looking up API key: %w", err)
	}

	// Check expiration.
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("API key expired at %s", key.ExpiresAt)
	}

	// Update last_used asynchronously — fire and forget.
	go func() {
		_ = a.Store.UpdateAPIKeyLastUsed(context.Background(), key.APIKeyID)
	}()

	role := key.Role
	if !IsValidRole(role) {
		role = RoleEngineer
	}

	return &APIKeyResult{
		APIKeyID:  key.APIKeyID,
		TenantID:  key.TenantID,
		KeyPrefix: key.KeyPrefix,
		Role:      role,
		Scopes:    key.Scopes,
		ExpiresAt: key.ExpiresAt,
	}, nil
}
