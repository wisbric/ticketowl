package tenant

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wisbric/core/pkg/platform"
)

var slugRegex = regexp.MustCompile(`^[a-z][a-z0-9_]{1,62}$`)

// TenantStore abstracts tenant CRUD operations so services can plug in their
// own sqlc-generated code instead of raw SQL.
type TenantStore interface {
	CreateTenant(ctx context.Context, name, slug string) (uuid.UUID, error)
	DeleteTenant(ctx context.Context, id uuid.UUID) error
}

// DefaultStore provides a raw-SQL TenantStore implementation.
type DefaultStore struct {
	Pool *pgxpool.Pool
}

func (s *DefaultStore) CreateTenant(ctx context.Context, name, slug string) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx,
		"INSERT INTO public.tenants (name, slug) VALUES ($1, $2) RETURNING id",
		name, slug,
	).Scan(&id)
	return id, err
}

func (s *DefaultStore) DeleteTenant(ctx context.Context, id uuid.UUID) error {
	_, err := s.Pool.Exec(ctx, "DELETE FROM public.tenants WHERE id = $1", id)
	return err
}

// Provisioner creates new tenants with their own database schema.
type Provisioner struct {
	DB            *pgxpool.Pool
	Store         TenantStore // if nil, uses DefaultStore with raw SQL
	DatabaseURL   string
	MigrationsDir string
	Logger        *slog.Logger
}

func (p *Provisioner) store() TenantStore {
	if p.Store != nil {
		return p.Store
	}
	return &DefaultStore{Pool: p.DB}
}

// Provision creates a new tenant: inserts the global row, creates the schema,
// and runs tenant migrations.
func (p *Provisioner) Provision(ctx context.Context, name, slug string) (*Info, error) {
	if !slugRegex.MatchString(slug) {
		return nil, fmt.Errorf("invalid tenant slug: %q", slug)
	}

	tenantID, err := p.store().CreateTenant(ctx, name, slug)
	if err != nil {
		return nil, fmt.Errorf("inserting tenant: %w", err)
	}

	schema := SchemaName(slug)

	// Create the tenant schema.
	if _, err := p.DB.Exec(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema)); err != nil {
		// Best-effort cleanup.
		_ = p.store().DeleteTenant(ctx, tenantID)
		return nil, fmt.Errorf("creating schema %s: %w", schema, err)
	}

	// Run tenant migrations against the new schema.
	tenantURL, err := WithSearchPath(p.DatabaseURL, schema)
	if err != nil {
		return nil, fmt.Errorf("building tenant database URL: %w", err)
	}

	if err := platform.RunTenantMigrations(tenantURL, p.MigrationsDir); err != nil {
		// Best-effort cleanup.
		_, _ = p.DB.Exec(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema))
		_ = p.store().DeleteTenant(ctx, tenantID)
		return nil, fmt.Errorf("running tenant migrations: %w", err)
	}

	p.Logger.Info("tenant provisioned",
		"tenant_id", tenantID,
		"slug", slug,
		"schema", schema,
	)

	return &Info{
		ID:     tenantID,
		Name:   name,
		Slug:   slug,
		Schema: schema,
	}, nil
}

// Deprovision drops the tenant schema and removes the global record.
func (p *Provisioner) Deprovision(ctx context.Context, slug string) error {
	schema := SchemaName(slug)

	if _, err := p.DB.Exec(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema)); err != nil {
		return fmt.Errorf("dropping schema %s: %w", schema, err)
	}

	// Look up tenant ID.
	var tenantID uuid.UUID
	err := p.DB.QueryRow(ctx,
		"SELECT id FROM public.tenants WHERE slug = $1", slug,
	).Scan(&tenantID)
	if err != nil {
		return fmt.Errorf("looking up tenant %q: %w", slug, err)
	}

	if err := p.store().DeleteTenant(ctx, tenantID); err != nil {
		return fmt.Errorf("deleting tenant record: %w", err)
	}

	p.Logger.Info("tenant deprovisioned", "slug", slug, "schema", schema)
	return nil
}

// WithSearchPath returns a modified database URL with the search_path set.
func WithSearchPath(databaseURL, schema string) (string, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return "", fmt.Errorf("parsing database URL: %w", err)
	}

	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()

	return u.String(), nil
}
