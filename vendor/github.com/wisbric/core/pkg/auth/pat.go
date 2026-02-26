package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
)

// PATPrefix identifies personal access tokens.
const PATPrefix = "nwl_pat_"

// MethodPAT indicates authentication via personal access token.
const MethodPAT = "pat"

// PATAuthResult holds resolved identity data from a PAT lookup.
type PATAuthResult struct {
	UserID      uuid.UUID
	Email       string
	DisplayName string
	Role        string
	TenantSlug  string
	TenantID    uuid.UUID
}

// PATAuthenticator validates personal access tokens across tenant schemas.
type PATAuthenticator struct {
	Store Storage
}

// NewPATAuthenticator creates a PAT authenticator.
func NewPATAuthenticator(store Storage) *PATAuthenticator {
	return &PATAuthenticator{Store: store}
}

// Authenticate validates a raw PAT string by looking up its prefix across tenants,
// verifying the hash, and checking expiry. Returns the resolved identity.
func (a *PATAuthenticator) Authenticate(ctx context.Context, rawToken string) (*PATAuthResult, error) {
	if len(rawToken) < len(PATPrefix)+8 {
		return nil, fmt.Errorf("token too short")
	}

	prefix := rawToken[:len(PATPrefix)+8]
	expectedHash := hashPAT(rawToken)

	result, err := a.Store.FindUserByPAT(ctx, expectedHash, prefix)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// Update last_used_at asynchronously.
	go func() {
		_ = a.Store.UpdatePATLastUsed(context.Background(), prefix)
	}()

	return result, nil
}

func hashPAT(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
