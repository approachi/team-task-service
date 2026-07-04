// Package httpserver runs the HTTP server with graceful shutdown: it stops
// accepting new connections on ctx cancellation, waits for in-flight
// requests to finish (bounded by a timeout), then closes the supplied
// resources (DB pool, Redis client, ...).
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type Config struct {
	Addr            string
	Handler         http.Handler
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

func Run(ctx context.Context, cfg Config, closers ...io.Closer) error {
	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      cfg.Handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("http server listening", "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen and serve: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	for _, c := range closers {
		if err := c.Close(); err != nil {
			slog.Error("close resource during shutdown", "error", err)
		}
	}

	slog.Info("shutdown complete")
	return nil
}
