package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/zhuk/team-task-service/internal/platform/middleware"
)

func TestRecoverer_TurnsPanicIntoFiveHundred(t *testing.T) {
	handler := middleware.Recoverer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	require.Equal(t, http.StatusInternalServerError, rw.Code)
}

// TestRecoverer_MustBeInnermostForMetricsToSeePanics locks in the router's
// required middleware order: Metrics (and Logging, by the same mechanism)
// must wrap Recoverer, not the other way around. If Recoverer sat outside
// Metrics, a panic would unwind straight past Metrics' post-next.ServeHTTP
// bookkeeping before Recoverer's deferred recover() ever caught it, and this
// test would see zero recorded requests instead of one 5xx.
func TestRecoverer_MustBeInnermostForMetricsToSeePanics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := middleware.NewMetrics(reg)

	r := chi.NewRouter()
	r.Use(m.Middleware)
	r.Use(middleware.Recoverer)
	r.Get("/boom", func(_ http.ResponseWriter, _ *http.Request) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)

	require.Equal(t, http.StatusInternalServerError, rw.Code)

	families, err := reg.Gather()
	require.NoError(t, err)

	var errorsSeries, requestsSeries bool
	for _, f := range families {
		switch f.GetName() {
		case "http_request_errors_total":
			errorsSeries = true
			require.Equal(t, float64(1), f.GetMetric()[0].GetCounter().GetValue())
		case "http_requests_total":
			requestsSeries = true
			require.Equal(t, float64(1), f.GetMetric()[0].GetCounter().GetValue())
		}
	}
	require.True(t, requestsSeries, "the panicking request must still be counted in http_requests_total")
	require.True(t, errorsSeries, "the panicking request must still be counted in http_request_errors_total")
}
