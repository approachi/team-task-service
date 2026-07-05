package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/zhuk/team-task-service/internal/apperr"
	"github.com/zhuk/team-task-service/internal/platform/httpx"
)

// Recoverer turns a panic anywhere downstream into a 500 error envelope
// instead of a crashed connection, logging the stack trace for diagnosis.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered",
					"request_id", RequestIDFromContext(r.Context()),
					"panic", rec,
					"stack", string(debug.Stack()),
				)
				httpx.WriteError(w, apperr.Internal(fmt.Errorf("panic: %v", rec)))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
