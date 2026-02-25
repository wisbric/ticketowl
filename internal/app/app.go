package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/wisbric/ticketowl/internal/auth"
	"github.com/wisbric/ticketowl/internal/config"
	"github.com/wisbric/ticketowl/internal/httpserver"
	"github.com/wisbric/ticketowl/internal/platform"
	"github.com/wisbric/ticketowl/internal/seed"
	"github.com/wisbric/ticketowl/internal/telemetry"
	"github.com/wisbric/ticketowl/internal/version"
)

// Run is the main application entry point. It reads config, connects to
// infrastructure, and starts the appropriate mode.
func Run(ctx context.Context, cfg *config.Config) error {
	logger := telemetry.NewLogger(cfg.LogFormat, cfg.LogLevel)
	slog.SetDefault(logger)

	logger.Info("starting ticketowl",
		"mode", cfg.Mode,
		"listen", cfg.ListenAddr(),
		"version", version.Version,
	)

	// Tracing
	shutdownTracer, err := telemetry.InitTracer(ctx, cfg.OTLPEndpoint, "ticketowl", version.Version)
	if err != nil {
		return fmt.Errorf("initializing tracer: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTracer(shutdownCtx); err != nil {
			logger.Error("shutting down tracer", "error", err)
		}
	}()

	// Metrics
	metricsReg := telemetry.NewMetricsRegistry()

	switch cfg.Mode {
	case "api":
		return runAPI(ctx, cfg, logger, metricsReg)
	case "worker":
		return runWorker(ctx, cfg, logger)
	case "seed":
		return runSeed(ctx, cfg, logger)
	case "seed-demo":
		return runSeedDemo(ctx, cfg, logger)
	case "migrate":
		return runMigrate(cfg, logger)
	default:
		return fmt.Errorf("unknown mode: %s (expected: api, worker, seed, seed-demo, migrate)", cfg.Mode)
	}
}

func runAPI(ctx context.Context, cfg *config.Config, logger *slog.Logger, metricsReg *prometheus.Registry) error {
	// Database
	db, err := platform.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer db.Close()

	// Redis
	rdb, err := platform.NewRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("connecting to redis: %w", err)
	}
	defer func() {
		if err := rdb.Close(); err != nil {
			logger.Error("closing redis", "error", err)
		}
	}()

	// Run global migrations before starting.
	if err := platform.RunGlobalMigrations(cfg.DatabaseURL, cfg.MigrationsGlobalDir); err != nil {
		logger.Warn("global migrations failed (may not exist yet)", "error", err)
	} else {
		logger.Info("global migrations applied")
	}

	// OIDC authenticator (optional).
	var oidcAuth *auth.OIDCAuthenticator
	if cfg.OIDCIssuerURL != "" && cfg.OIDCClientID != "" {
		oidcAuth, err = auth.NewOIDCAuthenticator(ctx, cfg.OIDCIssuerURL, cfg.OIDCClientID)
		if err != nil {
			return fmt.Errorf("initializing OIDC authenticator: %w", err)
		}
		logger.Info("OIDC authentication enabled", "issuer", cfg.OIDCIssuerURL)
	} else {
		logger.Info("OIDC authentication disabled (TICKETOWL_OIDC_ISSUER not set)")
	}

	srv := httpserver.NewServer(cfg, logger, db, rdb, metricsReg, oidcAuth)

	// Mount status endpoint.
	srv.APIRouter.Get("/status", srv.HandleStatus)

	// Domain handlers will be mounted here in later phases.

	httpSrv := &http.Server{
		Addr:         cfg.ListenAddr(),
		Handler:      srv,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("api server listening", "addr", cfg.ListenAddr())
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server: %w", err)
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutting down api server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpSrv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func runWorker(ctx context.Context, _ *config.Config, logger *slog.Logger) error {
	logger.Info("worker mode not yet implemented")
	<-ctx.Done()
	return nil
}

func runSeed(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	db, err := platform.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer db.Close()

	// Run global migrations first.
	if err := platform.RunGlobalMigrations(cfg.DatabaseURL, cfg.MigrationsGlobalDir); err != nil {
		return fmt.Errorf("running global migrations: %w", err)
	}
	logger.Info("global migrations applied")

	return seed.Run(ctx, db, cfg.DatabaseURL, cfg.MigrationsTenantDir, logger)
}

func runSeedDemo(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	db, err := platform.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer db.Close()

	// Run global migrations first.
	if err := platform.RunGlobalMigrations(cfg.DatabaseURL, cfg.MigrationsGlobalDir); err != nil {
		return fmt.Errorf("running global migrations: %w", err)
	}
	logger.Info("global migrations applied")

	return seed.RunDemo(ctx, db, cfg.DatabaseURL, cfg.MigrationsTenantDir, logger)
}

func runMigrate(cfg *config.Config, logger *slog.Logger) error {
	if err := platform.RunGlobalMigrations(cfg.DatabaseURL, cfg.MigrationsGlobalDir); err != nil {
		return fmt.Errorf("running global migrations: %w", err)
	}
	logger.Info("global migrations applied")
	return nil
}
