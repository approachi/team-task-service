package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zhuk/team-task-service/internal/platform/middleware"
)

func TestLimitBody_AllowsBodyUnderLimit(t *testing.T) {
	handler := middleware.LimitBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"a@b.com"}`))
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)
}

func TestLimitBody_RejectsBodyOverLimit(t *testing.T) {
	handler := middleware.LimitBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	oversized := strings.NewReader(strings.Repeat("a", middleware.MaxRequestBodyBytes+1))
	req := httptest.NewRequest(http.MethodPost, "/register", oversized)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	require.Equal(t, http.StatusRequestEntityTooLarge, rw.Code, "reading past the limit must error out instead of allocating the full body")
}
