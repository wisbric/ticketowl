package platform

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunGlobalMigrations applies migrations from the global directory to the public schema.
func RunGlobalMigrations(databaseURL, migrationsDir string) error {
	return runMigrations(databaseURL, migrationsDir)
}

// RunTenantMigrations applies tenant schema migrations.
// The databaseURL should include search_path set to the tenant schema.
func RunTenantMigrations(databaseURL, migrationsDir string) error {
	return runMigrations(databaseURL, migrationsDir)
}

func runMigrations(databaseURL, migrationsDir string) error {
	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsDir),
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf("creating migrator: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("running migrations: %w", err)
	}

	return nil
}
