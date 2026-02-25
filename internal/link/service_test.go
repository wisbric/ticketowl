package link_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/wisbric/ticketowl/internal/bookowl"
	"github.com/wisbric/ticketowl/internal/link"
	"github.com/wisbric/ticketowl/internal/nightowl"
)

// --- Mock NightOwl Client ---

type mockNightOwl struct {
	incidents map[string]*nightowl.Incident
}

func newMockNightOwl() *mockNightOwl {
	return &mockNightOwl{incidents: make(map[string]*nightowl.Incident)}
}

func (m *mockNightOwl) GetIncident(_ context.Context, incidentID string) (*nightowl.Incident, error) {
	inc, ok := m.incidents[incidentID]
	if !ok {
		return nil, &nightowl.APIError{StatusCode: 404, Message: "incident not found"}
	}
	return inc, nil
}

// --- Mock BookOwl Client ---

type mockBookOwl struct {
	articles map[string]*bookowl.Article
}

func newMockBookOwl() *mockBookOwl {
	return &mockBookOwl{articles: make(map[string]*bookowl.Article)}
}

func (m *mockBookOwl) GetArticle(_ context.Context, articleID string) (*bookowl.Article, error) {
	art, ok := m.articles[articleID]
	if !ok {
		return nil, &bookowl.APIError{StatusCode: 404, Message: "article not found"}
	}
	return art, nil
}

// --- Mock Link Store ---

type mockLinkStore struct {
	ticketMetaIDs map[int]uuid.UUID
	incidentLinks map[uuid.UUID][]link.IncidentLink
	articleLinks  map[uuid.UUID][]link.ArticleLink
	postMortems   map[uuid.UUID]*link.PostMortemLink
	nextID        int
}

func newMockLinkStore() *mockLinkStore {
	return &mockLinkStore{
		ticketMetaIDs: make(map[int]uuid.UUID),
		incidentLinks: make(map[uuid.UUID][]link.IncidentLink),
		articleLinks:  make(map[uuid.UUID][]link.ArticleLink),
		postMortems:   make(map[uuid.UUID]*link.PostMortemLink),
		nextID:        1,
	}
}

func (m *mockLinkStore) GetTicketMetaID(_ context.Context, zammadID int) (uuid.UUID, error) {
	id, ok := m.ticketMetaIDs[zammadID]
	if !ok {
		return uuid.UUID{}, fmt.Errorf("looking up ticket_meta for zammad_id %d: %w", zammadID, pgx.ErrNoRows)
	}
	return id, nil
}

func (m *mockLinkStore) ListIncidentLinks(_ context.Context, ticketMetaID uuid.UUID) ([]link.IncidentLink, error) {
	return m.incidentLinks[ticketMetaID], nil
}

func (m *mockLinkStore) CreateIncidentLink(_ context.Context, ticketMetaID, incidentID, linkedBy uuid.UUID, slug string) (*link.IncidentLink, error) {
	l := link.IncidentLink{
		ID:           uuid.New(),
		TicketMetaID: ticketMetaID,
		IncidentID:   incidentID,
		IncidentSlug: slug,
		LinkedBy:     linkedBy,
	}
	m.incidentLinks[ticketMetaID] = append(m.incidentLinks[ticketMetaID], l)
	return &l, nil
}

func (m *mockLinkStore) DeleteIncidentLink(_ context.Context, ticketMetaID, incidentID uuid.UUID) error {
	links := m.incidentLinks[ticketMetaID]
	for i, l := range links {
		if l.IncidentID == incidentID {
			m.incidentLinks[ticketMetaID] = append(links[:i], links[i+1:]...)
			return nil
		}
	}
	return pgx.ErrNoRows
}

func (m *mockLinkStore) ListArticleLinks(_ context.Context, ticketMetaID uuid.UUID) ([]link.ArticleLink, error) {
	return m.articleLinks[ticketMetaID], nil
}

func (m *mockLinkStore) CreateArticleLink(_ context.Context, ticketMetaID, articleID, linkedBy uuid.UUID, slug, title string) (*link.ArticleLink, error) {
	l := link.ArticleLink{
		ID:           uuid.New(),
		TicketMetaID: ticketMetaID,
		ArticleID:    articleID,
		ArticleSlug:  slug,
		ArticleTitle: title,
		LinkedBy:     linkedBy,
	}
	m.articleLinks[ticketMetaID] = append(m.articleLinks[ticketMetaID], l)
	return &l, nil
}

func (m *mockLinkStore) DeleteArticleLink(_ context.Context, ticketMetaID, articleID uuid.UUID) error {
	links := m.articleLinks[ticketMetaID]
	for i, l := range links {
		if l.ArticleID == articleID {
			m.articleLinks[ticketMetaID] = append(links[:i], links[i+1:]...)
			return nil
		}
	}
	return pgx.ErrNoRows
}

func (m *mockLinkStore) GetPostMortemLink(_ context.Context, ticketMetaID uuid.UUID) (*link.PostMortemLink, error) {
	pm, ok := m.postMortems[ticketMetaID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return pm, nil
}

func (m *mockLinkStore) CreatePostMortemLink(_ context.Context, ticketMetaID, postmortemID, createdBy uuid.UUID, url string) (*link.PostMortemLink, error) {
	pm := &link.PostMortemLink{
		ID:            uuid.New(),
		TicketMetaID:  ticketMetaID,
		PostMortemID:  postmortemID,
		PostMortemURL: url,
		CreatedBy:     createdBy,
	}
	m.postMortems[ticketMetaID] = pm
	return pm, nil
}

func testLogger() *slog.Logger {
	return slog.Default()
}

// --- Tests ---

func TestGetLinks_NoMetadata(t *testing.T) {
	store := newMockLinkStore()
	no := newMockNightOwl()
	bo := newMockBookOwl()

	svc := link.NewService(store, no, bo, testLogger())

	links, err := svc.GetLinks(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetLinks: unexpected error: %v", err)
	}
	if len(links.Incidents) != 0 {
		t.Errorf("len(Incidents) = %d, want 0", len(links.Incidents))
	}
	if len(links.Articles) != 0 {
		t.Errorf("len(Articles) = %d, want 0", len(links.Articles))
	}
	if links.PostMortem != nil {
		t.Error("PostMortem is non-nil, want nil")
	}
}

func TestGetLinks_WithLinks(t *testing.T) {
	store := newMockLinkStore()
	no := newMockNightOwl()
	bo := newMockBookOwl()

	metaID := uuid.New()
	store.ticketMetaIDs[42] = metaID
	store.incidentLinks[metaID] = []link.IncidentLink{
		{ID: uuid.New(), TicketMetaID: metaID, IncidentID: uuid.New(), IncidentSlug: "inc-1"},
	}
	store.articleLinks[metaID] = []link.ArticleLink{
		{ID: uuid.New(), TicketMetaID: metaID, ArticleID: uuid.New(), ArticleTitle: "Runbook"},
	}

	svc := link.NewService(store, no, bo, testLogger())

	links, err := svc.GetLinks(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetLinks: unexpected error: %v", err)
	}
	if len(links.Incidents) != 1 {
		t.Errorf("len(Incidents) = %d, want 1", len(links.Incidents))
	}
	if len(links.Articles) != 1 {
		t.Errorf("len(Articles) = %d, want 1", len(links.Articles))
	}
}

func TestLinkIncident_ValidatesWithNightOwl(t *testing.T) {
	store := newMockLinkStore()
	no := newMockNightOwl()
	bo := newMockBookOwl()

	metaID := uuid.New()
	incidentID := uuid.New()
	store.ticketMetaIDs[42] = metaID

	no.incidents[incidentID.String()] = &nightowl.Incident{
		ID:   incidentID.String(),
		Slug: "api-outage",
	}

	svc := link.NewService(store, no, bo, testLogger())

	linked, err := svc.LinkIncident(context.Background(), 42, incidentID.String(), uuid.New())
	if err != nil {
		t.Fatalf("LinkIncident: unexpected error: %v", err)
	}
	if linked.IncidentSlug != "api-outage" {
		t.Errorf("IncidentSlug = %q, want %q", linked.IncidentSlug, "api-outage")
	}
	if linked.IncidentID != incidentID {
		t.Errorf("IncidentID = %v, want %v", linked.IncidentID, incidentID)
	}
}

func TestLinkIncident_RejectsUnknownIncident(t *testing.T) {
	store := newMockLinkStore()
	no := newMockNightOwl()
	bo := newMockBookOwl()

	metaID := uuid.New()
	store.ticketMetaIDs[42] = metaID

	svc := link.NewService(store, no, bo, testLogger())

	_, err := svc.LinkIncident(context.Background(), 42, uuid.New().String(), uuid.New())
	if err == nil {
		t.Fatal("LinkIncident: expected error for unknown incident")
	}
	if !nightowl.IsNotFound(err) {
		t.Errorf("expected NightOwl NotFound, got: %v", err)
	}
}

func TestLinkIncident_RejectsUnknownTicket(t *testing.T) {
	store := newMockLinkStore()
	no := newMockNightOwl()
	bo := newMockBookOwl()

	incidentID := uuid.New()
	no.incidents[incidentID.String()] = &nightowl.Incident{
		ID:   incidentID.String(),
		Slug: "inc-1",
	}

	svc := link.NewService(store, no, bo, testLogger())

	// No ticket_meta for zammad_id 999.
	_, err := svc.LinkIncident(context.Background(), 999, incidentID.String(), uuid.New())
	if err == nil {
		t.Fatal("LinkIncident: expected error for unknown ticket")
	}
}

func TestUnlinkIncident(t *testing.T) {
	store := newMockLinkStore()
	no := newMockNightOwl()
	bo := newMockBookOwl()

	metaID := uuid.New()
	incidentID := uuid.New()
	store.ticketMetaIDs[42] = metaID
	store.incidentLinks[metaID] = []link.IncidentLink{
		{ID: uuid.New(), TicketMetaID: metaID, IncidentID: incidentID, IncidentSlug: "inc-1"},
	}

	svc := link.NewService(store, no, bo, testLogger())

	err := svc.UnlinkIncident(context.Background(), 42, incidentID.String())
	if err != nil {
		t.Fatalf("UnlinkIncident: unexpected error: %v", err)
	}

	if len(store.incidentLinks[metaID]) != 0 {
		t.Errorf("incident links remaining = %d, want 0", len(store.incidentLinks[metaID]))
	}
}

func TestLinkArticle_ValidatesWithBookOwl(t *testing.T) {
	store := newMockLinkStore()
	no := newMockNightOwl()
	bo := newMockBookOwl()

	metaID := uuid.New()
	articleID := uuid.New()
	store.ticketMetaIDs[42] = metaID

	bo.articles[articleID.String()] = &bookowl.Article{
		ID:    articleID.String(),
		Slug:  "restart-guide",
		Title: "Restart Guide",
	}

	svc := link.NewService(store, no, bo, testLogger())

	linked, err := svc.LinkArticle(context.Background(), 42, articleID.String(), uuid.New())
	if err != nil {
		t.Fatalf("LinkArticle: unexpected error: %v", err)
	}
	if linked.ArticleSlug != "restart-guide" {
		t.Errorf("ArticleSlug = %q, want %q", linked.ArticleSlug, "restart-guide")
	}
	if linked.ArticleTitle != "Restart Guide" {
		t.Errorf("ArticleTitle = %q, want %q", linked.ArticleTitle, "Restart Guide")
	}
}

func TestLinkArticle_RejectsUnknownArticle(t *testing.T) {
	store := newMockLinkStore()
	no := newMockNightOwl()
	bo := newMockBookOwl()

	metaID := uuid.New()
	store.ticketMetaIDs[42] = metaID

	svc := link.NewService(store, no, bo, testLogger())

	_, err := svc.LinkArticle(context.Background(), 42, uuid.New().String(), uuid.New())
	if err == nil {
		t.Fatal("LinkArticle: expected error for unknown article")
	}
	if !bookowl.IsNotFound(err) {
		t.Errorf("expected BookOwl NotFound, got: %v", err)
	}
}

func TestUnlinkArticle(t *testing.T) {
	store := newMockLinkStore()
	no := newMockNightOwl()
	bo := newMockBookOwl()

	metaID := uuid.New()
	articleID := uuid.New()
	store.ticketMetaIDs[42] = metaID
	store.articleLinks[metaID] = []link.ArticleLink{
		{ID: uuid.New(), TicketMetaID: metaID, ArticleID: articleID, ArticleTitle: "Guide"},
	}

	svc := link.NewService(store, no, bo, testLogger())

	err := svc.UnlinkArticle(context.Background(), 42, articleID.String())
	if err != nil {
		t.Fatalf("UnlinkArticle: unexpected error: %v", err)
	}

	if len(store.articleLinks[metaID]) != 0 {
		t.Errorf("article links remaining = %d, want 0", len(store.articleLinks[metaID]))
	}
}

func TestUnlinkArticle_NotFound(t *testing.T) {
	store := newMockLinkStore()
	no := newMockNightOwl()
	bo := newMockBookOwl()

	metaID := uuid.New()
	store.ticketMetaIDs[42] = metaID

	svc := link.NewService(store, no, bo, testLogger())

	err := svc.UnlinkArticle(context.Background(), 42, uuid.New().String())
	if err == nil {
		t.Fatal("UnlinkArticle: expected error for missing link")
	}
}
