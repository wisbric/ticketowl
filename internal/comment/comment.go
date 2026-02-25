package comment

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/wisbric/ticketowl/internal/zammad"
)

// ZammadClient defines the Zammad operations the comment service needs.
type ZammadClient interface {
	ListArticles(ctx context.Context, ticketID int) ([]zammad.Article, error)
	CreateArticle(ctx context.Context, req zammad.ArticleCreate) (*zammad.Article, error)
}

// NoteStore defines the database operations for internal notes.
type NoteStore interface {
	ListByTicketMetaID(ctx context.Context, ticketMetaID uuid.UUID) ([]InternalNote, error)
	Create(ctx context.Context, note *InternalNote) error
	GetTicketMetaID(ctx context.Context, zammadID int) (uuid.UUID, error)
}

// InternalNote is a note written in TicketOwl, never synced to Zammad.
type InternalNote struct {
	ID           uuid.UUID `json:"id"`
	TicketMetaID uuid.UUID `json:"ticket_meta_id"`
	AuthorID     uuid.UUID `json:"author_id"`
	AuthorName   string    `json:"author_name"`
	Body         string    `json:"body"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ThreadEntry is a unified view of a comment in the ticket thread.
// It may originate from either a Zammad article or a TicketOwl internal note.
type ThreadEntry struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"` // "zammad" | "internal"
	Type      string    `json:"type"`   // Zammad: "note"|"web"|"email"; internal: "internal_note"
	Sender    string    `json:"sender"` // Zammad: "Customer"|"Agent"|"System"; internal: author name
	Body      string    `json:"body"`
	Internal  bool      `json:"internal"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// AddNoteRequest is the request payload for adding an internal note.
type AddNoteRequest struct {
	Body string `json:"body"`
}

// AddReplyRequest is the request payload for adding a public reply via Zammad.
type AddReplyRequest struct {
	Body        string `json:"body"`
	ContentType string `json:"content_type,omitempty"` // defaults to "text/html"
}
