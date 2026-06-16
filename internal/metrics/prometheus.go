// Package metrics registers Prometheus collectors for the proxy.
// Registration happens in the package init() so that the collectors exist
// before any HTTP handler is wired.
//
// This is the ONLY file in the project that uses init().
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Namespace used for all metric names produced by this service.
const namespace = "llm_proxy"

// RequestDuration tracks per-request latency, broken down by backend, model,
// and the HTTP status code returned to the caller.
//
// Metric name: llm_proxy_request_duration_seconds
var RequestDuration *prometheus.HistogramVec

// RequestTotal counts completed requests by backend, model, and status code.
//
// Metric name: llm_proxy_requests_total
var RequestTotal *prometheus.CounterVec

// UpstreamRetries counts upstream retry attempts.
//
// Metric name: llm_proxy_upstream_retries_total
var UpstreamRetries *prometheus.CounterVec

func init() {
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "request_duration_seconds",
			Help:      "End-to-end request latency from client receipt to first byte sent, labeled by backend, model, and status code.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"backend", "model", "status_code"},
	)

	RequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "requests_total",
			Help:      "Total number of proxy requests completed.",
		},
		[]string{"backend", "model", "status_code"},
	)

	UpstreamRetries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "upstream_retries_total",
			Help:      "Number of upstream retry attempts, labeled by backend.",
		},
		[]string{"backend"},
	)
}
