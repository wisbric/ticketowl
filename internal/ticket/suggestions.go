package ticket

import (
	"context"
	"log/slog"

	"github.com/wisbric/ticketowl/internal/bookowl"
)

// BookOwlSearcher defines the BookOwl operations the suggestions service needs.
type BookOwlSearcher interface {
	SearchArticles(ctx context.Context, opts bookowl.SearchOptions) ([]bookowl.ArticleSummary, error)
}

// Suggestion is a BookOwl article suggested for a ticket.
type Suggestion struct {
	ID      string   `json:"id"`
	Slug    string   `json:"slug"`
	Title   string   `json:"title"`
	Excerpt string   `json:"excerpt"`
	Tags    []string `json:"tags"`
	URL     string   `json:"url"`
}

// GetSuggestions fetches runbook suggestions from BookOwl based on the ticket's
// title and tags. Returns an empty list if BookOwl is unreachable — never fails
// the ticket detail request.
func GetSuggestions(ctx context.Context, searcher BookOwlSearcher, title string, tags []string, logger *slog.Logger) []Suggestion {
	results, err := searcher.SearchArticles(ctx, bookowl.SearchOptions{
		Query: title,
		Tags:  tags,
		Limit: 5,
	})
	if err != nil {
		logger.Error("fetching suggestions from BookOwl", "error", err)
		return []Suggestion{}
	}

	suggestions := make([]Suggestion, len(results))
	for i, r := range results {
		suggestions[i] = Suggestion{
			ID:      r.ID,
			Slug:    r.Slug,
			Title:   r.Title,
			Excerpt: r.Excerpt,
			Tags:    r.Tags,
			URL:     r.URL,
		}
	}
	return suggestions
}
