package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/zhuk/team-task-service/internal/apperr"
	"github.com/zhuk/team-task-service/internal/platform/httpx"
	"github.com/zhuk/team-task-service/internal/platform/jwtauth"
)

type ctxKeyUserID struct{}

const bearerPrefix = "Bearer "

// Auth validates the JWT in the Authorization header and stores the
// authenticated user ID in the request context. Mount this only on route
// groups that require authentication.
func Auth(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, bearerPrefix) {
				httpx.WriteError(w, apperr.Unauthorized("missing or malformed Authorization header"))
				return
			}

			tokenString := strings.TrimPrefix(header, bearerPrefix)
			claims, err := jwtauth.Parse(secret, tokenString)
			if err != nil {
				httpx.WriteError(w, apperr.Unauthorized("invalid or expired token"))
				return
			}

			ctx := context.WithValue(r.Context(), ctxKeyUserID{}, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(ctxKeyUserID{}).(int64)
	return id, ok
}
