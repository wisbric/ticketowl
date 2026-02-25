package ticket_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/wisbric/ticketowl/internal/bookowl"
	"github.com/wisbric/ticketowl/internal/link"
	"github.com/wisbric/ticketowl/internal/nightowl"
	"github.com/wisbric/ticketowl/internal/ticket"
)

// --- Mock PostMortemStore ---

type mockPMStore struct {
	metaID        uuid.UUID
	metaErr       error
	existingPM    *link.PostMortemLink
	existingPMErr error
	incidentLinks []link.IncidentLink
	createdPM     *link.PostMortemLink
}

func (m *mockPMStore) GetTicketMetaID(_ context.Context, _ int) (uuid.UUID, error) {
	return m.metaID, m.metaErr
}

func (m *mockPMStore) GetPostMortemLink(_ context.Context, _ uuid.UUID) (*link.PostMortemLink, error) {
	return m.existingPM, m.existingPMErr
}

func (m *mockPMStore) CreatePostMortemLink(_ context.Context, _, _, _ uuid.UUID, url string) (*link.PostMortemLink, error) {
	if m.createdPM != nil {
		return m.createdPM, nil
	}
	return &link.PostMortemLink{
		ID:            uuid.New(),
		PostMortemURL: url,
	}, nil
}

func (m *mockPMStore) ListIncidentLinks(_ context.Context, _ uuid.UUID) ([]link.IncidentLink, error) {
	return m.incidentLinks, nil
}

// --- Mock PostMortemCreator ---

type mockPMCreator struct {
	result *bookowl.PostMortem
	err    error
}

func (m *mockPMCreator) CreatePostMortem(_ context.Context, _ bookowl.CreatePostMortemRequest) (*bookowl.PostMortem, error) {
	return m.result, m.err
}

// --- Mock NightOwlIncidentFetcher ---

type mockIncidentFetcher struct {
	incidents map[string]*nightowl.Incident
}

func (m *mockIncidentFetcher) GetIncident(_ context.Context, id string) (*nightowl.Incident, error) {
	inc, ok := m.incidents[id]
	if !ok {
		return nil, fmt.Errorf("incident not found")
	}
	return inc, nil
}

// --- Tests ---

func TestCreatePostMortem_HappyPath(t *testing.T) {
	pmID := uuid.New()
	store := &mockPMStore{
		metaID:        uuid.New(),
		existingPMErr: pgx.ErrNoRows,
		incidentLinks: []link.IncidentLink{},
	}
	creator := &mockPMCreator{
		result: &bookowl.PostMortem{
			ID:  pmID.String(),
			URL: "https://kb.example.com/pm/" + pmID.String(),
		},
	}
	fetcher := &mockIncidentFetcher{incidents: make(map[string]*nightowl.Incident)}

	result, err := ticket.CreatePostMortem(
		context.Background(), 42, "T42", "Outage", uuid.New(),
		store, creator, fetcher,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PostMortemID != pmID.String() {
		t.Errorf("PostMortemID = %q, want %q", result.PostMortemID, pmID.String())
	}
}

func TestCreatePostMortem_AlreadyExists_Returns409(t *testing.T) {
	existingURL := "https://kb.example.com/pm/existing"
	store := &mockPMStore{
		metaID: uuid.New(),
		existingPM: &link.PostMortemLink{
			PostMortemURL: existingURL,
		},
	}
	creator := &mockPMCreator{}
	fetcher := &mockIncidentFetcher{}

	_, err := ticket.CreatePostMortem(
		context.Background(), 42, "T42", "Outage", uuid.New(),
		store, creator, fetcher,
	)
	if err == nil {
		t.Fatal("expected ErrPostMortemExists")
	}
	var existsErr *ticket.ErrPostMortemExists
	if !errors.As(err, &existsErr) {
		t.Fatalf("expected ErrPostMortemExists, got %T: %v", err, err)
	}
	if existsErr.URL != existingURL {
		t.Errorf("URL = %q, want %q", existsErr.URL, existingURL)
	}
}

func TestCreatePostMortem_TicketNotTracked(t *testing.T) {
	store := &mockPMStore{
		metaErr: pgx.ErrNoRows,
	}
	creator := &mockPMCreator{}
	fetcher := &mockIncidentFetcher{}

	_, err := ticket.CreatePostMortem(
		context.Background(), 999, "T999", "Missing", uuid.New(),
		store, creator, fetcher,
	)
	if err == nil {
		t.Fatal("expected error for untracked ticket")
	}
}

func TestCreatePostMortem_WithIncidents(t *testing.T) {
	pmID := uuid.New()
	incID := uuid.New()
	store := &mockPMStore{
		metaID:        uuid.New(),
		existingPMErr: pgx.ErrNoRows,
		incidentLinks: []link.IncidentLink{
			{IncidentID: incID, IncidentSlug: "inc-001"},
		},
	}
	creator := &mockPMCreator{
		result: &bookowl.PostMortem{
			ID:  pmID.String(),
			URL: "https://kb.example.com/pm/" + pmID.String(),
		},
	}
	fetcher := &mockIncidentFetcher{
		incidents: map[string]*nightowl.Incident{
			incID.String(): {ID: incID.String(), Summary: "DB outage"},
		},
	}

	result, err := ticket.CreatePostMortem(
		context.Background(), 42, "T42", "Outage", uuid.New(),
		store, creator, fetcher,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PostMortemID != pmID.String() {
		t.Errorf("PostMortemID = %q, want %q", result.PostMortemID, pmID.String())
	}
}

func TestCreatePostMortem_NightOwlFetchFailure(t *testing.T) {
	// NightOwl failures are best-effort — should not fail the whole operation.
	pmID := uuid.New()
	incID := uuid.New()
	store := &mockPMStore{
		metaID:        uuid.New(),
		existingPMErr: pgx.ErrNoRows,
		incidentLinks: []link.IncidentLink{
			{IncidentID: incID, IncidentSlug: "inc-001"},
		},
	}
	creator := &mockPMCreator{
		result: &bookowl.PostMortem{
			ID:  pmID.String(),
			URL: "https://kb.example.com/pm/" + pmID.String(),
		},
	}
	// No incidents in the fetcher — all fetches will fail.
	fetcher := &mockIncidentFetcher{incidents: make(map[string]*nightowl.Incident)}

	result, err := ticket.CreatePostMortem(
		context.Background(), 42, "T42", "Outage", uuid.New(),
		store, creator, fetcher,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v (NightOwl failures should be best-effort)", err)
	}
	if result.PostMortemID != pmID.String() {
		t.Errorf("PostMortemID = %q, want %q", result.PostMortemID, pmID.String())
	}
}
