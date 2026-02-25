package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config holds all application configuration, loaded from environment variables.
type Config struct {
	// Mode selects the runtime mode: "api", "worker", "seed", "seed-demo", "migrate".
	Mode string `env:"TICKETOWL_MODE" envDefault:"api"`

	// Server
	Host string `env:"TICKETOWL_HOST" envDefault:"0.0.0.0"`
	Port int    `env:"TICKETOWL_PORT" envDefault:"8082"`

	// Database
	DatabaseURL string `env:"TICKETOWL_DB_URL" envDefault:"postgres://ticketowl:ticketowl@localhost:5434/ticketowl?sslmode=disable"`

	// Redis
	RedisURL string `env:"TICKETOWL_REDIS_URL" envDefault:"redis://localhost:6381/0"`

	// Logging
	LogLevel  string `env:"TICKETOWL_LOG_LEVEL" envDefault:"info"`
	LogFormat string `env:"TICKETOWL_LOG_FORMAT" envDefault:"json"`

	// Telemetry
	OTLPEndpoint string `env:"TICKETOWL_OTEL_ENDPOINT"`

	// Migrations
	MigrationsGlobalDir string `env:"TICKETOWL_MIGRATIONS_GLOBAL_DIR" envDefault:"migrations/global"`
	MigrationsTenantDir string `env:"TICKETOWL_MIGRATIONS_TENANT_DIR" envDefault:"migrations/tenant"`

	// CORS
	CORSAllowedOrigins []string `env:"TICKETOWL_CORS_ALLOWED_ORIGINS" envDefault:"*" envSeparator:","`

	// OIDC
	OIDCIssuerURL string `env:"TICKETOWL_OIDC_ISSUER"`
	OIDCClientID  string `env:"TICKETOWL_OIDC_CLIENT_ID"`

	// Encryption
	EncryptionKey string `env:"TICKETOWL_ENCRYPTION_KEY"`

	// NightOwl integration
	NightOwlAPIURL string `env:"TICKETOWL_NIGHTOWL_API_URL"`
	NightOwlAPIKey string `env:"TICKETOWL_NIGHTOWL_API_KEY"`

	// BookOwl integration
	BookOwlAPIURL string `env:"TICKETOWL_BOOKOWL_API_URL"`
	BookOwlAPIKey string `env:"TICKETOWL_BOOKOWL_API_KEY"`

	// Worker
	WorkerPollSeconds int `env:"TICKETOWL_WORKER_POLL_SECONDS" envDefault:"60"`

	// Zammad (for readyz check — per-tenant URL is in the DB, this is the primary instance)
	ZammadURL string `env:"TICKETOWL_ZAMMAD_URL"`

	// Dev mode
	DevMode bool `env:"TICKETOWL_DEV_MODE" envDefault:"false"`
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parsing config from env: %w", err)
	}
	return cfg, nil
}

// ListenAddr returns the address the HTTP server should listen on.
func (c *Config) ListenAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
