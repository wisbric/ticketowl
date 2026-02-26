package telemetry

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// NewLogger creates a structured logger. Format is "json" or "text".
// Level is one of: debug, info, warn, error.
func NewLogger(format, level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}
	var handler slog.Handler

	var w io.Writer = os.Stdout
	switch strings.ToLower(format) {
	case "text":
		handler = slog.NewTextHandler(w, opts)
	default:
		handler = slog.NewJSONHandler(w, opts)
	}

	return slog.New(handler)
}
