package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	promclient "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/zhuk/team-task-service/internal/platform/middleware"
)

func findMetricFamily(t *testing.T, families []*promclient.MetricFamily, name string) *promclient.MetricFamily {
	t.Helper()
	f := findMetricFamilyOptional(families, name)
	if f == nil {
		t.Fatalf("metric family %q not found", name)
	}
	return f
}

// findMetricFamilyOptional returns nil instead of failing the test — a
// CounterVec/HistogramVec with zero observed label combinations is
// entirely absent from Gather()'s output, not present-with-no-metrics, so
// "never incremented" must be asserted via a nil result here.
func findMetricFamilyOptional(families []*promclient.MetricFamily, name string) *promclient.MetricFamily {
	for _, f := range families {
		if f.GetName() == name {
			return f
		}
	}
	return nil
}

func labelValue(m *promclient.Metric, name string) string {
	for _, l := range m.GetLabel() {
		if l.GetName() == name {
			return l.GetValue()
		}
	}
	return ""
}

func TestMetrics_RecordsRequestAndDuration(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := middleware.NewMetrics(reg)

	r := chi.NewRouter()
	r.With(m.Middleware).Get("/api/v1/tasks/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/42", nil)
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)

	families, err := reg.Gather()
	require.NoError(t, err)

	requests := findMetricFamily(t, families, "http_requests_total")
	require.Len(t, requests.GetMetric(), 1)
	metric := requests.GetMetric()[0]
	require.Equal(t, float64(1), metric.GetCounter().GetValue())
	require.Equal(t, "/api/v1/tasks/{id}", labelValue(metric, "path"), "must use chi's route pattern, not the raw path")
	require.Equal(t, "201", labelValue(metric, "status"))

	duration := findMetricFamily(t, families, "http_request_duration_seconds")
	require.Len(t, duration.GetMetric(), 1)
	require.Equal(t, uint64(1), duration.GetMetric()[0].GetHistogram().GetSampleCount())

	errors := findMetricFamilyOptional(families, "http_request_errors_total")
	require.Nil(t, errors, "a 2xx response must not create any http_request_errors_total series")
}

func TestMetrics_RecordsServerErrors(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := middleware.NewMetrics(reg)

	r := chi.NewRouter()
	r.With(m.Middleware).Get("/boom", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)

	families, err := reg.Gather()
	require.NoError(t, err)

	errors := findMetricFamily(t, families, "http_request_errors_total")
	require.Len(t, errors.GetMetric(), 1)
	require.Equal(t, float64(1), errors.GetMetric()[0].GetCounter().GetValue())
}

func TestMetrics_UnmatchedRouteUsesFallbackLabel(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := middleware.NewMetrics(reg)

	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil) // no chi router, no RouteContext
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	families, err := reg.Gather()
	require.NoError(t, err)

	requests := findMetricFamily(t, families, "http_requests_total")
	require.Len(t, requests.GetMetric(), 1)
	require.Equal(t, "unmatched", labelValue(requests.GetMetric()[0], "path"))
}
