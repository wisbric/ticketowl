package zammad

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/wisbric/ticketowl/internal/telemetry"
)

var tracer = otel.Tracer("github.com/wisbric/ticketowl/internal/zammad")

const (
	maxRetries = 3
	baseDelay  = 250 * time.Millisecond
	multiplier = 2.0
	jitter     = 0.20
)

// Client communicates with a Zammad instance via REST API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	logger     *slog.Logger
}

// Option configures the Client.
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) { cl.httpClient = c }
}

// WithLogger sets a custom logger.
func WithLogger(l *slog.Logger) Option {
	return func(cl *Client) { cl.logger = l }
}

// New creates a new Zammad API client.
func New(baseURL, token string, opts ...Option) *Client {
	c := &Client{
		baseURL:    baseURL,
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     slog.Default(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// do executes an HTTP request with retry logic and OTel tracing.
func (c *Client) do(ctx context.Context, method, path string, body any) ([]byte, int, error) {
	ctx, span := tracer.Start(ctx, fmt.Sprintf("zammad.%s %s", method, path),
		trace.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("http.url", c.baseURL+path),
		),
	)
	defer span.End()

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	var lastErr error
	for attempt := range maxRetries {
		start := time.Now()

		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
		if err != nil {
			return nil, 0, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Authorization", "Token token="+c.token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("executing request: %w", err)
			c.logger.Warn("zammad request failed",
				"method", method,
				"path", path,
				"attempt", attempt+1,
				"error", err,
			)
			if attempt < maxRetries-1 {
				sleepWithJitter(attempt)
			}
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		duration := time.Since(start)
		telemetry.ZammadRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
		span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))

		if err != nil {
			return nil, resp.StatusCode, fmt.Errorf("reading response body: %w", err)
		}

		c.logger.Debug("zammad request",
			"method", method,
			"path", path,
			"status", resp.StatusCode,
			"duration_ms", duration.Milliseconds(),
			"attempt", attempt+1,
		)

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, resp.StatusCode, nil
		}

		// Client errors are not retried.
		if isClientError(resp.StatusCode) {
			return nil, resp.StatusCode, &ZammadError{
				StatusCode: resp.StatusCode,
				Message:    string(respBody),
			}
		}

		// Server error — retry.
		lastErr = &ZammadError{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}

		if attempt < maxRetries-1 {
			c.logger.Warn("zammad server error, retrying",
				"method", method,
				"path", path,
				"status", resp.StatusCode,
				"attempt", attempt+1,
			)
			sleepWithJitter(attempt)
		}

		// Reset bodyReader for retry.
		if body != nil {
			b, _ := json.Marshal(body)
			bodyReader = bytes.NewReader(b)
		}
	}

	return nil, 0, fmt.Errorf("zammad request failed after %d attempts: %w", maxRetries, lastErr)
}

// get is a convenience wrapper for GET requests.
func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	body, _, err := c.do(ctx, http.MethodGet, path, nil)
	return body, err
}

// post is a convenience wrapper for POST requests.
func (c *Client) post(ctx context.Context, path string, body any) ([]byte, error) {
	respBody, _, err := c.do(ctx, http.MethodPost, path, body)
	return respBody, err
}

// put is a convenience wrapper for PUT requests.
func (c *Client) put(ctx context.Context, path string, body any) ([]byte, error) {
	respBody, _, err := c.do(ctx, http.MethodPut, path, body)
	return respBody, err
}

func sleepWithJitter(attempt int) {
	delay := baseDelay
	for range attempt {
		delay = time.Duration(float64(delay) * multiplier)
	}
	jitterAmount := float64(delay) * jitter
	delay += time.Duration((rand.Float64()*2 - 1) * jitterAmount)
	time.Sleep(delay)
}
