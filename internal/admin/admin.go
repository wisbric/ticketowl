package admin

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ZammadConfig represents the tenant's Zammad connection configuration.
type ZammadConfig struct {
	ID            uuid.UUID  `json:"id"`
	URL           string     `json:"url"`
	APIToken      string     `json:"api_token"`
	WebhookSecret string     `json:"webhook_secret"`
	PauseStatuses []string   `json:"pause_statuses"`
	UpdatedAt     time.Time  `json:"updated_at"`
	UpdatedBy     *uuid.UUID `json:"updated_by,omitempty"`
}

// UpdateZammadConfigRequest is the payload for updating Zammad configuration.
type UpdateZammadConfigRequest struct {
	URL           string   `json:"url"`
	APIToken      string   `json:"api_token"`
	WebhookSecret string   `json:"webhook_secret"`
	PauseStatuses []string `json:"pause_statuses"`
}

// TestZammadRequest is the payload for testing a Zammad connection.
type TestZammadRequest struct {
	URL      string `json:"url"`
	APIToken string `json:"api_token"`
}

// IntegrationKey represents a NightOwl or BookOwl integration key.
type IntegrationKey struct {
	ID        uuid.UUID `json:"id"`
	Service   string    `json:"service"`
	APIKey    string    `json:"api_key"`
	APIURL    string    `json:"api_url"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UpdateIntegrationKeyRequest is the payload for updating an integration key.
type UpdateIntegrationKeyRequest struct {
	APIKey string `json:"api_key"`
	APIURL string `json:"api_url"`
}

// CustomerOrg represents a customer organisation in the admin context.
type CustomerOrg struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	OIDCGroup   string    `json:"oidc_group"`
	ZammadOrgID *int      `json:"zammad_org_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateCustomerOrgRequest is the payload for creating a customer org.
type CreateCustomerOrgRequest struct {
	Name        string `json:"name"`
	OIDCGroup   string `json:"oidc_group"`
	ZammadOrgID *int   `json:"zammad_org_id,omitempty"`
}

// UpdateCustomerOrgRequest is the payload for updating a customer org.
type UpdateCustomerOrgRequest struct {
	Name        *string `json:"name,omitempty"`
	OIDCGroup   *string `json:"oidc_group,omitempty"`
	ZammadOrgID *int    `json:"zammad_org_id,omitempty"`
}

// AutoTicketRule represents an auto-ticket creation rule.
type AutoTicketRule struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	Enabled         bool      `json:"enabled"`
	AlertGroup      *string   `json:"alert_group,omitempty"`
	MinSeverity     *string   `json:"min_severity,omitempty"`
	DefaultPriority string    `json:"default_priority"`
	DefaultGroup    *string   `json:"default_group,omitempty"`
	TitleTemplate   string    `json:"title_template"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateAutoTicketRuleRequest is the payload for creating an auto-ticket rule.
type CreateAutoTicketRuleRequest struct {
	Name            string  `json:"name"`
	Enabled         *bool   `json:"enabled,omitempty"`
	AlertGroup      *string `json:"alert_group,omitempty"`
	MinSeverity     *string `json:"min_severity,omitempty"`
	DefaultPriority string  `json:"default_priority"`
	DefaultGroup    *string `json:"default_group,omitempty"`
	TitleTemplate   string  `json:"title_template"`
}

// UpdateAutoTicketRuleRequest is the payload for updating an auto-ticket rule.
type UpdateAutoTicketRuleRequest struct {
	Name            *string `json:"name,omitempty"`
	Enabled         *bool   `json:"enabled,omitempty"`
	AlertGroup      *string `json:"alert_group,omitempty"`
	MinSeverity     *string `json:"min_severity,omitempty"`
	DefaultPriority *string `json:"default_priority,omitempty"`
	DefaultGroup    *string `json:"default_group,omitempty"`
	TitleTemplate   *string `json:"title_template,omitempty"`
}

// ConfigOverview is the combined configuration returned by GET /admin/config.
type ConfigOverview struct {
	Zammad   *ZammadConfigSummary   `json:"zammad,omitempty"`
	NightOwl *IntegrationKeySummary `json:"nightowl,omitempty"`
	BookOwl  *IntegrationKeySummary `json:"bookowl,omitempty"`
}

// ZammadConfigSummary is the public view of Zammad config (no secrets).
type ZammadConfigSummary struct {
	URL           string    `json:"url"`
	PauseStatuses []string  `json:"pause_statuses"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// IntegrationKeySummary is the public view of an integration key (no secret).
type IntegrationKeySummary struct {
	APIURL    string    `json:"api_url"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AdminStore defines the database operations the admin handler needs.
type AdminStore interface {
	GetZammadConfig(ctx context.Context) (*ZammadConfig, error)
	UpsertZammadConfig(ctx context.Context, req UpdateZammadConfigRequest, updatedBy uuid.UUID) (*ZammadConfig, error)
	GetIntegrationKey(ctx context.Context, service string) (*IntegrationKey, error)
	UpsertIntegrationKey(ctx context.Context, service string, req UpdateIntegrationKeyRequest) (*IntegrationKey, error)
	ListCustomerOrgs(ctx context.Context) ([]CustomerOrg, error)
	CreateCustomerOrg(ctx context.Context, req CreateCustomerOrgRequest) (*CustomerOrg, error)
	UpdateCustomerOrg(ctx context.Context, id uuid.UUID, req UpdateCustomerOrgRequest) (*CustomerOrg, error)
	DeleteCustomerOrg(ctx context.Context, id uuid.UUID) error
	ListAutoTicketRules(ctx context.Context) ([]AutoTicketRule, error)
	CreateAutoTicketRule(ctx context.Context, req CreateAutoTicketRuleRequest) (*AutoTicketRule, error)
	UpdateAutoTicketRule(ctx context.Context, id uuid.UUID, req UpdateAutoTicketRuleRequest) (*AutoTicketRule, error)
	DeleteAutoTicketRule(ctx context.Context, id uuid.UUID) error
}
