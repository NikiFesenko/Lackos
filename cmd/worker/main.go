// Binary worker consumes audit events from RabbitMQ and persists them to Postgres.
// It also runs the real-time alerting consumer for blocked and rate-limited activity.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mcp-gate/mcp-gate/internal/audit"
	"github.com/mcp-gate/mcp-gate/internal/config"
	"github.com/mcp-gate/mcp-gate/internal/db"
)

func main() {
	cfg := config.Load()

	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	slog.Info("worker starting")

	// ── Database ───────────────────────────────────────────────────────────
	pool, err := pgxpool.New(context.Background(), cfg.PostgresDSN)
	if err != nil {
		slog.Error("failed to create postgres pool", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		slog.Error("failed to connect to postgres", "err", err)
		os.Exit(1)
	}
	slog.Info("connected to postgres")

	queries := db.New(pool)

	// ── RabbitMQ Audit Writer Consumer ────────────────────────────────────
	writer, err := audit.NewWriter(queries, cfg.RabbitMQURL)
	if err != nil {
		slog.Error("failed to initialize audit writer", "err", err)
		os.Exit(1)
	}
	defer writer.Stop()

	if err := writer.Start(); err != nil {
		slog.Error("failed to start audit writer", "err", err)
		os.Exit(1)
	}

	// ── RabbitMQ Alerting Consumer ────────────────────────────────────────
	alertConsumer, err := audit.NewAlertConsumer(audit.NewLogAlerter(), cfg.RabbitMQURL)
	if err != nil {
		slog.Error("failed to initialize alert consumer", "err", err)
		os.Exit(1)
	}
	defer alertConsumer.Stop()

	if err := alertConsumer.Start(); err != nil {
		slog.Error("failed to start alert consumer", "err", err)
		os.Exit(1)
	}

	slog.Info("worker ready; listening for audit events and alerts")

	// Block until SIGTERM or SIGINT.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	slog.Info("worker shutting down")
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = writer.Stop()
	_ = alertConsumer.Stop()
	slog.Info("worker stopped")
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
