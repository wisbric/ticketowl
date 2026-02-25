package zammad_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/wisbric/ticketowl/internal/zammad"
)

// call records a single HTTP call made to the mock server.
type call struct {
	Method string
	Path   string
}

// MockServer is an httptest-based mock Zammad API server for unit tests.
type MockServer struct {
	t             *testing.T
	server        *httptest.Server
	mu            sync.Mutex
	tickets       map[int]zammad.Ticket
	articles      map[int][]zammad.Article // ticketID → articles
	users         map[int]zammad.User
	orgs          map[int]zammad.Organisation
	states        []zammad.TicketState
	priorities    []zammad.TicketPriority
	calls         []call
	nextTicketID  int
	nextArticleID int
}

// NewMockServer creates a new mock Zammad server and registers cleanup with t.
func NewMockServer(t *testing.T) *MockServer {
	t.Helper()

	m := &MockServer{
		t:             t,
		tickets:       make(map[int]zammad.Ticket),
		articles:      make(map[int][]zammad.Article),
		users:         make(map[int]zammad.User),
		orgs:          make(map[int]zammad.Organisation),
		nextTicketID:  100,
		nextArticleID: 1000,
		states: []zammad.TicketState{
			{ID: 1, Name: "new"},
			{ID: 2, Name: "open"},
			{ID: 3, Name: "pending reminder"},
			{ID: 5, Name: "closed"},
		},
		priorities: []zammad.TicketPriority{
			{ID: 1, Name: "1 low"},
			{ID: 2, Name: "2 normal"},
			{ID: 3, Name: "3 high"},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/tickets", m.handleListTickets)
	mux.HandleFunc("GET /api/v1/tickets/{id}", m.handleGetTicket)
	mux.HandleFunc("POST /api/v1/tickets", m.handleCreateTicket)
	mux.HandleFunc("PUT /api/v1/tickets/{id}", m.handleUpdateTicket)
	mux.HandleFunc("GET /api/v1/tickets/search", m.handleSearchTickets)
	mux.HandleFunc("GET /api/v1/ticket_articles/by_ticket/{id}", m.handleListArticles)
	mux.HandleFunc("POST /api/v1/ticket_articles", m.handleCreateArticle)
	mux.HandleFunc("GET /api/v1/users/{id}", m.handleGetUser)
	mux.HandleFunc("GET /api/v1/users/me", m.handleGetCurrentUser)
	mux.HandleFunc("GET /api/v1/organizations/{id}", m.handleGetOrganisation)
	mux.HandleFunc("GET /api/v1/ticket_states", m.handleListStates)
	mux.HandleFunc("GET /api/v1/ticket_priorities", m.handleListPriorities)

	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate auth header.
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Token token=") || auth == "Token token=" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		m.recordCall(r.Method, r.URL.Path)
		mux.ServeHTTP(w, r)
	}))

	t.Cleanup(m.server.Close)

	return m
}

// URL returns the base URL of the mock server.
func (m *MockServer) URL() string {
	return m.server.URL
}

// Client returns a Zammad client configured to talk to the mock server.
func (m *MockServer) Client() *zammad.Client {
	return zammad.New(m.server.URL, "mock-test-token")
}

// AddTicket adds a ticket to the mock server's in-memory store.
func (m *MockServer) AddTicket(t zammad.Ticket) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t.ID == 0 {
		t.ID = m.nextTicketID
		m.nextTicketID++
	}
	m.tickets[t.ID] = t
}

// AddArticle adds an article to the mock server's in-memory store.
func (m *MockServer) AddArticle(ticketID int, a zammad.Article) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a.ID == 0 {
		a.ID = m.nextArticleID
		m.nextArticleID++
	}
	a.TicketID = ticketID
	m.articles[ticketID] = append(m.articles[ticketID], a)
}

// AddUser adds a user to the mock server's in-memory store.
func (m *MockServer) AddUser(u zammad.User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[u.ID] = u
}

// AddOrganisation adds an organisation to the mock server's in-memory store.
func (m *MockServer) AddOrganisation(o zammad.Organisation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.orgs[o.ID] = o
}

// RequireCall asserts that a specific method+path was called at least once.
func (m *MockServer) RequireCall(t *testing.T, method, path string) {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, c := range m.calls {
		if c.Method == method && c.Path == path {
			return
		}
	}
	t.Errorf("expected call %s %s, but it was not made. Calls: %v", method, path, m.calls)
}

// RequireNoCall asserts that a specific method+path was never called.
func (m *MockServer) RequireNoCall(t *testing.T, method, path string) {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, c := range m.calls {
		if c.Method == method && c.Path == path {
			t.Errorf("expected no call %s %s, but it was made", method, path)
			return
		}
	}
}

// CallCount returns how many times a specific method+path was called.
func (m *MockServer) CallCount(method, path string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for _, c := range m.calls {
		if c.Method == method && c.Path == path {
			count++
		}
	}
	return count
}

func (m *MockServer) recordCall(method, path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, call{Method: method, Path: path})
}

func (m *MockServer) handleListTickets(w http.ResponseWriter, _ *http.Request) {
	m.mu.Lock()
	tickets := make([]zammad.Ticket, 0, len(m.tickets))
	for _, t := range m.tickets {
		tickets = append(tickets, t)
	}
	m.mu.Unlock()

	respondJSON(w, http.StatusOK, tickets)
}

func (m *MockServer) handleGetTicket(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	ticket, ok := m.tickets[id]
	m.mu.Unlock()

	if !ok {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, ticket)
}

func (m *MockServer) handleCreateTicket(w http.ResponseWriter, r *http.Request) {
	var req zammad.TicketCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	id := m.nextTicketID
	m.nextTicketID++
	ticket := zammad.Ticket{
		ID:         id,
		Number:     fmt.Sprintf("%d", 10000+id),
		Title:      req.Title,
		GroupID:    req.GroupID,
		CustomerID: req.CustomerID,
		StateID:    req.StateID,
		PriorityID: req.PriorityID,
	}
	if ticket.StateID == 0 {
		ticket.StateID = 1
		ticket.State = "new"
	}
	if ticket.PriorityID == 0 {
		ticket.PriorityID = 2
		ticket.Priority = "2 normal"
	}
	m.tickets[id] = ticket
	m.mu.Unlock()

	respondJSON(w, http.StatusCreated, ticket)
}

func (m *MockServer) handleUpdateTicket(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	ticket, ok := m.tickets[id]
	if !ok {
		m.mu.Unlock()
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	var req zammad.TicketUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.mu.Unlock()
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}

	if req.StateID != nil {
		ticket.StateID = *req.StateID
	}
	if req.PriorityID != nil {
		ticket.PriorityID = *req.PriorityID
	}
	if req.OwnerID != nil {
		ticket.OwnerID = *req.OwnerID
	}
	if req.GroupID != nil {
		ticket.GroupID = *req.GroupID
	}
	m.tickets[id] = ticket
	m.mu.Unlock()

	respondJSON(w, http.StatusOK, ticket)
}

func (m *MockServer) handleSearchTickets(w http.ResponseWriter, _ *http.Request) {
	// Simple: return all tickets (search filtering is Zammad's job).
	m.mu.Lock()
	tickets := make([]zammad.Ticket, 0, len(m.tickets))
	for _, t := range m.tickets {
		tickets = append(tickets, t)
	}
	m.mu.Unlock()

	respondJSON(w, http.StatusOK, tickets)
}

func (m *MockServer) handleListArticles(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	articles := m.articles[id]
	m.mu.Unlock()

	if articles == nil {
		articles = []zammad.Article{}
	}

	respondJSON(w, http.StatusOK, articles)
}

func (m *MockServer) handleCreateArticle(w http.ResponseWriter, r *http.Request) {
	var req zammad.ArticleCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	id := m.nextArticleID
	m.nextArticleID++
	article := zammad.Article{
		ID:          id,
		TicketID:    req.TicketID,
		Body:        req.Body,
		ContentType: req.ContentType,
		Internal:    req.Internal,
	}
	m.articles[req.TicketID] = append(m.articles[req.TicketID], article)
	m.mu.Unlock()

	respondJSON(w, http.StatusCreated, article)
}

func (m *MockServer) handleGetUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	user, ok := m.users[id]
	m.mu.Unlock()

	if !ok {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, user)
}

func (m *MockServer) handleGetCurrentUser(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, zammad.User{
		ID:        1,
		Login:     "admin@example.com",
		Firstname: "Admin",
		Lastname:  "User",
		Email:     "admin@example.com",
		Active:    true,
	})
}

func (m *MockServer) handleGetOrganisation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	org, ok := m.orgs[id]
	m.mu.Unlock()

	if !ok {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, org)
}

func (m *MockServer) handleListStates(w http.ResponseWriter, _ *http.Request) {
	m.mu.Lock()
	states := m.states
	m.mu.Unlock()

	respondJSON(w, http.StatusOK, states)
}

func (m *MockServer) handleListPriorities(w http.ResponseWriter, _ *http.Request) {
	m.mu.Lock()
	priorities := m.priorities
	m.mu.Unlock()

	respondJSON(w, http.StatusOK, priorities)
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
