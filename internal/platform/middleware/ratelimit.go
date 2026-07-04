package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/zhuk/team-task-service/internal/apperr"
	"github.com/zhuk/team-task-service/internal/platform/httpx"
)

// DefaultRateLimitPerMinute is the ТЗ-specified limit: 100 requests/minute
// per authenticated user.
const DefaultRateLimitPerMinute = 100

// DefaultAuthRateLimitPerMinute caps unauthenticated attempts against
// /register and /login per client IP. Deliberately stricter than the
// per-user limit: these endpoints have no user ID to key on and are the
// service's brute-force/credential-stuffing/mass-registration surface.
const DefaultAuthRateLimitPerMinute = 20

// RateLimit enforces a fixed-window request count per authenticated user,
// backed by Redis so the limit holds across multiple app instances. Mount
// only inside the authenticated route group, after Auth — it reads the
// user ID from context and does nothing if there isn't one.
func RateLimit(client *redis.Client, limit int, window time.Duration) func(http.Handler) http.Handler {
	return rateLimit(client, limit, window, func(r *http.Request) (string, bool) {
		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			// Auth didn't run first in the chain — nothing to key the
			// limit on. Should never happen given the router wiring.
			return "", false
		}
		return fmt.Sprintf("user:%d", userID), true
	})
}

// RateLimitByIP enforces a fixed-window request count per client IP,
// backed by Redis. Mount on pre-auth routes (/register, /login) where
// there is no authenticated user ID to key on — see RateLimit for the
// per-user variant used on the authenticated route group.
func RateLimitByIP(client *redis.Client, limit int, window time.Duration) func(http.Handler) http.Handler {
	return rateLimit(client, limit, window, func(r *http.Request) (string, bool) {
		return "ip:" + clientIP(r), true
	})
}

// clientIP returns the request's remote IP without its ephemeral port.
// Deliberately does not trust X-Forwarded-For or similar client-supplied
// headers: without a configured trusted-proxy allowlist, honoring them
// would let a client claim any key it likes and bypass the limit entirely.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// rateLimit is the shared fixed-window counter behind RateLimit and
// RateLimitByIP; keyFor supplies the per-request key (and may decline to
// rate-limit a request by returning ok=false).
func rateLimit(client *redis.Client, limit int, window time.Duration, keyFor func(r *http.Request) (key string, ok bool)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			keyPart, ok := keyFor(r)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			windowID := time.Now().Unix() / int64(window.Seconds())
			key := fmt.Sprintf("ratelimit:%s:%d", keyPart, windowID)

			count, err := client.Incr(r.Context(), key).Result()
			if err != nil {
				// Rate limiting is a protective measure, not a hard
				// dependency — a Redis outage should not take the API
				// down, so fail open.
				next.ServeHTTP(w, r)
				return
			}
			if count == 1 {
				client.Expire(r.Context(), key, window)
			}

			remaining := limit - int(count)
			if remaining < 0 {
				remaining = 0
			}
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

			if int(count) > limit {
				w.Header().Set("Retry-After", strconv.Itoa(int(window.Seconds())))
				httpx.WriteError(w, apperr.TooManyRequests("rate limit exceeded, try again later"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
