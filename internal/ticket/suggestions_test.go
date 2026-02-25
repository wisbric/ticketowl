package ticket_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/wisbric/ticketowl/internal/bookowl"
	"github.com/wisbric/ticketowl/internal/ticket"
)

// --- Mock BookOwlSearcher ---

type mockBookOwlSearcher struct {
	results []bookowl.ArticleSummary
	err     error
}

func (m *mockBookOwlSearcher) SearchArticles(_ context.Context, _ bookowl.SearchOptions) ([]bookowl.ArticleSummary, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.results, nil
}

func TestGetSuggestions_ReturnsResults(t *testing.T) {
	searcher := &mockBookOwlSearcher{
		results: []bookowl.ArticleSummary{
			{ID: "a1", Slug: "fix-printer", Title: "Fix Printer", Excerpt: "Steps to fix", Tags: []string{"printer"}, URL: "https://kb.example.com/fix-printer"},
			{ID: "a2", Slug: "reset-toner", Title: "Reset Toner", Excerpt: "Toner steps", Tags: []string{"printer", "toner"}, URL: "https://kb.example.com/reset-toner"},
		},
	}

	got := ticket.GetSuggestions(context.Background(), searcher, "Printer on fire", []string{"printer"}, slog.Default())
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].ID != "a1" {
		t.Errorf("got[0].ID = %q, want %q", got[0].ID, "a1")
	}
	if got[0].Title != "Fix Printer" {
		t.Errorf("got[0].Title = %q, want %q", got[0].Title, "Fix Printer")
	}
	if got[1].Slug != "reset-toner" {
		t.Errorf("got[1].Slug = %q, want %q", got[1].Slug, "reset-toner")
	}
}

func TestGetSuggestions_BookOwlUnreachable(t *testing.T) {
	searcher := &mockBookOwlSearcher{
		err: fmt.Errorf("connection refused"),
	}

	got := ticket.GetSuggestions(context.Background(), searcher, "Printer on fire", nil, slog.Default())
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0 (empty list when BookOwl unreachable)", len(got))
	}
}

func TestGetSuggestions_NoResults(t *testing.T) {
	searcher := &mockBookOwlSearcher{
		results: []bookowl.ArticleSummary{},
	}

	got := ticket.GetSuggestions(context.Background(), searcher, "Obscure issue", nil, slog.Default())
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
}
