// Binary worker consumes audit events from RabbitMQ and persists them to Postgres.
// It also houses the stub alerting consumer that will notify on suspicious activity.
//
// Phase 1: skeleton only — starts up, logs readiness, waits for shutdown signal.
// RabbitMQ consumers are wired in Phase 6.
package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mcp-gate/mcp-gate/internal/config"
)

func main() {
	cfg := config.Load()

	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	slog.Info("worker starting — no consumers registered yet (Phase 1 skeleton)")
	slog.Info("worker ready; waiting for shutdown signal")

	// Block until SIGTERM or SIGINT.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	slog.Info("worker shutting down")
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}
