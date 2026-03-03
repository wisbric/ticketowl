package ticket

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/wisbric/core/pkg/auth"
	"github.com/wisbric/core/pkg/httpserver"
	"github.com/wisbric/core/pkg/tenant"

	"github.com/wisbric/ticketowl/internal/clientresolver"
	"github.com/wisbric/ticketowl/internal/link"
	"github.com/wisbric/ticketowl/internal/telemetry"
	"github.com/wisbric/ticketowl/internal/zammad"
)

// Handler provides HTTP handlers for the tickets API.
type Handler struct {
	logger *slog.Logger
}

// NewHandler creates a ticket Handler.
func NewHandler(logger *slog.Logger) *Handler {
	return &Handler{
		logger: logger,
	}
}

// ChildRoutes holds optional sub-routers to nest under /tickets/{id}.
type ChildRoutes struct {
	Comments chi.Router
	Links    chi.Router
	SLA      chi.Router
}

// Routes returns a chi.Router with all ticket routes mounted.
// If children is non-nil, nested comment and link routes are mounted under /{id}.
func (h *Handler) Routes(children *ChildRoutes) chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.handleList)
	r.Post("/", h.handleCreate)
	r.Get("/metadata", h.handleMetadata)
	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", h.handleGet)
		r.Patch("/", h.handleUpdate)
		r.Delete("/", h.handleDelete)
		r.Get("/suggestions", h.handleGetSuggestions)
		r.Post("/postmortem", h.handleCreatePostMortem)

		if children != nil {
			if children.Comments != nil {
				r.Mount("/comments", children.Comments)
			}
			if children.Links != nil {
				r.Mount("/links", children.Links)
			}
			if children.SLA != nil {
				r.Mount("/sla", children.SLA)
			}
		}
	})
	return r
}

// service creates a per-request Service by resolving the Zammad client from the tenant DB.
func (h *Handler) service(r *http.Request) (*Service, error) {
	conn := tenant.ConnFromContext(r.Context())
	zClient, err := clientresolver.ZammadClient(r.Context(), conn, h.logger)
	if err != nil {
		return nil, err
	}
	var store TicketStore = NewStore(conn)
	return NewService(zClient, store, h.logger), nil
}

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	telemetry.TicketRequestsTotal.WithLabelValues("get").Inc()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid ticket ID")
		return
	}

	svc, err := h.service(r)
	if err != nil {
		h.logger.Error("resolving zammad client", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "zammad not configured")
		return
	}

	ticket, err := svc.Get(r.Context(), id)
	if err != nil {
		if zammad.IsNotFound(err) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "ticket not found")
			return
		}
		h.logger.Error("getting ticket", "error", err, "id", id)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to get ticket")
		return
	}

	httpserver.Respond(w, http.StatusOK, ticket)
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	telemetry.TicketRequestsTotal.WithLabelValues("list").Inc()
	q := r.URL.Query()

	opts := ListOptions{
		Query: q.Get("query"),
	}

	if v := q.Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			opts.Page = p
		}
	}
	if v := q.Get("per_page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			opts.PerPage = p
		}
	}
	if v := q.Get("order_by"); v != "" {
		opts.OrderBy = v
	}
	if v := q.Get("sort_by"); v != "" {
		opts.SortBy = v
	}

	svc, err := h.service(r)
	if err != nil {
		h.logger.Error("resolving zammad client", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "zammad not configured")
		return
	}

	tickets, err := svc.List(r.Context(), opts)
	if err != nil {
		h.logger.Error("listing tickets", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list tickets")
		return
	}

	httpserver.Respond(w, http.StatusOK, tickets)
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	telemetry.TicketRequestsTotal.WithLabelValues("create").Inc()
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	if req.Title == "" {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "title is required")
		return
	}
	if req.GroupID == 0 {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "group_id is required")
		return
	}

	svc, err := h.service(r)
	if err != nil {
		h.logger.Error("resolving zammad client", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "zammad not configured")
		return
	}

	callerEmail := ""
	if identity := auth.FromContext(r.Context()); identity != nil {
		callerEmail = identity.Email
	}

	ticket, err := svc.Create(r.Context(), req, callerEmail)
	if err != nil {
		if req.CustomerID == 0 && callerEmail == "" {
			httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "customer_id is required")
			return
		}
		h.logger.Error("creating ticket", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to create ticket")
		return
	}

	httpserver.Respond(w, http.StatusCreated, ticket)
}

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	telemetry.TicketRequestsTotal.WithLabelValues("update").Inc()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid ticket ID")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	svc, err := h.service(r)
	if err != nil {
		h.logger.Error("resolving zammad client", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "zammad not configured")
		return
	}

	ticket, err := svc.Update(r.Context(), id, req)
	if err != nil {
		if zammad.IsNotFound(err) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "ticket not found")
			return
		}
		h.logger.Error("updating ticket", "error", err, "id", id)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to update ticket")
		return
	}

	httpserver.Respond(w, http.StatusOK, ticket)
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	telemetry.TicketRequestsTotal.WithLabelValues("delete").Inc()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid ticket ID")
		return
	}

	svc, err := h.service(r)
	if err != nil {
		h.logger.Error("resolving zammad client", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "zammad not configured")
		return
	}

	if err := svc.Delete(r.Context(), id); err != nil {
		if zammad.IsNotFound(err) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "ticket not found")
			return
		}
		h.logger.Error("deleting ticket", "error", err, "id", id)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to delete ticket")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleGetSuggestions(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid ticket ID")
		return
	}

	conn := tenant.ConnFromContext(r.Context())

	// Resolve Zammad client (required).
	zClient, err := clientresolver.ZammadClient(r.Context(), conn, h.logger)
	if err != nil {
		h.logger.Error("resolving zammad client for suggestions", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "zammad not configured")
		return
	}

	// Resolve BookOwl client (optional — graceful degradation).
	bookowlClient, err := clientresolver.BookOwlClient(r.Context(), conn, h.logger)
	if err != nil {
		h.logger.Info("bookowl not configured, returning empty suggestions", "error", err)
		httpserver.Respond(w, http.StatusOK, []Suggestion{})
		return
	}

	t, err := zClient.GetTicket(r.Context(), id)
	if err != nil {
		if zammad.IsNotFound(err) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "ticket not found")
			return
		}
		h.logger.Error("getting ticket for suggestions", "error", err, "id", id)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to get ticket")
		return
	}

	suggestions := GetSuggestions(r.Context(), bookowlClient, t.Title, t.Tags, h.logger)
	httpserver.Respond(w, http.StatusOK, suggestions)
}

func (h *Handler) handleCreatePostMortem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid ticket ID")
		return
	}

	conn := tenant.ConnFromContext(r.Context())

	// Resolve Zammad client (required).
	zClient, err := clientresolver.ZammadClient(r.Context(), conn, h.logger)
	if err != nil {
		h.logger.Error("resolving zammad client for post-mortem", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "zammad not configured")
		return
	}

	// Resolve BookOwl + NightOwl clients (required for post-mortem creation).
	bookowlClient, err := clientresolver.BookOwlClient(r.Context(), conn, h.logger)
	if err != nil {
		httpserver.RespondError(w, http.StatusServiceUnavailable, "unavailable", "bookowl not configured")
		return
	}
	nightowlClient, err := clientresolver.NightOwlClient(r.Context(), conn, h.logger)
	if err != nil {
		httpserver.RespondError(w, http.StatusServiceUnavailable, "unavailable", "nightowl not configured")
		return
	}

	t, err := zClient.GetTicket(r.Context(), id)
	if err != nil {
		if zammad.IsNotFound(err) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "ticket not found")
			return
		}
		h.logger.Error("getting ticket for post-mortem", "error", err, "id", id)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to get ticket")
		return
	}

	createdBy := uuid.Nil
	identity := auth.FromContext(r.Context())
	if identity != nil && identity.UserID != nil {
		createdBy = *identity.UserID
	}

	pmStore := link.NewStore(conn)

	result, err := CreatePostMortem(r.Context(), id, t.Number, t.Title, createdBy, pmStore, bookowlClient, nightowlClient)
	if err != nil {
		var existsErr *ErrPostMortemExists
		if errors.As(err, &existsErr) {
			httpserver.Respond(w, http.StatusConflict, map[string]string{
				"error":          "conflict",
				"message":        "post-mortem already exists",
				"postmortem_url": existsErr.URL,
			})
			return
		}
		h.logger.Error("creating post-mortem", "error", err, "ticket_id", id)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to create post-mortem")
		return
	}

	httpserver.Respond(w, http.StatusCreated, result)
}

// TicketMetadata holds Zammad states, priorities, and groups for the frontend.
type TicketMetadata struct {
	States     []zammad.TicketState    `json:"states"`
	Priorities []zammad.TicketPriority `json:"priorities"`
	Groups     []zammad.TicketGroup    `json:"groups"`
}

func (h *Handler) handleMetadata(w http.ResponseWriter, r *http.Request) {
	conn := tenant.ConnFromContext(r.Context())
	zClient, err := clientresolver.ZammadClient(r.Context(), conn, h.logger)
	if err != nil {
		h.logger.Error("resolving zammad client for metadata", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "zammad not configured")
		return
	}

	states, err := zClient.ListStates(r.Context())
	if err != nil {
		h.logger.Error("listing states", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list states")
		return
	}

	priorities, err := zClient.ListPriorities(r.Context())
	if err != nil {
		h.logger.Error("listing priorities", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list priorities")
		return
	}

	groups, err := zClient.ListGroups(r.Context())
	if err != nil {
		h.logger.Error("listing groups", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list groups")
		return
	}

	httpserver.Respond(w, http.StatusOK, TicketMetadata{
		States:     states,
		Priorities: priorities,
		Groups:     groups,
	})
}
