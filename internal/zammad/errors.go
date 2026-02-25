package zammad

import (
	"errors"
	"fmt"
)

// ZammadError represents an error response from the Zammad API.
type ZammadError struct {
	StatusCode int
	Message    string
}

func (e *ZammadError) Error() string {
	return fmt.Sprintf("zammad: %d %s", e.StatusCode, e.Message)
}

// IsNotFound reports whether the error is a 404 from Zammad.
func IsNotFound(err error) bool {
	var ze *ZammadError
	if errors.As(err, &ze) {
		return ze.StatusCode == 404
	}
	return false
}

// IsUnauthorised reports whether the error is a 401 or 403 from Zammad.
func IsUnauthorised(err error) bool {
	var ze *ZammadError
	if errors.As(err, &ze) {
		return ze.StatusCode == 401 || ze.StatusCode == 403
	}
	return false
}

// isClientError reports whether the status code should not be retried.
func isClientError(statusCode int) bool {
	switch statusCode {
	case 400, 401, 403, 404, 422:
		return true
	}
	return false
}
