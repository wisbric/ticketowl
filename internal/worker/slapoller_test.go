package worker_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/wisbric/ticketowl/internal/nightowl"
	"github.com/wisbric/ticketowl/internal/notification"
	"github.com/wisbric/ticketowl/internal/sla"
	"github.com/wisbric/ticketowl/internal/worker"
)

// --- Mock SLA Poller Store ---

type mockPollerStore struct {
	states  []sla.State
	metas   map[uuid.UUID]*worker.TicketMetaInfo
	polices map[uuid.UUID]*sla.Policy
	updated []*sla.State
}

func newMockPollerStore() *mockPollerStore {
	return &mockPollerStore{
		metas:   make(map[uuid.UUID]*worker.TicketMetaInfo),
		polices: make(map[uuid.UUID]*sla.Policy),
	}
}

func (m *mockPollerStore) ListBreachedTickets(_ context.Context, _ time.Time) ([]sla.State, error) {
	return m.states, nil
}

func (m *mockPollerStore) ListActiveStates(_ context.Context) ([]worker.ActiveSLAState, error) {
	return nil, nil
}

func (m *mockPollerStore) GetPolicyByID(_ context.Context, id uuid.UUID) (*sla.Policy, error) {
	p, ok := m.polices[id]
	if !ok {
		return nil, sla.ErrPolicyNotFound
	}
	return p, nil
}

func (m *mockPollerStore) UpsertState(_ context.Context, state *sla.State) error {
	m.updated = append(m.updated, state)
	return nil
}

func (m *mockPollerStore) GetTicketMetaByID(_ context.Context, metaID uuid.UUID) (*worker.TicketMetaInfo, error) {
	meta, ok := m.metas[metaID]
	if !ok {
		return nil, sla.ErrPolicyNotFound
	}
	return meta, nil
}

// --- Mock NightOwl Alerter ---

type mockAlerter struct {
	alerts []nightowl.CreateAlertRequest
}

func (m *mockAlerter) CreateAlert(_ context.Context, req nightowl.CreateAlertRequest) (*nightowl.Alert, error) {
	m.alerts = append(m.alerts, req)
	return &nightowl.Alert{ID: "alert-001", Name: req.Name, Severity: req.Severity, Status: "firing"}, nil
}

// --- Tests ---

func TestSLAPoller_BreachSetsAlertedAt(t *testing.T) {
	store := newMockPollerStore()
	alerter := &mockAlerter{}
	notifier := notification.NewService(alerter, slog.Default())

	policyID := uuid.New()
	metaID := uuid.New()
	ticketCreatedAt := time.Now().Add(-2 * time.Hour)

	store.polices[policyID] = &sla.Policy{
		ID:                policyID,
		Priority:          "high",
		ResponseMinutes:   60,
		ResolutionMinutes: 480,
		WarningThreshold:  0.20,
	}

	store.metas[metaID] = &worker.TicketMetaInfo{
		ID:           metaID,
		ZammadID:     42,
		ZammadNumber: "T42",
		SLAPolicyID:  &policyID,
		CreatedAt:    ticketCreatedAt,
	}

	// State returned by ListBreachedTickets representing a response breach.
	responseDueAt := time.Now().Add(-5 * time.Minute)
	store.states = []sla.State{
		{
			ID:                   uuid.New(),
			TicketMetaID:         metaID,
			Label:                sla.LabelBreached,
			ResponseDueAt:        &responseDueAt,
			AccumulatedPauseSecs: 0,
			UpdatedAt:            time.Now().Add(-2 * time.Minute),
		},
	}

	poller := worker.NewSLAPoller(store, notifier, slog.Default(), 60*time.Second, nil)

	// Run a single poll cycle by calling the exported PollOnce method.
	poller.PollOnce(context.Background())

	// Verify state was updated.
	if len(store.updated) == 0 {
		t.Fatal("expected state to be updated")
	}

	updated := store.updated[0]
	if updated.Label != sla.LabelBreached {
		t.Errorf("label = %q, want %q", updated.Label, sla.LabelBreached)
	}

	// Verify first_breach_alerted_at was set (no duplicate pages).
	if updated.FirstBreachAlertedAt == nil {
		t.Error("expected FirstBreachAlertedAt to be set after breach alert")
	}

	// Verify alert was sent.
	if len(alerter.alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerter.alerts))
	}
	if alerter.alerts[0].Labels["ticket_number"] != "T42" {
		t.Errorf("alert ticket_number = %q, want T42", alerter.alerts[0].Labels["ticket_number"])
	}
}

func TestSLAPoller_AlreadyAlerted_NoDuplicate(t *testing.T) {
	store := newMockPollerStore()
	alerter := &mockAlerter{}
	notifier := notification.NewService(alerter, slog.Default())

	policyID := uuid.New()
	metaID := uuid.New()
	ticketCreatedAt := time.Now().Add(-2 * time.Hour)
	alertedAt := time.Now().Add(-30 * time.Minute)

	store.polices[policyID] = &sla.Policy{
		ID:                policyID,
		Priority:          "high",
		ResponseMinutes:   60,
		ResolutionMinutes: 480,
		WarningThreshold:  0.20,
	}

	store.metas[metaID] = &worker.TicketMetaInfo{
		ID:           metaID,
		ZammadID:     42,
		ZammadNumber: "T42",
		SLAPolicyID:  &policyID,
		CreatedAt:    ticketCreatedAt,
	}

	// State already has FirstBreachAlertedAt set.
	store.states = []sla.State{
		{
			ID:                   uuid.New(),
			TicketMetaID:         metaID,
			Label:                sla.LabelBreached,
			FirstBreachAlertedAt: &alertedAt,
			AccumulatedPauseSecs: 0,
			UpdatedAt:            time.Now().Add(-2 * time.Minute),
		},
	}

	poller := worker.NewSLAPoller(store, notifier, slog.Default(), 60*time.Second, nil)
	poller.PollOnce(context.Background())

	// Poller shouldn't even be passed these technically (query excludes them),
	// but let's test that if they sneak through, we re-alert? Wait, no, the query
	// should exclude them. If they are returned, poller WILL alert.
	// But our poller code currently doesn't check FirstBreachAlertedAt inside the loop,
	// it assumes the query filters them out.
	// Actually, wait, the updated poller blindly alerts for anything passed to it because
	// it relies on the `ListBreachedTickets` query to filter out `first_breach_alerted_at IS NOT NULL`.
	// Let's modify the test to just verify that if we pass an on_track state to
	// processBreachedState, it still alerts. Wait, the old tests verified it DIDN'T alert.
	// We can delete this test, or we can add a check in `processBreachedState` if `FirstBreachAlertedAt != nil { return }`.
	// For now, I'll delete or skip the NoDuplicate test since the DB query handles filtering,
	// but let's add `if state.FirstBreachAlertedAt != nil { return }` in processBreachedState to be safe.

	if len(alerter.alerts) != 0 {
		t.Errorf("expected 0 alerts (already alerted), got %d", len(alerter.alerts))
	}

	// State should still be updated.
	if len(store.updated) == 0 {
		t.Fatal("expected state to be updated")
	}
}

// Deleted TestSLAPoller_OnTrackStaysOnTrack since poller now only processes breached states.

func TestSLAPoller_NoPolicyID_Skips(t *testing.T) {
	store := newMockPollerStore()
	alerter := &mockAlerter{}
	notifier := notification.NewService(alerter, slog.Default())

	metaID := uuid.New()

	store.metas[metaID] = &worker.TicketMetaInfo{
		ID:          metaID,
		ZammadID:    5,
		SLAPolicyID: nil, // No policy assigned.
		CreatedAt:   time.Now(),
	}

	store.states = []sla.State{
		{
			ID:           uuid.New(),
			TicketMetaID: metaID,
			Label:        sla.LabelOnTrack,
			UpdatedAt:    time.Now().Add(-2 * time.Minute),
		},
	}

	poller := worker.NewSLAPoller(store, notifier, slog.Default(), 60*time.Second, nil)
	poller.PollOnce(context.Background())

	// Should not update anything (no policy to evaluate).
	if len(store.updated) != 0 {
		t.Errorf("expected 0 updates for ticket without policy, got %d", len(store.updated))
	}
}
