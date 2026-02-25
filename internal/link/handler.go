package link

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/wisbric/ticketowl/internal/auth"
	"github.com/wisbric/ticketowl/internal/bookowl"
	"github.com/wisbric/ticketowl/internal/httpserver"
	"github.com/wisbric/ticketowl/internal/nightowl"
	"github.com/wisbric/ticketowl/internal/tenant"
)

// Handler provides HTTP handlers for the links API.
type Handler struct {
	logger   *slog.Logger
	nightowl NightOwlClient
	bookowl  BookOwlClient
}

// NewHandler creates a link Handler.
func NewHandler(logger *slog.Logger, nightowl NightOwlClient, bookowl BookOwlClient) *Handler {
	return &Handler{
		logger:   logger,
		nightowl: nightowl,
		bookowl:  bookowl,
	}
}

// Routes returns a chi.Router with all link routes mounted.
// Expects to be mounted at /api/v1/tickets/{id}/links.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.handleGetLinks)
	r.Post("/incident", h.handleLinkIncident)
	r.Delete("/incident/{incident_id}", h.handleUnlinkIncident)
	r.Post("/article", h.handleLinkArticle)
	r.Delete("/article/{article_id}", h.handleUnlinkArticle)
	return r
}

// service creates a per-request Service.
func (h *Handler) service(r *http.Request) *Service {
	conn := tenant.ConnFromContext(r.Context())
	var store LinkStore = NewStore(conn)
	return NewService(store, h.nightowl, h.bookowl, h.logger)
}

func (h *Handler) handleGetLinks(w http.ResponseWriter, r *http.Request) {
	zammadID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid ticket ID")
		return
	}

	svc := h.service(r)
	links, err := svc.GetLinks(r.Context(), zammadID)
	if err != nil {
		h.logger.Error("getting links", "error", err, "zammad_id", zammadID)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to get links")
		return
	}

	httpserver.Respond(w, http.StatusOK, links)
}

func (h *Handler) handleLinkIncident(w http.ResponseWriter, r *http.Request) {
	zammadID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid ticket ID")
		return
	}

	var req CreateIncidentLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if req.IncidentID == "" {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "incident_id is required")
		return
	}

	linkedBy := callerUUID(r)

	svc := h.service(r)
	link, err := svc.LinkIncident(r.Context(), zammadID, req.IncidentID, linkedBy)
	if err != nil {
		if nightowl.IsNotFound(err) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "incident not found in NightOwl")
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "ticket metadata not found")
			return
		}
		h.logger.Error("linking incident", "error", err, "zammad_id", zammadID)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to link incident")
		return
	}

	httpserver.Respond(w, http.StatusCreated, link)
}

func (h *Handler) handleUnlinkIncident(w http.ResponseWriter, r *http.Request) {
	zammadID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid ticket ID")
		return
	}

	incidentID := chi.URLParam(r, "incident_id")

	svc := h.service(r)
	if err := svc.UnlinkIncident(r.Context(), zammadID, incidentID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "link not found")
			return
		}
		h.logger.Error("unlinking incident", "error", err, "zammad_id", zammadID)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to unlink incident")
		return
	}

	httpserver.Respond(w, http.StatusNoContent, nil)
}

func (h *Handler) handleLinkArticle(w http.ResponseWriter, r *http.Request) {
	zammadID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid ticket ID")
		return
	}

	var req CreateArticleLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if req.ArticleID == "" {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "article_id is required")
		return
	}

	linkedBy := callerUUID(r)

	svc := h.service(r)
	link, err := svc.LinkArticle(r.Context(), zammadID, req.ArticleID, linkedBy)
	if err != nil {
		if bookowl.IsNotFound(err) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "article not found in BookOwl")
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "ticket metadata not found")
			return
		}
		h.logger.Error("linking article", "error", err, "zammad_id", zammadID)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to link article")
		return
	}

	httpserver.Respond(w, http.StatusCreated, link)
}

func (h *Handler) handleUnlinkArticle(w http.ResponseWriter, r *http.Request) {
	zammadID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid ticket ID")
		return
	}

	articleID := chi.URLParam(r, "article_id")

	svc := h.service(r)
	if err := svc.UnlinkArticle(r.Context(), zammadID, articleID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "link not found")
			return
		}
		h.logger.Error("unlinking article", "error", err, "zammad_id", zammadID)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to unlink article")
		return
	}

	httpserver.Respond(w, http.StatusNoContent, nil)
}

func callerUUID(r *http.Request) uuid.UUID {
	id := auth.FromContext(r.Context())
	if id != nil && id.UserID != nil {
		return *id.UserID
	}
	return uuid.UUID{}
}
