package tenant

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wisbric/ticketowl/internal/platform"
)

var slugRegex = regexp.MustCompile(`^[a-z][a-z0-9_]{1,62}$`)

// Provisioner creates new tenants with their own database schema.
type Provisioner struct {
	DB            *pgxpool.Pool
	DatabaseURL   string
	MigrationsDir string
	Logger        *slog.Logger
}

// Provision creates a new tenant: inserts the global row, creates the schema,
// and runs tenant migrations.
func (p *Provisioner) Provision(ctx context.Context, name, slug string) (*Info, error) {
	if !slugRegex.MatchString(slug) {
		return nil, fmt.Errorf("invalid tenant slug: %q", slug)
	}

	// Insert into global.tenants.
	var tenantID uuid.UUID
	err := p.DB.QueryRow(ctx,
		"INSERT INTO global.tenants (name, slug) VALUES ($1, $2) RETURNING id",
		name, slug,
	).Scan(&tenantID)
	if err != nil {
		return nil, fmt.Errorf("inserting tenant: %w", err)
	}

	schema := SchemaName(slug)

	// Create the tenant schema.
	if _, err := p.DB.Exec(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema)); err != nil {
		// Best-effort cleanup.
		_, _ = p.DB.Exec(ctx, "DELETE FROM global.tenants WHERE id = $1", tenantID)
		return nil, fmt.Errorf("creating schema %s: %w", schema, err)
	}

	// Run tenant migrations against the new schema.
	tenantURL, err := withSearchPath(p.DatabaseURL, schema)
	if err != nil {
		return nil, fmt.Errorf("building tenant database URL: %w", err)
	}

	if err := platform.RunTenantMigrations(tenantURL, p.MigrationsDir); err != nil {
		// Best-effort cleanup.
		_, _ = p.DB.Exec(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema))
		_, _ = p.DB.Exec(ctx, "DELETE FROM global.tenants WHERE id = $1", tenantID)
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

// withSearchPath returns a modified database URL with the search_path set.
func withSearchPath(databaseURL, schema string) (string, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return "", fmt.Errorf("parsing database URL: %w", err)
	}

	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()

	return u.String(), nil
}
