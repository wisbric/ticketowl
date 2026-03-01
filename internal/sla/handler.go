package sla

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/wisbric/core/pkg/httpserver"
	"github.com/wisbric/core/pkg/tenant"
)

// Handler provides HTTP handlers for the SLA API.
type Handler struct {
	logger *slog.Logger
}

// NewHandler creates an SLA Handler.
func NewHandler(logger *slog.Logger) *Handler {
	return &Handler{logger: logger}
}

// PolicyRoutes returns a chi.Router for SLA policy management.
// Mounted at /api/v1/sla/policies.
func (h *Handler) PolicyRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.handleListPolicies)
	r.Post("/", h.handleCreatePolicy)
	r.Route("/{id}", func(r chi.Router) {
		r.Put("/", h.handleUpdatePolicy)
		r.Delete("/", h.handleDeletePolicy)
	})
	return r
}

// TicketSLARoutes returns a chi.Router for per-ticket SLA state.
// Mounted at /api/v1/tickets/{id}/sla.
func (h *Handler) TicketSLARoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.handleGetTicketSLA)
	return r
}

func (h *Handler) service(r *http.Request) *Service {
	conn := tenant.ConnFromContext(r.Context())
	var store SLAStore = NewStore(conn)
	return NewService(store, h.logger)
}

func (h *Handler) handleListPolicies(w http.ResponseWriter, r *http.Request) {
	svc := h.service(r)
	policies, err := svc.ListPolicies(r.Context())
	if err != nil {
		h.logger.Error("listing SLA policies", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list policies")
		return
	}
	if policies == nil {
		policies = []Policy{}
	}
	httpserver.Respond(w, http.StatusOK, policies)
}

func (h *Handler) handleCreatePolicy(w http.ResponseWriter, r *http.Request) {
	var req CreatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if req.Name == "" || req.Priority == "" {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "name and priority are required")
		return
	}
	if req.ResponseMinutes <= 0 || req.ResolutionMinutes <= 0 {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "response_minutes and resolution_minutes must be positive")
		return
	}

	svc := h.service(r)
	policy, err := svc.CreatePolicy(r.Context(), req)
	if err != nil {
		h.logger.Error("creating SLA policy", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to create policy")
		return
	}
	httpserver.Respond(w, http.StatusCreated, policy)
}

func (h *Handler) handleUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid policy ID")
		return
	}

	var req UpdatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	svc := h.service(r)
	policy, err := svc.UpdatePolicy(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "policy not found")
			return
		}
		h.logger.Error("updating SLA policy", "error", err, "id", id)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to update policy")
		return
	}
	httpserver.Respond(w, http.StatusOK, policy)
}

func (h *Handler) handleDeletePolicy(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid policy ID")
		return
	}

	svc := h.service(r)
	if err := svc.DeletePolicy(r.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "policy not found")
			return
		}
		h.logger.Error("deleting SLA policy", "error", err, "id", id)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to delete policy")
		return
	}
	httpserver.Respond(w, http.StatusNoContent, nil)
}

func (h *Handler) handleGetTicketSLA(w http.ResponseWriter, r *http.Request) {
	// The {id} param is the Zammad ticket ID (integer) from the parent ticket router.
	zammadID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid ticket ID")
		return
	}

	svc := h.service(r)
	state, err := svc.GetTicketSLAByZammadID(r.Context(), zammadID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "SLA state not found")
			return
		}
		h.logger.Error("getting ticket SLA", "error", err, "zammad_id", zammadID)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to get SLA state")
		return
	}
	httpserver.Respond(w, http.StatusOK, state)
}
