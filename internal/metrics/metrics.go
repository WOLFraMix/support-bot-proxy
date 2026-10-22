package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Counter: сколько всего запросов пришло на эндпоинт
	SupportRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "support_bot_requests_total",
			Help: "Total number of support requests processed",
		},
		[]string{"handler", "status"},
	)

	// Histogram: сколько времени занимает обработка запроса (в секундах)
	SupportRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "support_bot_request_duration_seconds",
			Help:    "Duration of support request processing in seconds",
			Buckets: prometheus.ExponentialBuckets(0.01, 2, 10), // 0.01s, 0.02s, ..., до ~10s
		},
		[]string{"handler"},
	)
)

func Register() {
	prometheus.MustRegister(SupportRequestsTotal)
	prometheus.MustRegister(SupportRequestDuration)
}
