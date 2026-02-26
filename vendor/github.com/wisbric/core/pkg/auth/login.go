package auth

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

// LoginRequest is the JSON body for POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse is the JSON response for a successful login.
type LoginResponse struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}

// UserInfo is the public user information returned in auth responses.
type UserInfo struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

// AuthConfigResponse tells the frontend which auth methods are available.
type AuthConfigResponse struct {
	OIDCEnabled  bool   `json:"oidc_enabled"`
	OIDCName     string `json:"oidc_name"`
	LocalEnabled bool   `json:"local_enabled"`
}

// LoginHandler handles local email/password login and auth discovery.
type LoginHandler struct {
	sessionMgr  *SessionManager
	store       Storage
	logger      *slog.Logger
	oidcEnabled bool
}

// NewLoginHandler creates a new login handler.
func NewLoginHandler(sm *SessionManager, store Storage, logger *slog.Logger, oidcEnabled bool) *LoginHandler {
	return &LoginHandler{
		sessionMgr:  sm,
		store:       store,
		logger:      logger,
		oidcEnabled: oidcEnabled,
	}
}

// HandleLogin authenticates a user with email/password and returns a session JWT.
func (h *LoginHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	if req.Email == "" || req.Password == "" {
		respondErr(w, http.StatusBadRequest, "bad_request", "email and password are required")
		return
	}

	// Look up the user across all tenant schemas.
	userRow, tenantSlug, tenantID, err := h.findUserByEmail(r.Context(), req.Email)
	if err != nil {
		h.logger.Warn("login: user lookup failed", "email", req.Email, "error", err)
		respondErr(w, http.StatusUnauthorized, "unauthorized", "invalid email or password")
		return
	}

	// Verify password.
	if userRow.PasswordHash == nil || *userRow.PasswordHash == "" {
		h.logger.Warn("login: user has no password set", "email", req.Email)
		respondErr(w, http.StatusUnauthorized, "unauthorized", "invalid email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*userRow.PasswordHash), []byte(req.Password)); err != nil {
		respondErr(w, http.StatusUnauthorized, "unauthorized", "invalid email or password")
		return
	}

	// Issue session token.
	claims := SessionClaims{
		Subject:    userRow.DisplayName,
		Email:      userRow.Email,
		Role:       userRow.Role,
		TenantSlug: tenantSlug,
		TenantID:   tenantID,
		UserID:     userRow.ID.String(),
		Method:     "local",
	}

	token, err := h.sessionMgr.IssueToken(claims)
	if err != nil {
		h.logger.Error("login: issuing token", "error", err)
		respondErr(w, http.StatusInternalServerError, "internal", "failed to issue token")
		return
	}

	// Set session cookie (browser clients).
	_ = h.sessionMgr.IssueCookie(w, claims)

	respondJSON(w, http.StatusOK, LoginResponse{
		Token: token,
		User: UserInfo{
			ID:          userRow.ID.String(),
			Email:       userRow.Email,
			DisplayName: userRow.DisplayName,
			Role:        userRow.Role,
		},
	})
}

// HandleAuthConfig returns the available authentication methods.
func (h *LoginHandler) HandleAuthConfig(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, AuthConfigResponse{
		OIDCEnabled:  h.oidcEnabled,
		OIDCName:     "Sign in with SSO",
		LocalEnabled: true,
	})
}

// HandleMe returns the current user's info from a session cookie or Bearer token.
func (h *LoginHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	// Try session cookie first, then Bearer token.
	var claims *SessionClaims
	if c, err := h.sessionMgr.ValidateCookie(r); err == nil {
		claims = c
	} else {
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) < 8 {
			respondErr(w, http.StatusUnauthorized, "unauthorized", "no token provided")
			return
		}

		token := authHeader[7:] // strip "Bearer "
		c, err := h.sessionMgr.ValidateToken(token)
		if err != nil {
			respondErr(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
			return
		}
		claims = c
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"id":           claims.UserID,
		"email":        claims.Email,
		"display_name": claims.Subject,
		"role":         claims.Role,
		"tenant_slug":  claims.TenantSlug,
	})
}

// HandleLogout clears the session cookie and returns success.
func (h *LoginHandler) HandleLogout(w http.ResponseWriter, _ *http.Request) {
	h.sessionMgr.ClearCookie(w)
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// respondJSON writes a JSON response with the given status code.
func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// findUserByEmail searches across all tenant schemas for a user with the given email.
func (h *LoginHandler) findUserByEmail(ctx context.Context, email string) (*UserRow, string, string, error) {
	return h.store.FindUserByEmail(ctx, email)
}
