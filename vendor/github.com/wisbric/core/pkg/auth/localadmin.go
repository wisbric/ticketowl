package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// LocalAdminLoginRequest is the JSON body for POST /auth/local.
type LocalAdminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Tenant   string `json:"tenant"`
}

// LocalAdminLoginResponse is the JSON response for a successful local admin login.
type LocalAdminLoginResponse struct {
	Token      string   `json:"token"`
	MustChange bool     `json:"must_change"`
	User       UserInfo `json:"user"`
}

// ChangePasswordRequest is the JSON body for POST /auth/change-password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// localAdminRow represents a row from public.local_admins.
type localAdminRow struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	Username     string
	PasswordHash string
	MustChange   bool
	LastLoginAt  *time.Time
}

// LocalAdminHandler handles local admin authentication endpoints.
type LocalAdminHandler struct {
	sessionMgr  *SessionManager
	store       Storage
	logger      *slog.Logger
	rateLimiter LoginRateLimiter
}

// NewLocalAdminHandler creates a new local admin handler.
func NewLocalAdminHandler(sm *SessionManager, store Storage, logger *slog.Logger, rl LoginRateLimiter) *LocalAdminHandler {
	return &LocalAdminHandler{
		sessionMgr:  sm,
		store:       store,
		logger:      logger,
		rateLimiter: rl,
	}
}

// HandleLocalLogin authenticates a local admin with username/password.
func (h *LocalAdminHandler) HandleLocalLogin(w http.ResponseWriter, r *http.Request) {
	var req LocalAdminLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	if req.Username == "" || req.Password == "" {
		respondErr(w, http.StatusBadRequest, "bad_request", "username and password are required")
		return
	}

	// Rate limit check.
	ip := clientIP(r)
	if h.rateLimiter != nil {
		result, err := h.rateLimiter.Check(r.Context(), ip)
		if err != nil {
			h.logger.Error("rate limit check failed", "error", err)
		} else if !result.Allowed {
			retryAfter := int(time.Until(result.RetryAt).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
			respondJSON(w, http.StatusTooManyRequests, map[string]any{
				"error":       "rate_limited",
				"message":     "too many login attempts",
				"retry_after": retryAfter,
			})
			return
		}
	}

	// Look up local admin. If tenant is specified, look in that tenant only.
	admin, tenantSlug, err := h.findLocalAdmin(r.Context(), req.Username, req.Tenant)
	if err != nil {
		h.logger.Warn("local admin login: lookup failed", "username", req.Username, "error", err)
		if h.rateLimiter != nil {
			_ = h.rateLimiter.Record(r.Context(), ip)
		}
		respondErr(w, http.StatusUnauthorized, "unauthorized", "invalid username or password")
		return
	}

	// Verify password.
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		if h.rateLimiter != nil {
			_ = h.rateLimiter.Record(r.Context(), ip)
		}
		respondErr(w, http.StatusUnauthorized, "unauthorized", "invalid username or password")
		return
	}

	// Reset rate limit on success.
	if h.rateLimiter != nil {
		_ = h.rateLimiter.Reset(r.Context(), ip)
	}

	// Update last_login_at.
	go func() {
		_ = h.store.UpdateLocalAdminLastLogin(context.Background(), admin.ID)
	}()

	// Issue session token.
	claims := SessionClaims{
		Subject:    admin.Username,
		Email:      admin.Username + "@local",
		Role:       RoleAdmin,
		TenantSlug: tenantSlug,
		TenantID:   admin.TenantID.String(),
		UserID:     admin.ID.String(),
		Method:     MethodLocal,
	}

	token, err := h.sessionMgr.IssueToken(claims)
	if err != nil {
		h.logger.Error("local admin login: issuing token", "error", err)
		respondErr(w, http.StatusInternalServerError, "internal", "failed to issue token")
		return
	}

	// Set session cookie (browser clients).
	_ = h.sessionMgr.IssueCookie(w, claims)

	respondJSON(w, http.StatusOK, LocalAdminLoginResponse{
		Token:      token,
		MustChange: admin.MustChange,
		User: UserInfo{
			ID:          admin.ID.String(),
			Email:       admin.Username + "@local",
			DisplayName: "Local Admin",
			Role:        RoleAdmin,
		},
	})
}

// HandleChangePassword handles the forced password change flow.
func (h *LocalAdminHandler) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		respondErr(w, http.StatusBadRequest, "bad_request", "current_password and new_password are required")
		return
	}

	// Validate new password requirements: >= 12 chars, upper+lower, number or symbol.
	if err := validatePassword(req.NewPassword); err != nil {
		respondErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	// Extract current user from session cookie or Bearer token.
	var claims *SessionClaims
	if c, err := h.sessionMgr.ValidateCookie(r); err == nil {
		claims = c
	} else {
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) < 8 {
			respondErr(w, http.StatusUnauthorized, "unauthorized", "no token provided")
			return
		}
		token := authHeader[7:]
		c, err := h.sessionMgr.ValidateToken(token)
		if err != nil {
			respondErr(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
			return
		}
		claims = c
	}

	if claims.Method != MethodLocal {
		respondErr(w, http.StatusBadRequest, "bad_request", "password change is only available for local admin accounts")
		return
	}

	adminID, err := uuid.Parse(claims.UserID)
	if err != nil {
		respondErr(w, http.StatusBadRequest, "bad_request", "invalid user ID")
		return
	}

	// Fetch the admin to verify current password.
	currentHash, err := h.store.GetLocalAdminPasswordHash(r.Context(), adminID)
	if err != nil {
		h.logger.Error("change password: admin lookup", "error", err)
		respondErr(w, http.StatusInternalServerError, "internal", "failed to look up admin")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.CurrentPassword)); err != nil {
		respondErr(w, http.StatusUnauthorized, "unauthorized", "current password is incorrect")
		return
	}

	// Hash new password.
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 12)
	if err != nil {
		h.logger.Error("change password: hashing", "error", err)
		respondErr(w, http.StatusInternalServerError, "internal", "failed to hash password")
		return
	}

	// Update password and clear must_change.
	err = h.store.UpdateLocalAdminPassword(r.Context(), adminID, string(newHash), false)
	if err != nil {
		h.logger.Error("change password: update", "error", err)
		respondErr(w, http.StatusInternalServerError, "internal", "failed to update password")
		return
	}

	// Issue a new session token with the updated state.
	newClaims := SessionClaims{
		Subject:    claims.Subject,
		Email:      claims.Email,
		Role:       claims.Role,
		TenantSlug: claims.TenantSlug,
		TenantID:   claims.TenantID,
		UserID:     claims.UserID,
		Method:     MethodLocal,
	}

	newToken, err := h.sessionMgr.IssueToken(newClaims)
	if err != nil {
		h.logger.Error("change password: issuing new token", "error", err)
		respondErr(w, http.StatusInternalServerError, "internal", "failed to issue new token")
		return
	}

	// Set session cookie (browser clients).
	_ = h.sessionMgr.IssueCookie(w, newClaims)

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"token":  newToken,
	})
}

// HandleAuthConfig returns the available auth methods for a tenant.
// This is an updated version that checks OIDC config from the database.
func (h *LocalAdminHandler) HandleAuthConfig(w http.ResponseWriter, r *http.Request) {
	tenant := r.URL.Query().Get("tenant")

	oidcEnabled := false
	oidcName := "Sign in with SSO"

	if tenant != "" {
		oidcEnabled = h.CheckOIDCEnabled(r.Context(), tenant)
	}

	respondJSON(w, http.StatusOK, AuthConfigResponse{
		OIDCEnabled:  oidcEnabled,
		OIDCName:     oidcName,
		LocalEnabled: true,
	})
}

// CheckOIDCEnabled checks if OIDC is configured and enabled for a tenant.
func (h *LocalAdminHandler) CheckOIDCEnabled(ctx context.Context, tenantSlug string) bool {
	config, err := h.store.GetOIDCConfig(ctx, tenantSlug)
	if err != nil {
		return false
	}
	return config.Enabled
}

// findLocalAdmin looks up a local admin by username across tenants or in a specific tenant.
func (h *LocalAdminHandler) findLocalAdmin(ctx context.Context, username, tenantSlug string) (*LocalAdminRow, string, error) {
	return h.store.FindLocalAdmin(ctx, username, tenantSlug)
}

// validatePassword checks password requirements: >= 12 chars, upper+lower, number or symbol.
func validatePassword(pw string) error {
	if len(pw) < 12 {
		return fmt.Errorf("password must be at least 12 characters")
	}

	var hasUpper, hasLower, hasDigitOrSymbol bool
	for _, r := range pw {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigitOrSymbol = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasDigitOrSymbol = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasDigitOrSymbol {
		return fmt.Errorf("password must contain at least one number or symbol")
	}

	return nil
}
