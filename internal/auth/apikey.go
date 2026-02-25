package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// APIKeyAuthenticator validates API keys against the database.
type APIKeyAuthenticator struct {
	DB *pgxpool.Pool
}

// APIKeyResult holds the resolved identity data from an API key lookup.
type APIKeyResult struct {
	APIKeyID uuid.UUID
	TenantID uuid.UUID
}

// Authenticate hashes the raw key, looks it up in global.api_keys, and
// returns the associated tenant.
func (a *APIKeyAuthenticator) Authenticate(ctx context.Context, rawKey string) (*APIKeyResult, error) {
	if rawKey == "" {
		return nil, fmt.Errorf("empty API key")
	}

	hash := HashAPIKey(rawKey)

	var keyID, tenantID uuid.UUID
	err := a.DB.QueryRow(ctx,
		"SELECT id, tenant_id FROM global.api_keys WHERE key_hash = $1",
		hash,
	).Scan(&keyID, &tenantID)
	if err != nil {
		return nil, fmt.Errorf("looking up API key: %w", err)
	}

	// Update last_used asynchronously.
	go func() {
		_, _ = a.DB.Exec(context.Background(),
			"UPDATE global.api_keys SET last_used_at = now() WHERE id = $1",
			keyID,
		)
	}()

	return &APIKeyResult{
		APIKeyID: keyID,
		TenantID: tenantID,
	}, nil
}
