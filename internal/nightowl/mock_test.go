package nightowl_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/wisbric/ticketowl/internal/nightowl"
)

// call records a single HTTP call made to the mock server.
type call struct {
	Method string
	Path   string
}

// MockServer is an httptest-based mock NightOwl API server for unit tests.
type MockServer struct {
	t         *testing.T
	server    *httptest.Server
	mu        sync.Mutex
	incidents map[string]nightowl.Incident
	oncall    map[string]nightowl.OnCallInfo // service → oncall
	alerts    []nightowl.Alert
	calls     []call
	nextAlert int
}

// NewMockServer creates a new mock NightOwl server and registers cleanup with t.
func NewMockServer(t *testing.T) *MockServer {
	t.Helper()

	m := &MockServer{
		t:         t,
		incidents: make(map[string]nightowl.Incident),
		oncall:    make(map[string]nightowl.OnCallInfo),
		nextAlert: 1,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/incidents/{id}", m.handleGetIncident)
	mux.HandleFunc("GET /api/v1/incidents", m.handleListIncidents)
	mux.HandleFunc("GET /api/v1/oncall/{service}", m.handleGetOnCall)
	mux.HandleFunc("POST /api/v1/alerts", m.handleCreateAlert)

	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
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

// Client returns a NightOwl client configured to talk to the mock server.
func (m *MockServer) Client() *nightowl.Client {
	return nightowl.New(m.server.URL, "mock-test-key")
}

// AddIncident adds an incident to the mock server's in-memory store.
func (m *MockServer) AddIncident(i nightowl.Incident) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.incidents[i.ID] = i
}

// AddOnCall sets the on-call info for a service.
func (m *MockServer) AddOnCall(service string, info nightowl.OnCallInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.oncall[service] = info
}

// Alerts returns all alerts that were created.
func (m *MockServer) Alerts() []nightowl.Alert {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]nightowl.Alert, len(m.alerts))
	copy(out, m.alerts)
	return out
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

func (m *MockServer) handleGetIncident(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	m.mu.Lock()
	incident, ok := m.incidents[id]
	m.mu.Unlock()

	if !ok {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, incident)
}

func (m *MockServer) handleListIncidents(w http.ResponseWriter, _ *http.Request) {
	m.mu.Lock()
	incidents := make([]nightowl.Incident, 0, len(m.incidents))
	for _, i := range m.incidents {
		incidents = append(incidents, i)
	}
	m.mu.Unlock()

	respondJSON(w, http.StatusOK, incidents)
}

func (m *MockServer) handleGetOnCall(w http.ResponseWriter, r *http.Request) {
	service := r.PathValue("service")

	m.mu.Lock()
	info, ok := m.oncall[service]
	m.mu.Unlock()

	if !ok {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, info)
}

func (m *MockServer) handleCreateAlert(w http.ResponseWriter, r *http.Request) {
	var req nightowl.CreateAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	id := m.nextAlert
	m.nextAlert++
	alert := nightowl.Alert{
		ID:       fmt.Sprintf("alert-%d", id),
		Name:     req.Name,
		Severity: req.Severity,
		Status:   "firing",
	}
	m.alerts = append(m.alerts, alert)
	m.mu.Unlock()

	respondJSON(w, http.StatusCreated, alert)
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
