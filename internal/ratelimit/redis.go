// Package ratelimit provides a Redis-backed sliding-window rate limiter
// that satisfies the Limiter interface.
package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// RedisLimiter implements Limiter using a Redis sorted-set sliding window.
//
// Algorithm per request for key K:
//  1. ZREMRANGEBYSCORE K -inf (now - windowSec)   — evict expired entries
//  2. ZADD K now now                               — record this request
//  3. ZCARD K                                      — count in-window requests
//  4. EXPIRE K windowSec                           — auto-cleanup
//
// Steps 1-4 are pipelined in a single round-trip.
type RedisLimiter struct {
	client     *redis.Client
	windowSec  time.Duration
	maxReqs    int64
}

// NewRedisLimiter constructs a RedisLimiter and verifies connectivity.
// redisURL must be a valid redis:// or rediss:// URL.
func NewRedisLimiter(redisURL string, windowSec, maxRequests int) (*RedisLimiter, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("redis: parse url: %w", err)
	}

	client := redis.NewClient(opts)

	// Verify connectivity at startup so we fail fast rather than at first request.
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("redis: ping: %w", err)
	}

	return &RedisLimiter{
		client:    client,
		windowSec: time.Duration(windowSec) * time.Second,
		maxReqs:   int64(maxRequests),
	}, nil
}

// Allow implements Limiter.
// key should be a stable, hashed identifier for the caller (e.g. SHA-256 of the API key).
func (r *RedisLimiter) Allow(ctx context.Context, key string) (bool, error) {
	// TODO: implement Redis sorted-set sliding window pipeline
	// Sketch:
	//   now := time.Now()
	//   pipe := r.client.Pipeline()
	//   pipe.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(now.Add(-r.windowSec).UnixMilli(), 10))
	//   pipe.ZAdd(ctx, key, redis.Z{Score: float64(now.UnixMilli()), Member: now.UnixNano()})
	//   cardCmd := pipe.ZCard(ctx, key)
	//   pipe.Expire(ctx, key, r.windowSec)
	//   if _, err := pipe.Exec(ctx); err != nil { return false, err }
	//   return cardCmd.Val() <= r.maxReqs, nil
	return true, nil
}

// NewMiddleware returns an http.Handler middleware that enforces rate limits.
// The key is extracted from the Authorization header (after "Bearer ").
// On quota exceeded it writes HTTP 429 with a Retry-After header.
func NewMiddleware(l Limiter, log zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO: extract key from request context (set by auth middleware)
			// allowed, err := l.Allow(r.Context(), key)
			// if err != nil { ... 500 }
			// if !allowed { ... 429 }
			_ = l
			_ = log
			next.ServeHTTP(w, r)
		})
	}
}
