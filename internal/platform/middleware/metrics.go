package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds the three ТЗ-required series: request count, error count,
// and response time. Constructed with an injectable prometheus.Registerer
// (prometheus.DefaultRegisterer in production, a fresh prometheus.Registry
// in tests) so tests don't collide with global state across runs.
type Metrics struct {
	requestsTotal *prometheus.CounterVec
	errorsTotal   *prometheus.CounterVec
	duration      *prometheus.HistogramVec
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		requestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests processed, labeled by method, route pattern, and status code.",
		}, []string{"method", "path", "status"}),
		errorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_request_errors_total",
			Help: "Total HTTP requests that resulted in a server error (status >= 500).",
		}, []string{"method", "path", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds, labeled by method and route pattern.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path"}),
	}
	reg.MustRegister(m.requestsTotal, m.errorsTotal, m.duration)
	return m
}

// Middleware records all three series for every request. Mounted before
// auth (top-level) so it covers public endpoints too, not just
// authenticated ones.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		path := routePattern(r)
		status := strconv.Itoa(rec.status)

		m.requestsTotal.WithLabelValues(r.Method, path, status).Inc()
		if rec.status >= http.StatusInternalServerError {
			m.errorsTotal.WithLabelValues(r.Method, path, status).Inc()
		}
		m.duration.WithLabelValues(r.Method, path).Observe(time.Since(start).Seconds())
	})
}

// routePattern reads chi's matched route pattern (e.g. "/tasks/{id}")
// instead of the raw path, to keep the "path" label's cardinality bounded.
// Populated by chi's router as it matches, so it's only readable after
// next.ServeHTTP has run.
func routePattern(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if pattern := rctx.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	return "unmatched"
}
