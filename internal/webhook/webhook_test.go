package webhook_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/wisbric/core/pkg/tenant"

	"github.com/wisbric/ticketowl/internal/webhook"
	"github.com/wisbric/ticketowl/internal/zammad"
)

const testSecret = "test-webhook-secret"

func testLogger() *slog.Logger {
	return slog.Default()
}

// staticSecret returns a SecretResolver that always returns the given secret.
func staticSecret(secret string) webhook.SecretResolver {
	return func(r *http.Request) (string, error) {
		return secret, nil
	}
}

func testTenantContext() context.Context {
	return tenant.NewContext(context.Background(), &tenant.Info{
		ID:     uuid.New(),
		Slug:   "acme",
		Name:   "Acme Corp",
		Schema: "tenant_acme",
	})
}

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	// Use a miniredis-like approach: we'll use a real Redis if available,
	// otherwise skip. For unit tests we use the alicebob/miniredis package
	// but since we want to keep deps minimal, we'll use a fakeredis approach.
	// Actually, let's use the redis client pointed at a test-only server.
	// For CI, we'll use the miniredis in-process server.
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6381",
	})

	// Try to ping — if Redis isn't available, skip the test.
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skipf("Redis not available at localhost:6381: %v (skipping webhook integration tests)", err)
	}

	// Clean up the test stream after the test.
	t.Cleanup(func() {
		rdb.Del(context.Background(), webhook.StreamKey("acme"))
		_ = rdb.Close()
	})

	return rdb
}

// --- Zammad Webhook Tests ---

func TestZammadWebhook_ValidSignature(t *testing.T) {
	rdb := newTestRedis(t)

	payload := map[string]any{
		"event": "ticket.create",
		"ticket": map[string]any{
			"id":       42,
			"number":   "T42",
			"state_id": 1,
			"state":    "new",
			"priority": "2 normal",
		},
	}
	body, _ := json.Marshal(payload)
	sig := zammad.ComputeWebhookSignature(body, testSecret)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/zammad", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature", sig)
	req = req.WithContext(testTenantContext())

	rr := httptest.NewRecorder()
	handler := webhook.HandleZammad(rdb, staticSecret(testSecret), testLogger())
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	// Verify event was pushed to stream.
	msgs, err := rdb.XRange(context.Background(), webhook.StreamKey("acme"), "-", "+").Result()
	if err != nil {
		t.Fatalf("XRange: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("expected at least one message in the stream")
	}

	source, ok := msgs[len(msgs)-1].Values["source"].(string)
	if !ok || source != "zammad" {
		t.Errorf("source = %q, want zammad", source)
	}

	data, ok := msgs[len(msgs)-1].Values["data"].(string)
	if !ok {
		t.Fatal("data field not found in stream message")
	}

	var evt webhook.ZammadEvent
	if err := json.Unmarshal([]byte(data), &evt); err != nil {
		t.Fatalf("unmarshaling event data: %v", err)
	}
	if evt.Event != "ticket.create" {
		t.Errorf("event = %q, want ticket.create", evt.Event)
	}
	if evt.TicketID != 42 {
		t.Errorf("ticket_id = %d, want 42", evt.TicketID)
	}
}

func TestZammadWebhook_InvalidSignature(t *testing.T) {
	rdb := newTestRedis(t)

	payload := map[string]any{
		"event":  "ticket.create",
		"ticket": map[string]any{"id": 42, "number": "T42"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/zammad", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature", "sha1=invalidhex")
	req = req.WithContext(testTenantContext())

	rr := httptest.NewRecorder()
	handler := webhook.HandleZammad(rdb, staticSecret(testSecret), testLogger())
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestZammadWebhook_TamperedBody(t *testing.T) {
	rdb := newTestRedis(t)

	originalBody := []byte(`{"event":"ticket.create","ticket":{"id":42,"number":"T42"}}`)
	sig := zammad.ComputeWebhookSignature(originalBody, testSecret)

	// Send a different body with the original signature.
	tamperedBody := []byte(`{"event":"ticket.create","ticket":{"id":99,"number":"T99"}}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/zammad", bytes.NewReader(tamperedBody))
	req.Header.Set("X-Hub-Signature", sig)
	req = req.WithContext(testTenantContext())

	rr := httptest.NewRecorder()
	handler := webhook.HandleZammad(rdb, staticSecret(testSecret), testLogger())
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d (tampered body should fail)", rr.Code, http.StatusUnauthorized)
	}
}

func TestZammadWebhook_WrongSecret(t *testing.T) {
	rdb := newTestRedis(t)

	payload := map[string]any{
		"event":  "ticket.create",
		"ticket": map[string]any{"id": 42, "number": "T42"},
	}
	body, _ := json.Marshal(payload)
	sig := zammad.ComputeWebhookSignature(body, "wrong-secret")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/zammad", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature", sig)
	req = req.WithContext(testTenantContext())

	rr := httptest.NewRecorder()
	handler := webhook.HandleZammad(rdb, staticSecret(testSecret), testLogger())
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d (wrong secret should fail)", rr.Code, http.StatusUnauthorized)
	}
}

func TestZammadWebhook_MissingSignature(t *testing.T) {
	rdb := newTestRedis(t)

	payload := map[string]any{
		"event":  "ticket.create",
		"ticket": map[string]any{"id": 42, "number": "T42"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/zammad", bytes.NewReader(body))
	// No X-Hub-Signature header.
	req = req.WithContext(testTenantContext())

	rr := httptest.NewRecorder()
	handler := webhook.HandleZammad(rdb, staticSecret(testSecret), testLogger())
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d (missing signature should fail)", rr.Code, http.StatusUnauthorized)
	}
}

func TestZammadWebhook_MissingEvent(t *testing.T) {
	rdb := newTestRedis(t)

	payload := map[string]any{
		"ticket": map[string]any{"id": 42, "number": "T42"},
	}
	body, _ := json.Marshal(payload)
	sig := zammad.ComputeWebhookSignature(body, testSecret)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/zammad", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature", sig)
	req = req.WithContext(testTenantContext())

	rr := httptest.NewRecorder()
	handler := webhook.HandleZammad(rdb, staticSecret(testSecret), testLogger())
	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (missing event should be bad request)", rr.Code, http.StatusBadRequest)
	}
}

func TestZammadWebhook_MissingTicket(t *testing.T) {
	rdb := newTestRedis(t)

	payload := map[string]any{
		"event": "ticket.create",
	}
	body, _ := json.Marshal(payload)
	sig := zammad.ComputeWebhookSignature(body, testSecret)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/zammad", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature", sig)
	req = req.WithContext(testTenantContext())

	rr := httptest.NewRecorder()
	handler := webhook.HandleZammad(rdb, staticSecret(testSecret), testLogger())
	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

// --- NightOwl Webhook Tests ---

func TestNightOwlWebhook_Valid(t *testing.T) {
	rdb := newTestRedis(t)

	payload := map[string]any{
		"event": "incident.created",
		"incident": map[string]any{
			"id":       "inc-001",
			"slug":     "INC-001",
			"summary":  "Server down",
			"severity": "critical",
			"status":   "open",
			"service":  "api-gateway",
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/nightowl", bytes.NewReader(body))
	req = req.WithContext(testTenantContext())

	rr := httptest.NewRecorder()
	handler := webhook.HandleNightOwl(rdb, testLogger())
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	// Verify event was pushed to stream.
	msgs, err := rdb.XRange(context.Background(), webhook.StreamKey("acme"), "-", "+").Result()
	if err != nil {
		t.Fatalf("XRange: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("expected at least one message in the stream")
	}

	lastMsg := msgs[len(msgs)-1]
	source, _ := lastMsg.Values["source"].(string)
	if source != "nightowl" {
		t.Errorf("source = %q, want nightowl", source)
	}

	data, _ := lastMsg.Values["data"].(string)
	var evt webhook.NightOwlEvent
	if err := json.Unmarshal([]byte(data), &evt); err != nil {
		t.Fatalf("unmarshaling event data: %v", err)
	}
	if evt.Event != "incident.created" {
		t.Errorf("event = %q, want incident.created", evt.Event)
	}
	if evt.IncidentID != "inc-001" {
		t.Errorf("incident_id = %q, want inc-001", evt.IncidentID)
	}
}

func TestNightOwlWebhook_MissingEvent(t *testing.T) {
	rdb := newTestRedis(t)

	payload := map[string]any{
		"incident": map[string]any{"id": "inc-001"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/nightowl", bytes.NewReader(body))
	req = req.WithContext(testTenantContext())

	rr := httptest.NewRecorder()
	handler := webhook.HandleNightOwl(rdb, testLogger())
	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestNightOwlWebhook_MissingIncident(t *testing.T) {
	rdb := newTestRedis(t)

	payload := map[string]any{
		"event":    "incident.created",
		"incident": map[string]any{},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/nightowl", bytes.NewReader(body))
	req = req.WithContext(testTenantContext())

	rr := httptest.NewRecorder()
	handler := webhook.HandleNightOwl(rdb, testLogger())
	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (empty incident ID should fail)", rr.Code, http.StatusBadRequest)
	}
}

func TestNightOwlWebhook_InvalidJSON(t *testing.T) {
	rdb := newTestRedis(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/nightowl", bytes.NewReader([]byte("not json")))
	req = req.WithContext(testTenantContext())

	rr := httptest.NewRecorder()
	handler := webhook.HandleNightOwl(rdb, testLogger())
	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestStreamKey(t *testing.T) {
	key := webhook.StreamKey("acme")
	if key != "ticketowl:acme:events" {
		t.Errorf("StreamKey = %q, want ticketowl:acme:events", key)
	}
}
