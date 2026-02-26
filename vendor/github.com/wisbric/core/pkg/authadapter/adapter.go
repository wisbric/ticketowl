package authadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wisbric/core/pkg/auth"
)

// TenantQuerier provides tenant lookup operations. Each service implements this
// using its own sqlc-generated code.
type TenantQuerier interface {
	GetTenantBySlug(ctx context.Context, slug string) (*auth.TenantResult, error)
	ListTenants(ctx context.Context) ([]auth.TenantResult, error)
}

// BaseAdapter implements the auth.Storage methods that are identical across
// services — cross-tenant user/PAT scanning, local admin CRUD, and OIDC config.
// Services embed this and add their own sqlc-specific methods (GetAPIKeyByHash,
// GetTenant, GetTenantBySlug, ListTenants, FindOrCreateOIDCUser, etc.).
type BaseAdapter struct {
	Pool *pgxpool.Pool
	TQ   TenantQuerier
}

func (a *BaseAdapter) FindUserByEmail(ctx context.Context, email string) (*auth.UserRow, string, string, error) {
	tenants, err := a.TQ.ListTenants(ctx)
	if err != nil {
		return nil, "", "", fmt.Errorf("listing tenants: %w", err)
	}

	for _, t := range tenants {
		conn, err := a.Pool.Acquire(ctx)
		if err != nil {
			return nil, "", "", fmt.Errorf("acquiring connection: %w", err)
		}

		_, err = conn.Exec(ctx, fmt.Sprintf("SET search_path TO tenant_%s, public", t.Slug))
		if err != nil {
			conn.Release()
			continue
		}

		var u auth.UserRow
		err = conn.QueryRow(ctx,
			"SELECT id, external_id, email, display_name, timezone, phone, slack_user_id, role, is_active, password_hash FROM users WHERE email = $1 AND is_active = true",
			email,
		).Scan(
			&u.ID, &u.ExternalID, &u.Email, &u.DisplayName, &u.Timezone,
			&u.Phone, &u.SlackUserID, &u.Role, &u.IsActive, &u.PasswordHash,
		)
		conn.Release()

		if err == nil {
			return &u, t.Slug, t.ID.String(), nil
		}
	}

	return nil, "", "", fmt.Errorf("user not found")
}

func (a *BaseAdapter) FindUserByPAT(ctx context.Context, expectedHash, prefix string) (*auth.PATAuthResult, error) {
	tenants, err := a.TQ.ListTenants(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing tenants: %w", err)
	}

	for _, t := range tenants {
		conn, err := a.Pool.Acquire(ctx)
		if err != nil {
			return nil, fmt.Errorf("acquiring connection: %w", err)
		}

		schema := fmt.Sprintf("tenant_%s", t.Slug)
		_, err = conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", schema))
		if err != nil {
			conn.Release()
			continue
		}

		var tokenHash string
		var userID uuid.UUID
		var expiresAt *time.Time
		err = conn.QueryRow(ctx,
			"SELECT token_hash, user_id, expires_at FROM personal_access_tokens WHERE prefix = $1",
			prefix,
		).Scan(&tokenHash, &userID, &expiresAt)

		if err != nil {
			conn.Release()
			continue
		}

		if tokenHash != expectedHash {
			conn.Release()
			return nil, fmt.Errorf("invalid token")
		}

		if expiresAt != nil && expiresAt.Before(time.Now()) {
			conn.Release()
			return nil, fmt.Errorf("token expired at %s", expiresAt)
		}

		var email, displayName, role string
		err = conn.QueryRow(ctx,
			"SELECT email, display_name, role FROM users WHERE id = $1 AND is_active = true",
			userID,
		).Scan(&email, &displayName, &role)
		conn.Release()

		if err != nil {
			return nil, fmt.Errorf("looking up user for PAT: %w", err)
		}

		return &auth.PATAuthResult{
			UserID:      userID,
			Email:       email,
			DisplayName: displayName,
			Role:        role,
			TenantSlug:  t.Slug,
			TenantID:    t.ID,
		}, nil
	}

	return nil, fmt.Errorf("token not found")
}

func (a *BaseAdapter) UpdatePATLastUsed(ctx context.Context, prefix string) error {
	tenants, err := a.TQ.ListTenants(ctx)
	if err != nil {
		return err
	}

	for _, t := range tenants {
		c, err := a.Pool.Acquire(ctx)
		if err != nil {
			return err
		}
		schema := fmt.Sprintf("tenant_%s", t.Slug)
		_, _ = c.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", schema))
		tag, err := c.Exec(ctx, "UPDATE personal_access_tokens SET last_used_at = now() WHERE prefix = $1", prefix)
		c.Release()
		if err == nil && tag.RowsAffected() > 0 {
			return nil
		}
	}
	return nil
}

func (a *BaseAdapter) GetDevAdminUser(ctx context.Context, tenantSlug string) (uuid.UUID, string, string, error) {
	if _, err := a.TQ.GetTenantBySlug(ctx, tenantSlug); err != nil {
		return uuid.Nil, "", "", err
	}
	schema := fmt.Sprintf("tenant_%s", tenantSlug)
	var userID uuid.UUID
	var email, displayName string
	err := a.Pool.QueryRow(ctx,
		fmt.Sprintf("SELECT id, email, display_name FROM %s.users WHERE role = 'admin' AND is_active = true LIMIT 1", schema),
	).Scan(&userID, &email, &displayName)
	return userID, email, displayName, err
}

func (a *BaseAdapter) FindLocalAdmin(ctx context.Context, username, tenantSlug string) (*auth.LocalAdminRow, string, error) {
	if tenantSlug != "" {
		t, err := a.TQ.GetTenantBySlug(ctx, tenantSlug)
		if err != nil {
			return nil, "", fmt.Errorf("looking up tenant %s: %w", tenantSlug, err)
		}

		var admin auth.LocalAdminRow
		err = a.Pool.QueryRow(ctx,
			"SELECT id, tenant_id, username, password_hash, must_change FROM public.local_admins WHERE tenant_id = $1 AND username = $2",
			t.ID, username,
		).Scan(&admin.ID, &admin.TenantID, &admin.Username, &admin.PasswordHash, &admin.MustChange)
		if err != nil {
			return nil, "", fmt.Errorf("local admin not found for tenant %s: %w", tenantSlug, err)
		}
		return &admin, t.Slug, nil
	}

	tenants, err := a.TQ.ListTenants(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("listing tenants: %w", err)
	}

	for _, t := range tenants {
		var admin auth.LocalAdminRow
		err := a.Pool.QueryRow(ctx,
			"SELECT id, tenant_id, username, password_hash, must_change FROM public.local_admins WHERE tenant_id = $1 AND username = $2",
			t.ID, username,
		).Scan(&admin.ID, &admin.TenantID, &admin.Username, &admin.PasswordHash, &admin.MustChange)
		if err == nil {
			return &admin, t.Slug, nil
		}
	}

	return nil, "", fmt.Errorf("local admin not found")
}

func (a *BaseAdapter) UpdateLocalAdminLastLogin(ctx context.Context, adminID uuid.UUID) error {
	_, err := a.Pool.Exec(ctx, "UPDATE public.local_admins SET last_login_at = now() WHERE id = $1", adminID)
	return err
}

func (a *BaseAdapter) GetLocalAdminPasswordHash(ctx context.Context, adminID uuid.UUID) (string, error) {
	var currentHash string
	err := a.Pool.QueryRow(ctx, "SELECT password_hash FROM public.local_admins WHERE id = $1", adminID).Scan(&currentHash)
	return currentHash, err
}

func (a *BaseAdapter) UpdateLocalAdminPassword(ctx context.Context, adminID uuid.UUID, newHash string, mustChange bool) error {
	_, err := a.Pool.Exec(ctx,
		"UPDATE public.local_admins SET password_hash = $1, must_change = $2, updated_at = now() WHERE id = $3",
		newHash, mustChange, adminID,
	)
	return err
}

func (a *BaseAdapter) ResetLocalAdminPassword(ctx context.Context, tenantID uuid.UUID, newHash string) error {
	_, err := a.Pool.Exec(ctx,
		"UPDATE public.local_admins SET password_hash = $1, must_change = true, updated_at = now() WHERE tenant_id = $2",
		newHash, tenantID,
	)
	return err
}

func (a *BaseAdapter) GetOIDCConfig(ctx context.Context, tenantSlug string) (*auth.OIDCConfigRow, error) {
	t, err := a.TQ.GetTenantBySlug(ctx, tenantSlug)
	if err != nil {
		return nil, fmt.Errorf("looking up tenant: %w", err)
	}

	conn, err := a.Pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	schema := fmt.Sprintf("tenant_%s", t.Slug)
	if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", schema)); err != nil {
		return nil, fmt.Errorf("setting search_path: %w", err)
	}

	var row auth.OIDCConfigRow
	err = conn.QueryRow(ctx,
		"SELECT id, issuer_url, client_id, enabled, tested_at FROM oidc_config LIMIT 1",
	).Scan(&row.ID, &row.IssuerURL, &row.ClientID, &row.Enabled, &row.TestedAt)
	if err != nil {
		return nil, fmt.Errorf("querying oidc_config: %w", err)
	}
	return &row, nil
}

func (a *BaseAdapter) UpsertOIDCConfig(ctx context.Context, tenantSlug, issuerURL, clientID, encryptedSecret string, enabled bool) error {
	t, err := a.TQ.GetTenantBySlug(ctx, tenantSlug)
	if err != nil {
		return fmt.Errorf("looking up tenant: %w", err)
	}

	conn, err := a.Pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	schema := fmt.Sprintf("tenant_%s", t.Slug)
	if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", schema)); err != nil {
		return fmt.Errorf("setting search_path: %w", err)
	}

	var existingID uuid.UUID
	err = conn.QueryRow(ctx, "SELECT id FROM oidc_config LIMIT 1").Scan(&existingID)
	if err != nil {
		_, err = conn.Exec(ctx,
			"INSERT INTO oidc_config (issuer_url, client_id, client_secret, enabled) VALUES ($1, $2, $3, $4)",
			issuerURL, clientID, encryptedSecret, enabled,
		)
	} else {
		_, err = conn.Exec(ctx,
			"UPDATE oidc_config SET issuer_url = $1, client_id = $2, client_secret = $3, enabled = $4, updated_at = now() WHERE id = $5",
			issuerURL, clientID, encryptedSecret, enabled, existingID,
		)
	}
	return err
}

func (a *BaseAdapter) UpdateOIDCTestedAt(ctx context.Context, tenantSlug string, testedAt time.Time) error {
	t, err := a.TQ.GetTenantBySlug(ctx, tenantSlug)
	if err != nil {
		return err
	}

	conn, err := a.Pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	schema := fmt.Sprintf("tenant_%s", t.Slug)
	if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", schema)); err != nil {
		return err
	}

	_, err = conn.Exec(ctx, "UPDATE oidc_config SET tested_at = $1", testedAt)
	return err
}
