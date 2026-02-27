package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OIDCClaims are the JWT claims we extract for authentication.
type OIDCClaims struct {
	Subject    string `json:"sub"`
	Email      string `json:"email"`
	TenantSlug string `json:"tenant_slug"`
	Role       string `json:"role"`
	OrgID      string `json:"org_id"`
}

// OIDCAuthenticator validates OIDC JWTs and extracts claims.
type OIDCAuthenticator struct {
	Verifier *oidc.IDTokenVerifier
	provider *oidc.Provider
}

// NewOIDCAuthenticator creates an authenticator by performing OIDC discovery
// against the issuer URL. This makes a network call to fetch the provider's
// public keys.
func NewOIDCAuthenticator(ctx context.Context, issuerURL, clientID string) (*OIDCAuthenticator, error) {
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, fmt.Errorf("discovering OIDC provider %s: %w", issuerURL, err)
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})

	return &OIDCAuthenticator{Verifier: verifier, provider: provider}, nil
}

// Endpoint returns the OAuth2 endpoint discovered from the OIDC provider.
func (a *OIDCAuthenticator) Endpoint() oauth2.Endpoint {
	return a.provider.Endpoint()
}

// Authenticate validates a Bearer token and returns the extracted claims.
func (a *OIDCAuthenticator) Authenticate(ctx context.Context, bearerToken string) (*OIDCClaims, error) {
	token := strings.TrimPrefix(bearerToken, "Bearer ")
	token = strings.TrimPrefix(token, "bearer ")
	token = strings.TrimSpace(token)

	if token == "" {
		return nil, fmt.Errorf("empty bearer token")
	}

	idToken, err := a.Verifier.Verify(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("verifying token: %w", err)
	}

	var claims OIDCClaims
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("extracting claims: %w", err)
	}

	if claims.Subject == "" {
		return nil, fmt.Errorf("token missing sub claim")
	}
	if claims.TenantSlug == "" {
		return nil, fmt.Errorf("token missing tenant_slug claim")
	}
	if claims.Role == "" {
		claims.Role = RoleEngineer
	}
	if !IsValidRole(claims.Role) {
		claims.Role = RoleEngineer
	}

	return &claims, nil
}
