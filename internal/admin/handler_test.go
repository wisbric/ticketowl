package admin_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/wisbric/ticketowl/internal/admin"
)

// --- Mock AdminStore ---

type mockStore struct {
	zammadConfig    *admin.ZammadConfig
	integrationKeys map[string]*admin.IntegrationKey
	customerOrgs    []admin.CustomerOrg
	autoTicketRules []admin.AutoTicketRule
}

func newMockStore() *mockStore {
	return &mockStore{
		integrationKeys: make(map[string]*admin.IntegrationKey),
	}
}

func (m *mockStore) GetZammadConfig(_ context.Context) (*admin.ZammadConfig, error) {
	if m.zammadConfig == nil {
		return nil, pgx.ErrNoRows
	}
	return m.zammadConfig, nil
}

func (m *mockStore) UpsertZammadConfig(_ context.Context, req admin.UpdateZammadConfigRequest, updatedBy uuid.UUID) (*admin.ZammadConfig, error) {
	m.zammadConfig = &admin.ZammadConfig{
		ID:            uuid.New(),
		URL:           req.URL,
		APIToken:      req.APIToken,
		WebhookSecret: req.WebhookSecret,
		PauseStatuses: req.PauseStatuses,
		UpdatedAt:     time.Now(),
		UpdatedBy:     &updatedBy,
	}
	return m.zammadConfig, nil
}

func (m *mockStore) GetIntegrationKey(_ context.Context, service string) (*admin.IntegrationKey, error) {
	key, ok := m.integrationKeys[service]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return key, nil
}

func (m *mockStore) UpsertIntegrationKey(_ context.Context, service string, req admin.UpdateIntegrationKeyRequest) (*admin.IntegrationKey, error) {
	key := &admin.IntegrationKey{
		ID:        uuid.New(),
		Service:   service,
		APIKey:    req.APIKey,
		APIURL:    req.APIURL,
		UpdatedAt: time.Now(),
	}
	m.integrationKeys[service] = key
	return key, nil
}

func (m *mockStore) ListCustomerOrgs(_ context.Context) ([]admin.CustomerOrg, error) {
	return m.customerOrgs, nil
}

func (m *mockStore) CreateCustomerOrg(_ context.Context, req admin.CreateCustomerOrgRequest) (*admin.CustomerOrg, error) {
	org := &admin.CustomerOrg{
		ID:          uuid.New(),
		Name:        req.Name,
		OIDCGroup:   req.OIDCGroup,
		ZammadOrgID: req.ZammadOrgID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	m.customerOrgs = append(m.customerOrgs, *org)
	return org, nil
}

func (m *mockStore) UpdateCustomerOrg(_ context.Context, id uuid.UUID, req admin.UpdateCustomerOrgRequest) (*admin.CustomerOrg, error) {
	for i, o := range m.customerOrgs {
		if o.ID == id {
			if req.Name != nil {
				m.customerOrgs[i].Name = *req.Name
			}
			if req.OIDCGroup != nil {
				m.customerOrgs[i].OIDCGroup = *req.OIDCGroup
			}
			if req.ZammadOrgID != nil {
				m.customerOrgs[i].ZammadOrgID = req.ZammadOrgID
			}
			return &m.customerOrgs[i], nil
		}
	}
	return nil, pgx.ErrNoRows
}

func (m *mockStore) DeleteCustomerOrg(_ context.Context, id uuid.UUID) error {
	for i, o := range m.customerOrgs {
		if o.ID == id {
			m.customerOrgs = append(m.customerOrgs[:i], m.customerOrgs[i+1:]...)
			return nil
		}
	}
	return pgx.ErrNoRows
}

func (m *mockStore) ListAutoTicketRules(_ context.Context) ([]admin.AutoTicketRule, error) {
	return m.autoTicketRules, nil
}

func (m *mockStore) CreateAutoTicketRule(_ context.Context, req admin.CreateAutoTicketRuleRequest) (*admin.AutoTicketRule, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	rule := &admin.AutoTicketRule{
		ID:              uuid.New(),
		Name:            req.Name,
		Enabled:         enabled,
		AlertGroup:      req.AlertGroup,
		MinSeverity:     req.MinSeverity,
		DefaultPriority: req.DefaultPriority,
		DefaultGroup:    req.DefaultGroup,
		TitleTemplate:   req.TitleTemplate,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	m.autoTicketRules = append(m.autoTicketRules, *rule)
	return rule, nil
}

func (m *mockStore) UpdateAutoTicketRule(_ context.Context, id uuid.UUID, req admin.UpdateAutoTicketRuleRequest) (*admin.AutoTicketRule, error) {
	for i, r := range m.autoTicketRules {
		if r.ID == id {
			if req.Name != nil {
				m.autoTicketRules[i].Name = *req.Name
			}
			if req.Enabled != nil {
				m.autoTicketRules[i].Enabled = *req.Enabled
			}
			return &m.autoTicketRules[i], nil
		}
	}
	return nil, pgx.ErrNoRows
}

func (m *mockStore) DeleteAutoTicketRule(_ context.Context, id uuid.UUID) error {
	for i, r := range m.autoTicketRules {
		if r.ID == id {
			m.autoTicketRules = append(m.autoTicketRules[:i], m.autoTicketRules[i+1:]...)
			return nil
		}
	}
	return pgx.ErrNoRows
}

// --- Tests ---

func TestUpsertZammadConfig(t *testing.T) {
	store := newMockStore()

	req := admin.UpdateZammadConfigRequest{
		URL:           "https://zammad.example.com",
		APIToken:      "secret-token",
		WebhookSecret: "hmac-secret",
		PauseStatuses: []string{"pending customer"},
	}

	cfg, err := store.UpsertZammadConfig(context.Background(), req, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.URL != req.URL {
		t.Errorf("URL = %q, want %q", cfg.URL, req.URL)
	}
	if len(cfg.PauseStatuses) != 1 || cfg.PauseStatuses[0] != "pending customer" {
		t.Errorf("PauseStatuses = %v, want [pending customer]", cfg.PauseStatuses)
	}

	// Verify stored.
	got, err := store.GetZammadConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.URL != req.URL {
		t.Errorf("stored URL = %q, want %q", got.URL, req.URL)
	}
}

func TestUpsertIntegrationKey(t *testing.T) {
	store := newMockStore()

	req := admin.UpdateIntegrationKeyRequest{
		APIKey: "my-api-key",
		APIURL: "https://nightowl.example.com",
	}

	key, err := store.UpsertIntegrationKey(context.Background(), "nightowl", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key.Service != "nightowl" {
		t.Errorf("Service = %q, want %q", key.Service, "nightowl")
	}
	if key.APIURL != req.APIURL {
		t.Errorf("APIURL = %q, want %q", key.APIURL, req.APIURL)
	}
}

func TestCustomerOrgCRUD(t *testing.T) {
	store := newMockStore()

	// Create.
	zammadOrgID := 42
	org, err := store.CreateCustomerOrg(context.Background(), admin.CreateCustomerOrgRequest{
		Name:        "Acme Corp",
		OIDCGroup:   "acme-customers",
		ZammadOrgID: &zammadOrgID,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if org.Name != "Acme Corp" {
		t.Errorf("Name = %q, want %q", org.Name, "Acme Corp")
	}

	// List.
	orgs, err := store.ListCustomerOrgs(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(orgs) != 1 {
		t.Fatalf("len = %d, want 1", len(orgs))
	}

	// Update.
	newName := "Acme Inc"
	updated, err := store.UpdateCustomerOrg(context.Background(), org.ID, admin.UpdateCustomerOrgRequest{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Acme Inc" {
		t.Errorf("updated Name = %q, want %q", updated.Name, "Acme Inc")
	}

	// Delete.
	err = store.DeleteCustomerOrg(context.Background(), org.ID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	orgs, _ = store.ListCustomerOrgs(context.Background())
	if len(orgs) != 0 {
		t.Errorf("len after delete = %d, want 0", len(orgs))
	}
}

func TestAutoTicketRuleCRUD(t *testing.T) {
	store := newMockStore()

	alertGroup := "kubernetes-"
	minSev := "high"

	// Create.
	rule, err := store.CreateAutoTicketRule(context.Background(), admin.CreateAutoTicketRuleRequest{
		Name:            "K8s High Severity",
		AlertGroup:      &alertGroup,
		MinSeverity:     &minSev,
		DefaultPriority: "high",
		TitleTemplate:   "{{.AlertName}}: {{.Summary}}",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if rule.Name != "K8s High Severity" {
		t.Errorf("Name = %q, want %q", rule.Name, "K8s High Severity")
	}
	if !rule.Enabled {
		t.Error("new rule should be enabled by default")
	}

	// List.
	rules, err := store.ListAutoTicketRules(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("len = %d, want 1", len(rules))
	}

	// Update.
	disabled := false
	updated, err := store.UpdateAutoTicketRule(context.Background(), rule.ID, admin.UpdateAutoTicketRuleRequest{
		Enabled: &disabled,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Enabled {
		t.Error("expected rule to be disabled after update")
	}

	// Delete.
	err = store.DeleteAutoTicketRule(context.Background(), rule.ID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	rules, _ = store.ListAutoTicketRules(context.Background())
	if len(rules) != 0 {
		t.Errorf("len after delete = %d, want 0", len(rules))
	}
}

func TestDeleteCustomerOrg_NotFound(t *testing.T) {
	store := newMockStore()
	err := store.DeleteCustomerOrg(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for missing org")
	}
}

func TestDeleteAutoTicketRule_NotFound(t *testing.T) {
	store := newMockStore()
	err := store.DeleteAutoTicketRule(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for missing rule")
	}
}
