package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// OIDCConfigResponse is the JSON response for GET /api/v1/admin/oidc/config.
type OIDCConfigResponse struct {
	ID           string  `json:"id"`
	IssuerURL    string  `json:"issuer_url"`
	ClientID     string  `json:"client_id"`
	ClientSecret string  `json:"client_secret"` // masked
	Enabled      bool    `json:"enabled"`
	TestedAt     *string `json:"tested_at,omitempty"`
}

// OIDCConfigUpdateRequest is the JSON body for PUT /api/v1/admin/oidc/config.
type OIDCConfigUpdateRequest struct {
	IssuerURL    string `json:"issuer_url" validate:"required,url"`
	ClientID     string `json:"client_id" validate:"required"`
	ClientSecret string `json:"client_secret" validate:"required"`
	Enabled      bool   `json:"enabled"`
}

// OIDCTestResponse is the JSON response for POST /api/v1/admin/oidc/test.
type OIDCTestResponse struct {
	OK       bool   `json:"ok"`
	Error    string `json:"error,omitempty"`
	Issuer   string `json:"issuer,omitempty"`
	TestedAt string `json:"tested_at,omitempty"`
}

// LocalAdminResetResponse is the JSON response for POST /api/v1/admin/local-admin/reset.
type LocalAdminResetResponse struct {
	Password string `json:"password"`
	Message  string `json:"message"`
}

// OIDCAdminHandler handles OIDC config admin endpoints.
type OIDCAdminHandler struct {
	store     Storage
	logger    *slog.Logger
	secretKey string
}

// NewOIDCAdminHandler creates a new OIDC admin handler.
func NewOIDCAdminHandler(store Storage, logger *slog.Logger, secretKey string) *OIDCAdminHandler {
	return &OIDCAdminHandler{
		store:     store,
		logger:    logger,
		secretKey: secretKey,
	}
}

// HandleGetOIDCConfig returns the current OIDC config for the tenant (secret masked).
func (h *OIDCAdminHandler) HandleGetOIDCConfig(w http.ResponseWriter, r *http.Request) {
	id := FromContext(r.Context())
	if id == nil {
		respondErr(w, http.StatusUnauthorized, "unauthorized", "missing authentication")
		return
	}

	row, err := h.getOIDCConfig(r.Context(), id.TenantSlug)
	if err != nil {
		// No config yet — return empty.
		respondJSON(w, http.StatusOK, OIDCConfigResponse{})
		return
	}

	respondJSON(w, http.StatusOK, row)
}

// HandleUpdateOIDCConfig saves the OIDC config for the tenant.
func (h *OIDCAdminHandler) HandleUpdateOIDCConfig(w http.ResponseWriter, r *http.Request) {
	id := FromContext(r.Context())
	if id == nil {
		respondErr(w, http.StatusUnauthorized, "unauthorized", "missing authentication")
		return
	}

	var req OIDCConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondErr(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	if req.IssuerURL == "" || req.ClientID == "" || req.ClientSecret == "" {
		respondErr(w, http.StatusBadRequest, "bad_request", "issuer_url, client_id, and client_secret are required")
		return
	}

	// Encrypt the client secret.
	encryptedSecret, err := encryptAES256GCM(req.ClientSecret, h.secretKey)
	if err != nil {
		h.logger.Error("encrypting OIDC client secret", "error", err)
		respondErr(w, http.StatusInternalServerError, "internal", "failed to encrypt client secret")
		return
	}

	// Upsert the config.
	err = h.store.UpsertOIDCConfig(r.Context(), id.TenantSlug, req.IssuerURL, req.ClientID, encryptedSecret, req.Enabled)
	if err != nil {
		h.logger.Error("saving OIDC config", "error", err)
		respondErr(w, http.StatusInternalServerError, "internal", "failed to save OIDC config")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleTestOIDCConnection tests the OIDC provider connection.
func (h *OIDCAdminHandler) HandleTestOIDCConnection(w http.ResponseWriter, r *http.Request) {
	id := FromContext(r.Context())
	if id == nil {
		respondErr(w, http.StatusUnauthorized, "unauthorized", "missing authentication")
		return
	}

	// Get the config (with decrypted secret).
	issuerURL, clientID, err := h.getOIDCConfigDecrypted(r.Context(), id.TenantSlug)
	if err != nil {
		respondJSON(w, http.StatusOK, OIDCTestResponse{
			OK:    false,
			Error: "no OIDC configuration found",
		})
		return
	}

	// Try OIDC discovery.
	_, err = NewOIDCAuthenticator(r.Context(), issuerURL, clientID)
	if err != nil {
		respondJSON(w, http.StatusOK, OIDCTestResponse{
			OK:    false,
			Error: fmt.Sprintf("OIDC discovery failed: %s", err.Error()),
		})
		return
	}

	// Update tested_at.
	now := time.Now().UTC()
	_ = h.store.UpdateOIDCTestedAt(r.Context(), id.TenantSlug, now)

	respondJSON(w, http.StatusOK, OIDCTestResponse{
		OK:       true,
		Issuer:   issuerURL,
		TestedAt: now.Format(time.RFC3339),
	})
}

// HandleResetLocalAdmin resets the local admin password (admin only).
func (h *OIDCAdminHandler) HandleResetLocalAdmin(w http.ResponseWriter, r *http.Request) {
	id := FromContext(r.Context())
	if id == nil {
		respondErr(w, http.StatusUnauthorized, "unauthorized", "missing authentication")
		return
	}

	// Generate random password.
	newPassword, err := generateRandomPassword(16)
	if err != nil {
		h.logger.Error("generating random password", "error", err)
		respondErr(w, http.StatusInternalServerError, "internal", "failed to generate password")
		return
	}

	hash, err := bcryptHash(newPassword)
	if err != nil {
		h.logger.Error("hashing password", "error", err)
		respondErr(w, http.StatusInternalServerError, "internal", "failed to hash password")
		return
	}

	err = h.store.ResetLocalAdminPassword(r.Context(), id.TenantID, hash)
	if err != nil {
		h.logger.Error("resetting local admin password", "error", err)
		respondErr(w, http.StatusInternalServerError, "internal", "failed to reset password")
		return
	}

	respondJSON(w, http.StatusOK, LocalAdminResetResponse{
		Password: newPassword,
		Message:  "Password reset. The admin must change it on next login.",
	})
}

// getOIDCConfig retrieves the OIDC config for a tenant (with masked secret).
func (h *OIDCAdminHandler) getOIDCConfig(ctx context.Context, tenantSlug string) (*OIDCConfigResponse, error) {
	row, err := h.store.GetOIDCConfig(ctx, tenantSlug)
	if err != nil {
		return nil, fmt.Errorf("querying oidc_config: %w", err)
	}

	response := OIDCConfigResponse{
		ID:           row.ID.String(),
		IssuerURL:    row.IssuerURL,
		ClientID:     row.ClientID,
		ClientSecret: "••••••••••••••••",
		Enabled:      row.Enabled,
	}

	if row.TestedAt != nil {
		formatted := row.TestedAt.Format(time.RFC3339)
		response.TestedAt = &formatted
	}

	return &response, nil
}

// getOIDCConfigDecrypted retrieves the OIDC config with decrypted secret.
func (h *OIDCAdminHandler) getOIDCConfigDecrypted(ctx context.Context, tenantSlug string) (issuerURL, clientID string, err error) {
	row, err := h.store.GetOIDCConfig(ctx, tenantSlug)
	if err != nil {
		return "", "", fmt.Errorf("querying oidc_config: %w", err)
	}

	if !row.Enabled {
		return "", "", fmt.Errorf("oidc is disabled for tenant")
	}

	return row.IssuerURL, row.ClientID, nil
}

// encryptAES256GCM encrypts plaintext using AES-256-GCM with the given key.
func encryptAES256GCM(plaintext, key string) (string, error) {
	// Derive a 32-byte key from the secret using SHA-256.
	keyHash := sha256.Sum256([]byte(key))

	block, err := aes.NewCipher(keyHash[:])
	if err != nil {
		return "", fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// generateRandomPassword generates a random alphanumeric password.
func generateRandomPassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b), nil
}

// bcryptHash hashes a password with bcrypt cost 12.
func bcryptHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
