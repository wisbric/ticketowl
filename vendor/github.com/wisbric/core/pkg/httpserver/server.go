package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/wisbric/core/pkg/auth"
	"github.com/wisbric/core/pkg/tenant"
	"github.com/wisbric/core/pkg/version"
)

// ServerConfig holds the parameters NewServer needs, decoupled from any
// service-specific configuration struct.
type ServerConfig struct {
	CORSAllowedOrigins []string
	ZammadURL          string // optional, for readyz
}

// Server holds the HTTP server dependencies.
type Server struct {
	Router    *chi.Mux
	APIRouter chi.Router // authenticated, tenant-scoped /api/v1 sub-router
	Logger    *slog.Logger
	DB        *pgxpool.Pool
	Redis     *redis.Client
	Metrics   *prometheus.Registry
	zammadURL string // primary Zammad URL for readyz check
	startedAt time.Time
}

// NewServer creates an HTTP server with middleware and health/metrics endpoints.
// oidcAuth may be nil when OIDC is not configured.
// Domain handlers should be mounted on APIRouter after calling NewServer.
func NewServer(cfg ServerConfig, logger *slog.Logger, db *pgxpool.Pool, rdb *redis.Client, metricsReg *prometheus.Registry, sessionMgr *auth.SessionManager, oidcAuth *auth.OIDCAuthenticator, patAuth *auth.PATAuthenticator, authStore auth.Storage) *Server {
	s := &Server{
		Router:    chi.NewRouter(),
		Logger:    logger,
		DB:        db,
		Redis:     rdb,
		Metrics:   metricsReg,
		zammadURL: cfg.ZammadURL,
		startedAt: time.Now(),
	}

	// Global middleware
	s.Router.Use(RequestID)
	s.Router.Use(Logger(logger))
	s.Router.Use(Metrics)
	s.Router.Use(middleware.Recoverer)
	s.Router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key", "X-Request-ID", "X-Tenant-Slug"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health endpoints (unauthenticated)
	s.Router.Get("/healthz", s.handleHealthz)
	s.Router.Get("/readyz", s.handleReadyz)

	// Prometheus metrics (unauthenticated)
	s.Router.Handle("/metrics", promhttp.HandlerFor(metricsReg, promhttp.HandlerOpts{}))

	// Authenticated, tenant-scoped API routes.
	s.Router.Route("/api/v1", func(r chi.Router) {
		// 1. Authenticate: OIDC JWT → API key → dev header fallback.
		r.Use(auth.Middleware(sessionMgr, oidcAuth, patAuth, authStore, logger))

		// 2. Resolve tenant and set search_path from the authenticated identity.
		r.Use(tenant.Middleware(db, &authContextResolver{}, logger))

		// 3. Require valid authentication on all /api/v1 routes.
		r.Use(auth.RequireAuth)

		// Debug endpoint.
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			t := tenant.FromContext(r.Context())
			id := auth.FromContext(r.Context())
			Respond(w, http.StatusOK, map[string]string{
				"tenant":  t.Slug,
				"schema":  t.Schema,
				"subject": id.Subject,
				"role":    id.Role,
				"method":  id.Method,
			})
		})

		// Store reference so domain handlers can be mounted externally.
		s.APIRouter = r
	})

	return s
}

// authContextResolver reads the tenant slug from the auth Identity stored in
// the request context by the auth middleware.
type authContextResolver struct{}

func (authContextResolver) Resolve(r *http.Request) (string, error) {
	id := auth.FromContext(r.Context())
	if id != nil && id.TenantSlug != "" {
		return id.TenantSlug, nil
	}
	return "", fmt.Errorf("no authenticated tenant")
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Router.ServeHTTP(w, r)
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	Respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	type checkResult struct {
		Name   string `json:"name"`
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
	}

	var checks []checkResult
	allOK := true

	// Database check.
	if err := s.DB.Ping(ctx); err != nil {
		s.Logger.Error("readiness check: database ping failed", "error", err)
		checks = append(checks, checkResult{Name: "database", Status: "fail", Error: err.Error()})
		allOK = false
	} else {
		checks = append(checks, checkResult{Name: "database", Status: "ok"})
	}

	// Redis check.
	if err := s.Redis.Ping(ctx).Err(); err != nil {
		s.Logger.Error("readiness check: redis ping failed", "error", err)
		checks = append(checks, checkResult{Name: "redis", Status: "fail", Error: err.Error()})
		allOK = false
	} else {
		checks = append(checks, checkResult{Name: "redis", Status: "ok"})
	}

	// Zammad check.
	if s.zammadURL != "" {
		if err := s.checkZammad(ctx); err != nil {
			s.Logger.Error("readiness check: zammad unreachable", "error", err, "url", s.zammadURL)
			checks = append(checks, checkResult{Name: "zammad", Status: "fail", Error: err.Error()})
			allOK = false
		} else {
			checks = append(checks, checkResult{Name: "zammad", Status: "ok"})
		}
	}

	status := "ready"
	httpStatus := http.StatusOK
	if !allOK {
		status = "unavailable"
		httpStatus = http.StatusServiceUnavailable
	}

	Respond(w, httpStatus, map[string]any{
		"status": status,
		"checks": checks,
	})
}

// checkZammad verifies that the Zammad instance is reachable.
func (s *Server) checkZammad(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.zammadURL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("connecting to zammad: %w", err)
	}
	_ = resp.Body.Close()

	// Any HTTP response means Zammad is reachable (even 401/403).
	return nil
}

// HandleStatus returns system health information.
func (s *Server) HandleStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uptime := time.Since(s.startedAt)

	resp := map[string]any{
		"status":         "ok",
		"version":        version.Version,
		"commit_sha":     version.Commit,
		"uptime":         uptime.Truncate(time.Second).String(),
		"uptime_seconds": int64(uptime.Seconds()),
	}

	if err := s.DB.Ping(ctx); err != nil {
		resp["database"] = "error"
		resp["status"] = "degraded"
	} else {
		resp["database"] = "ok"
	}

	if err := s.Redis.Ping(ctx).Err(); err != nil {
		resp["redis"] = "error"
		resp["status"] = "degraded"
	} else {
		resp["redis"] = "ok"
	}

	Respond(w, http.StatusOK, resp)
}
