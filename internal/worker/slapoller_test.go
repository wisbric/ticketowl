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

func (m *mockPollerStore) ListStaleStates(_ context.Context, _ time.Time) ([]sla.State, error) {
	return m.states, nil
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

	// State is on_track with no prior breach alert — response SLA breached (>60 min elapsed).
	store.states = []sla.State{
		{
			ID:                   uuid.New(),
			TicketMetaID:         metaID,
			Label:                sla.LabelOnTrack,
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

	// Should NOT send another alert.
	if len(alerter.alerts) != 0 {
		t.Errorf("expected 0 alerts (already alerted), got %d", len(alerter.alerts))
	}

	// State should still be updated.
	if len(store.updated) == 0 {
		t.Fatal("expected state to be updated")
	}
}

func TestSLAPoller_OnTrackStaysOnTrack(t *testing.T) {
	store := newMockPollerStore()
	alerter := &mockAlerter{}
	notifier := notification.NewService(alerter, slog.Default())

	policyID := uuid.New()
	metaID := uuid.New()
	ticketCreatedAt := time.Now().Add(-10 * time.Minute)

	store.polices[policyID] = &sla.Policy{
		ID:                policyID,
		Priority:          "normal",
		ResponseMinutes:   60,
		ResolutionMinutes: 480,
		WarningThreshold:  0.20,
	}

	store.metas[metaID] = &worker.TicketMetaInfo{
		ID:           metaID,
		ZammadID:     10,
		ZammadNumber: "T10",
		SLAPolicyID:  &policyID,
		CreatedAt:    ticketCreatedAt,
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

	if len(alerter.alerts) != 0 {
		t.Errorf("expected 0 alerts for on_track, got %d", len(alerter.alerts))
	}

	if len(store.updated) == 0 {
		t.Fatal("expected state to be updated")
	}
	if store.updated[0].Label != sla.LabelOnTrack {
		t.Errorf("label = %q, want on_track", store.updated[0].Label)
	}
}

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
