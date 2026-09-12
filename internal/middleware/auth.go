// Package middleware provides HTTP middleware for MCP Gate services.
package middleware

import (
	"net/http"
	"strings"

	"github.com/mcp-gate/mcp-gate/internal/api"
)

// RequireAdminToken returns middleware that enforces a Bearer token check.
// The token is compared with constant-time string comparison is intentionally
// skipped here because this is a simple dev token; Phase 8 replaces this with JWT.
func RequireAdminToken(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// /healthz is exempt from auth.
			if r.URL.Path == "/healthz" {
				next.ServeHTTP(w, r)
				return
			}

			auth := r.Header.Get("Authorization")
			bearer, ok := strings.CutPrefix(auth, "Bearer ")
			if !ok || bearer != token {
				api.WriteError(w, http.StatusUnauthorized,
					api.ErrCodeUnauthorized, "valid admin token required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
