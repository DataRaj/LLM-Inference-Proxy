// Package retry provides a pure exponential-backoff helper with full jitter.
// No network calls, no global state — only math.
package retry

import (
	"math"
	"math/rand"
	"time"
)

// Backoff returns a jittered backoff duration for the given retry attempt.
//
// Algorithm: "Full Jitter" as described in the AWS Architecture Blog.
//   sleep = random_between(0, min(cap, base * 2^attempt))
//
// Parameters:
//   - attempt: zero-indexed retry count (0 = first retry)
//   - base:    minimum backoff (e.g. 200 ms)
//   - cap:     maximum backoff ceiling (e.g. 10 s)
//
// The caller is responsible for sleeping:
//
//	time.Sleep(retry.Backoff(attempt, 200*time.Millisecond, 10*time.Second))
func Backoff(attempt int, base, cap time.Duration) time.Duration {
	// Clamp attempt to prevent float64 overflow on large retry counts.
	if attempt > 62 {
		attempt = 62
	}

	// ceiling = min(cap, base * 2^attempt)
	ceiling := time.Duration(math.Min(
		float64(cap),
		float64(base)*math.Pow(2, float64(attempt)),
	))

	// Full jitter: uniform random in [0, ceiling)
	// rand.Int63n panics on n <= 0 so guard that edge case.
	if ceiling <= 0 {
		return 0
	}

	return time.Duration(rand.Int63n(int64(ceiling))) //nolint:gosec // non-crypto jitter is intentional
}
