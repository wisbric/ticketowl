package telemetry

import "github.com/prometheus/client_golang/prometheus"

var TicketRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "ticketowl",
		Name:      "ticket_requests_total",
		Help:      "Total number of ticket-related API requests.",
	},
	[]string{"operation"},
)

var SLABreachesTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: "ticketowl",
		Name:      "sla_breaches_total",
		Help:      "Total number of SLA breaches detected.",
	},
)

var ZammadRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "ticketowl",
		Name:      "zammad_request_duration_seconds",
		Help:      "Zammad API request duration in seconds.",
		Buckets:   []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	},
	[]string{"method", "endpoint"},
)

var WorkerPollDuration = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Namespace: "ticketowl",
		Name:      "worker_poll_duration_seconds",
		Help:      "SLA polling loop duration in seconds.",
		Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
	},
)

// All returns all TicketOwl-specific metrics for registration.
func All() []prometheus.Collector {
	return []prometheus.Collector{
		TicketRequestsTotal,
		SLABreachesTotal,
		ZammadRequestDuration,
		WorkerPollDuration,
	}
}
