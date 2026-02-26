package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// validate is a package-level, concurrency-safe validator instance.
var validate = validator.New(validator.WithRequiredStructEnabled())

// ValidationError represents a single field validation failure.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrorResponse is the error envelope returned for invalid requests.
type ValidationErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Details []ValidationError `json:"details"`
}

// Decode reads a JSON request body into dst. It enforces a max body size and
// disallows unknown fields. Returns an error suitable for display to the client.
func Decode(r *http.Request, dst any) error {
	const maxBody = 1 << 20 // 1 MiB

	body := http.MaxBytesReader(nil, r.Body, maxBody)
	defer func() { _ = body.Close() }()

	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		switch {
		case errors.As(err, &maxBytesErr):
			return fmt.Errorf("request body too large (max 1 MiB)")
		case errors.Is(err, io.EOF):
			return fmt.Errorf("request body is empty")
		default:
			return fmt.Errorf("invalid JSON: %w", err)
		}
	}

	// Reject trailing data after the first JSON value.
	if dec.More() {
		return fmt.Errorf("request body must contain a single JSON object")
	}

	return nil
}

// Validate runs struct-tag validation on v and returns field-level errors.
func Validate(v any) []ValidationError {
	err := validate.Struct(v)
	if err == nil {
		return nil
	}

	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return []ValidationError{{Field: "", Message: err.Error()}}
	}

	out := make([]ValidationError, 0, len(ve))
	for _, fe := range ve {
		out = append(out, ValidationError{
			Field:   jsonFieldName(fe),
			Message: fieldErrorMessage(fe),
		})
	}
	return out
}

// DecodeAndValidate is a convenience helper that decodes a JSON body and
// validates the result. On failure it writes a 400 response and returns false.
func DecodeAndValidate(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := Decode(r, dst); err != nil {
		RespondError(w, http.StatusBadRequest, "bad_request", err.Error())
		return false
	}

	if errs := Validate(dst); len(errs) > 0 {
		RespondValidationError(w, errs)
		return false
	}

	return true
}

// RespondValidationError writes a 422 response with field-level validation errors.
func RespondValidationError(w http.ResponseWriter, errs []ValidationError) {
	Respond(w, http.StatusUnprocessableEntity, ValidationErrorResponse{
		Error:   "validation_error",
		Message: "one or more fields failed validation",
		Details: errs,
	})
}

// jsonFieldName converts the validator's field name to the JSON field name
// (lowercase first segment of the namespace after the struct name).
func jsonFieldName(fe validator.FieldError) string {
	ns := fe.Namespace()
	// Namespace looks like "CreateAlertRequest.Title" — drop the struct prefix.
	if idx := strings.Index(ns, "."); idx >= 0 {
		ns = ns[idx+1:]
	}
	// Convert PascalCase to snake_case for the first level only.
	return toSnakeCase(ns)
}

// fieldErrorMessage returns a human-readable message for a field error.
func fieldErrorMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "this field is required"
	case "email":
		return "must be a valid email address"
	case "uuid":
		return "must be a valid UUID"
	case "min":
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fe.Param())
	case "url":
		return "must be a valid URL"
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", fe.Param())
	case "lte":
		return fmt.Sprintf("must be less than or equal to %s", fe.Param())
	default:
		return fmt.Sprintf("failed on '%s' validation", fe.Tag())
	}
}

// toSnakeCase converts PascalCase/camelCase to snake_case.
func toSnakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r + 32)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// URLParamUUID extracts a URL parameter and parses it as a UUID.
// It returns an error if the parameter is missing or invalid.
func URLParamUUID(r *http.Request, key string) (uuid.UUID, error) {
	val := chi.URLParam(r, key)
	if val == "" {
		return uuid.Nil, fmt.Errorf("missing parameter %s", key)
	}
	return uuid.Parse(val)
}
