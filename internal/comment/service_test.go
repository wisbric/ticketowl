package comment_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/wisbric/ticketowl/internal/comment"
	"github.com/wisbric/ticketowl/internal/zammad"
)

// --- Mock Zammad Client ---

type mockZammad struct {
	articles map[int][]zammad.Article
	created  []zammad.Article
}

func newMockZammad() *mockZammad {
	return &mockZammad{articles: make(map[int][]zammad.Article)}
}

func (m *mockZammad) ListArticles(_ context.Context, ticketID int) ([]zammad.Article, error) {
	arts, ok := m.articles[ticketID]
	if !ok {
		return []zammad.Article{}, nil
	}
	return arts, nil
}

func (m *mockZammad) CreateArticle(_ context.Context, req zammad.ArticleCreate) (*zammad.Article, error) {
	a := zammad.Article{
		ID:          100 + len(m.created),
		TicketID:    req.TicketID,
		Type:        "web",
		Sender:      "Agent",
		Body:        req.Body,
		ContentType: req.ContentType,
		Internal:    req.Internal,
		CreatedBy:   "agent@example.com",
		CreatedAt:   time.Now(),
	}
	m.created = append(m.created, a)
	m.articles[req.TicketID] = append(m.articles[req.TicketID], a)
	return &a, nil
}

// --- Mock Note Store ---

type mockNoteStore struct {
	notes map[uuid.UUID][]comment.InternalNote
	metas map[int]uuid.UUID // zammadID → metaID
}

func newMockNoteStore() *mockNoteStore {
	return &mockNoteStore{
		notes: make(map[uuid.UUID][]comment.InternalNote),
		metas: make(map[int]uuid.UUID),
	}
}

func (m *mockNoteStore) ListByTicketMetaID(_ context.Context, ticketMetaID uuid.UUID) ([]comment.InternalNote, error) {
	return m.notes[ticketMetaID], nil
}

func (m *mockNoteStore) Create(_ context.Context, note *comment.InternalNote) error {
	m.notes[note.TicketMetaID] = append(m.notes[note.TicketMetaID], *note)
	return nil
}

func (m *mockNoteStore) GetTicketMetaID(_ context.Context, zammadID int) (uuid.UUID, error) {
	id, ok := m.metas[zammadID]
	if !ok {
		return uuid.Nil, pgx.ErrNoRows
	}
	return id, nil
}

func testLogger() *slog.Logger {
	return slog.Default()
}

// --- Tests ---

func TestListThread_MergesAndSorts(t *testing.T) {
	z := newMockZammad()
	store := newMockNoteStore()

	metaID := uuid.New()
	store.metas[42] = metaID

	// Zammad article at t+10min.
	z.articles[42] = []zammad.Article{
		{
			ID:        1,
			TicketID:  42,
			Type:      "web",
			Sender:    "Customer",
			Body:      "Help me!",
			Internal:  false,
			CreatedBy: "customer@example.com",
			CreatedAt: time.Date(2026, 2, 25, 10, 10, 0, 0, time.UTC),
		},
		{
			ID:        3,
			TicketID:  42,
			Type:      "note",
			Sender:    "Agent",
			Body:      "Zammad internal note",
			Internal:  true,
			CreatedBy: "agent@example.com",
			CreatedAt: time.Date(2026, 2, 25, 10, 30, 0, 0, time.UTC),
		},
	}

	// Internal note at t+20min (between the two Zammad articles).
	store.notes[metaID] = []comment.InternalNote{
		{
			ID:           uuid.New(),
			TicketMetaID: metaID,
			AuthorID:     uuid.New(),
			AuthorName:   "agent@example.com",
			Body:         "TicketOwl internal note",
			CreatedAt:    time.Date(2026, 2, 25, 10, 20, 0, 0, time.UTC),
		},
	}

	svc := comment.NewService(z, store, testLogger())
	thread, err := svc.ListThread(context.Background(), 42)
	if err != nil {
		t.Fatalf("ListThread: unexpected error: %v", err)
	}

	if len(thread) != 3 {
		t.Fatalf("len(thread) = %d, want 3", len(thread))
	}

	// Verify chronological order.
	if thread[0].Source != "zammad" || thread[0].Body != "Help me!" {
		t.Errorf("thread[0]: want zammad 'Help me!', got source=%q body=%q", thread[0].Source, thread[0].Body)
	}
	if thread[1].Source != "internal" || thread[1].Body != "TicketOwl internal note" {
		t.Errorf("thread[1]: want internal 'TicketOwl internal note', got source=%q body=%q", thread[1].Source, thread[1].Body)
	}
	if thread[2].Source != "zammad" || thread[2].Body != "Zammad internal note" {
		t.Errorf("thread[2]: want zammad 'Zammad internal note', got source=%q body=%q", thread[2].Source, thread[2].Body)
	}
}

func TestListThread_NoMetadata(t *testing.T) {
	z := newMockZammad()
	store := newMockNoteStore()

	// Ticket 99 has no metadata in TicketOwl.
	z.articles[99] = []zammad.Article{
		{
			ID:        1,
			TicketID:  99,
			Type:      "web",
			Sender:    "Customer",
			Body:      "First message",
			CreatedBy: "customer@example.com",
			CreatedAt: time.Now(),
		},
	}

	svc := comment.NewService(z, store, testLogger())
	thread, err := svc.ListThread(context.Background(), 99)
	if err != nil {
		t.Fatalf("ListThread: unexpected error: %v", err)
	}

	if len(thread) != 1 {
		t.Fatalf("len(thread) = %d, want 1", len(thread))
	}
	if thread[0].Source != "zammad" {
		t.Errorf("source = %q, want zammad", thread[0].Source)
	}
}

func TestListThread_EmptyThread(t *testing.T) {
	z := newMockZammad()
	store := newMockNoteStore()

	svc := comment.NewService(z, store, testLogger())
	thread, err := svc.ListThread(context.Background(), 999)
	if err != nil {
		t.Fatalf("ListThread: unexpected error: %v", err)
	}

	if len(thread) != 0 {
		t.Errorf("len(thread) = %d, want 0", len(thread))
	}
}

func TestAddPublicReply(t *testing.T) {
	z := newMockZammad()
	store := newMockNoteStore()

	svc := comment.NewService(z, store, testLogger())
	entry, err := svc.AddPublicReply(context.Background(), 42, comment.AddReplyRequest{
		Body: "Here is the fix",
	})
	if err != nil {
		t.Fatalf("AddPublicReply: unexpected error: %v", err)
	}

	if entry.Source != "zammad" {
		t.Errorf("source = %q, want zammad", entry.Source)
	}
	if entry.Body != "Here is the fix" {
		t.Errorf("body = %q, want %q", entry.Body, "Here is the fix")
	}
	if entry.Internal {
		t.Error("expected public reply (internal=false)")
	}

	// Verify it was created in mock Zammad.
	if len(z.created) != 1 {
		t.Fatalf("expected 1 created article, got %d", len(z.created))
	}
	if z.created[0].Internal {
		t.Error("zammad article should not be internal")
	}
}

func TestAddInternalNote(t *testing.T) {
	z := newMockZammad()
	store := newMockNoteStore()

	metaID := uuid.New()
	store.metas[42] = metaID

	authorID := uuid.New()
	svc := comment.NewService(z, store, testLogger())
	entry, err := svc.AddInternalNote(context.Background(), 42, authorID, "agent@example.com", comment.AddNoteRequest{
		Body: "Internal observation",
	})
	if err != nil {
		t.Fatalf("AddInternalNote: unexpected error: %v", err)
	}

	if entry.Source != "internal" {
		t.Errorf("source = %q, want internal", entry.Source)
	}
	if entry.Body != "Internal observation" {
		t.Errorf("body = %q, want %q", entry.Body, "Internal observation")
	}
	if !entry.Internal {
		t.Error("expected internal note (internal=true)")
	}
	if entry.Type != "internal_note" {
		t.Errorf("type = %q, want internal_note", entry.Type)
	}

	// Verify stored in mock store.
	notes := store.notes[metaID]
	if len(notes) != 1 {
		t.Fatalf("expected 1 stored note, got %d", len(notes))
	}
	if notes[0].AuthorID != authorID {
		t.Errorf("author_id = %v, want %v", notes[0].AuthorID, authorID)
	}
}

func TestAddInternalNote_NoMetadata(t *testing.T) {
	z := newMockZammad()
	store := newMockNoteStore()

	// Ticket 99 has no metadata — should error.
	svc := comment.NewService(z, store, testLogger())
	_, err := svc.AddInternalNote(context.Background(), 99, uuid.New(), "agent", comment.AddNoteRequest{
		Body: "Note on untracked ticket",
	})
	if err == nil {
		t.Fatal("expected error for untracked ticket")
	}
}

func TestListThread_InternalNotesMarkedInternal(t *testing.T) {
	z := newMockZammad()
	store := newMockNoteStore()

	metaID := uuid.New()
	store.metas[42] = metaID

	z.articles[42] = []zammad.Article{
		{
			ID: 1, TicketID: 42, Type: "web", Sender: "Customer",
			Body: "Public", Internal: false, CreatedAt: time.Now(),
		},
	}
	store.notes[metaID] = []comment.InternalNote{
		{
			ID: uuid.New(), TicketMetaID: metaID, AuthorName: "agent",
			Body: "Secret", CreatedAt: time.Now().Add(time.Minute),
		},
	}

	svc := comment.NewService(z, store, testLogger())
	thread, err := svc.ListThread(context.Background(), 42)
	if err != nil {
		t.Fatalf("ListThread: unexpected error: %v", err)
	}

	var internalCount int
	for _, e := range thread {
		if e.Internal {
			internalCount++
		}
	}
	if internalCount != 1 {
		t.Errorf("expected 1 internal entry, got %d", internalCount)
	}
}

func TestThreadEntryFields(t *testing.T) {
	z := newMockZammad()
	store := newMockNoteStore()

	metaID := uuid.New()
	store.metas[42] = metaID
	noteID := uuid.New()

	z.articles[42] = []zammad.Article{
		{
			ID: 5, TicketID: 42, Type: "email", Sender: "Agent",
			Body: "Email reply", ContentType: "text/html", Internal: false,
			CreatedBy: "agent@example.com",
			CreatedAt: time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC),
		},
	}
	store.notes[metaID] = []comment.InternalNote{
		{
			ID: noteID, TicketMetaID: metaID,
			AuthorID: uuid.New(), AuthorName: "admin@example.com",
			Body:      "Check KB",
			CreatedAt: time.Date(2026, 2, 25, 10, 5, 0, 0, time.UTC),
		},
	}

	svc := comment.NewService(z, store, testLogger())
	thread, err := svc.ListThread(context.Background(), 42)
	if err != nil {
		t.Fatalf("ListThread: unexpected error: %v", err)
	}

	if len(thread) != 2 {
		t.Fatalf("len(thread) = %d, want 2", len(thread))
	}

	// Zammad entry.
	ze := thread[0]
	if ze.ID != "5" {
		t.Errorf("zammad ID = %q, want %q", ze.ID, "5")
	}
	if ze.Type != "email" {
		t.Errorf("zammad type = %q, want email", ze.Type)
	}
	if ze.Sender != "Agent" {
		t.Errorf("zammad sender = %q, want Agent", ze.Sender)
	}

	// Internal entry.
	ie := thread[1]
	if ie.ID != noteID.String() {
		t.Errorf("internal ID = %q, want %q", ie.ID, noteID.String())
	}
	if ie.Type != "internal_note" {
		t.Errorf("internal type = %q, want internal_note", ie.Type)
	}
	if ie.CreatedBy != "admin@example.com" {
		t.Errorf("internal created_by = %q, want admin@example.com", ie.CreatedBy)
	}
}
