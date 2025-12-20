package server

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	reqCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "requests_total"},
		[]string{"method", "status"},
	)

	latency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "request_latency_seconds"},
		[]string{"method"},
	)
)

func init() {
	prometheus.MustRegister(reqCount, latency)
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

func ObserveLatency(method string, start time.Time) {
	latency.WithLabelValues(method).Observe(time.Since(start).Seconds())
}

func IncSuccess(method string) {
	reqCount.WithLabelValues(method, "success").Inc()
}

func IncError(method string) {
	reqCount.WithLabelValues(method, "error").Inc()
}

func IncNotFound(method string) {
	reqCount.WithLabelValues(method, "not_found").Inc()
}
