// Package middleware provides the middleware composition helper for the proxy.
// It defines the canonical ordering: metrics → auth → ratelimit → proxy handler.
package middleware

import "net/http"

// Chain composes a slice of middleware functions into a single middleware.
// Execution order matches the slice order: middlewares[0] wraps middlewares[1]
// which wraps ... which wraps the final handler.
//
// Usage in main.go:
//
//	r.Use(middleware.Chain(
//	    metricsMiddleware,
//	    authMiddleware,
//	    rateLimitMiddleware,
//	))
//
// This results in the call stack:
//
//	request → metrics → auth → ratelimit → handler → ratelimit → auth → metrics → response
func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		// Apply in reverse so that middlewares[0] is the outermost wrapper.
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}
