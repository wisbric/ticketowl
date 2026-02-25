package auth

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Middleware returns an HTTP middleware that authenticates the caller via
// OIDC JWT, API key, or dev header and stores the resulting Identity in
// the request context.
//
// Authentication precedence:
//  1. Authorization: Bearer <jwt> → OIDC validation
//  2. X-API-Key: <raw-key>       → API key hash lookup
//  3. X-Tenant-Slug: <slug>      → Development-only fallback (no real auth)
func Middleware(oidcAuth *OIDCAuthenticator, db *pgxpool.Pool, logger *slog.Logger) func(http.Handler) http.Handler {
	apikeyAuth := &APIKeyAuthenticator{DB: db}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var identity *Identity

			// 1. Try Bearer token (OIDC JWT).
			if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") || strings.HasPrefix(authHeader, "bearer ") {
				if oidcAuth == nil {
					logger.Warn("JWT presented but OIDC is not configured")
					respondErr(w, http.StatusUnauthorized, "unauthorized", "OIDC not configured")
					return
				}

				claims, err := oidcAuth.Authenticate(r.Context(), authHeader)
				if err != nil {
					logger.Warn("OIDC authentication failed", "error", err)
					respondErr(w, http.StatusUnauthorized, "unauthorized", "invalid token")
					return
				}

				identity = &Identity{
					Subject:    claims.Subject,
					Email:      claims.Email,
					Role:       claims.Role,
					TenantSlug: claims.TenantSlug,
					Method:     MethodOIDC,
				}

				if claims.OrgID != "" {
					orgID, err := uuid.Parse(claims.OrgID)
					if err == nil {
						identity.OrgID = &orgID
					}
				}

				logger.Debug("authenticated via OIDC",
					"sub", claims.Subject,
					"email", claims.Email,
					"tenant", claims.TenantSlug,
				)
			}

			// 2. Try API key.
			if identity == nil {
				if rawKey := r.Header.Get("X-API-Key"); rawKey != "" {
					result, err := apikeyAuth.Authenticate(r.Context(), rawKey)
					if err != nil {
						logger.Warn("API key authentication failed", "error", err)
						respondErr(w, http.StatusUnauthorized, "unauthorized", "invalid API key")
						return
					}

					// Look up tenant slug from tenant ID.
					var slug string
					err = db.QueryRow(r.Context(),
						"SELECT slug FROM global.tenants WHERE id = $1",
						result.TenantID,
					).Scan(&slug)
					if err != nil {
						logger.Error("tenant lookup for API key failed", "tenant_id", result.TenantID, "error", err)
						respondErr(w, http.StatusUnauthorized, "unauthorized", "tenant not found")
						return
					}

					identity = &Identity{
						Subject:    fmt.Sprintf("apikey:%s", result.APIKeyID),
						Role:       RoleAdmin,
						TenantSlug: slug,
						TenantID:   result.TenantID,
						APIKeyID:   &result.APIKeyID,
						Method:     MethodAPIKey,
					}

					logger.Debug("authenticated via API key",
						"key_id", result.APIKeyID,
						"tenant_slug", slug,
					)
				}
			}

			// 3. Dev-mode fallback: X-Tenant-Slug header.
			if identity == nil {
				if slug := r.Header.Get("X-Tenant-Slug"); slug != "" {
					devID := uuid.Nil
					identity = &Identity{
						Subject:    "dev:anonymous",
						Email:      "dev@localhost",
						Role:       RoleAdmin,
						TenantSlug: slug,
						TenantID:   devID,
						UserID:     &devID,
						Method:     MethodDev,
					}

					logger.Debug("dev-mode authentication", "tenant_slug", slug)
				}
			}

			if identity == nil {
				respondErr(w, http.StatusUnauthorized, "unauthorized", "no valid authentication provided")
				return
			}

			ctx := NewContext(r.Context(), identity)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func respondErr(w http.ResponseWriter, status int, errStr, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   errStr,
		"message": message,
	})
}
