package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/redis/go-redis/v9"

	"github.com/wisbric/core/pkg/httpserver"
	"github.com/wisbric/core/pkg/tenant"

	"github.com/wisbric/ticketowl/internal/clientresolver"
	"github.com/wisbric/ticketowl/internal/zammad"
)

// SecretResolver resolves the Zammad webhook secret for the current request.
type SecretResolver func(r *http.Request) (string, error)

// DBSecretResolver returns a SecretResolver that reads the webhook secret
// from the tenant's zammad_config in the database.
func DBSecretResolver(logger *slog.Logger) SecretResolver {
	return func(r *http.Request) (string, error) {
		conn := tenant.ConnFromContext(r.Context())
		return clientresolver.WebhookSecret(r.Context(), conn)
	}
}

// HandleZammad returns an HTTP handler for inbound Zammad webhooks.
// It uses the provided SecretResolver to get the per-tenant webhook secret,
// validates the HMAC-SHA1 signature, parses the payload, and pushes the
// event to the tenant's Redis stream synchronously.
func HandleZammad(rdb *redis.Client, resolveSecret SecretResolver, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Read body.
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("reading webhook body", "error", err)
			httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "failed to read body")
			return
		}

		// 2. Resolve webhook secret.
		webhookSecret, err := resolveSecret(r)
		if err != nil {
			logger.Error("resolving webhook secret", "error", err)
			httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "webhook secret not configured")
			return
		}

		// 3. Validate HMAC-SHA1 signature before doing anything else.
		sig := r.Header.Get("X-Hub-Signature")
		if err := zammad.ValidateWebhookSignature(body, webhookSecret, sig); err != nil {
			logger.Warn("invalid zammad webhook signature", "error", err)
			httpserver.RespondError(w, http.StatusUnauthorized, "unauthorized", "invalid signature")
			return
		}

		// 4. Parse payload.
		var payload zammad.WebhookPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			logger.Error("parsing zammad webhook payload", "error", err)
			httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "invalid payload")
			return
		}

		if payload.Event == "" || payload.Ticket == nil {
			httpserver.RespondError(w, http.StatusBadRequest, "bad_request", "missing event or ticket")
			return
		}

		// 5. Push to Redis stream synchronously.
		tenantInfo := tenant.FromContext(r.Context())
		streamKey := StreamKey(tenantInfo.Slug)

		evt := ZammadEvent{
			Event:    payload.Event,
			TicketID: payload.Ticket.ID,
			Number:   payload.Ticket.Number,
			StateID:  payload.Ticket.StateID,
			State:    payload.Ticket.State,
			Priority: payload.Ticket.Priority,
			Group:    payload.Ticket.Group,
		}

		if err := pushEvent(r.Context(), rdb, streamKey, "zammad", evt); err != nil {
			logger.Error("pushing zammad event to redis", "error", err, "event", payload.Event)
			httpserver.RespondError(w, http.StatusInternalServerError, "internal_error", "failed to queue event")
			return
		}

		logger.Info("zammad webhook received",
			"event", payload.Event,
			"ticket_id", payload.Ticket.ID,
			"stream", streamKey,
		)

		// 6. Return 200 immediately.
		httpserver.Respond(w, http.StatusOK, map[string]string{"status": "queued"})
	}
}

// pushEvent marshals the event data and pushes it to a Redis stream via XADD.
func pushEvent(ctx context.Context, rdb *redis.Client, streamKey, source string, data any) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling event data: %w", err)
	}

	return rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]interface{}{
			"source": source,
			"data":   string(dataJSON),
		},
	}).Err()
}
