package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
)

// OIDCFlowHandler handles the OAuth2 Authorization Code flow.
type OIDCFlowHandler struct {
	oauth2Cfg  *oauth2.Config
	oidcAuth   *OIDCAuthenticator
	sessionMgr *SessionManager
	store      Storage
	redis      *redis.Client
	logger     *slog.Logger

	// SuccessURL is where the browser is redirected after successful authentication.
	// Defaults to "/" if empty.
	SuccessURL string
}

// NewOIDCFlowHandler creates a handler for the full OIDC Authorization Code flow.
func NewOIDCFlowHandler(
	oauth2Cfg *oauth2.Config,
	oidcAuth *OIDCAuthenticator,
	sm *SessionManager,
	store Storage,
	rdb *redis.Client,
	logger *slog.Logger,
) *OIDCFlowHandler {
	return &OIDCFlowHandler{
		oauth2Cfg:  oauth2Cfg,
		oidcAuth:   oidcAuth,
		sessionMgr: sm,
		store:      store,
		redis:      rdb,
		logger:     logger,
	}
}

// HandleLogin redirects the user to the OIDC identity provider.
// It accepts an optional ?tenant= query parameter. If not provided, it
// defaults to the first tenant in the database (single-tenant convenience).
func (h *OIDCFlowHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := randomState()
	if err != nil {
		respondErr(w, http.StatusInternalServerError, "internal", "failed to generate state")
		return
	}

	// Resolve tenant from query param or default to first tenant.
	tenantSlug := r.URL.Query().Get("tenant")
	if tenantSlug == "" {
		tenants, listErr := h.store.ListTenants(r.Context())
		if listErr != nil || len(tenants) == 0 {
			h.logger.Error("oidc: no tenants available for OIDC login", "error", listErr)
			respondErr(w, http.StatusInternalServerError, "internal", "no tenants configured")
			return
		}
		tenantSlug = tenants[0].Slug
	}

	// Store state → tenant mapping in Redis with 10 minute TTL.
	if err := h.redis.Set(r.Context(), "oidc_state:"+state, tenantSlug, 10*time.Minute).Err(); err != nil {
		h.logger.Error("oidc: storing state in redis", "error", err)
		respondErr(w, http.StatusInternalServerError, "internal", "failed to store state")
		return
	}

	url := h.oauth2Cfg.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusFound)
}

// HandleCallback handles the IdP callback after authentication.
func (h *OIDCFlowHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Verify state and retrieve the tenant slug stored during login.
	state := r.URL.Query().Get("state")
	if state == "" {
		respondErr(w, http.StatusBadRequest, "bad_request", "missing state parameter")
		return
	}

	tenantSlug, err := h.redis.GetDel(ctx, "oidc_state:"+state).Result()
	if err != nil || tenantSlug == "" {
		respondErr(w, http.StatusBadRequest, "bad_request", "invalid or expired state")
		return
	}

	// Check for error from IdP.
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		desc := r.URL.Query().Get("error_description")
		h.logger.Warn("oidc: IdP returned error", "error", errParam, "description", desc)
		respondErr(w, http.StatusUnauthorized, "unauthorized", "authentication failed: "+errParam)
		return
	}

	// Exchange code for tokens.
	code := r.URL.Query().Get("code")
	if code == "" {
		respondErr(w, http.StatusBadRequest, "bad_request", "missing code parameter")
		return
	}

	oauth2Token, err := h.oauth2Cfg.Exchange(ctx, code)
	if err != nil {
		h.logger.Error("oidc: code exchange failed", "error", err)
		respondErr(w, http.StatusUnauthorized, "unauthorized", "code exchange failed")
		return
	}

	// Extract and verify the ID token (without requiring tenant_slug claim).
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		respondErr(w, http.StatusUnauthorized, "unauthorized", "no id_token in response")
		return
	}

	claims, err := h.oidcAuth.AuthenticateCallbackToken(ctx, rawIDToken)
	if err != nil {
		h.logger.Error("oidc: token verification failed", "error", err)
		respondErr(w, http.StatusUnauthorized, "unauthorized", "invalid id_token")
		return
	}

	// Set tenant from the OAuth state (not from the token).
	claims.TenantSlug = tenantSlug

	// Look up or create the user in the tenant schema.
	userRow, tenantID, err := h.findOrCreateUser(ctx, claims)
	if err != nil {
		h.logger.Error("oidc: user lookup/create failed", "error", err, "tenant", tenantSlug)
		respondErr(w, http.StatusInternalServerError, "internal", "failed to resolve user")
		return
	}

	// Issue session JWT — use the DB display name (resolved from OIDC claims
	// by the adapter), not the raw subject UUID.
	sessClaims := SessionClaims{
		Subject:    userRow.DisplayName,
		Email:      claims.Email,
		Role:       claims.Role,
		TenantSlug: claims.TenantSlug,
		TenantID:   tenantID,
		UserID:     userRow.ID.String(),
		Method:     "oidc",
	}

	// Set session cookie (browser clients).
	if err := h.sessionMgr.IssueCookie(w, sessClaims); err != nil {
		h.logger.Error("oidc: issuing session token", "error", err)
		respondErr(w, http.StatusInternalServerError, "internal", "failed to issue token")
		return
	}

	// Redirect to frontend.
	successURL := h.SuccessURL
	if successURL == "" {
		successURL = "/"
	}
	http.Redirect(w, r, successURL, http.StatusFound)
}

// findOrCreateUser resolves an OIDC user to a database user row.
func (h *OIDCFlowHandler) findOrCreateUser(ctx context.Context, claims *OIDCClaims) (*UserRow, string, error) {
	return h.store.FindOrCreateOIDCUser(ctx, claims.TenantSlug, claims.Subject, claims.Email, claims.DisplayName(), claims.Role)
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}
