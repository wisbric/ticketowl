package ticket

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"github.com/wisbric/ticketowl/internal/zammad"
)

// Service encapsulates ticket business logic.
// It fetches ticket content live from Zammad and enriches with TicketOwl metadata.
type Service struct {
	zammad ZammadClient
	store  TicketStore
	logger *slog.Logger
}

// NewService creates a ticket Service.
func NewService(zammad ZammadClient, store TicketStore, logger *slog.Logger) *Service {
	return &Service{
		zammad: zammad,
		store:  store,
		logger: logger,
	}
}

// Get fetches a ticket live from Zammad and enriches it with TicketOwl metadata.
func (s *Service) Get(ctx context.Context, zammadID int) (*EnrichedTicket, error) {
	t, err := s.zammad.GetTicket(ctx, zammadID)
	if err != nil {
		return nil, fmt.Errorf("fetching ticket from zammad: %w", err)
	}

	meta, err := s.store.GetByZammadID(ctx, zammadID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("fetching ticket metadata: %w", err)
	}

	et := enrichFromZammad(t, meta)
	return &et, nil
}

// List fetches tickets from Zammad and enriches with TicketOwl metadata.
func (s *Service) List(ctx context.Context, opts ListOptions) ([]EnrichedTicket, error) {
	orderBy := opts.OrderBy
	if orderBy == "" {
		orderBy = "created_at"
	}
	sortBy := opts.SortBy
	if sortBy == "" {
		sortBy = "desc"
	}

	zammadOpts := zammad.ListTicketsOptions{
		Page:     opts.Page,
		PerPage:  opts.PerPage,
		StateIDs: opts.StateIDs,
		GroupIDs: opts.GroupIDs,
		OrgID:    opts.OrgID,
		OrderBy:  orderBy,
		SortBy:   sortBy,
	}

	var tickets []zammad.Ticket
	var err error

	if opts.Query != "" {
		tickets, err = s.zammad.SearchTickets(ctx, opts.Query, zammadOpts)
	} else {
		tickets, err = s.zammad.ListTickets(ctx, zammadOpts)
	}
	if err != nil {
		return nil, fmt.Errorf("fetching tickets from zammad: %w", err)
	}

	if len(tickets) == 0 {
		return []EnrichedTicket{}, nil
	}

	// Batch-load metadata for all returned tickets.
	zammadIDs := make([]int, len(tickets))
	for i, t := range tickets {
		zammadIDs[i] = t.ID
	}

	metas, err := s.store.ListByZammadIDs(ctx, zammadIDs)
	if err != nil {
		s.logger.Warn("failed to load ticket metadata, returning without enrichment",
			"error", err,
			"count", len(zammadIDs),
		)
		metas = nil
	}

	metaByZammadID := make(map[int]*Meta, len(metas))
	for i := range metas {
		metaByZammadID[metas[i].ZammadID] = &metas[i]
	}

	result := make([]EnrichedTicket, len(tickets))
	for i, t := range tickets {
		result[i] = enrichFromZammad(&t, metaByZammadID[t.ID])
	}

	return result, nil
}

// Create creates a ticket in Zammad and records metadata in TicketOwl.
func (s *Service) Create(ctx context.Context, req CreateRequest, callerEmail string) (*EnrichedTicket, error) {
	zReq := zammad.TicketCreateRequest{
		Title:      req.Title,
		GroupID:    req.GroupID,
		CustomerID: req.CustomerID,
		StateID:    req.StateID,
		PriorityID: req.PriorityID,
	}
	// If no customer_id provided, look up or create the customer in Zammad by email.
	if zReq.CustomerID == 0 && callerEmail != "" {
		user, err := s.zammad.SearchUsersByEmail(ctx, callerEmail)
		if err != nil {
			return nil, fmt.Errorf("searching for customer %s: %w", callerEmail, err)
		}
		if user == nil {
			user, err = s.zammad.CreateUser(ctx, callerEmail, "", "")
			if err != nil {
				// User may already exist but search missed them — retry search.
				user, searchErr := s.zammad.SearchUsersByEmail(ctx, callerEmail)
				if searchErr != nil || user == nil {
					return nil, fmt.Errorf("creating customer %s in zammad: %w", callerEmail, err)
				}
			}
		}
		zReq.CustomerID = user.ID
	}
	if zReq.CustomerID == 0 {
		return nil, fmt.Errorf("customer_id is required when caller email is not available")
	}

	if req.Body != "" {
		zReq.Article = &zammad.ArticleCreate{
			Body:        req.Body,
			ContentType: "text/html",
		}
	}

	t, err := s.zammad.CreateTicket(ctx, zReq)
	if err != nil {
		return nil, fmt.Errorf("creating ticket in zammad: %w", err)
	}

	meta, err := s.store.Upsert(ctx, t.ID, t.Number, nil)
	if err != nil {
		s.logger.Error("failed to store ticket metadata after zammad create",
			"error", err,
			"zammad_id", t.ID,
		)
	}

	et := enrichFromZammad(t, meta)
	return &et, nil
}

// Delete deletes a ticket in Zammad.
func (s *Service) Delete(ctx context.Context, zammadID int) error {
	if err := s.zammad.DeleteTicket(ctx, zammadID); err != nil {
		return fmt.Errorf("deleting ticket in zammad: %w", err)
	}
	return nil
}

// Update updates a ticket in Zammad.
func (s *Service) Update(ctx context.Context, zammadID int, req UpdateRequest) (*EnrichedTicket, error) {
	zReq := zammad.TicketUpdateRequest{
		StateID:    req.StateID,
		PriorityID: req.PriorityID,
		OwnerID:    req.OwnerID,
		GroupID:    req.GroupID,
	}

	t, err := s.zammad.UpdateTicket(ctx, zammadID, zReq)
	if err != nil {
		return nil, fmt.Errorf("updating ticket in zammad: %w", err)
	}

	meta, err := s.store.GetByZammadID(ctx, zammadID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.logger.Warn("failed to load ticket metadata after update",
			"error", err,
			"zammad_id", zammadID,
		)
	}

	et := enrichFromZammad(t, meta)
	return &et, nil
}
