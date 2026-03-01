package comment

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/wisbric/core/pkg/auth"
	"github.com/wisbric/core/pkg/httpserver"
	"github.com/wisbric/core/pkg/tenant"

	"github.com/wisbric/ticketowl/internal/clientresolver"
	"github.com/wisbric/ticketowl/internal/zammad"
)

// Handler provides HTTP handlers for the comment/thread API.
type Handler struct {
	logger *slog.Logger
}

// NewHandler creates a comment Handler.
func NewHandler(logger *slog.Logger) *Handler {
	return &Handler{
		logger: logger,
	}
}

// Routes returns a chi.Router for comment endpoints.
// Mounted at /api/v1/tickets/{id}/comments.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.handleListThread)
	r.Post("/", h.handleAddPublicReply)
	r.Post("/internal", h.handleAddInternalNote)
	return r
}

func (h *Handler) service(r *http.Request) (*Service, error) {
	conn := tenant.ConnFromContext(r.Context())
	zClient, err := clientresolver.ZammadClient(r.Context(), conn, h.logger)
	if err != nil {
		return nil, err
	}
	var store NoteStore = NewStore(conn)
	return NewService(zClient, store, h.logger), nil
}

func (h *Handler) ticketID(r *http.Request) (int, bool) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return 0, false
	}
	return id, true
}

func (h *Handler) handleListThread(w http.ResponseWriter, r *http.Request) {
	ticketID, ok := h.ticketID(r)
	if !ok {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid ticket ID")
		return
	}

	svc, err := h.service(r)
	if err != nil {
		h.logger.Error("resolving zammad client", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "zammad not configured")
		return
	}

	thread, err := svc.ListThread(r.Context(), ticketID)
	if err != nil {
		if zammad.IsNotFound(err) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "ticket not found")
			return
		}
		h.logger.Error("listing thread", "error", err, "ticket_id", ticketID)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list thread")
		return
	}

	if thread == nil {
		thread = []ThreadEntry{}
	}
	httpserver.Respond(w, http.StatusOK, thread)
}

func (h *Handler) handleAddPublicReply(w http.ResponseWriter, r *http.Request) {
	ticketID, ok := h.ticketID(r)
	if !ok {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid ticket ID")
		return
	}

	var req AddReplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if req.Body == "" {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "body is required")
		return
	}

	svc, err := h.service(r)
	if err != nil {
		h.logger.Error("resolving zammad client", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "zammad not configured")
		return
	}

	entry, err := svc.AddPublicReply(r.Context(), ticketID, req)
	if err != nil {
		h.logger.Error("adding public reply", "error", err, "ticket_id", ticketID)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to add reply")
		return
	}

	httpserver.Respond(w, http.StatusCreated, entry)
}

func (h *Handler) handleAddInternalNote(w http.ResponseWriter, r *http.Request) {
	ticketID, ok := h.ticketID(r)
	if !ok {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid ticket ID")
		return
	}

	var req AddNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if req.Body == "" {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "body is required")
		return
	}

	identity := auth.FromContext(r.Context())
	authorID := uuid.Nil
	authorName := "unknown"
	if identity != nil {
		if identity.UserID != nil {
			authorID = *identity.UserID
		}
		if identity.Email != "" {
			authorName = identity.Email
		}
	}

	svc, err := h.service(r)
	if err != nil {
		h.logger.Error("resolving zammad client", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "zammad not configured")
		return
	}

	entry, err := svc.AddInternalNote(r.Context(), ticketID, authorID, authorName, req)
	if err != nil {
		h.logger.Error("adding internal note", "error", err, "ticket_id", ticketID)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to add note")
		return
	}

	httpserver.Respond(w, http.StatusCreated, entry)
}
