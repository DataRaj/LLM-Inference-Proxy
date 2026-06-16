// Package ratelimit defines the rate-limiting interface used throughout the proxy.
// Internal packages depend on this interface, never on a concrete Redis implementation.
package ratelimit

import "context"

// Limiter is the single-method contract for sliding-window rate limiting.
// Implementations must be safe for concurrent use.
//
// Allow returns (true, nil) when the request is within quota.
// Allow returns (false, nil) when the request exceeds the quota — the caller
// should respond with HTTP 429; this is not an error condition.
// Allow returns (false, err) on infrastructure failures (e.g. Redis timeout).
type Limiter interface {
	// Allow reports whether the key is permitted to make a request.
	// key is typically a hashed API key or client IP.
	Allow(ctx context.Context, key string) (bool, error)
}
