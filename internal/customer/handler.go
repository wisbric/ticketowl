package customer

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wisbric/core/pkg/auth"
	"github.com/wisbric/core/pkg/httpserver"
	"github.com/wisbric/core/pkg/tenant"
	"github.com/wisbric/ticketowl/internal/zammad"
)

// Handler provides HTTP handlers for the customer portal.
type Handler struct {
	logger *slog.Logger
	zammad ZammadClient
}

// NewHandler creates a customer Handler.
func NewHandler(logger *slog.Logger, zammad ZammadClient) *Handler {
	return &Handler{
		logger: logger,
		zammad: zammad,
	}
}

// Routes returns a chi.Router with all portal routes mounted.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/tickets", h.handleListTickets)
	r.Route("/tickets/{id}", func(r chi.Router) {
		r.Get("/", h.handleGetTicket)
		r.Post("/reply", h.handleReply)
		r.Get("/articles", h.handleGetArticles)
	})
	return r
}

// service creates a per-request Service from the tenant-scoped connection.
func (h *Handler) service(r *http.Request) *Service {
	conn := tenant.ConnFromContext(r.Context())
	var store CustomerStore = NewStore(conn)
	return NewService(store, h.zammad, h.logger)
}

// orgID extracts the org UUID from the authenticated identity.
// Returns nil if the caller has no org_id (non-customer).
func orgID(r *http.Request) *auth.Identity {
	return auth.FromContext(r.Context())
}

func (h *Handler) handleListTickets(w http.ResponseWriter, r *http.Request) {
	identity := orgID(r)
	if identity == nil || identity.OrgID == nil {
		httpserver.RespondError(w, http.StatusForbidden, "forbidden", "customer org_id required")
		return
	}

	svc := h.service(r)
	tickets, err := svc.ListMyTickets(r.Context(), *identity.OrgID)
	if err != nil {
		h.logger.Error("listing portal tickets", "error", err, "org_id", identity.OrgID)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list tickets")
		return
	}
	if tickets == nil {
		tickets = []PortalTicket{}
	}
	httpserver.Respond(w, http.StatusOK, tickets)
}

func (h *Handler) handleGetTicket(w http.ResponseWriter, r *http.Request) {
	identity := orgID(r)
	if identity == nil || identity.OrgID == nil {
		httpserver.RespondError(w, http.StatusForbidden, "forbidden", "customer org_id required")
		return
	}

	zammadID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid ticket ID")
		return
	}

	svc := h.service(r)
	detail, err := svc.GetMyTicket(r.Context(), *identity.OrgID, zammadID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			httpserver.RespondError(w, http.StatusForbidden, "forbidden", "ticket not accessible")
			return
		}
		if zammad.IsNotFound(err) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "ticket not found")
			return
		}
		h.logger.Error("getting portal ticket", "error", err, "zammad_id", zammadID)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to get ticket")
		return
	}

	httpserver.Respond(w, http.StatusOK, detail)
}

func (h *Handler) handleReply(w http.ResponseWriter, r *http.Request) {
	identity := orgID(r)
	if identity == nil || identity.OrgID == nil {
		httpserver.RespondError(w, http.StatusForbidden, "forbidden", "customer org_id required")
		return
	}

	zammadID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid ticket ID")
		return
	}

	var req ReplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if req.Body == "" {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "body is required")
		return
	}

	svc := h.service(r)
	if err := svc.AddReply(r.Context(), *identity.OrgID, zammadID, req.Body); err != nil {
		if errors.Is(err, ErrForbidden) {
			httpserver.RespondError(w, http.StatusForbidden, "forbidden", "ticket not accessible")
			return
		}
		if zammad.IsNotFound(err) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "ticket not found")
			return
		}
		h.logger.Error("adding portal reply", "error", err, "zammad_id", zammadID)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to add reply")
		return
	}

	httpserver.Respond(w, http.StatusCreated, map[string]string{"status": "ok"})
}

func (h *Handler) handleGetArticles(w http.ResponseWriter, r *http.Request) {
	identity := orgID(r)
	if identity == nil || identity.OrgID == nil {
		httpserver.RespondError(w, http.StatusForbidden, "forbidden", "customer org_id required")
		return
	}

	zammadID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid ticket ID")
		return
	}

	svc := h.service(r)
	articles, err := svc.GetLinkedArticles(r.Context(), *identity.OrgID, zammadID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			httpserver.RespondError(w, http.StatusForbidden, "forbidden", "ticket not accessible")
			return
		}
		if zammad.IsNotFound(err) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "ticket not found")
			return
		}
		h.logger.Error("getting portal articles", "error", err, "zammad_id", zammadID)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to get articles")
		return
	}
	if articles == nil {
		articles = []PortalArticle{}
	}
	httpserver.Respond(w, http.StatusOK, articles)
}
