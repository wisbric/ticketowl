package ticket_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/wisbric/ticketowl/internal/ticket"
	"github.com/wisbric/ticketowl/internal/zammad"
)

// --- Mock Zammad Client ---

type mockZammad struct {
	tickets map[int]*zammad.Ticket
	created []zammad.Ticket
}

func newMockZammad() *mockZammad {
	return &mockZammad{tickets: make(map[int]*zammad.Ticket)}
}

func (m *mockZammad) GetTicket(_ context.Context, id int) (*zammad.Ticket, error) {
	t, ok := m.tickets[id]
	if !ok {
		return nil, &zammad.ZammadError{StatusCode: 404, Message: "not found"}
	}
	return t, nil
}

func (m *mockZammad) ListTickets(_ context.Context, _ zammad.ListTicketsOptions) ([]zammad.Ticket, error) {
	tickets := make([]zammad.Ticket, 0, len(m.tickets))
	for _, t := range m.tickets {
		tickets = append(tickets, *t)
	}
	return tickets, nil
}

func (m *mockZammad) CreateTicket(_ context.Context, req zammad.TicketCreateRequest) (*zammad.Ticket, error) {
	id := 1000 + len(m.created)
	t := zammad.Ticket{
		ID:         id,
		Number:     fmt.Sprintf("T%d", id),
		Title:      req.Title,
		GroupID:    req.GroupID,
		CustomerID: req.CustomerID,
		StateID:    req.StateID,
		PriorityID: req.PriorityID,
	}
	if t.StateID == 0 {
		t.StateID = 1
		t.State = "new"
	}
	if t.PriorityID == 0 {
		t.PriorityID = 2
		t.Priority = "2 normal"
	}
	m.created = append(m.created, t)
	m.tickets[id] = &t
	return &t, nil
}

func (m *mockZammad) UpdateTicket(_ context.Context, id int, req zammad.TicketUpdateRequest) (*zammad.Ticket, error) {
	t, ok := m.tickets[id]
	if !ok {
		return nil, &zammad.ZammadError{StatusCode: 404, Message: "not found"}
	}
	if req.StateID != nil {
		t.StateID = *req.StateID
	}
	if req.PriorityID != nil {
		t.PriorityID = *req.PriorityID
	}
	if req.OwnerID != nil {
		t.OwnerID = *req.OwnerID
	}
	if req.GroupID != nil {
		t.GroupID = *req.GroupID
	}
	return t, nil
}

func (m *mockZammad) SearchTickets(_ context.Context, _ string, _ zammad.ListTicketsOptions) ([]zammad.Ticket, error) {
	return m.ListTickets(context.Background(), zammad.ListTicketsOptions{})
}

// --- Mock Ticket Store ---

type mockTicketStore struct {
	metas map[int]*ticket.Meta
}

func newMockTicketStore() *mockTicketStore {
	return &mockTicketStore{metas: make(map[int]*ticket.Meta)}
}

func (m *mockTicketStore) GetByZammadID(_ context.Context, zammadID int) (*ticket.Meta, error) {
	meta, ok := m.metas[zammadID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return meta, nil
}

func (m *mockTicketStore) Upsert(_ context.Context, zammadID int, zammadNumber string, slaPolicyID *uuid.UUID) (*ticket.Meta, error) {
	meta := &ticket.Meta{
		ID:           uuid.New(),
		ZammadID:     zammadID,
		ZammadNumber: zammadNumber,
		SLAPolicyID:  slaPolicyID,
	}
	m.metas[zammadID] = meta
	return meta, nil
}

func (m *mockTicketStore) ListByZammadIDs(_ context.Context, zammadIDs []int) ([]ticket.Meta, error) {
	var metas []ticket.Meta
	for _, id := range zammadIDs {
		if meta, ok := m.metas[id]; ok {
			metas = append(metas, *meta)
		}
	}
	return metas, nil
}

func testLogger() *slog.Logger {
	return slog.Default()
}

// --- Tests ---

func TestGet_LiveFromZammad(t *testing.T) {
	z := newMockZammad()
	store := newMockTicketStore()

	z.tickets[42] = &zammad.Ticket{
		ID:     42,
		Number: "T42",
		Title:  "Printer on fire",
		State:  "open",
	}

	svc := ticket.NewService(z, store, testLogger())

	got, err := svc.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("Get: unexpected error: %v", err)
	}
	if got.ID != 42 {
		t.Errorf("ID = %d, want 42", got.ID)
	}
	if got.Title != "Printer on fire" {
		t.Errorf("Title = %q, want %q", got.Title, "Printer on fire")
	}
	// No metadata exists, so MetaID should be nil.
	if got.MetaID != nil {
		t.Errorf("MetaID = %v, want nil (no metadata stored)", got.MetaID)
	}
}

func TestGet_EnrichedWithMetadata(t *testing.T) {
	z := newMockZammad()
	store := newMockTicketStore()

	z.tickets[42] = &zammad.Ticket{
		ID:     42,
		Number: "T42",
		Title:  "Printer on fire",
		State:  "open",
	}

	slaPolicyID := uuid.New()
	metaID := uuid.New()
	store.metas[42] = &ticket.Meta{
		ID:          metaID,
		ZammadID:    42,
		SLAPolicyID: &slaPolicyID,
	}

	svc := ticket.NewService(z, store, testLogger())

	got, err := svc.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("Get: unexpected error: %v", err)
	}
	if got.MetaID == nil {
		t.Fatal("MetaID is nil, expected non-nil")
	}
	if *got.MetaID != metaID {
		t.Errorf("MetaID = %v, want %v", *got.MetaID, metaID)
	}
	if got.SLAPolicyID == nil || *got.SLAPolicyID != slaPolicyID {
		t.Errorf("SLAPolicyID = %v, want %v", got.SLAPolicyID, slaPolicyID)
	}
}

func TestGet_NotFound(t *testing.T) {
	z := newMockZammad()
	store := newMockTicketStore()

	svc := ticket.NewService(z, store, testLogger())

	_, err := svc.Get(context.Background(), 999)
	if err == nil {
		t.Fatal("Get: expected error for missing ticket")
	}
	if !zammad.IsNotFound(err) {
		t.Errorf("expected IsNotFound=true, got false; err=%v", err)
	}
}

func TestList_EnrichesBatch(t *testing.T) {
	z := newMockZammad()
	store := newMockTicketStore()

	z.tickets[1] = &zammad.Ticket{ID: 1, Number: "T1", Title: "First"}
	z.tickets[2] = &zammad.Ticket{ID: 2, Number: "T2", Title: "Second"}

	metaID := uuid.New()
	store.metas[1] = &ticket.Meta{ID: metaID, ZammadID: 1}

	svc := ticket.NewService(z, store, testLogger())

	got, err := svc.List(context.Background(), ticket.ListOptions{})
	if err != nil {
		t.Fatalf("List: unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}

	// One ticket should have metadata, the other should not.
	var withMeta, withoutMeta int
	for _, et := range got {
		if et.MetaID != nil {
			withMeta++
		} else {
			withoutMeta++
		}
	}
	if withMeta != 1 || withoutMeta != 1 {
		t.Errorf("withMeta=%d, withoutMeta=%d; want 1,1", withMeta, withoutMeta)
	}
}

func TestList_EmptyResult(t *testing.T) {
	z := newMockZammad()
	store := newMockTicketStore()

	svc := ticket.NewService(z, store, testLogger())

	got, err := svc.List(context.Background(), ticket.ListOptions{})
	if err != nil {
		t.Fatalf("List: unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len(got) = %d, want 0", len(got))
	}
}

func TestList_WithQuery(t *testing.T) {
	z := newMockZammad()
	store := newMockTicketStore()

	z.tickets[1] = &zammad.Ticket{ID: 1, Number: "T1", Title: "Searchable"}

	svc := ticket.NewService(z, store, testLogger())

	got, err := svc.List(context.Background(), ticket.ListOptions{Query: "searchable"})
	if err != nil {
		t.Fatalf("List: unexpected error: %v", err)
	}
	if len(got) == 0 {
		t.Error("expected at least one result from search")
	}
}

func TestCreate_StoresMetadata(t *testing.T) {
	z := newMockZammad()
	store := newMockTicketStore()

	svc := ticket.NewService(z, store, testLogger())

	got, err := svc.Create(context.Background(), ticket.CreateRequest{
		Title:   "New ticket",
		GroupID: 1,
	})
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	if got.Title != "New ticket" {
		t.Errorf("Title = %q, want %q", got.Title, "New ticket")
	}
	if got.ID == 0 {
		t.Error("expected non-zero ticket ID")
	}
	// Should have metadata stored.
	if got.MetaID == nil {
		t.Error("MetaID is nil, expected metadata to be stored after create")
	}

	// Verify stored in mock store.
	if _, ok := store.metas[got.ID]; !ok {
		t.Error("expected metadata to be stored in the store")
	}
}

func TestUpdate_ReturnsUpdated(t *testing.T) {
	z := newMockZammad()
	store := newMockTicketStore()

	z.tickets[42] = &zammad.Ticket{
		ID:         42,
		Number:     "T42",
		Title:      "Old title",
		StateID:    1,
		PriorityID: 2,
	}

	svc := ticket.NewService(z, store, testLogger())

	newState := 5
	got, err := svc.Update(context.Background(), 42, ticket.UpdateRequest{
		StateID: &newState,
	})
	if err != nil {
		t.Fatalf("Update: unexpected error: %v", err)
	}
	if got.StateID != 5 {
		t.Errorf("StateID = %d, want 5", got.StateID)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	z := newMockZammad()
	store := newMockTicketStore()

	svc := ticket.NewService(z, store, testLogger())

	newState := 5
	_, err := svc.Update(context.Background(), 999, ticket.UpdateRequest{
		StateID: &newState,
	})
	if err == nil {
		t.Fatal("Update: expected error for missing ticket")
	}
	if !zammad.IsNotFound(err) {
		t.Errorf("expected IsNotFound=true, got false; err=%v", err)
	}
}
