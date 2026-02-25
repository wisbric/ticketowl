package bookowl

import (
	"context"
	"encoding/json"
	"fmt"
)

// Article represents a full BookOwl knowledge base article.
type Article struct {
	ID     string   `json:"id"`
	Slug   string   `json:"slug"`
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Tags   []string `json:"tags"`
	URL    string   `json:"url"`
	Status string   `json:"status"`
}

// GetArticle returns a single article by ID.
func (c *Client) GetArticle(ctx context.Context, articleID string) (*Article, error) {
	path := fmt.Sprintf("/api/v1/articles/%s", articleID)

	body, err := c.get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("getting article %s: %w", articleID, err)
	}

	var article Article
	if err := json.Unmarshal(body, &article); err != nil {
		return nil, fmt.Errorf("decoding article %s: %w", articleID, err)
	}

	return &article, nil
}
