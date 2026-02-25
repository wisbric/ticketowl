package tenant

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Resolver identifies the tenant for the current request.
type Resolver interface {
	Resolve(r *http.Request) (slug string, err error)
}

// Middleware returns an HTTP middleware that resolves the tenant, acquires a
// database connection, sets the PostgreSQL search_path, and stores both the
// tenant info and the scoped connection in the request context.
//
// The connection is released after the downstream handler returns.
func Middleware(pool *pgxpool.Pool, resolver Resolver, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slug, err := resolver.Resolve(r)
			if err != nil {
				respondError(w, http.StatusUnauthorized, "unauthorized", "tenant resolution failed")
				return
			}

			// Look up tenant in the global registry.
			var tenantID [16]byte
			var tenantName string
			err = pool.QueryRow(r.Context(),
				"SELECT id, name FROM global.tenants WHERE slug = $1 AND NOT suspended",
				slug,
			).Scan(&tenantID, &tenantName)
			if err != nil {
				logger.Warn("tenant not found", "slug", slug, "error", err)
				respondError(w, http.StatusUnauthorized, "unauthorized", "unknown tenant")
				return
			}

			schema := SchemaName(slug)

			// Acquire a dedicated connection and set search_path.
			conn, err := pool.Acquire(r.Context())
			if err != nil {
				logger.Error("acquiring database connection", "error", err)
				respondError(w, http.StatusServiceUnavailable, "unavailable", "database connection unavailable")
				return
			}
			defer conn.Release()

			if _, err := conn.Exec(r.Context(), fmt.Sprintf("SET search_path TO %s, public", schema)); err != nil {
				logger.Error("setting search_path", "schema", schema, "error", err)
				respondError(w, http.StatusInternalServerError, "internal", "database configuration error")
				return
			}

			info := &Info{
				ID:     tenantID,
				Name:   tenantName,
				Slug:   slug,
				Schema: schema,
			}

			ctx := NewContext(r.Context(), info)
			ctx = NewConnContext(ctx, conn)

			logger.Debug("tenant resolved",
				"tenant_id", tenantID,
				"slug", slug,
				"schema", schema,
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// respondError writes a JSON error response without importing httpserver.
func respondError(w http.ResponseWriter, status int, errStr, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   errStr,
		"message": message,
	})
}
