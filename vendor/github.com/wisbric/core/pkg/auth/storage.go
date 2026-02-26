package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TenantResult represents a tenant record needed for authentication.
type TenantResult struct {
	ID   uuid.UUID
	Slug string
}

// UserRow represents the necessary user fields for login.
type UserRow struct {
	ID           uuid.UUID
	ExternalID   *string
	Email        string
	DisplayName  string
	Timezone     string
	Phone        *string
	SlackUserID  *string
	Role         string
	IsActive     bool
	PasswordHash *string
}

// LocalAdminRow represents the necessary admin fields for login.
type LocalAdminRow struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	Username     string
	PasswordHash string
	MustChange   bool
}

// OIDCConfigRow represents a tenant's OIDC configuration.
type OIDCConfigRow struct {
	ID           uuid.UUID
	IssuerURL    string
	ClientID     string
	ClientSecret string
	Enabled      bool
	TestedAt     *time.Time
}

// Storage abstracts the database operations required by the auth package.
// This allows the core `auth` package to be decoupled from any specific microservice's database schema.
type Storage interface {
	// API Keys
	GetAPIKeyByHash(ctx context.Context, hash string) (*APIKeyResult, error)
	UpdateAPIKeyLastUsed(ctx context.Context, keyID uuid.UUID) error

	// Tenants
	GetTenant(ctx context.Context, tenantID uuid.UUID) (*TenantResult, error)
	GetTenantBySlug(ctx context.Context, slug string) (*TenantResult, error)
	ListTenants(ctx context.Context) ([]TenantResult, error)

	// Users (Cross-tenant search)
	FindUserByEmail(ctx context.Context, email string) (*UserRow, string, string, error)
	FindUserByPAT(ctx context.Context, expectedHash, prefix string) (*PATAuthResult, error)
	UpdatePATLastUsed(ctx context.Context, prefix string) error

	// OIDC Users
	FindOrCreateOIDCUser(ctx context.Context, tenantSlug, subject, email, role string) (*UserRow, string, error)

	// Dev Mode
	GetDevAdminUser(ctx context.Context, tenantSlug string) (uuid.UUID, string, string, error)

	// Local Admins
	FindLocalAdmin(ctx context.Context, username, tenantSlug string) (*LocalAdminRow, string, error)
	UpdateLocalAdminLastLogin(ctx context.Context, adminID uuid.UUID) error
	GetLocalAdminPasswordHash(ctx context.Context, adminID uuid.UUID) (string, error)
	UpdateLocalAdminPassword(ctx context.Context, adminID uuid.UUID, newHash string, mustChange bool) error
	ResetLocalAdminPassword(ctx context.Context, tenantID uuid.UUID, newHash string) error

	// OIDC Admin
	GetOIDCConfig(ctx context.Context, tenantSlug string) (*OIDCConfigRow, error)
	UpsertOIDCConfig(ctx context.Context, tenantSlug, issuerURL, clientID, encryptedSecret string, enabled bool) error
	UpdateOIDCTestedAt(ctx context.Context, tenantSlug string, testedAt time.Time) error
}
