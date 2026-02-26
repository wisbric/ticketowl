package seed

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/wisbric/core/pkg/auth"
	"github.com/wisbric/core/pkg/tenant"
)

// DevAPIKey is the development API key (never use in production).
const DevAPIKey = "to_dev_seed_key_do_not_use_in_production"

// Run creates the "acme" dev tenant with seed data. Idempotent — re-running
// will ensure all resources exist without duplicating them.
func Run(ctx context.Context, db *pgxpool.Pool, databaseURL, migrationsDir string, logger *slog.Logger) error {
	// Check if tenant already exists.
	var exists bool
	var tenantID uuid.UUID
	err := db.QueryRow(ctx,
		"SELECT id FROM global.tenants WHERE slug = 'acme'",
	).Scan(&tenantID)
	exists = err == nil

	if exists {
		logger.Info("seed: acme tenant already exists, ensuring local admin")
		if err := ensureLocalAdmin(ctx, db, tenantID, logger); err != nil {
			return err
		}
		return nil
	}

	// Provision the tenant.
	prov := &tenant.Provisioner{
		DB:            db,
		DatabaseURL:   databaseURL,
		MigrationsDir: migrationsDir,
		Logger:        logger,
	}

	info, err := prov.Provision(ctx, "Acme Corp", "acme")
	if err != nil {
		return fmt.Errorf("provisioning acme tenant: %w", err)
	}

	// Insert dev API key.
	keyHash := auth.HashAPIKey(DevAPIKey)
	_, err = db.Exec(ctx,
		"INSERT INTO global.api_keys (tenant_id, key_hash, description) VALUES ($1, $2, $3)",
		info.ID, keyHash, "Development seed key",
	)
	if err != nil {
		return fmt.Errorf("inserting dev API key: %w", err)
	}

	// Acquire a tenant-scoped connection for seed data.
	conn, err := db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", info.Schema)); err != nil {
		return fmt.Errorf("setting search_path: %w", err)
	}

	// Seed default SLA policies.
	if err := seedSLAPolicies(ctx, conn); err != nil {
		return fmt.Errorf("seeding SLA policies: %w", err)
	}

	// Seed Zammad config (dev placeholder).
	if err := seedZammadConfig(ctx, conn); err != nil {
		return fmt.Errorf("seeding Zammad config: %w", err)
	}

	// Seed integration keys (dev placeholders).
	if err := seedIntegrationKeys(ctx, conn); err != nil {
		return fmt.Errorf("seeding integration keys: %w", err)
	}

	// Create local admin.
	if err := ensureLocalAdmin(ctx, db, info.ID, logger); err != nil {
		return err
	}

	logger.Info("seed: acme tenant created successfully",
		"tenant_id", info.ID,
		"api_key", DevAPIKey,
	)

	return nil
}

// ensureLocalAdmin creates the local admin account if it doesn't already exist.
// Uses ON CONFLICT to be idempotent.
func ensureLocalAdmin(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, logger *slog.Logger) error {
	localAdminPassword := "ticketowl-admin"
	adminPasswordHash, err := bcrypt.GenerateFromPassword([]byte(localAdminPassword), 12)
	if err != nil {
		return fmt.Errorf("hashing local admin password: %w", err)
	}

	tag, err := pool.Exec(ctx,
		"INSERT INTO public.local_admins (tenant_id, username, password_hash, must_change) VALUES ($1, 'admin', $2, true) ON CONFLICT (tenant_id) DO NOTHING",
		tenantID, string(adminPasswordHash),
	)
	if err != nil {
		return fmt.Errorf("creating local admin: %w", err)
	}

	if tag.RowsAffected() > 0 {
		logger.Info("seed: created local admin", "username", "admin", "password", localAdminPassword)
	} else {
		logger.Info("seed: local admin already exists")
	}
	return nil
}

func seedSLAPolicies(ctx context.Context, conn *pgxpool.Conn) error {
	policies := []struct {
		name              string
		priority          string
		responseMinutes   int
		resolutionMinutes int
	}{
		{"Critical SLA", "critical", 15, 60},
		{"High SLA", "high", 30, 240},
		{"Normal SLA", "normal", 60, 480},
		{"Low SLA", "low", 120, 1440},
	}

	for _, p := range policies {
		_, err := conn.Exec(ctx,
			`INSERT INTO sla_policies (name, priority, response_minutes, resolution_minutes, is_default)
			 VALUES ($1, $2, $3, $4, true)`,
			p.name, p.priority, p.responseMinutes, p.resolutionMinutes,
		)
		if err != nil {
			return fmt.Errorf("inserting SLA policy %s: %w", p.name, err)
		}
	}

	return nil
}

func seedZammadConfig(ctx context.Context, conn *pgxpool.Conn) error {
	_, err := conn.Exec(ctx,
		`INSERT INTO zammad_config (url, api_token, webhook_secret)
		 VALUES ($1, $2, $3)`,
		"http://localhost:3003",
		"dev-zammad-token-placeholder",
		"dev-webhook-secret-placeholder",
	)
	return err
}

func seedIntegrationKeys(ctx context.Context, conn *pgxpool.Conn) error {
	integrations := []struct {
		service string
		apiKey  string
		apiURL  string
	}{
		{"nightowl", "ow_dev_seed_key_do_not_use_in_production", "http://localhost:8080"},
		{"bookowl", "bw_dev_seed_key_do_not_use_in_production", "http://localhost:8081"},
	}

	for _, i := range integrations {
		_, err := conn.Exec(ctx,
			`INSERT INTO integration_keys (service, api_key, api_url)
			 VALUES ($1, $2, $3)`,
			i.service, i.apiKey, i.apiURL,
		)
		if err != nil {
			return fmt.Errorf("inserting integration key %s: %w", i.service, err)
		}
	}

	return nil
}
