package webhook

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/wisbric/ticketowl/internal/httpserver"
	"github.com/wisbric/ticketowl/internal/tenant"
)

// nightOwlPayload is the inbound payload from a NightOwl webhook.
type nightOwlPayload struct {
	Event    string             `json:"event"`
	Incident nightOwlIncidentPL `json:"incident"`
}

type nightOwlIncidentPL struct {
	ID         string  `json:"id"`
	Slug       string  `json:"slug"`
	Summary    string  `json:"summary"`
	Severity   string  `json:"severity"`
	Status     string  `json:"status"`
	Service    string  `json:"service"`
	CreatedAt  string  `json:"created_at"`
	ResolvedAt *string `json:"resolved_at,omitempty"`
}

// HandleNightOwl returns an HTTP handler for inbound NightOwl webhooks.
// It validates the API key, parses the payload, and pushes the event to
// the tenant's Redis stream synchronously.
func HandleNightOwl(rdb *redis.Client, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Auth is handled by the auth middleware (X-API-Key validated upstream).

		// 1. Read body.
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("reading nightowl webhook body", "error", err)
			httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "failed to read body")
			return
		}

		// 2. Parse payload.
		var payload nightOwlPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			logger.Error("parsing nightowl webhook payload", "error", err)
			httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid payload")
			return
		}

		if payload.Event == "" || payload.Incident.ID == "" {
			httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "missing event or incident")
			return
		}

		// 3. Push to Redis stream synchronously.
		tenantInfo := tenant.FromContext(r.Context())
		streamKey := StreamKey(tenantInfo.Slug)

		evt := NightOwlEvent{
			Event:      payload.Event,
			IncidentID: payload.Incident.ID,
			Slug:       payload.Incident.Slug,
			Summary:    payload.Incident.Summary,
			Severity:   payload.Incident.Severity,
			Status:     payload.Incident.Status,
			Service:    payload.Incident.Service,
		}

		if err := pushEvent(r.Context(), rdb, streamKey, "nightowl", evt); err != nil {
			logger.Error("pushing nightowl event to redis", "error", err, "event", payload.Event)
			httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to queue event")
			return
		}

		logger.Info("nightowl webhook received",
			"event", payload.Event,
			"incident_id", payload.Incident.ID,
			"stream", streamKey,
		)

		// 4. Return 200 immediately.
		httpserver.Respond(w, http.StatusOK, map[string]string{"status": "queued"})
	}
}
