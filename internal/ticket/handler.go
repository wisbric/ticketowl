package ticket

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/wisbric/ticketowl/internal/auth"
	"github.com/wisbric/ticketowl/internal/httpserver"
	"github.com/wisbric/ticketowl/internal/link"
	"github.com/wisbric/ticketowl/internal/telemetry"
	"github.com/wisbric/ticketowl/internal/tenant"
	"github.com/wisbric/ticketowl/internal/zammad"
)

// Handler provides HTTP handlers for the tickets API.
type Handler struct {
	logger        *slog.Logger
	zammad        ZammadClient
	bookowlSearch BookOwlSearcher
	bookowlPM     PostMortemCreator
	nightowlFetch NightOwlIncidentFetcher
}

// NewHandler creates a ticket Handler.
func NewHandler(logger *slog.Logger, zammad ZammadClient) *Handler {
	return &Handler{
		logger: logger,
		zammad: zammad,
	}
}

// WithEnrichment sets the optional BookOwl and NightOwl clients for suggestions and post-mortems.
func (h *Handler) WithEnrichment(bookowlSearch BookOwlSearcher, bookowlPM PostMortemCreator, nightowlFetch NightOwlIncidentFetcher) *Handler {
	h.bookowlSearch = bookowlSearch
	h.bookowlPM = bookowlPM
	h.nightowlFetch = nightowlFetch
	return h
}

// Routes returns a chi.Router with all ticket routes mounted.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.handleList)
	r.Post("/", h.handleCreate)
	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", h.handleGet)
		r.Patch("/", h.handleUpdate)
		r.Get("/suggestions", h.handleGetSuggestions)
		r.Post("/postmortem", h.handleCreatePostMortem)
	})
	return r
}

// service creates a per-request Service from the tenant-scoped connection.
func (h *Handler) service(r *http.Request) *Service {
	conn := tenant.ConnFromContext(r.Context())
	var store TicketStore = NewStore(conn)
	return NewService(h.zammad, store, h.logger)
}

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	telemetry.TicketRequestsTotal.WithLabelValues("get").Inc()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid ticket ID")
		return
	}

	svc := h.service(r)
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

	svc := h.service(r)
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

	svc := h.service(r)
	ticket, err := svc.Create(r.Context(), req)
	if err != nil {
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

	svc := h.service(r)
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

func (h *Handler) handleGetSuggestions(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid ticket ID")
		return
	}

	if h.bookowlSearch == nil {
		httpserver.Respond(w, http.StatusOK, []Suggestion{})
		return
	}

	t, err := h.zammad.GetTicket(r.Context(), id)
	if err != nil {
		if zammad.IsNotFound(err) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "ticket not found")
			return
		}
		h.logger.Error("getting ticket for suggestions", "error", err, "id", id)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to get ticket")
		return
	}

	suggestions := GetSuggestions(r.Context(), h.bookowlSearch, t.Title, t.Tags, h.logger)
	httpserver.Respond(w, http.StatusOK, suggestions)
}

func (h *Handler) handleCreatePostMortem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid ticket ID")
		return
	}

	if h.bookowlPM == nil || h.nightowlFetch == nil {
		httpserver.RespondError(w, http.StatusServiceUnavailable, "unavailable", "post-mortem creation not configured")
		return
	}

	t, err := h.zammad.GetTicket(r.Context(), id)
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

	conn := tenant.ConnFromContext(r.Context())
	pmStore := link.NewStore(conn)

	result, err := CreatePostMortem(r.Context(), id, t.Number, t.Title, createdBy, pmStore, h.bookowlPM, h.nightowlFetch)
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
