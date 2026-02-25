package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Respond writes a JSON response with the given status code.
func Respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("encoding response", "error", err)
	}
}

// ErrorResponse is the standard JSON error envelope.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// RespondError writes a JSON error response.
func RespondError(w http.ResponseWriter, status int, err string, message string) {
	Respond(w, status, ErrorResponse{
		Error:   err,
		Message: message,
	})
}
