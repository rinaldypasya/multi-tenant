package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	WorkerProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "worker_messages_processed_total",
			Help: "Total number of messages processed by workers",
		},
		[]string{"tenant"},
	)

	WorkerActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "worker_active_goroutines",
			Help: "Number of active worker goroutines per tenant",
		},
		[]string{"tenant"},
	)

	QueueDepth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "queue_depth",
			Help: "Current RabbitMQ queue depth per tenant",
		},
		[]string{"tenant"},
	)
)

// Init registers metrics with Prometheus
func Init() {
	prometheus.MustRegister(WorkerProcessed)
	prometheus.MustRegister(WorkerActive)
	prometheus.MustRegister(QueueDepth)
}

// Handler returns the Prometheus metrics HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}
