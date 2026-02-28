package config

import (
	coreconfig "github.com/wisbric/core/pkg/config"
)

// Config holds all TicketOwl-specific configuration, embedding shared infra fields.
type Config struct {
	coreconfig.BaseConfig

	// Zammad
	ZammadURL   string `env:"TICKETOWL_ZAMMAD_URL"`
	ZammadToken string `env:"TICKETOWL_ZAMMAD_TOKEN"`

	// NightOwl integration
	NightOwlAPIURL string `env:"TICKETOWL_NIGHTOWL_API_URL"`
	NightOwlAPIKey string `env:"TICKETOWL_NIGHTOWL_API_KEY"`

	// BookOwl integration
	BookOwlAPIURL string `env:"TICKETOWL_BOOKOWL_API_URL"`
	BookOwlAPIKey string `env:"TICKETOWL_BOOKOWL_API_KEY"`

	// Cross-service links (public URLs for sidebar navigation)
	NightOwlURL string `env:"TICKETOWL_NIGHTOWL_URL"`
	BookOwlURL  string `env:"TICKETOWL_BOOKOWL_URL"`

	// Worker
	WorkerPollSeconds int `env:"TICKETOWL_WORKER_POLL_SECONDS" envDefault:"60"`

	// OIDC (TicketOwl-specific: Authorization Code flow)
	OIDCClientSecret string `env:"TICKETOWL_OIDC_CLIENT_SECRET"`
	OIDCRedirectURL  string `env:"TICKETOWL_OIDC_REDIRECT_URL" envDefault:"http://localhost:3002/auth/oidc/callback"`

	// Session
	SessionSecret string `env:"TICKETOWL_SESSION_SECRET"`
	SessionMaxAge string `env:"TICKETOWL_SESSION_MAX_AGE" envDefault:"24h"`

	// Admin
	AdminPassword string `env:"TICKETOWL_ADMIN_PASSWORD"`

	// Encryption
	EncryptionKey string `env:"TICKETOWL_ENCRYPTION_KEY"`
}
