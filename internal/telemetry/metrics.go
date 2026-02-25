package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// HTTPRequestDuration tracks HTTP request latency.
var HTTPRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "ticketowl",
		Subsystem: "api",
		Name:      "request_duration_seconds",
		Help:      "HTTP request duration in seconds.",
		Buckets:   prometheus.DefBuckets,
	},
	[]string{"method", "path", "status"},
)

// TicketRequestsTotal counts ticket-related API requests.
var TicketRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "ticketowl",
		Name:      "ticket_requests_total",
		Help:      "Total number of ticket-related API requests.",
	},
	[]string{"operation"},
)

// SLABreachesTotal counts SLA breaches detected.
var SLABreachesTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: "ticketowl",
		Name:      "sla_breaches_total",
		Help:      "Total number of SLA breaches detected.",
	},
)

// ZammadRequestDuration tracks Zammad API call latency.
var ZammadRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "ticketowl",
		Name:      "zammad_request_duration_seconds",
		Help:      "Zammad API request duration in seconds.",
		Buckets:   []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	},
	[]string{"method", "endpoint"},
)

// WorkerPollDuration tracks SLA polling loop duration.
var WorkerPollDuration = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Namespace: "ticketowl",
		Name:      "worker_poll_duration_seconds",
		Help:      "SLA polling loop duration in seconds.",
		Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
	},
)

// NewMetricsRegistry creates a Prometheus registry with default and custom collectors.
func NewMetricsRegistry() *prometheus.Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		HTTPRequestDuration,
		TicketRequestsTotal,
		SLABreachesTotal,
		ZammadRequestDuration,
		WorkerPollDuration,
	)
	return reg
}
