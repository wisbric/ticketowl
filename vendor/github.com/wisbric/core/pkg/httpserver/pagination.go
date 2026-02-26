package httpserver

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	// DefaultPageSize is the default number of items per page.
	DefaultPageSize = 25
	// MaxPageSize is the maximum allowed page size.
	MaxPageSize = 100
)

// --- Cursor-based pagination (for alerts and other time-ordered data) ---

// Cursor represents a position in a cursor-paginated result set.
// It encodes a timestamp + ID pair for stable, keyset-based pagination.
type Cursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

// EncodeCursor serialises a cursor to a URL-safe opaque string.
func EncodeCursor(c Cursor) string {
	raw := fmt.Sprintf("%d:%s", c.CreatedAt.UnixMicro(), c.ID.String())
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor parses a cursor string back into its components.
func DecodeCursor(s string) (Cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, fmt.Errorf("decoding cursor: %w", err)
	}

	parts := strings.SplitN(string(raw), ":", 2)
	if len(parts) != 2 {
		return Cursor{}, fmt.Errorf("invalid cursor format")
	}

	usec, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return Cursor{}, fmt.Errorf("invalid cursor timestamp: %w", err)
	}

	id, err := uuid.Parse(parts[1])
	if err != nil {
		return Cursor{}, fmt.Errorf("invalid cursor id: %w", err)
	}

	return Cursor{
		CreatedAt: time.UnixMicro(usec).UTC(),
		ID:        id,
	}, nil
}

// CursorParams holds the parsed query parameters for cursor-based pagination.
type CursorParams struct {
	After *Cursor // nil means start from the beginning
	Limit int
}

// ParseCursorParams extracts cursor pagination parameters from the request.
func ParseCursorParams(r *http.Request) (CursorParams, error) {
	p := CursorParams{Limit: DefaultPageSize}

	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return p, fmt.Errorf("limit must be a positive integer")
		}
		if n > MaxPageSize {
			n = MaxPageSize
		}
		p.Limit = n
	}

	if v := r.URL.Query().Get("after"); v != "" {
		c, err := DecodeCursor(v)
		if err != nil {
			return p, fmt.Errorf("invalid cursor: %w", err)
		}
		p.After = &c
	}

	return p, nil
}

// CursorPage is the response envelope for cursor-paginated results.
type CursorPage[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"next_cursor,omitempty"`
	HasMore    bool    `json:"has_more"`
}

// NewCursorPage builds a CursorPage from a result set. Pass the function that
// extracts the cursor fields from the last item. Items should be fetched with
// limit+1 to detect whether more rows exist.
func NewCursorPage[T any](items []T, limit int, cursorFn func(T) Cursor) CursorPage[T] {
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	page := CursorPage[T]{
		Items:   items,
		HasMore: hasMore,
	}

	if hasMore && len(items) > 0 {
		c := EncodeCursor(cursorFn(items[len(items)-1]))
		page.NextCursor = &c
	}

	return page
}

// --- Offset-based pagination (for knowledge base, runbooks, etc.) ---

// OffsetParams holds the parsed query parameters for offset-based pagination.
type OffsetParams struct {
	Page     int
	PageSize int
	Offset   int // computed from Page and PageSize
}

// ParseOffsetParams extracts offset pagination parameters from the request.
func ParseOffsetParams(r *http.Request) (OffsetParams, error) {
	p := OffsetParams{Page: 1, PageSize: DefaultPageSize}

	if v := r.URL.Query().Get("page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return p, fmt.Errorf("page must be a positive integer")
		}
		p.Page = n
	}

	if v := r.URL.Query().Get("page_size"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return p, fmt.Errorf("page_size must be a positive integer")
		}
		if n > MaxPageSize {
			n = MaxPageSize
		}
		p.PageSize = n
	}

	p.Offset = (p.Page - 1) * p.PageSize
	return p, nil
}

// OffsetPage is the response envelope for offset-paginated results.
type OffsetPage[T any] struct {
	Items      []T `json:"items"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// NewOffsetPage builds an OffsetPage from a result set and total count.
func NewOffsetPage[T any](items []T, params OffsetParams, totalItems int) OffsetPage[T] {
	totalPages := 0
	if params.PageSize > 0 {
		totalPages = (totalItems + params.PageSize - 1) / params.PageSize
	}

	return OffsetPage[T]{
		Items:      items,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
}
