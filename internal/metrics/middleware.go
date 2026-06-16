// Package metrics provides HTTP middleware that records request duration
// and status codes using the Prometheus collectors registered in prometheus.go.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/rs/zerolog"
)

// responseRecorder wraps http.ResponseWriter to capture the status code
// written by a downstream handler so the middleware can record it.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader intercepts the status code before passing it on.
func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

// NewMiddleware returns an http.Handler middleware that measures:
//   - request duration (histogram)
//   - request count (counter)
//
// Labels "backend" and "model" are read from the request context where the
// proxy handler stores them after backend selection. If absent (e.g. for
// /healthz), they default to "unknown".
func NewMiddleware(log zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rr := &responseRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // default; overridden by WriteHeader
			}

			next.ServeHTTP(rr, r)

			duration := time.Since(start)

			// TODO: read backend/model from context after proxy handler populates them
			backend := backendFromCtx(r)
			model := modelFromCtx(r)
			statusStr := strconv.Itoa(rr.statusCode)

			RequestDuration.
				WithLabelValues(backend, model, statusStr).
				Observe(duration.Seconds())

			RequestTotal.
				WithLabelValues(backend, model, statusStr).
				Inc()

			log.Debug().
				Str("backend", backend).
				Str("model", model).
				Str("status", statusStr).
				Dur("duration_ms", duration).
				Msg("metrics: request recorded")
		})
	}
}

// backendFromCtx extracts the backend label from the request context.
// Returns "unknown" if not set (routes that don't go through the proxy).
func backendFromCtx(r *http.Request) string {
	// TODO: define and use a typed context key shared with proxy package
	return "unknown"
}

// modelFromCtx extracts the model label from the request context.
func modelFromCtx(r *http.Request) string {
	// TODO: define and use a typed context key shared with proxy package
	return "unknown"
}
