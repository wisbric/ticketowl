package bookowl_test

import (
	"context"
	"testing"

	"github.com/wisbric/ticketowl/internal/bookowl"
)

func TestSearchArticles(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	m.SetSearchResults([]bookowl.ArticleSummary{
		{ID: "art-1", Title: "Restart Guide", Excerpt: "How to restart services"},
		{ID: "art-2", Title: "Troubleshooting DB", Excerpt: "Common DB issues"},
	})

	articles, err := client.SearchArticles(context.Background(), bookowl.SearchOptions{
		Query: "restart",
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("SearchArticles: unexpected error: %v", err)
	}
	if len(articles) != 2 {
		t.Errorf("len(articles) = %d, want 2", len(articles))
	}
	if articles[0].Title != "Restart Guide" {
		t.Errorf("articles[0].Title = %q, want %q", articles[0].Title, "Restart Guide")
	}

	m.RequireCall(t, "GET", "/api/v1/articles/search")
}

func TestSearchArticles_Empty(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	articles, err := client.SearchArticles(context.Background(), bookowl.SearchOptions{
		Query: "nonexistent",
	})
	if err != nil {
		t.Fatalf("SearchArticles: unexpected error: %v", err)
	}
	if len(articles) != 0 {
		t.Errorf("len(articles) = %d, want 0", len(articles))
	}
}

func TestGetArticle(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	m.AddArticle(bookowl.Article{
		ID:    "art-1",
		Slug:  "restart-guide",
		Title: "Restart Guide",
		Body:  "Step 1: SSH into the server...",
		Tags:  []string{"operations", "restart"},
		URL:   "https://bookowl.example.com/articles/restart-guide",
	})

	article, err := client.GetArticle(context.Background(), "art-1")
	if err != nil {
		t.Fatalf("GetArticle: unexpected error: %v", err)
	}
	if article.ID != "art-1" {
		t.Errorf("ID = %q, want %q", article.ID, "art-1")
	}
	if article.Title != "Restart Guide" {
		t.Errorf("Title = %q, want %q", article.Title, "Restart Guide")
	}
	if article.Body != "Step 1: SSH into the server..." {
		t.Errorf("Body = %q, want %q", article.Body, "Step 1: SSH into the server...")
	}

	m.RequireCall(t, "GET", "/api/v1/articles/art-1")
}

func TestGetArticle_NotFound(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	_, err := client.GetArticle(context.Background(), "does-not-exist")
	if err == nil {
		t.Fatal("GetArticle: expected error for missing article")
	}
	if !bookowl.IsNotFound(err) {
		t.Errorf("expected IsNotFound=true, got false; err=%v", err)
	}
}

func TestCreatePostMortem(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	pm, err := client.CreatePostMortem(context.Background(), bookowl.CreatePostMortemRequest{
		Title:        "Post-Mortem: API Outage",
		TicketID:     "ticket-42",
		TicketNumber: "00042",
		IncidentIDs:  []string{"inc-001", "inc-002"},
		Summary:      "API was down for 30 minutes due to DB failover",
		Tags:         []string{"outage", "database"},
	})
	if err != nil {
		t.Fatalf("CreatePostMortem: unexpected error: %v", err)
	}
	if pm.ID == "" {
		t.Error("expected non-empty post-mortem ID")
	}
	if pm.URL == "" {
		t.Error("expected non-empty post-mortem URL")
	}

	m.RequireCall(t, "POST", "/api/v1/postmortems")

	pms := m.PostMortems()
	if len(pms) != 1 {
		t.Errorf("len(postmortems) = %d, want 1", len(pms))
	}
}

func TestUnauthorized(t *testing.T) {
	m := NewMockServer(t)
	client := bookowl.New(m.URL(), "")

	_, err := client.GetArticle(context.Background(), "art-1")
	if err == nil {
		t.Fatal("expected error for unauthorized request")
	}
}

func TestCallCount(t *testing.T) {
	m := NewMockServer(t)
	client := m.Client()

	m.AddArticle(bookowl.Article{ID: "art-1", Title: "Test"})

	_, _ = client.GetArticle(context.Background(), "art-1")
	_, _ = client.GetArticle(context.Background(), "art-1")

	if got := m.CallCount("GET", "/api/v1/articles/art-1"); got != 2 {
		t.Errorf("CallCount = %d, want 2", got)
	}
}

func TestRequireNoCall(t *testing.T) {
	m := NewMockServer(t)
	m.RequireNoCall(t, "DELETE", "/api/v1/articles/art-1")
}

func TestIsNotFound(t *testing.T) {
	if bookowl.IsNotFound(nil) {
		t.Error("IsNotFound(nil) = true, want false")
	}

	err := &bookowl.APIError{StatusCode: 404, Message: "not found"}
	if !bookowl.IsNotFound(err) {
		t.Error("IsNotFound(404) = false, want true")
	}

	err = &bookowl.APIError{StatusCode: 500, Message: "server error"}
	if bookowl.IsNotFound(err) {
		t.Error("IsNotFound(500) = true, want false")
	}
}

func TestIsUnauthorised(t *testing.T) {
	if bookowl.IsUnauthorised(nil) {
		t.Error("IsUnauthorised(nil) = true, want false")
	}

	for _, code := range []int{401, 403} {
		err := &bookowl.APIError{StatusCode: code, Message: "denied"}
		if !bookowl.IsUnauthorised(err) {
			t.Errorf("IsUnauthorised(%d) = false, want true", code)
		}
	}

	err := &bookowl.APIError{StatusCode: 404, Message: "not found"}
	if bookowl.IsUnauthorised(err) {
		t.Error("IsUnauthorised(404) = true, want false")
	}
}
