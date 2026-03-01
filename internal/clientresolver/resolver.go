package clientresolver

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/wisbric/ticketowl/internal/admin"
	"github.com/wisbric/ticketowl/internal/bookowl"
	"github.com/wisbric/ticketowl/internal/db"
	"github.com/wisbric/ticketowl/internal/nightowl"
	"github.com/wisbric/ticketowl/internal/zammad"
)

// ZammadClient reads the tenant's zammad_config and returns a configured client.
func ZammadClient(ctx context.Context, dbtx db.DBTX, logger *slog.Logger) (*zammad.Client, error) {
	store := admin.NewStore(dbtx)
	cfg, err := store.GetZammadConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting zammad config: %w", err)
	}
	return zammad.New(cfg.URL, cfg.APIToken, zammad.WithLogger(logger)), nil
}

// NightOwlClient reads the tenant's nightowl integration key and returns a configured client.
func NightOwlClient(ctx context.Context, dbtx db.DBTX, logger *slog.Logger) (*nightowl.Client, error) {
	store := admin.NewStore(dbtx)
	key, err := store.GetIntegrationKey(ctx, "nightowl")
	if err != nil {
		return nil, fmt.Errorf("getting nightowl integration key: %w", err)
	}
	return nightowl.New(key.APIURL, key.APIKey, nightowl.WithLogger(logger)), nil
}

// BookOwlClient reads the tenant's bookowl integration key and returns a configured client.
func BookOwlClient(ctx context.Context, dbtx db.DBTX, logger *slog.Logger) (*bookowl.Client, error) {
	store := admin.NewStore(dbtx)
	key, err := store.GetIntegrationKey(ctx, "bookowl")
	if err != nil {
		return nil, fmt.Errorf("getting bookowl integration key: %w", err)
	}
	return bookowl.New(key.APIURL, key.APIKey, bookowl.WithLogger(logger)), nil
}

// WebhookSecret reads the tenant's Zammad webhook secret from the DB.
func WebhookSecret(ctx context.Context, dbtx db.DBTX) (string, error) {
	store := admin.NewStore(dbtx)
	cfg, err := store.GetZammadConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("getting zammad config for webhook secret: %w", err)
	}
	return cfg.WebhookSecret, nil
}
