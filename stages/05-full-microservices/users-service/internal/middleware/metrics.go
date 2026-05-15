package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func PrometheusMetrics(service string) func(http.Handler) http.Handler {
	requestDuration := promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:        "http_request_duration_seconds",
		Help:        "HTTP request latency in seconds",
		ConstLabels: prometheus.Labels{"service": service},
		Buckets:     []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
	}, []string{"method", "endpoint", "status"})

	requestsTotal := promauto.NewCounterVec(prometheus.CounterOpts{
		Name:        "http_requests_total",
		Help:        "Total number of HTTP requests",
		ConstLabels: prometheus.Labels{"service": service},
	}, []string{"method", "endpoint", "status"})

	requestsInFlight := promauto.NewGauge(prometheus.GaugeOpts{
		Name:        "http_requests_in_flight",
		Help:        "Number of HTTP requests currently being processed",
		ConstLabels: prometheus.Labels{"service": service},
	})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/metrics" || r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			requestsInFlight.Inc()
			defer requestsInFlight.Dec()

			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = r.URL.Path
			}

			status := strconv.Itoa(rec.status)
			requestDuration.WithLabelValues(r.Method, route, status).Observe(time.Since(start).Seconds())
			requestsTotal.WithLabelValues(r.Method, route, status).Inc()
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}
