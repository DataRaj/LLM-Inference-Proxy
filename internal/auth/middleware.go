// Package auth provides HTTP middleware for API key authentication.
// Keys are validated against the static list supplied at startup via Config —
// this package never touches Redis or environment variables directly.
package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
)

// contextKey is an unexported type for context value keys in this package.
type contextKey int

const (
	// keyContextKey stores the validated API key identifier in request context.
	keyContextKey contextKey = iota
)

// Middleware is a configured auth middleware instance.
// Construct it with NewMiddleware; use its Handler method as chi middleware.
type Middleware struct {
	// validKeys is a set for O(1) lookup.
	// Built once at construction time; never mutated after that.
	validKeys map[string]struct{}
	log       zerolog.Logger
}

// NewMiddleware constructs an auth Middleware from the list of valid API keys.
// The returned function satisfies the chi middleware signature func(http.Handler) http.Handler.
func NewMiddleware(apiKeys []string, log zerolog.Logger) func(http.Handler) http.Handler {
	m := &Middleware{
		validKeys: make(map[string]struct{}, len(apiKeys)),
		log:       log,
	}
	for _, k := range apiKeys {
		m.validKeys[k] = struct{}{}
	}
	return m.handler
}

// handler is the actual net/http middleware function.
func (m *Middleware) handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, ok := extractBearerToken(r)
		if !ok || !m.isValid(key) {
			m.log.Warn().
				Str("path", r.URL.Path).
				Str("remote_addr", r.RemoteAddr).
				Msg("auth: rejected request — missing or invalid API key")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Propagate the validated key so downstream middleware (rate-limiter)
		// can use it as a stable scoping identifier without re-parsing headers.
		ctx := context.WithValue(r.Context(), keyContextKey, key)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// isValid reports whether key exists in the pre-built valid-key set.
func (m *Middleware) isValid(key string) bool {
	_, ok := m.validKeys[key]
	return ok
}

// extractBearerToken parses the Authorization header.
// Returns ("", false) if the header is absent or not in "Bearer <token>" form.
func extractBearerToken(r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", false
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}
	return token, true
}

// KeyFromContext retrieves the authenticated API key from the request context.
// Returns ("", false) if no key was stored (e.g. on unauthenticated routes).
// Other packages (e.g. ratelimit) call this instead of re-parsing headers.
func KeyFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(keyContextKey).(string)
	return v, ok
}
