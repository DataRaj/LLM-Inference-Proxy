// Package proxy implements the HTTP handler for the /v1/chat/completions endpoint.
// It decodes the request, selects an upstream backend via the Router,
// and delegates the actual HTTP call to the UpstreamClient.
package proxy

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/dataraj/llm-proxy/internal/models"
)

// Handler is the http.Handler for /v1/chat/completions.
// It holds references to the router and upstream client, both injected at construction.
type Handler struct {
	router   *Router
	upstream *UpstreamClient
	log      zerolog.Logger
}

// NewHandler constructs a Handler. Both router and upstream must be non-nil.
func NewHandler(router *Router, upstream *UpstreamClient, log zerolog.Logger) *Handler {
	return &Handler{
		router:   router,
		upstream: upstream,
		log:      log,
	}
}

// ServeHTTP implements http.Handler.
//
// Request lifecycle:
//  1. Decode JSON body into ChatCompletionRequest.
//  2. Validate required fields (model, messages).
//  3. Check http.Flusher if stream=true — return 500 if not supported.
//  4. Select backend via Router.
//  5. Build upstream *http.Request with auth header forwarding.
//  6. Delegate to UpstreamClient.Do — which handles retries and streaming.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// ── 1. Decode request body ────────────────────────────────────────────────
	var req models.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error().Err(err).Msg("handler: failed to decode request body")
		http.Error(w, "bad request: invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// ── 2. Validate ───────────────────────────────────────────────────────────
	if req.Model == "" {
		http.Error(w, "bad request: model is required", http.StatusBadRequest)
		return
	}
	if len(req.Messages) == 0 {
		http.Error(w, "bad request: messages cannot be empty", http.StatusBadRequest)
		return
	}

	// ── 3. Streaming capability check ────────────────────────────────────────
	if req.Stream {
		if _, ok := w.(http.Flusher); !ok {
			h.log.Error().Msg("handler: ResponseWriter does not implement http.Flusher")
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}
	}

	// ── 4. Backend selection ──────────────────────────────────────────────────
	backend, err := h.router.Select(req.Model)
	if err != nil {
		h.log.Error().Err(err).Str("model", req.Model).Msg("handler: no backend for model")
		http.Error(w, "no backend available for requested model", http.StatusBadGateway)
		return
	}

	h.log.Info().
		Str("backend", backend.Name).
		Str("model", req.Model).
		Bool("stream", req.Stream).
		Msg("handler: routing request")

	// ── 5. Build upstream request ─────────────────────────────────────────────
	// TODO: construct *http.Request targeting backend.BaseURL + "/v1/chat/completions"
	//   - re-encode body (or pipe original reader with body buffering)
	//   - forward Authorization header if the upstream requires its own key
	//   - attach request context

	// ── 6. Delegate to UpstreamClient ────────────────────────────────────────
	// TODO: call h.upstream.Do(r.Context(), upstreamReq, w, backend.Name)
}
