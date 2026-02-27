package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/wisbric/core/pkg/auth"
	"github.com/wisbric/core/pkg/httpserver"
	"github.com/wisbric/core/pkg/platform"
	coretelemetry "github.com/wisbric/core/pkg/telemetry"
	"github.com/wisbric/core/pkg/tenant"
	"github.com/wisbric/core/pkg/version"

	"github.com/wisbric/ticketowl/internal/authadapter"
	"github.com/wisbric/ticketowl/internal/config"
	"github.com/wisbric/ticketowl/internal/db"
	"github.com/wisbric/ticketowl/internal/nightowl"
	"github.com/wisbric/ticketowl/internal/notification"
	"github.com/wisbric/ticketowl/internal/seed"
	ticketowlmetrics "github.com/wisbric/ticketowl/internal/telemetry"
	"github.com/wisbric/ticketowl/internal/worker"
)

// Run is the main application entry point. It reads config, connects to
// infrastructure, and starts the appropriate mode.
func Run(ctx context.Context, cfg *config.Config) error {
	logger := coretelemetry.NewLogger(cfg.LogFormat, cfg.LogLevel)
	slog.SetDefault(logger)

	logger.Info("starting ticketowl",
		"mode", cfg.Mode,
		"listen", cfg.ListenAddr(),
		"version", version.Version,
	)

	// Tracing
	shutdownTracer, err := coretelemetry.InitTracer(ctx, cfg.OTLPEndpoint, "ticketowl", version.Version)
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
	metricsReg := coretelemetry.NewMetricsRegistry(ticketowlmetrics.All()...)

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

	// Session manager.
	sessionSecret := cfg.SessionSecret
	if sessionSecret == "" {
		if !cfg.DevMode {
			return fmt.Errorf("missing TICKETOWL_SESSION_SECRET (required when DEV_MODE=false)")
		}
		sessionSecret = auth.GenerateDevSecret()
		logger.Info("session: using auto-generated dev secret (DEV_MODE=true)")
	}
	sessionMaxAge, err := time.ParseDuration(cfg.SessionMaxAge)
	if err != nil {
		return fmt.Errorf("parsing session max age %q: %w", cfg.SessionMaxAge, err)
	}
	sessionMgr, err := auth.NewSessionManager(sessionSecret, sessionMaxAge)
	if err != nil {
		return fmt.Errorf("creating session manager: %w", err)
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

	// Auth storage adapter.
	authStore := authadapter.New(db)

	// PAT authenticator.
	patAuth := auth.NewPATAuthenticator(authStore)

	srv := httpserver.NewServer(httpserver.ServerConfig{
		CORSAllowedOrigins: cfg.CORSAllowedOrigins,
		ZammadURL:          cfg.ZammadURL,
		DevMode:            cfg.DevMode,
	}, logger, db, rdb, metricsReg, sessionMgr, oidcAuth, patAuth, authStore)

	// --- Auth routes (public, pre-authentication) ---

	// Rate limiter: 10 failed attempts per IP per 15 minutes.
	rateLimiter := auth.NewRateLimiter(rdb, 10, 15*time.Minute)

	// Local admin login and change-password.
	localAdminHandler := auth.NewLocalAdminHandler(sessionMgr, authStore, logger, rateLimiter)
	srv.Router.Post("/auth/local", localAdminHandler.HandleLocalLogin)
	srv.Router.Post("/auth/change-password", localAdminHandler.HandleChangePassword)
	srv.Router.Get("/auth/config", localAdminHandler.HandleAuthConfig)

	// Login handler (session info + logout).
	loginHandler := auth.NewLoginHandler(sessionMgr, authStore, logger, oidcAuth != nil, rateLimiter)
	srv.Router.Post("/auth/login", loginHandler.HandleLogin)
	srv.Router.Get("/auth/me", loginHandler.HandleMe)
	srv.Router.Post("/auth/logout", loginHandler.HandleLogout)

	// Public status endpoint (no auth required — used by about page).
	srv.Router.Get("/status", srv.HandleStatus)

	// Authenticated status endpoint (backward compat).
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

func runWorker(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	// Database
	pool, err := platform.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

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

	// NightOwl client for SLA breach alerts.
	noClient := nightowl.New(cfg.NightOwlAPIURL, cfg.NightOwlAPIKey, nightowl.WithLogger(logger))
	notifier := notification.NewService(noClient, logger)

	// List all active tenants.
	q := db.New(pool)
	tenants, err := q.ListTenants(ctx)
	if err != nil {
		return fmt.Errorf("listing tenants: %w", err)
	}

	if len(tenants) == 0 {
		logger.Warn("no tenants found, worker idle")
		<-ctx.Done()
		return nil
	}

	// Launch a Worker per tenant (SLA poller + event processor).
	var wg sync.WaitGroup
	for _, t := range tenants {
		if t.Suspended {
			logger.Info("skipping suspended tenant", "slug", t.Slug)
			continue
		}

		schema := tenant.SchemaName(t.Slug)
		store := worker.NewPollerStore(pool, schema)
		eventHandler := worker.NewDefaultEventHandler(logger.With("tenant", t.Slug))

		w := worker.New(rdb, store, notifier, eventHandler, worker.Config{
			PollIntervalSeconds: cfg.WorkerPollSeconds,
			TenantSlug:          t.Slug,
		}, logger.With("tenant", t.Slug))

		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = w.Run(ctx)
		}()

		logger.Info("worker started for tenant", "slug", t.Slug)
	}

	<-ctx.Done()
	logger.Info("worker shutting down, waiting for tenant workers")
	wg.Wait()
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
