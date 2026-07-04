// Package main wires config, storage, services, handlers, and the HTTP
// server together, and runs the server with graceful shutdown.
//
// @title			Team Task Service API
// @version			1.0
// @description		REST API for team task management (Phase 1: auth, teams, tasks, audit history).
// @BasePath		/api/v1
// @securityDefinitions.apikey	BearerAuth
// @in				header
// @name			Authorization
// @description		Type "Bearer" followed by a space and the JWT token.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	_ "github.com/zhuk/team-task-service/docs"

	"github.com/zhuk/team-task-service/internal/config"
	"github.com/zhuk/team-task-service/internal/handler"
	"github.com/zhuk/team-task-service/internal/notify"
	"github.com/zhuk/team-task-service/internal/platform/breaker"
	"github.com/zhuk/team-task-service/internal/platform/cache"
	"github.com/zhuk/team-task-service/internal/platform/db"
	"github.com/zhuk/team-task-service/internal/platform/httpserver"
	"github.com/zhuk/team-task-service/internal/repository"
	"github.com/zhuk/team-task-service/internal/router"
	"github.com/zhuk/team-task-service/internal/service"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal startup error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(config.Path())
	if err != nil {
		return err
	}

	setupLogger(cfg.Log.Level)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbConn, err := db.Connect(cfg.DB)
	if err != nil {
		return err
	}

	redisClient, err := cache.Connect(ctx, cfg.Redis)
	if err != nil {
		return err
	}

	userRepo := repository.NewUserRepository(dbConn)
	teamRepo := repository.NewTeamRepository(dbConn)
	taskRepo := repository.NewTaskRepository(dbConn)
	analyticsRepo := repository.NewAnalyticsRepository(dbConn)

	// Cache-aside decorator over the team task list (see docs/COVER_LETTER.md,
	// "Ключевое техническое решение") — wired here only; TaskService is
	// unaware it exists.
	var cachedTaskRepo service.TaskRepository = repository.NewCachingTaskRepository(taskRepo, redisClient, repository.TaskListCacheTTL)

	// Circuit breaker decorator over the invite-email notifier (Phase 4) —
	// wired here only; TeamService is unaware it exists.
	var notifier notify.Notifier = breaker.NewNotifier(notify.NewLogNotifier(), breaker.NotifierSettings())

	authSvc := service.NewAuthService(userRepo, []byte(cfg.JWT.Secret), cfg.JWT.TTL.Duration)
	teamSvc := service.NewTeamService(teamRepo, userRepo, notifier)
	taskSvc := service.NewTaskService(cachedTaskRepo, teamRepo)
	analyticsSvc := service.NewAnalyticsService(analyticsRepo)

	handlers := router.Handlers{
		Auth:   handler.NewAuthHandler(authSvc),
		Team:   handler.NewTeamHandler(teamSvc),
		Task:   handler.NewTaskHandler(taskSvc),
		Report: handler.NewReportHandler(analyticsSvc),
	}

	mux := router.New(handlers, router.Config{
		JWTSecret:   []byte(cfg.JWT.Secret),
		RedisClient: redisClient,
		MetricsReg:  prometheus.NewRegistry(),
	})

	serverCfg := httpserver.Config{
		Addr:            cfg.HTTP.Addr,
		Handler:         mux,
		ReadTimeout:     time.Duration(cfg.HTTP.ReadTimeout.Duration),
		WriteTimeout:    time.Duration(cfg.HTTP.WriteTimeout.Duration),
		IdleTimeout:     time.Duration(cfg.HTTP.IdleTimeout.Duration),
		ShutdownTimeout: time.Duration(cfg.HTTP.ShutdownTimeout.Duration),
	}

	return httpserver.Run(ctx, serverCfg, dbConn, redisClient)
}

func setupLogger(level string) {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})))
}
