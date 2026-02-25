package bookowl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// SearchOptions controls article search filtering.
type SearchOptions struct {
	Query string
	Tags  []string
	Limit int
}

// ArticleSummary is a brief article representation returned by search.
type ArticleSummary struct {
	ID      string   `json:"id"`
	Slug    string   `json:"slug"`
	Title   string   `json:"title"`
	Excerpt string   `json:"excerpt"`
	Tags    []string `json:"tags"`
	URL     string   `json:"url"`
}

// SearchArticles searches for knowledge base articles matching the given options.
func (c *Client) SearchArticles(ctx context.Context, opts SearchOptions) ([]ArticleSummary, error) {
	params := url.Values{}
	if opts.Query != "" {
		params.Set("query", opts.Query)
	}
	if len(opts.Tags) > 0 {
		params.Set("tags", strings.Join(opts.Tags, ","))
	}
	if opts.Limit > 0 {
		params.Set("limit", strconv.Itoa(opts.Limit))
	}

	path := "/api/v1/articles/search?" + params.Encode()

	body, err := c.get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("searching articles: %w", err)
	}

	var articles []ArticleSummary
	if err := json.Unmarshal(body, &articles); err != nil {
		return nil, fmt.Errorf("decoding article search results: %w", err)
	}

	return articles, nil
}
