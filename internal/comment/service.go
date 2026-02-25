package comment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/wisbric/ticketowl/internal/zammad"
)

// Service encapsulates comment/thread business logic.
type Service struct {
	zammad ZammadClient
	store  NoteStore
	logger *slog.Logger
}

// NewService creates a comment Service.
func NewService(zammad ZammadClient, store NoteStore, logger *slog.Logger) *Service {
	return &Service{
		zammad: zammad,
		store:  store,
		logger: logger,
	}
}

// ListThread merges Zammad articles and TicketOwl internal notes into a single
// chronologically sorted slice. The two sources have different types, so this
// method unifies them into ThreadEntry view models.
func (s *Service) ListThread(ctx context.Context, zammadTicketID int) ([]ThreadEntry, error) {
	// Fetch Zammad articles (live).
	articles, err := s.zammad.ListArticles(ctx, zammadTicketID)
	if err != nil {
		return nil, fmt.Errorf("listing zammad articles: %w", err)
	}

	// Fetch internal notes from TicketOwl DB.
	var notes []InternalNote
	metaID, err := s.store.GetTicketMetaID(ctx, zammadTicketID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("looking up ticket metadata: %w", err)
	}
	if err == nil {
		notes, err = s.store.ListByTicketMetaID(ctx, metaID)
		if err != nil {
			return nil, fmt.Errorf("listing internal notes: %w", err)
		}
	}

	// Build unified thread.
	entries := make([]ThreadEntry, 0, len(articles)+len(notes))

	for _, a := range articles {
		entries = append(entries, threadEntryFromArticle(a))
	}
	for _, n := range notes {
		entries = append(entries, threadEntryFromNote(n))
	}

	// Sort chronologically.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.Before(entries[j].CreatedAt)
	})

	return entries, nil
}

// AddPublicReply creates a public reply in Zammad (visible to customers).
func (s *Service) AddPublicReply(ctx context.Context, zammadTicketID int, req AddReplyRequest) (*ThreadEntry, error) {
	contentType := req.ContentType
	if contentType == "" {
		contentType = "text/html"
	}

	article, err := s.zammad.CreateArticle(ctx, zammad.ArticleCreate{
		TicketID:    zammadTicketID,
		TypeID:      9, // web
		Body:        req.Body,
		ContentType: contentType,
		Internal:    false,
		SenderID:    2, // Agent
	})
	if err != nil {
		return nil, fmt.Errorf("creating zammad article: %w", err)
	}

	entry := threadEntryFromArticle(*article)
	return &entry, nil
}

// AddInternalNote creates an internal note in TicketOwl (never synced to Zammad).
func (s *Service) AddInternalNote(ctx context.Context, zammadTicketID int, authorID uuid.UUID, authorName string, req AddNoteRequest) (*ThreadEntry, error) {
	metaID, err := s.store.GetTicketMetaID(ctx, zammadTicketID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("ticket not tracked in ticketowl: %w", err)
		}
		return nil, fmt.Errorf("looking up ticket metadata: %w", err)
	}

	now := time.Now()
	note := &InternalNote{
		ID:           uuid.New(),
		TicketMetaID: metaID,
		AuthorID:     authorID,
		AuthorName:   authorName,
		Body:         req.Body,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.store.Create(ctx, note); err != nil {
		return nil, fmt.Errorf("creating internal note: %w", err)
	}

	entry := threadEntryFromNote(*note)
	return &entry, nil
}

func threadEntryFromArticle(a zammad.Article) ThreadEntry {
	return ThreadEntry{
		ID:        strconv.Itoa(a.ID),
		Source:    "zammad",
		Type:      a.Type,
		Sender:    a.Sender,
		Body:      a.Body,
		Internal:  a.Internal,
		CreatedBy: a.CreatedBy,
		CreatedAt: a.CreatedAt,
	}
}

func threadEntryFromNote(n InternalNote) ThreadEntry {
	return ThreadEntry{
		ID:        n.ID.String(),
		Source:    "internal",
		Type:      "internal_note",
		Sender:    n.AuthorName,
		Body:      n.Body,
		Internal:  true,
		CreatedBy: n.AuthorName,
		CreatedAt: n.CreatedAt,
	}
}
