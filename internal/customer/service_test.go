package customer_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/wisbric/ticketowl/internal/customer"
	"github.com/wisbric/ticketowl/internal/link"
	"github.com/wisbric/ticketowl/internal/sla"
	"github.com/wisbric/ticketowl/internal/zammad"
)

// --- Mock CustomerStore ---

type mockStore struct {
	orgs         map[uuid.UUID]*customer.Org
	ticketMetas  map[int]uuid.UUID
	articleLinks map[uuid.UUID][]link.ArticleLink
	slaStates    map[uuid.UUID]*sla.State
}

func newMockStore() *mockStore {
	return &mockStore{
		orgs:         make(map[uuid.UUID]*customer.Org),
		ticketMetas:  make(map[int]uuid.UUID),
		articleLinks: make(map[uuid.UUID][]link.ArticleLink),
		slaStates:    make(map[uuid.UUID]*sla.State),
	}
}

func (m *mockStore) GetOrg(_ context.Context, orgID uuid.UUID) (*customer.Org, error) {
	org, ok := m.orgs[orgID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return org, nil
}

func (m *mockStore) GetTicketMetaID(_ context.Context, zammadID int) (uuid.UUID, error) {
	id, ok := m.ticketMetas[zammadID]
	if !ok {
		return uuid.UUID{}, pgx.ErrNoRows
	}
	return id, nil
}

func (m *mockStore) ListArticleLinks(_ context.Context, ticketMetaID uuid.UUID) ([]link.ArticleLink, error) {
	return m.articleLinks[ticketMetaID], nil
}

func (m *mockStore) GetSLAState(_ context.Context, ticketMetaID uuid.UUID) (*sla.State, error) {
	st, ok := m.slaStates[ticketMetaID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return st, nil
}

// --- Mock ZammadClient ---

type mockZammad struct {
	tickets  map[int]*zammad.Ticket
	articles map[int][]zammad.Article
	created  []zammad.ArticleCreate
}

func newMockZammad() *mockZammad {
	return &mockZammad{
		tickets:  make(map[int]*zammad.Ticket),
		articles: make(map[int][]zammad.Article),
	}
}

func (m *mockZammad) ListTickets(_ context.Context, opts zammad.ListTicketsOptions) ([]zammad.Ticket, error) {
	var result []zammad.Ticket
	for _, t := range m.tickets {
		if opts.OrgID != nil && t.OrganisationID != *opts.OrgID {
			continue
		}
		result = append(result, *t)
	}
	return result, nil
}

func (m *mockZammad) GetTicket(_ context.Context, id int) (*zammad.Ticket, error) {
	t, ok := m.tickets[id]
	if !ok {
		return nil, &zammad.ZammadError{StatusCode: 404, Message: "not found"}
	}
	return t, nil
}

func (m *mockZammad) ListArticles(_ context.Context, ticketID int) ([]zammad.Article, error) {
	return m.articles[ticketID], nil
}

func (m *mockZammad) CreateArticle(_ context.Context, req zammad.ArticleCreate) (*zammad.Article, error) {
	m.created = append(m.created, req)
	return &zammad.Article{
		ID:        100 + len(m.created),
		TicketID:  req.TicketID,
		Body:      req.Body,
		Sender:    "Customer",
		Internal:  false,
		CreatedAt: time.Now(),
	}, nil
}

// --- Tests ---

func TestListMyTickets_OrgScoped(t *testing.T) {
	store := newMockStore()
	z := newMockZammad()

	orgID := uuid.New()
	zammadOrgID := 10
	store.orgs[orgID] = &customer.Org{
		ID:          orgID,
		Name:        "Acme Corp",
		ZammadOrgID: &zammadOrgID,
	}

	// Two tickets: one belongs to org 10, the other to org 20.
	z.tickets[1] = &zammad.Ticket{ID: 1, Number: "T1", Title: "Org 10 ticket", State: "open", OrganisationID: 10}
	z.tickets[2] = &zammad.Ticket{ID: 2, Number: "T2", Title: "Org 20 ticket", State: "open", OrganisationID: 20}

	svc := customer.NewService(store, z, slog.Default())

	tickets, err := svc.ListMyTickets(context.Background(), orgID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tickets) != 1 {
		t.Fatalf("len = %d, want 1 (only org 10 tickets)", len(tickets))
	}
	if tickets[0].ID != 1 {
		t.Errorf("ticket ID = %d, want 1", tickets[0].ID)
	}
}

func TestListMyTickets_OrgNotFound(t *testing.T) {
	store := newMockStore()
	z := newMockZammad()

	svc := customer.NewService(store, z, slog.Default())

	_, err := svc.ListMyTickets(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for unknown org")
	}
}

func TestGetMyTicket_OrgScoping(t *testing.T) {
	store := newMockStore()
	z := newMockZammad()

	orgID := uuid.New()
	zammadOrgID := 10
	store.orgs[orgID] = &customer.Org{
		ID:          orgID,
		Name:        "Acme Corp",
		ZammadOrgID: &zammadOrgID,
	}

	z.tickets[1] = &zammad.Ticket{ID: 1, Number: "T1", Title: "My ticket", State: "open", OrganisationID: 10}
	z.articles[1] = []zammad.Article{
		{ID: 100, TicketID: 1, Body: "Public comment", Sender: "Agent", Internal: false, CreatedAt: time.Now()},
		{ID: 101, TicketID: 1, Body: "Secret note", Sender: "Agent", Internal: true, CreatedAt: time.Now()},
	}

	svc := customer.NewService(store, z, slog.Default())

	detail, err := svc.GetMyTicket(context.Background(), orgID, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.ID != 1 {
		t.Errorf("ticket ID = %d, want 1", detail.ID)
	}
	// Only public comments should be included.
	if len(detail.Comments) != 1 {
		t.Fatalf("comments len = %d, want 1 (internal notes excluded)", len(detail.Comments))
	}
	if detail.Comments[0].Body != "Public comment" {
		t.Errorf("comment body = %q, want %q", detail.Comments[0].Body, "Public comment")
	}
}

func TestGetMyTicket_ForbiddenForOtherOrg(t *testing.T) {
	store := newMockStore()
	z := newMockZammad()

	orgID := uuid.New()
	zammadOrgID := 10
	store.orgs[orgID] = &customer.Org{
		ID:          orgID,
		Name:        "Acme Corp",
		ZammadOrgID: &zammadOrgID,
	}

	// Ticket belongs to org 20, not org 10.
	z.tickets[1] = &zammad.Ticket{ID: 1, Number: "T1", Title: "Other org ticket", State: "open", OrganisationID: 20}

	svc := customer.NewService(store, z, slog.Default())

	_, err := svc.GetMyTicket(context.Background(), orgID, 1)
	if err == nil {
		t.Fatal("expected forbidden error")
	}
	if !errors.Is(err, customer.ErrForbidden) {
		t.Errorf("expected ErrForbidden in error chain, got: %v", err)
	}
}

func TestAddReply_Success(t *testing.T) {
	store := newMockStore()
	z := newMockZammad()

	orgID := uuid.New()
	zammadOrgID := 10
	store.orgs[orgID] = &customer.Org{
		ID:          orgID,
		Name:        "Acme Corp",
		ZammadOrgID: &zammadOrgID,
	}
	z.tickets[1] = &zammad.Ticket{ID: 1, Number: "T1", Title: "My ticket", State: "open", OrganisationID: 10}

	svc := customer.NewService(store, z, slog.Default())

	err := svc.AddReply(context.Background(), orgID, 1, "Thanks for the update!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(z.created) != 1 {
		t.Fatalf("expected 1 article created, got %d", len(z.created))
	}
	if z.created[0].Internal {
		t.Error("reply should not be internal")
	}
}

func TestAddReply_ForbiddenForOtherOrg(t *testing.T) {
	store := newMockStore()
	z := newMockZammad()

	orgID := uuid.New()
	zammadOrgID := 10
	store.orgs[orgID] = &customer.Org{
		ID:          orgID,
		Name:        "Acme Corp",
		ZammadOrgID: &zammadOrgID,
	}
	z.tickets[1] = &zammad.Ticket{ID: 1, Number: "T1", Title: "Other org", State: "open", OrganisationID: 20}

	svc := customer.NewService(store, z, slog.Default())

	err := svc.AddReply(context.Background(), orgID, 1, "Should fail")
	if err == nil {
		t.Fatal("expected forbidden error")
	}
}

func TestGetLinkedArticles_OrgScoped(t *testing.T) {
	store := newMockStore()
	z := newMockZammad()

	orgID := uuid.New()
	zammadOrgID := 10
	store.orgs[orgID] = &customer.Org{
		ID:          orgID,
		Name:        "Acme Corp",
		ZammadOrgID: &zammadOrgID,
	}
	z.tickets[1] = &zammad.Ticket{ID: 1, Number: "T1", Title: "My ticket", State: "open", OrganisationID: 10}

	metaID := uuid.New()
	store.ticketMetas[1] = metaID
	store.articleLinks[metaID] = []link.ArticleLink{
		{ArticleID: uuid.New(), ArticleSlug: "kb-setup", ArticleTitle: "Setup Guide"},
	}

	svc := customer.NewService(store, z, slog.Default())

	articles, err := svc.GetLinkedArticles(context.Background(), orgID, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(articles) != 1 {
		t.Fatalf("len = %d, want 1", len(articles))
	}
	if articles[0].Title != "Setup Guide" {
		t.Errorf("title = %q, want %q", articles[0].Title, "Setup Guide")
	}
}
