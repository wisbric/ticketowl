package authadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wisbric/core/pkg/auth"
	"github.com/wisbric/core/pkg/authadapter"
	"github.com/wisbric/ticketowl/internal/db"
)

// Adapter implements auth.Storage for TicketOwl. It embeds the shared
// BaseAdapter for cross-tenant scan methods and adds sqlc-specific operations.
type Adapter struct {
	authadapter.BaseAdapter
	pool *pgxpool.Pool
}

// New creates a new auth storage adapter.
func New(pool *pgxpool.Pool) *Adapter {
	a := &Adapter{pool: pool}
	a.BaseAdapter = authadapter.BaseAdapter{Pool: pool, TQ: a}
	return a
}

// --- sqlc-based methods (service-specific) ---

func (a *Adapter) GetAPIKeyByHash(ctx context.Context, hash string) (*auth.APIKeyResult, error) {
	q := db.New(a.pool)
	key, err := q.GetAPIKeyByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	var expiresAt *time.Time
	// ticketowl doesn't support ExpiresAt yet
	return &auth.APIKeyResult{
		APIKeyID:  key.ID,
		TenantID:  key.TenantID,
		KeyPrefix: "",
		Role:      "admin",
		Scopes:    []string{},
		ExpiresAt: expiresAt,
	}, nil
}

func (a *Adapter) UpdateAPIKeyLastUsed(ctx context.Context, keyID uuid.UUID) error {
	q := db.New(a.pool)
	return q.UpdateAPIKeyLastUsed(ctx, keyID)
}

func (a *Adapter) GetTenant(ctx context.Context, tenantID uuid.UUID) (*auth.TenantResult, error) {
	q := db.New(a.pool)
	t, err := q.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return &auth.TenantResult{ID: t.ID, Slug: t.Slug}, nil
}

func (a *Adapter) GetTenantBySlug(ctx context.Context, slug string) (*auth.TenantResult, error) {
	q := db.New(a.pool)
	t, err := q.GetTenantBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return &auth.TenantResult{ID: t.ID, Slug: t.Slug}, nil
}

func (a *Adapter) ListTenants(ctx context.Context) ([]auth.TenantResult, error) {
	q := db.New(a.pool)
	tenants, err := q.ListTenants(ctx)
	if err != nil {
		return nil, err
	}
	var res []auth.TenantResult
	for _, t := range tenants {
		res = append(res, auth.TenantResult{ID: t.ID, Slug: t.Slug})
	}
	return res, nil
}

func (a *Adapter) FindOrCreateOIDCUser(ctx context.Context, tenantSlug, subject, email, role string) (*auth.UserRow, string, error) {
	return nil, "", fmt.Errorf("OIDC user provisioning not supported in ticketowl")
}
