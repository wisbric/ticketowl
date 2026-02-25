package admin

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/wisbric/ticketowl/internal/db"
)

// Store provides database operations for admin configuration.
type Store struct {
	dbtx db.DBTX
}

// NewStore creates an admin Store backed by the given database connection.
func NewStore(dbtx db.DBTX) *Store {
	return &Store{dbtx: dbtx}
}

// --- Zammad Config ---

// GetZammadConfig returns the tenant's Zammad configuration.
func (s *Store) GetZammadConfig(ctx context.Context) (*ZammadConfig, error) {
	var c ZammadConfig
	err := s.dbtx.QueryRow(ctx,
		`SELECT id, url, api_token, webhook_secret, pause_statuses, updated_at, updated_by
		 FROM zammad_config LIMIT 1`).
		Scan(&c.ID, &c.URL, &c.APIToken, &c.WebhookSecret, &c.PauseStatuses, &c.UpdatedAt, &c.UpdatedBy)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// UpsertZammadConfig inserts or updates the Zammad configuration.
func (s *Store) UpsertZammadConfig(ctx context.Context, req UpdateZammadConfigRequest, updatedBy uuid.UUID) (*ZammadConfig, error) {
	var c ZammadConfig
	err := s.dbtx.QueryRow(ctx,
		`INSERT INTO zammad_config (url, api_token, webhook_secret, pause_statuses, updated_by)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (id) DO UPDATE SET
		   url = EXCLUDED.url,
		   api_token = EXCLUDED.api_token,
		   webhook_secret = EXCLUDED.webhook_secret,
		   pause_statuses = EXCLUDED.pause_statuses,
		   updated_by = EXCLUDED.updated_by,
		   updated_at = now()
		 RETURNING id, url, api_token, webhook_secret, pause_statuses, updated_at, updated_by`,
		req.URL, req.APIToken, req.WebhookSecret, req.PauseStatuses, updatedBy).
		Scan(&c.ID, &c.URL, &c.APIToken, &c.WebhookSecret, &c.PauseStatuses, &c.UpdatedAt, &c.UpdatedBy)
	if err != nil {
		return nil, fmt.Errorf("upserting zammad_config: %w", err)
	}
	return &c, nil
}

// --- Integration Keys ---

// GetIntegrationKey returns an integration key by service name.
func (s *Store) GetIntegrationKey(ctx context.Context, service string) (*IntegrationKey, error) {
	var k IntegrationKey
	err := s.dbtx.QueryRow(ctx,
		`SELECT id, service, api_key, api_url, updated_at
		 FROM integration_keys WHERE service = $1`, service).
		Scan(&k.ID, &k.Service, &k.APIKey, &k.APIURL, &k.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

// UpsertIntegrationKey inserts or updates an integration key.
func (s *Store) UpsertIntegrationKey(ctx context.Context, service string, req UpdateIntegrationKeyRequest) (*IntegrationKey, error) {
	var k IntegrationKey
	err := s.dbtx.QueryRow(ctx,
		`INSERT INTO integration_keys (service, api_key, api_url)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (service) DO UPDATE SET
		   api_key = EXCLUDED.api_key,
		   api_url = EXCLUDED.api_url,
		   updated_at = now()
		 RETURNING id, service, api_key, api_url, updated_at`,
		service, req.APIKey, req.APIURL).
		Scan(&k.ID, &k.Service, &k.APIKey, &k.APIURL, &k.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("upserting integration_key: %w", err)
	}
	return &k, nil
}

// --- Customer Orgs ---

// ListCustomerOrgs returns all customer organisations.
func (s *Store) ListCustomerOrgs(ctx context.Context) ([]CustomerOrg, error) {
	rows, err := s.dbtx.Query(ctx,
		`SELECT id, name, oidc_group, zammad_org_id, created_at, updated_at
		 FROM customer_orgs ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("listing customer_orgs: %w", err)
	}
	defer rows.Close()

	var orgs []CustomerOrg
	for rows.Next() {
		var o CustomerOrg
		if err := rows.Scan(&o.ID, &o.Name, &o.OIDCGroup, &o.ZammadOrgID, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning customer_org: %w", err)
		}
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}

// CreateCustomerOrg inserts a new customer organisation.
func (s *Store) CreateCustomerOrg(ctx context.Context, req CreateCustomerOrgRequest) (*CustomerOrg, error) {
	var o CustomerOrg
	err := s.dbtx.QueryRow(ctx,
		`INSERT INTO customer_orgs (name, oidc_group, zammad_org_id)
		 VALUES ($1, $2, $3)
		 RETURNING id, name, oidc_group, zammad_org_id, created_at, updated_at`,
		req.Name, req.OIDCGroup, req.ZammadOrgID).
		Scan(&o.ID, &o.Name, &o.OIDCGroup, &o.ZammadOrgID, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating customer_org: %w", err)
	}
	return &o, nil
}

// UpdateCustomerOrg updates an existing customer organisation.
func (s *Store) UpdateCustomerOrg(ctx context.Context, id uuid.UUID, req UpdateCustomerOrgRequest) (*CustomerOrg, error) {
	var o CustomerOrg
	err := s.dbtx.QueryRow(ctx,
		`UPDATE customer_orgs SET
		   name = COALESCE($2, name),
		   oidc_group = COALESCE($3, oidc_group),
		   zammad_org_id = COALESCE($4, zammad_org_id),
		   updated_at = now()
		 WHERE id = $1
		 RETURNING id, name, oidc_group, zammad_org_id, created_at, updated_at`,
		id, req.Name, req.OIDCGroup, req.ZammadOrgID).
		Scan(&o.ID, &o.Name, &o.OIDCGroup, &o.ZammadOrgID, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("updating customer_org: %w", err)
	}
	return &o, nil
}

// DeleteCustomerOrg deletes a customer organisation.
func (s *Store) DeleteCustomerOrg(ctx context.Context, id uuid.UUID) error {
	tag, err := s.dbtx.Exec(ctx, `DELETE FROM customer_orgs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting customer_org: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// --- Auto-Ticket Rules ---

// ListAutoTicketRules returns all auto-ticket rules.
func (s *Store) ListAutoTicketRules(ctx context.Context) ([]AutoTicketRule, error) {
	rows, err := s.dbtx.Query(ctx,
		`SELECT id, name, enabled, alert_group, min_severity,
		        default_priority, default_group, title_template, created_at, updated_at
		 FROM auto_ticket_rules ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("listing auto_ticket_rules: %w", err)
	}
	defer rows.Close()

	var rules []AutoTicketRule
	for rows.Next() {
		var r AutoTicketRule
		if err := rows.Scan(&r.ID, &r.Name, &r.Enabled, &r.AlertGroup, &r.MinSeverity,
			&r.DefaultPriority, &r.DefaultGroup, &r.TitleTemplate, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning auto_ticket_rule: %w", err)
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

// CreateAutoTicketRule inserts a new auto-ticket rule.
func (s *Store) CreateAutoTicketRule(ctx context.Context, req CreateAutoTicketRuleRequest) (*AutoTicketRule, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	var r AutoTicketRule
	err := s.dbtx.QueryRow(ctx,
		`INSERT INTO auto_ticket_rules (name, enabled, alert_group, min_severity, default_priority, default_group, title_template)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, name, enabled, alert_group, min_severity, default_priority, default_group, title_template, created_at, updated_at`,
		req.Name, enabled, req.AlertGroup, req.MinSeverity, req.DefaultPriority, req.DefaultGroup, req.TitleTemplate).
		Scan(&r.ID, &r.Name, &r.Enabled, &r.AlertGroup, &r.MinSeverity,
			&r.DefaultPriority, &r.DefaultGroup, &r.TitleTemplate, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating auto_ticket_rule: %w", err)
	}
	return &r, nil
}

// UpdateAutoTicketRule updates an existing auto-ticket rule.
func (s *Store) UpdateAutoTicketRule(ctx context.Context, id uuid.UUID, req UpdateAutoTicketRuleRequest) (*AutoTicketRule, error) {
	var r AutoTicketRule
	err := s.dbtx.QueryRow(ctx,
		`UPDATE auto_ticket_rules SET
		   name = COALESCE($2, name),
		   enabled = COALESCE($3, enabled),
		   alert_group = COALESCE($4, alert_group),
		   min_severity = COALESCE($5, min_severity),
		   default_priority = COALESCE($6, default_priority),
		   default_group = COALESCE($7, default_group),
		   title_template = COALESCE($8, title_template),
		   updated_at = now()
		 WHERE id = $1
		 RETURNING id, name, enabled, alert_group, min_severity, default_priority, default_group, title_template, created_at, updated_at`,
		id, req.Name, req.Enabled, req.AlertGroup, req.MinSeverity, req.DefaultPriority, req.DefaultGroup, req.TitleTemplate).
		Scan(&r.ID, &r.Name, &r.Enabled, &r.AlertGroup, &r.MinSeverity,
			&r.DefaultPriority, &r.DefaultGroup, &r.TitleTemplate, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("updating auto_ticket_rule: %w", err)
	}
	return &r, nil
}

// DeleteAutoTicketRule deletes an auto-ticket rule.
func (s *Store) DeleteAutoTicketRule(ctx context.Context, id uuid.UUID) error {
	tag, err := s.dbtx.Exec(ctx, `DELETE FROM auto_ticket_rules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting auto_ticket_rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
