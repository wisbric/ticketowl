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
	Subject           string   `json:"sub"`
	Email             string   `json:"email"`
	Name              string   `json:"name"`
	PreferredUsername string   `json:"preferred_username"`
	TenantSlug        string   `json:"tenant_slug"`
	Role              string   `json:"role"`
	OrgID             string   `json:"org_id"`
	RealmRoles        []string `json:"realm_roles"`
	Groups            []string `json:"groups"`
}

// DisplayName returns the best available display name from the OIDC claims,
// preferring the full name, then preferred_username, then email, then subject.
func (c *OIDCClaims) DisplayName() string {
	if c.Name != "" {
		return c.Name
	}
	if c.PreferredUsername != "" {
		return c.PreferredUsername
	}
	if c.Email != "" {
		return c.Email
	}
	return c.Subject
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
// It requires the tenant_slug claim for service-to-service API authentication.
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
	claims.resolveRole()

	return &claims, nil
}

// AuthenticateCallbackToken validates an OIDC ID token from the Authorization
// Code flow. Unlike Authenticate, it does not require the tenant_slug claim
// because the tenant is resolved from the OAuth state parameter.
func (a *OIDCAuthenticator) AuthenticateCallbackToken(ctx context.Context, rawToken string) (*OIDCClaims, error) {
	idToken, err := a.Verifier.Verify(ctx, rawToken)
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
	claims.resolveRole()

	return &claims, nil
}

// resolveRole determines the user's role from the available JWT claims.
// It checks (in order): explicit "role" claim, realm_roles array, groups array.
func (c *OIDCClaims) resolveRole() {
	if c.Role != "" && IsValidRole(c.Role) {
		return
	}

	// Check realm_roles for known roles (highest-privilege first).
	for _, role := range ValidRoles {
		for _, r := range c.RealmRoles {
			if r == role {
				c.Role = role
				return
			}
		}
	}

	// Check groups for role-like group names (e.g. "/admins" or "admins").
	groupRoleMap := map[string]string{
		"admins":   RoleAdmin,
		"managers": RoleManager,
	}
	for _, g := range c.Groups {
		name := strings.TrimPrefix(g, "/")
		if role, ok := groupRoleMap[name]; ok {
			c.Role = role
			return
		}
	}

	c.Role = RoleEngineer
}
