package zammad

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Article represents a Zammad article (comment).
type Article struct {
	ID          int       `json:"id"`
	TicketID    int       `json:"ticket_id"`
	Type        string    `json:"type"`
	Sender      string    `json:"sender"`
	Body        string    `json:"body"`
	ContentType string    `json:"content_type"`
	Internal    bool      `json:"internal"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// ArticleCreate is the payload for creating an article.
type ArticleCreate struct {
	TicketID    int    `json:"ticket_id"`
	TypeID      int    `json:"type_id"`
	Body        string `json:"body"`
	ContentType string `json:"content_type"`
	Internal    bool   `json:"internal"`
	SenderID    int    `json:"sender_id"`
}

// ListArticles returns all articles for a ticket.
func (c *Client) ListArticles(ctx context.Context, ticketID int) ([]Article, error) {
	path := fmt.Sprintf("/api/v1/ticket_articles/by_ticket/%d?expand=true", ticketID)

	body, err := c.get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("listing articles for ticket %d: %w", ticketID, err)
	}

	var articles []Article
	if err := json.Unmarshal(body, &articles); err != nil {
		return nil, fmt.Errorf("decoding articles: %w", err)
	}

	return articles, nil
}

// CreateArticle creates a new article (comment) on a ticket.
func (c *Client) CreateArticle(ctx context.Context, req ArticleCreate) (*Article, error) {
	body, err := c.post(ctx, "/api/v1/ticket_articles", req)
	if err != nil {
		return nil, fmt.Errorf("creating article: %w", err)
	}

	var article Article
	if err := json.Unmarshal(body, &article); err != nil {
		return nil, fmt.Errorf("decoding created article: %w", err)
	}

	return &article, nil
}
