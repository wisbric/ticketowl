package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/wisbric/core/pkg/auth"
	"github.com/wisbric/core/pkg/httpserver"
	"github.com/wisbric/core/pkg/tenant"
)

// ZammadTester can test connectivity to a Zammad instance.
type ZammadTester interface {
	TestConnection(ctx context.Context, url, apiToken string) error
}

// Handler provides HTTP handlers for the admin API.
type Handler struct {
	logger       *slog.Logger
	zammadTester ZammadTester
	managed      bool
}

// NewHandler creates an admin Handler.
func NewHandler(logger *slog.Logger) *Handler {
	return &Handler{logger: logger}
}

// WithZammadTester sets the optional Zammad connectivity tester.
func (h *Handler) WithZammadTester(t ZammadTester) *Handler {
	h.zammadTester = t
	return h
}

// WithManaged marks Zammad as a managed (platform-deployed) instance.
func (h *Handler) WithManaged(managed bool) *Handler {
	h.managed = managed
	return h
}

// Routes returns a chi.Router with all admin routes mounted.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// Configuration
	r.Get("/config", h.handleGetConfig)
	r.Put("/config/zammad", h.handleUpdateZammadConfig)
	r.Post("/config/zammad/test", h.handleTestZammad)
	r.Post("/config/zammad/test-stored", h.handleTestStoredZammad)
	r.Put("/config/zammad/pause-statuses", h.handleUpdatePauseStatuses)
	r.Put("/config/nightowl", h.handleUpdateIntegrationKey("nightowl"))
	r.Put("/config/bookowl", h.handleUpdateIntegrationKey("bookowl"))

	// Customer orgs
	r.Get("/customers", h.handleListCustomerOrgs)
	r.Post("/customers", h.handleCreateCustomerOrg)
	r.Route("/customers/{id}", func(r chi.Router) {
		r.Put("/", h.handleUpdateCustomerOrg)
		r.Delete("/", h.handleDeleteCustomerOrg)
	})

	// Auto-ticket rules
	r.Get("/rules", h.handleListRules)
	r.Post("/rules", h.handleCreateRule)
	r.Route("/rules/{id}", func(r chi.Router) {
		r.Put("/", h.handleUpdateRule)
		r.Delete("/", h.handleDeleteRule)
	})

	// Group-roster mappings
	r.Get("/group-roster-mappings", h.handleListGroupRosterMappings)
	r.Post("/group-roster-mappings", h.handleCreateGroupRosterMapping)
	r.Route("/group-roster-mappings/{id}", func(r chi.Router) {
		r.Put("/", h.handleUpdateGroupRosterMapping)
		r.Delete("/", h.handleDeleteGroupRosterMapping)
	})

	return r
}

// store creates a per-request Store from the tenant-scoped connection.
func (h *Handler) store(r *http.Request) *Store {
	conn := tenant.ConnFromContext(r.Context())
	return NewStore(conn)
}

func callerUUID(r *http.Request) uuid.UUID {
	id := auth.FromContext(r.Context())
	if id != nil && id.UserID != nil {
		return *id.UserID
	}
	return uuid.UUID{}
}

// --- Config ---

func (h *Handler) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	s := h.store(r)
	overview := ConfigOverview{
		Managed: h.managed,
	}

	zammadCfg, err := s.GetZammadConfig(r.Context())
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		h.logger.Error("getting zammad config", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to get config")
		return
	}
	if zammadCfg != nil {
		overview.Zammad = &ZammadConfigSummary{
			URL:           zammadCfg.URL,
			HasToken:      zammadCfg.APIToken != "",
			PauseStatuses: zammadCfg.PauseStatuses,
			UpdatedAt:     zammadCfg.UpdatedAt,
		}
	}

	nightowlKey, err := s.GetIntegrationKey(r.Context(), "nightowl")
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		h.logger.Error("getting nightowl integration key", "error", err)
	}
	if nightowlKey != nil {
		overview.NightOwl = &IntegrationKeySummary{
			APIURL:    nightowlKey.APIURL,
			UpdatedAt: nightowlKey.UpdatedAt,
		}
	}

	bookowlKey, err := s.GetIntegrationKey(r.Context(), "bookowl")
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		h.logger.Error("getting bookowl integration key", "error", err)
	}
	if bookowlKey != nil {
		overview.BookOwl = &IntegrationKeySummary{
			APIURL:    bookowlKey.APIURL,
			UpdatedAt: bookowlKey.UpdatedAt,
		}
	}

	httpserver.Respond(w, http.StatusOK, overview)
}

func (h *Handler) handleUpdateZammadConfig(w http.ResponseWriter, r *http.Request) {
	var req UpdateZammadConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if req.URL == "" || req.APIToken == "" {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "url and api_token are required")
		return
	}

	s := h.store(r)
	cfg, err := s.UpsertZammadConfig(r.Context(), req, callerUUID(r))
	if err != nil {
		h.logger.Error("updating zammad config", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to update config")
		return
	}

	// Return the summary (no secrets).
	httpserver.Respond(w, http.StatusOK, ZammadConfigSummary{
		URL:           cfg.URL,
		PauseStatuses: cfg.PauseStatuses,
		UpdatedAt:     cfg.UpdatedAt,
	})
}

func (h *Handler) handleTestZammad(w http.ResponseWriter, r *http.Request) {
	if h.zammadTester == nil {
		httpserver.RespondError(w, http.StatusServiceUnavailable, "unavailable", "zammad test not configured")
		return
	}

	var req TestZammadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if req.URL == "" || req.APIToken == "" {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "url and api_token are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := h.zammadTester.TestConnection(ctx, req.URL, req.APIToken); err != nil {
		httpserver.Respond(w, http.StatusOK, map[string]any{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	httpserver.Respond(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

func (h *Handler) handleTestStoredZammad(w http.ResponseWriter, r *http.Request) {
	if h.zammadTester == nil {
		httpserver.RespondError(w, http.StatusServiceUnavailable, "unavailable", "zammad test not configured")
		return
	}

	s := h.store(r)
	cfg, err := s.GetZammadConfig(r.Context())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "zammad not configured")
			return
		}
		h.logger.Error("getting zammad config for test", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to get config")
		return
	}
	if cfg.URL == "" || cfg.APIToken == "" {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "zammad url or token not configured")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := h.zammadTester.TestConnection(ctx, cfg.URL, cfg.APIToken); err != nil {
		httpserver.Respond(w, http.StatusOK, map[string]any{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	httpserver.Respond(w, http.StatusOK, map[string]any{
		"success": true,
	})
}

func (h *Handler) handleUpdatePauseStatuses(w http.ResponseWriter, r *http.Request) {
	var req UpdateZammadPauseStatusesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	s := h.store(r)
	if err := s.UpdateZammadPauseStatuses(r.Context(), req.PauseStatuses, callerUUID(r)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "zammad not configured")
			return
		}
		h.logger.Error("updating pause statuses", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to update pause statuses")
		return
	}

	httpserver.Respond(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) handleUpdateIntegrationKey(service string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req UpdateIntegrationKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
			return
		}
		if req.APIURL == "" || req.APIKey == "" {
			httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "api_url and api_key are required")
			return
		}

		s := h.store(r)
		key, err := s.UpsertIntegrationKey(r.Context(), service, req)
		if err != nil {
			h.logger.Error("updating integration key", "error", err, "service", service)
			httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", fmt.Sprintf("failed to update %s config", service))
			return
		}

		httpserver.Respond(w, http.StatusOK, IntegrationKeySummary{
			APIURL:    key.APIURL,
			UpdatedAt: key.UpdatedAt,
		})
	}
}

// --- Customer Orgs ---

func (h *Handler) handleListCustomerOrgs(w http.ResponseWriter, r *http.Request) {
	s := h.store(r)
	orgs, err := s.ListCustomerOrgs(r.Context())
	if err != nil {
		h.logger.Error("listing customer orgs", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list customer orgs")
		return
	}
	if orgs == nil {
		orgs = []CustomerOrg{}
	}
	httpserver.Respond(w, http.StatusOK, orgs)
}

func (h *Handler) handleCreateCustomerOrg(w http.ResponseWriter, r *http.Request) {
	var req CreateCustomerOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if req.Name == "" || req.OIDCGroup == "" {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "name and oidc_group are required")
		return
	}

	s := h.store(r)
	org, err := s.CreateCustomerOrg(r.Context(), req)
	if err != nil {
		h.logger.Error("creating customer org", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to create customer org")
		return
	}
	httpserver.Respond(w, http.StatusCreated, org)
}

func (h *Handler) handleUpdateCustomerOrg(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid customer org ID")
		return
	}

	var req UpdateCustomerOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	s := h.store(r)
	org, err := s.UpdateCustomerOrg(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "customer org not found")
			return
		}
		h.logger.Error("updating customer org", "error", err, "id", id)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to update customer org")
		return
	}
	httpserver.Respond(w, http.StatusOK, org)
}

func (h *Handler) handleDeleteCustomerOrg(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid customer org ID")
		return
	}

	s := h.store(r)
	if err := s.DeleteCustomerOrg(r.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "customer org not found")
			return
		}
		h.logger.Error("deleting customer org", "error", err, "id", id)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to delete customer org")
		return
	}
	httpserver.Respond(w, http.StatusNoContent, nil)
}

// --- Auto-Ticket Rules ---

func (h *Handler) handleListRules(w http.ResponseWriter, r *http.Request) {
	s := h.store(r)
	rules, err := s.ListAutoTicketRules(r.Context())
	if err != nil {
		h.logger.Error("listing auto-ticket rules", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list rules")
		return
	}
	if rules == nil {
		rules = []AutoTicketRule{}
	}
	httpserver.Respond(w, http.StatusOK, rules)
}

func (h *Handler) handleCreateRule(w http.ResponseWriter, r *http.Request) {
	var req CreateAutoTicketRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if req.Name == "" {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "name is required")
		return
	}
	if req.DefaultPriority == "" {
		req.DefaultPriority = "normal"
	}
	if req.TitleTemplate == "" {
		req.TitleTemplate = "{{.AlertName}}: {{.Summary}}"
	}

	s := h.store(r)
	rule, err := s.CreateAutoTicketRule(r.Context(), req)
	if err != nil {
		h.logger.Error("creating auto-ticket rule", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to create rule")
		return
	}
	httpserver.Respond(w, http.StatusCreated, rule)
}

func (h *Handler) handleUpdateRule(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid rule ID")
		return
	}

	var req UpdateAutoTicketRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	s := h.store(r)
	rule, err := s.UpdateAutoTicketRule(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "rule not found")
			return
		}
		h.logger.Error("updating auto-ticket rule", "error", err, "id", id)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to update rule")
		return
	}
	httpserver.Respond(w, http.StatusOK, rule)
}

func (h *Handler) handleDeleteRule(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid rule ID")
		return
	}

	s := h.store(r)
	if err := s.DeleteAutoTicketRule(r.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "rule not found")
			return
		}
		h.logger.Error("deleting auto-ticket rule", "error", err, "id", id)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to delete rule")
		return
	}
	httpserver.Respond(w, http.StatusNoContent, nil)
}

// --- Group-Roster Mappings ---

func (h *Handler) handleListGroupRosterMappings(w http.ResponseWriter, r *http.Request) {
	s := h.store(r)
	mappings, err := s.ListGroupRosterMappings(r.Context())
	if err != nil {
		h.logger.Error("listing group-roster mappings", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to list mappings")
		return
	}
	if mappings == nil {
		mappings = []GroupRosterMapping{}
	}
	httpserver.Respond(w, http.StatusOK, mappings)
}

func (h *Handler) handleCreateGroupRosterMapping(w http.ResponseWriter, r *http.Request) {
	var req CreateGroupRosterMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if req.ZammadGroup == "" || req.RosterID == "" {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "zammad_group and roster_id are required")
		return
	}

	s := h.store(r)
	mapping, err := s.CreateGroupRosterMapping(r.Context(), req)
	if err != nil {
		h.logger.Error("creating group-roster mapping", "error", err)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to create mapping")
		return
	}
	httpserver.Respond(w, http.StatusCreated, mapping)
}

func (h *Handler) handleUpdateGroupRosterMapping(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid mapping ID")
		return
	}

	var req UpdateGroupRosterMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	s := h.store(r)
	mapping, err := s.UpdateGroupRosterMapping(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "mapping not found")
			return
		}
		h.logger.Error("updating group-roster mapping", "error", err, "id", id)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to update mapping")
		return
	}
	httpserver.Respond(w, http.StatusOK, mapping)
}

func (h *Handler) handleDeleteGroupRosterMapping(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid mapping ID")
		return
	}

	s := h.store(r)
	if err := s.DeleteGroupRosterMapping(r.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpserver.RespondError(w, http.StatusNotFound, "not_found", "mapping not found")
			return
		}
		h.logger.Error("deleting group-roster mapping", "error", err, "id", id)
		httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to delete mapping")
		return
	}
	httpserver.Respond(w, http.StatusNoContent, nil)
}
