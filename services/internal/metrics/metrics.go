// Package metrics holds the Prometheus metrics shared by the lab services.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// DurationBuckets resolve both fast requests and the 2-4 s range.
var DurationBuckets = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 1.5, 2, 2.5, 3, 3.5, 4, 5, 10}

// HTTP records request counts and durations for one service.
type HTTP struct {
	Registry *prometheus.Registry
	service  string
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

// NewHTTP registers the request metrics for service on a new registry.
func NewHTTP(service string) *HTTP {
	m := &HTTP{
		Registry: prometheus.NewRegistry(),
		service:  service,
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP requests served, by route template and status code.",
		}, []string{"service", "endpoint", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration, by route template.",
			Buckets: DurationBuckets,
		}, []string{"service", "endpoint"}),
	}
	m.Registry.MustRegister(m.requests, m.duration)
	return m
}

// Observe records one request. endpoint must be a route template.
func (m *HTTP) Observe(endpoint string, status int, d time.Duration) {
	m.requests.WithLabelValues(m.service, endpoint, strconv.Itoa(status)).Inc()
	m.duration.WithLabelValues(m.service, endpoint).Observe(d.Seconds())
}

// Handler serves the registry in the Prometheus exposition format.
func (m *HTTP) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{})
}
