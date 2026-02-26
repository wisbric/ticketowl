package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// HTTPRequestDuration tracks HTTP request latency. Shared across all services.
var HTTPRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "wisbric",
		Subsystem: "api",
		Name:      "request_duration_seconds",
		Help:      "HTTP request duration in seconds.",
		Buckets:   prometheus.DefBuckets,
	},
	[]string{"method", "path", "status"},
)

// NewMetricsRegistry creates a Prometheus registry with Go/process collectors,
// the shared HTTPRequestDuration metric, and any additional service-specific
// collectors passed as arguments.
func NewMetricsRegistry(extra ...prometheus.Collector) *prometheus.Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		HTTPRequestDuration,
	)
	for _, c := range extra {
		reg.MustRegister(c)
	}
	return reg
}
