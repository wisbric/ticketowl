package bookowl_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/wisbric/ticketowl/internal/bookowl"
)

// call records a single HTTP call made to the mock server.
type call struct {
	Method string
	Path   string
}

// MockServer is an httptest-based mock BookOwl API server for unit tests.
type MockServer struct {
	t            *testing.T
	server       *httptest.Server
	mu           sync.Mutex
	articles     map[string]bookowl.Article
	searchResult []bookowl.ArticleSummary
	postMortems  []bookowl.PostMortem
	calls        []call
	nextPM       int
}

// NewMockServer creates a new mock BookOwl server and registers cleanup with t.
func NewMockServer(t *testing.T) *MockServer {
	t.Helper()

	m := &MockServer{
		t:        t,
		articles: make(map[string]bookowl.Article),
		nextPM:   1,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/articles/search", m.handleSearchArticles)
	mux.HandleFunc("GET /api/v1/articles/{id}", m.handleGetArticle)
	mux.HandleFunc("POST /api/v1/postmortems", m.handleCreatePostMortem)

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

// Client returns a BookOwl client configured to talk to the mock server.
func (m *MockServer) Client() *bookowl.Client {
	return bookowl.New(m.server.URL, "mock-test-key")
}

// AddArticle adds an article to the mock server's in-memory store.
func (m *MockServer) AddArticle(a bookowl.Article) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.articles[a.ID] = a
}

// SetSearchResults sets the articles returned by search.
func (m *MockServer) SetSearchResults(results []bookowl.ArticleSummary) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.searchResult = results
}

// PostMortems returns all post-mortems that were created.
func (m *MockServer) PostMortems() []bookowl.PostMortem {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]bookowl.PostMortem, len(m.postMortems))
	copy(out, m.postMortems)
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

func (m *MockServer) handleSearchArticles(w http.ResponseWriter, _ *http.Request) {
	m.mu.Lock()
	results := m.searchResult
	m.mu.Unlock()

	if results == nil {
		results = []bookowl.ArticleSummary{}
	}

	respondJSON(w, http.StatusOK, results)
}

func (m *MockServer) handleGetArticle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	m.mu.Lock()
	article, ok := m.articles[id]
	m.mu.Unlock()

	if !ok {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, article)
}

func (m *MockServer) handleCreatePostMortem(w http.ResponseWriter, r *http.Request) {
	var req bookowl.CreatePostMortemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	id := m.nextPM
	m.nextPM++
	pm := bookowl.PostMortem{
		ID:  fmt.Sprintf("pm-%d", id),
		URL: fmt.Sprintf("https://bookowl.example.com/postmortems/pm-%d", id),
	}
	m.postMortems = append(m.postMortems, pm)
	m.mu.Unlock()

	respondJSON(w, http.StatusCreated, pm)
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
