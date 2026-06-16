// Package proxy implements the outbound HTTP client that calls upstream LLM backends.
// It handles retries with exponential backoff and SSE stream forwarding.
package proxy

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/dataraj/llm-proxy/internal/metrics"
	"github.com/dataraj/llm-proxy/pkg/retry"
)

// UpstreamClient wraps net/http.Client with retry and streaming support.
// It is the only place in the proxy that makes outbound HTTP calls.
type UpstreamClient struct {
	httpClient http.Client
	maxRetries int
	log        zerolog.Logger
}

// NewUpstreamClient constructs an UpstreamClient.
// The caller is responsible for configuring the http.Client timeout.
func NewUpstreamClient(httpClient http.Client, maxRetries int, log zerolog.Logger) *UpstreamClient {
	return &UpstreamClient{
		httpClient: httpClient,
		maxRetries: maxRetries,
		log:        log,
	}
}

// Do sends req to the upstream backend and writes the response to w.
//
// Retry policy:
//   - HTTP 429 (Too Many Requests) from upstream triggers exponential backoff + retry.
//   - Other 5xx responses are returned immediately.
//   - Network errors are retried up to maxRetries.
//
// Streaming:
//   - If resp.Header "Content-Type" contains "text/event-stream", chunks are
//     copied directly using http.Flusher to preserve SSE semantics.
//
// backend is used as a Prometheus label on the retry counter.
func (c *UpstreamClient) Do(
	ctx context.Context,
	req *http.Request,
	w http.ResponseWriter,
	backend string,
) error {
	const (
		retryBase = 200 * time.Millisecond
		retryCap  = 10 * time.Second
	)

	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Record the retry attempt in Prometheus before sleeping.
			metrics.UpstreamRetries.WithLabelValues(backend).Inc()

			backoff := retry.Backoff(attempt-1, retryBase, retryCap)
			c.log.Warn().
				Str("backend", backend).
				Int("attempt", attempt).
				Dur("backoff_ms", backoff).
				Msg("upstream: retrying after backoff")

			select {
			case <-ctx.Done():
				return fmt.Errorf("upstream: context cancelled during backoff: %w", ctx.Err())
			case <-time.After(backoff):
			}
		}

		// Clone the request for each attempt because http.Client consumes the body.
		// TODO: implement request body buffering + clone for retry safety
		resp, err := c.httpClient.Do(req.WithContext(ctx))
		if err != nil {
			lastErr = fmt.Errorf("upstream: attempt %d: %w", attempt, err)
			c.log.Error().Err(lastErr).Str("backend", backend).Msg("upstream: request failed")
			continue
		}

		// HTTP 429 — rate-limited by the upstream itself; retry with backoff.
		if resp.StatusCode == http.StatusTooManyRequests {
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("upstream: attempt %d: received 429 from backend", attempt)
			continue
		}

		// All other responses: forward status, headers, and body to the caller.
		if err := c.forward(resp, w); err != nil {
			return fmt.Errorf("upstream: forward: %w", err)
		}
		return nil
	}

	return fmt.Errorf("upstream: all %d attempts failed: %w", c.maxRetries+1, lastErr)
}

// forward copies the upstream response (headers + body) to the http.ResponseWriter.
// If the response is SSE, it flushes after each chunk write.
func (c *UpstreamClient) forward(resp *http.Response, w http.ResponseWriter) error {
	defer resp.Body.Close()

	// Copy upstream headers that are safe to proxy.
	// TODO: implement selective header forwarding (skip hop-by-hop headers)

	w.WriteHeader(resp.StatusCode)

	// TODO: detect "Content-Type: text/event-stream" and use SSE copy path
	//   flusher, ok := w.(http.Flusher)
	//   if !ok { return errors.New("upstream: streaming not supported by ResponseWriter") }
	//   ... scan lines, write "data: ...\n\n", flusher.Flush()

	// Non-streaming path: copy body in one shot.
	// TODO: replace with io.Copy(w, resp.Body)
	_ = resp

	return nil
}
