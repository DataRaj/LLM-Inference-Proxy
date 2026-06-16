// Package main holds the application entrypoint and wiring.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// BackendConfig describes a single upstream LLM backend.
type BackendConfig struct {
	Name    string   // human-readable label, e.g. "openai-us-east"
	BaseURL string   // e.g. "https://api.openai.com"
	Models  []string // model IDs this backend serves, e.g. ["gpt-4o", "gpt-4-turbo"]
	Weight  int      // relative weight for weighted round-robin (1–100)
}

// Config holds every configurable value for the proxy.
// It is parsed once at startup via Load() and passed by value or pointer —
// internal packages never call os.Getenv themselves.
type Config struct {
	// HTTP server
	ServerPort      string
	ReadTimeoutSec  time.Duration
	WriteTimeoutSec time.Duration
	IdleTimeoutSec  time.Duration

	// Upstream
	UpstreamTimeoutSec time.Duration
	MaxRetries         int

	// Redis
	RedisURL string

	// Rate limiting
	RateLimitWindowSec  int
	RateLimitMaxRequests int

	// Auth — comma-separated API keys from env; resolved to a slice at load time
	APIKeys []string

	// Backends — JSON-encoded or structured via numbered env vars; resolved at load time
	// Example env pattern:
	//   BACKEND_0_NAME=openai-primary
	//   BACKEND_0_URL=https://api.openai.com
	//   BACKEND_0_MODELS=gpt-4o,gpt-4-turbo
	//   BACKEND_0_WEIGHT=10
	Backends []BackendConfig
}

// Load reads every configuration value from the environment and returns a
// fully-populated Config. Returns an error if any required variable is absent
// or unparseable.
func Load() (Config, error) {
	cfg := Config{}

	// --- server ---
	cfg.ServerPort = envStringRequired("SERVER_PORT")
	if cfg.ServerPort == "" {
		cfg.ServerPort = "8080" // sensible default
	}

	readSec, err := envDurationSec("READ_TIMEOUT_SEC", 5)
	if err != nil {
		return Config{}, fmt.Errorf("config: READ_TIMEOUT_SEC: %w", err)
	}
	cfg.ReadTimeoutSec = readSec

	writeSec, err := envDurationSec("WRITE_TIMEOUT_SEC", 60)
	if err != nil {
		return Config{}, fmt.Errorf("config: WRITE_TIMEOUT_SEC: %w", err)
	}
	cfg.WriteTimeoutSec = writeSec

	idleSec, err := envDurationSec("IDLE_TIMEOUT_SEC", 120)
	if err != nil {
		return Config{}, fmt.Errorf("config: IDLE_TIMEOUT_SEC: %w", err)
	}
	cfg.IdleTimeoutSec = idleSec

	// --- upstream ---
	upstreamSec, err := envDurationSec("UPSTREAM_TIMEOUT_SEC", 30)
	if err != nil {
		return Config{}, fmt.Errorf("config: UPSTREAM_TIMEOUT_SEC: %w", err)
	}
	cfg.UpstreamTimeoutSec = upstreamSec

	maxRetries, err := envInt("MAX_RETRIES", 3)
	if err != nil {
		return Config{}, fmt.Errorf("config: MAX_RETRIES: %w", err)
	}
	cfg.MaxRetries = maxRetries

	// --- redis ---
	cfg.RedisURL = envStringRequired("REDIS_URL")

	// --- rate limit ---
	windowSec, err := envInt("RATE_LIMIT_WINDOW_SEC", 60)
	if err != nil {
		return Config{}, fmt.Errorf("config: RATE_LIMIT_WINDOW_SEC: %w", err)
	}
	cfg.RateLimitWindowSec = windowSec

	maxReqs, err := envInt("RATE_LIMIT_MAX_REQUESTS", 100)
	if err != nil {
		return Config{}, fmt.Errorf("config: RATE_LIMIT_MAX_REQUESTS: %w", err)
	}
	cfg.RateLimitMaxRequests = maxReqs

	// --- auth ---
	cfg.APIKeys = envStringSlice("API_KEYS", ",")

	// --- backends ---
	backends, err := loadBackends()
	if err != nil {
		return Config{}, fmt.Errorf("config: backends: %w", err)
	}
	cfg.Backends = backends

	return cfg, nil
}

// loadBackends resolves numbered BACKEND_N_* env vars into []BackendConfig.
// It iterates indices 0..N until BACKEND_N_NAME is absent.
func loadBackends() ([]BackendConfig, error) {
	// TODO: implement numbered env var iteration
	// Pattern:
	//   for i := 0; ; i++ {
	//       name := os.Getenv(fmt.Sprintf("BACKEND_%d_NAME", i))
	//       if name == "" { break }
	//       ...
	//   }
	_ = strconv.Itoa // keep import used during scaffolding
	return nil, nil
}

// -------------------------------------------------------------------------
// internal helpers — os.Getenv wrappers, only called from Load()
// -------------------------------------------------------------------------

func envStringRequired(key string) string {
	return os.Getenv(key)
}

func envStringSlice(key, sep string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// envInt returns the integer value of key, or defaultVal if key is unset.
func envInt(key string, defaultVal int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultVal, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("must be an integer, got %q: %w", raw, err)
	}
	return v, nil
}

// envDurationSec returns a time.Duration for a key expressed in whole seconds.
func envDurationSec(key string, defaultSec int) (time.Duration, error) {
	secs, err := envInt(key, defaultSec)
	if err != nil {
		return 0, err
	}
	return time.Duration(secs) * time.Second, nil
}
