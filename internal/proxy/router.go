// Package proxy implements backend selection logic for the LLM proxy.
// The Router is a pure, stateful struct — it holds backend configuration
// and selection state (e.g. round-robin counter) but has no global variables.
package proxy

import (
	"fmt"
	"sync/atomic"
)

// BackendConfig mirrors cmd/api.BackendConfig to avoid import cycles.
// main.go converts its config.BackendConfig into this type during wiring.
type BackendConfig struct {
	Name    string
	BaseURL string
	Models  []string
	Weight  int
}

// Router selects an upstream backend for a given model identifier.
// It is safe for concurrent use.
type Router struct {
	// backends holds all registered backends indexed by position.
	backends []BackendConfig

	// modelIndex maps a model ID to the slice of backend indices that serve it.
	modelIndex map[string][]int

	// counter is used for round-robin selection; incremented atomically.
	counter atomic.Uint64
}

// NewRouter constructs a Router from a slice of backend configs.
// Returns an error if backends is empty.
func NewRouter(backends []BackendConfig) (*Router, error) {
	if len(backends) == 0 {
		return nil, fmt.Errorf("proxy: router: at least one backend is required")
	}

	// Build model → backend-indices index
	idx := make(map[string][]int, len(backends)*2)
	for i, b := range backends {
		for _, model := range b.Models {
			idx[model] = append(idx[model], i)
		}
	}

	return &Router{
		backends:   backends,
		modelIndex: idx,
	}, nil
}

// Select returns the backend that should handle a request for model.
// Selection strategy: round-robin across backends serving the model.
// Returns an error if no backend is configured for the model.
func (r *Router) Select(model string) (BackendConfig, error) {
	indices, ok := r.modelIndex[model]
	if !ok || len(indices) == 0 {
		return BackendConfig{}, fmt.Errorf("proxy: router: no backend configured for model %q", model)
	}

	// Atomically increment and mod-select to avoid holding a lock.
	n := r.counter.Add(1)
	chosen := indices[int(n)%len(indices)]
	return r.backends[chosen], nil
}
