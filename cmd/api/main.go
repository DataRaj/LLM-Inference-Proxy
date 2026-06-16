
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"

	"github.com/dataraj/llm-proxy/internal/auth"
	"github.com/dataraj/llm-proxy/internal/metrics"
	"github.com/dataraj/llm-proxy/internal/middleware"
	"github.com/dataraj/llm-proxy/internal/proxy"
	"github.com/dataraj/llm-proxy/internal/ratelimit"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "startup error: %v\n", err)
		os.Exit(1)
	}
}

// run is separated from main so that deferred cleanup runs before os.Exit.
func run() error {
	// ── 1. Config ────────────────────────────────────────────────────────────
	cfg, err := Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// ── 2. Logger ────────────────────────────────────────────────────────────
	logger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("service", "llm-proxy").
		Logger()

	logger.Info().Msg("configuration loaded")

	// ── 3. Wire dependencies ─────────────────────────────────────────────────

	// Redis-backed rate limiter
	limiter, err := ratelimit.NewRedisLimiter(
		cfg.RedisURL,
		cfg.RateLimitWindowSec,
		cfg.RateLimitMaxRequests,
	)
	if err != nil {
		return fmt.Errorf("build redis limiter: %w", err)
	}

	// Auth middleware — closed over the key list from config
	authMiddleware := auth.NewMiddleware(cfg.APIKeys, logger)

	// Rate-limit middleware
	rateLimitMiddleware := ratelimit.NewMiddleware(limiter, logger)

	// Metrics middleware (histogram/counter already registered in metrics init)
	metricsMiddleware := metrics.NewMiddleware(logger)

	// Proxy handler — router + upstream client
	// Convert config.BackendConfig → proxy.BackendConfig at the wiring boundary.
	proxyRouter, err := proxy.NewRouter(toProxyBackends(cfg.Backends))
	if err != nil {
		return fmt.Errorf("build proxy router: %w", err)
	}

	upstreamClient := proxy.NewUpstreamClient(http.Client{
		Timeout: cfg.UpstreamTimeoutSec,
	}, cfg.MaxRetries, logger)

	proxyHandler := proxy.NewHandler(proxyRouter, upstreamClient, logger)

	// ── 4. HTTP router ───────────────────────────────────────────────────────
	r := chi.NewRouter()

	// Global middleware chain: metrics → auth → ratelimit
	r.Use(middleware.Chain(
		metricsMiddleware,
		authMiddleware,
		rateLimitMiddleware,
	))

	// Protected routes
	r.Post("/v1/chat/completions", proxyHandler.ServeHTTP)

	// Unprotected routes — registered outside the chain
	r.Get("/healthz", healthzHandler)
	r.Handle("/metrics", promhttp.Handler())

	// ── 5. HTTP server + graceful shutdown ───────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      r,
		ReadTimeout:  cfg.ReadTimeoutSec,
		WriteTimeout: cfg.WriteTimeoutSec,
		IdleTimeout:  cfg.IdleTimeoutSec,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info().Str("addr", srv.Addr).Msg("server listening")
		serverErr <- srv.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		logger.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	}

	drainCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(drainCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	logger.Info().Msg("server stopped cleanly")
	return nil
}

// healthzHandler returns 200 OK with no auth required.
func healthzHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// toProxyBackends converts the cmd-layer BackendConfig slice into the proxy
// package's equivalent type. The conversion lives here — at the wiring
// boundary — to prevent the proxy package from importing cmd types.
func toProxyBackends(in []BackendConfig) []proxy.BackendConfig {
	out := make([]proxy.BackendConfig, len(in))
	for i, b := range in {
		out[i] = proxy.BackendConfig{
			Name:    b.Name,
			BaseURL: b.BaseURL,
			Models:  b.Models,
			Weight:  b.Weight,
		}
	}
	return out
}
