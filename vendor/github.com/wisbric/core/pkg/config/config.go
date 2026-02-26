package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// BaseConfig holds shared infrastructure configuration fields.
// Services embed this and add their own service-specific fields.
type BaseConfig struct {
	Mode string `env:"APP_MODE" envDefault:"api"`

	// Server
	Host string `env:"APP_HOST" envDefault:"0.0.0.0"`
	Port int    `env:"APP_PORT" envDefault:"8080"`

	// Database
	DatabaseURL string `env:"DATABASE_URL" envDefault:"postgres://localhost:5432/app?sslmode=disable"`

	// Redis
	RedisURL string `env:"REDIS_URL" envDefault:"redis://localhost:6379/0"`

	// Logging
	LogLevel  string `env:"LOG_LEVEL" envDefault:"info"`
	LogFormat string `env:"LOG_FORMAT" envDefault:"json"`

	// Telemetry
	OTLPEndpoint string `env:"OTEL_ENDPOINT"`

	// Migrations
	MigrationsGlobalDir string `env:"MIGRATIONS_GLOBAL_DIR" envDefault:"migrations/global"`
	MigrationsTenantDir string `env:"MIGRATIONS_TENANT_DIR" envDefault:"migrations/tenant"`

	// CORS
	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS" envDefault:"*" envSeparator:","`

	// OIDC
	OIDCIssuerURL string `env:"OIDC_ISSUER"`
	OIDCClientID  string `env:"OIDC_CLIENT_ID"`

	// Dev mode
	DevMode bool `env:"DEV_MODE" envDefault:"false"`
}

// Load reads configuration from environment variables into a struct of type T.
// T should embed BaseConfig and add service-specific fields.
func Load[T any]() (*T, error) {
	cfg := new(T)
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parsing config from env: %w", err)
	}
	return cfg, nil
}

// ListenAddr returns the address the HTTP server should listen on.
func (c *BaseConfig) ListenAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
