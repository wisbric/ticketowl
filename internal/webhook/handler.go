package webhook

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

// Handler provides HTTP handlers for inbound webhooks.
type Handler struct {
	redis         *redis.Client
	webhookSecret string
	logger        *slog.Logger
}

// NewHandler creates a webhook Handler.
func NewHandler(rdb *redis.Client, webhookSecret string, logger *slog.Logger) *Handler {
	return &Handler{
		redis:         rdb,
		webhookSecret: webhookSecret,
		logger:        logger,
	}
}

// Routes returns a chi.Router for webhook endpoints.
// Mounted at /api/v1/webhooks.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/zammad", HandleZammad(h.redis, h.webhookSecret, h.logger))
	r.Post("/nightowl", HandleNightOwl(h.redis, h.logger))
	return r
}
