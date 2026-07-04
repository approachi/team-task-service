// Package router assembles the chi router: middleware chain, route table,
// Swagger UI, and metrics endpoint.
package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/zhuk/team-task-service/internal/handler"
	"github.com/zhuk/team-task-service/internal/platform/middleware"
)

type Handlers struct {
	Auth   *handler.AuthHandler
	Team   *handler.TeamHandler
	Task   *handler.TaskHandler
	Report *handler.ReportHandler
}

// Config bundles the cross-cutting infra New needs beyond the handlers:
// the JWT secret for Auth, a Redis client for per-user rate limiting, and
// a Prometheus registry for metrics + the /metrics scrape endpoint.
type Config struct {
	JWTSecret   []byte
	RedisClient *redis.Client
	MetricsReg  *prometheus.Registry
}

func New(h Handlers, cfg Config) http.Handler {
	r := chi.NewRouter()

	metrics := middleware.NewMetrics(cfg.MetricsReg)

	r.Use(middleware.RequestID)
	r.Use(middleware.LimitBody)
	r.Use(middleware.Logging)
	r.Use(metrics.Middleware) // pre-auth: covers public endpoints too
	// Recoverer must be innermost (closest to the handlers): if it ran
	// outside Logging/Metrics instead, a panic would unwind straight past
	// their post-next.ServeHTTP bookkeeping (the access-log line, the
	// requests/errors/duration series) before Recoverer's deferred recover()
	// ever caught it — the worst class of failure would go unlogged and
	// unmetered.
	r.Use(middleware.Recoverer)
	// TODO: добавить CORS-middleware (allowlist origin из конфига) и
	// заголовки безопасности (X-Content-Type-Options, Referrer-Policy) —
	// сейчас их нет вовсе, см. docs/COVER_LETTER.md, раздел "Надёжность и
	// безопасность".

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))
	r.Handle("/metrics", promhttp.HandlerFor(cfg.MetricsReg, promhttp.HandlerOpts{}))

	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			// /register and /login have no authenticated user to key
			// RateLimit on, so they get their own IP-keyed limiter — see
			// docs/COVER_LETTER.md. Without this they were wide open to
			// credential-stuffing and mass-registration.
			r.Use(middleware.RateLimitByIP(cfg.RedisClient, middleware.DefaultAuthRateLimitPerMinute, time.Minute))
			r.Post("/register", h.Auth.Register)
			r.Post("/login", h.Auth.Login)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))
			r.Use(middleware.RateLimit(cfg.RedisClient, middleware.DefaultRateLimitPerMinute, time.Minute))

			r.Post("/teams", h.Team.Create)
			r.Get("/teams", h.Team.List)
			r.Post("/teams/{id}/invite", h.Team.Invite)

			r.Post("/tasks", h.Task.Create)
			r.Get("/tasks", h.Task.List)
			r.Put("/tasks/{id}", h.Task.Update)
			r.Get("/tasks/{id}/history", h.Task.History)
			// TODO: нет DELETE /tasks/{id} и нет удаления/исключения участника
			// из команды — эндпоинты не реализованы (см. docs/COVER_LETTER.md,
			// раздел "Функциональность"). Решить: мягкое удаление (deleted_at)
			// или жёсткое, и что делать с задачами удалённого участника при
			// исключении.

			r.Get("/reports/teams-summary", h.Report.TeamsSummary)
			r.Get("/reports/top-creators", h.Report.TopCreators)
			r.Get("/reports/orphaned-assignees", h.Report.OrphanedAssignees)
		})
	})

	return r
}
