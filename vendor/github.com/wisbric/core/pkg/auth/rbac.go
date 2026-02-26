package auth

import (
	"encoding/json"
	"net/http"
)

// roleLevel maps roles to a numeric privilege level for comparison.
var roleLevel = map[string]int{
	RoleAdmin:    40,
	RoleManager:  30,
	RoleEngineer: 20,
	RoleReadonly: 10,
}

// RequireAuth rejects requests that have no authenticated identity.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if FromContext(r.Context()) == nil {
			respondErr(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole returns middleware that rejects requests whose identity does not
// hold one of the listed roles. Roles are checked by exact match.
func RequireRole(allowed ...string) func(http.Handler) http.Handler {
	set := make(map[string]struct{}, len(allowed))
	for _, r := range allowed {
		set[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := FromContext(r.Context())
			if id == nil {
				respondForbidden(w, "authentication required")
				return
			}
			if _, ok := set[id.Role]; !ok {
				respondForbidden(w, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireMinRole returns middleware that rejects requests whose identity has a
// lower privilege level than the given minimum role. This allows hierarchical
// checks: RequireMinRole(RoleManager) permits admin and manager.
func RequireMinRole(minRole string) func(http.Handler) http.Handler {
	minLevel := roleLevel[minRole]

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := FromContext(r.Context())
			if id == nil {
				respondForbidden(w, "authentication required")
				return
			}
			if roleLevel[id.Role] < minLevel {
				respondForbidden(w, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func respondForbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   "forbidden",
		"message": message,
	})
}
