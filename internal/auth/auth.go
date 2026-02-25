package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/google/uuid"
)

// Roles supported by the RBAC system.
const (
	RoleAdmin    = "admin"
	RoleAgent    = "agent"
	RoleViewer   = "viewer"
	RoleCustomer = "customer"
)

// ValidRoles lists all known roles in descending privilege order.
var ValidRoles = []string{RoleAdmin, RoleAgent, RoleViewer, RoleCustomer}

// Method describes how the caller was authenticated.
const (
	MethodOIDC   = "oidc"
	MethodAPIKey = "apikey"
	MethodDev    = "dev"
)

// Identity represents the authenticated caller for the current request.
type Identity struct {
	Subject    string     // OIDC sub or "apikey:<prefix>"
	Email      string     // User email (empty for API keys)
	Role       string     // One of the Role* constants
	TenantSlug string     // Resolved tenant slug
	TenantID   uuid.UUID  // Resolved tenant ID
	UserID     *uuid.UUID // Non-nil for OIDC-authenticated users
	APIKeyID   *uuid.UUID // Non-nil for API key authentication
	OrgID      *uuid.UUID // Non-nil for customer users (org scoping)
	Method     string     // One of the Method* constants
}

type ctxKey string

const identityKey ctxKey = "auth_identity"

// NewContext stores the identity in the context.
func NewContext(ctx context.Context, id *Identity) context.Context {
	return context.WithValue(ctx, identityKey, id)
}

// FromContext extracts the identity from the context.
// Returns nil if no identity is set.
func FromContext(ctx context.Context) *Identity {
	v, _ := ctx.Value(identityKey).(*Identity)
	return v
}

// IsValidRole reports whether role is a recognised RBAC role.
func IsValidRole(role string) bool {
	for _, r := range ValidRoles {
		if r == role {
			return true
		}
	}
	return false
}

// HashAPIKey returns the SHA-256 hex digest of a raw API key.
func HashAPIKey(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
