package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// package middleware (white-box), not middleware_test — needed to inject
// a user ID via the unexported ctxKeyUserID without adding an
// exported-just-for-tests setter to the public API.

func withUserID(r *http.Request, userID int64) *http.Request {
	ctx := context.WithValue(r.Context(), ctxKeyUserID{}, userID)
	return r.WithContext(ctx)
}

func newTestRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func TestRateLimit_AllowsUnderLimitAndBlocksOver(t *testing.T) {
	client := newTestRedisClient(t)
	limiter := RateLimit(client, 3, time.Minute)
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	do := func(userID int64) *httptest.ResponseRecorder {
		req := withUserID(httptest.NewRequest(http.MethodGet, "/tasks", nil), userID)
		rw := httptest.NewRecorder()
		handler.ServeHTTP(rw, req)
		return rw
	}

	for i := 0; i < 3; i++ {
		rw := do(1)
		require.Equal(t, http.StatusOK, rw.Code, "request %d must be allowed", i+1)
	}

	rw := do(1)
	require.Equal(t, http.StatusTooManyRequests, rw.Code, "4th request over the limit must be rejected")
	require.NotEmpty(t, rw.Header().Get("Retry-After"))

	rwOtherUser := do(2)
	require.Equal(t, http.StatusOK, rwOtherUser.Code, "a different user must have an independent limit")
}

func TestRateLimit_NoUserIDPassesThrough(t *testing.T) {
	client := newTestRedisClient(t)
	limiter := RateLimit(client, 1, time.Minute)
	called := false
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/tasks", nil) // no user ID in context
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	require.True(t, called, "must pass through when Auth didn't run first")
	require.Equal(t, http.StatusOK, rw.Code)
}

func TestRateLimitByIP_AllowsUnderLimitAndBlocksOver(t *testing.T) {
	client := newTestRedisClient(t)
	limiter := RateLimitByIP(client, 3, time.Minute)
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	do := func(remoteAddr string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/register", nil)
		req.RemoteAddr = remoteAddr
		rw := httptest.NewRecorder()
		handler.ServeHTTP(rw, req)
		return rw
	}

	for i := 0; i < 3; i++ {
		rw := do("203.0.113.1:5000")
		require.Equal(t, http.StatusOK, rw.Code, "request %d must be allowed", i+1)
	}

	rw := do("203.0.113.1:5001") // same IP, different ephemeral port
	require.Equal(t, http.StatusTooManyRequests, rw.Code, "4th request from the same IP must be rejected")
	require.NotEmpty(t, rw.Header().Get("Retry-After"))

	rwOtherIP := do("203.0.113.2:5000")
	require.Equal(t, http.StatusOK, rwOtherIP.Code, "a different IP must have an independent limit")
}

func TestRateLimitByIP_NoAuthRequired(t *testing.T) {
	client := newTestRedisClient(t)
	limiter := RateLimitByIP(client, 1, time.Minute)
	called := false
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/login", nil) // no user ID in context, unlike RateLimit
	req.RemoteAddr = "203.0.113.5:5000"
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	require.True(t, called)
	require.Equal(t, http.StatusOK, rw.Code)
}

func TestRateLimit_RedisDownFailsOpen(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	mr.Close()

	limiter := RateLimit(client, 1, time.Minute)
	called := false
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := withUserID(httptest.NewRequest(http.MethodGet, "/tasks", nil), 1)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	require.True(t, called, "a Redis outage must not block requests")
	require.Equal(t, http.StatusOK, rw.Code)
}
