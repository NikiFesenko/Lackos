// Binary proxy is the MCP-facing gateway that authenticates agents, enforces
// policy, and forwards requests to downstream MCP servers.
//
// Phase 1: skeleton only — exposes GET /healthz.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mcp-gate/mcp-gate/internal/config"
)

func main() {
	cfg := config.Load()

	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthzHandler("proxy"))

	addr := fmt.Sprintf(":%d", cfg.ProxyPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine so we can listen for shutdown signals.
	go func() {
		slog.Info("proxy listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("proxy server error", "err", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown on SIGTERM / SIGINT.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	slog.Info("proxy shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("proxy shutdown error", "err", err)
		os.Exit(1)
	}
	slog.Info("proxy stopped")
}

// healthzHandler returns a simple JSON health response for the given service name.
func healthzHandler(service string) http.HandlerFunc {
	body, _ := json.Marshal(map[string]string{"status": "ok", "service": service})
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(body) //nolint:errcheck
	}
}

// newLogger creates a structured slog.Logger at the configured level.
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
