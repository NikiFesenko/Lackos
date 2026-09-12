// Binary adminapi serves the REST API consumed by the React admin dashboard.
// It allows HR ops/security admins to manage users, roles, policies,
// downstream servers, and view the audit log.
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

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mcp-gate/mcp-gate/internal/config"
	"github.com/mcp-gate/mcp-gate/internal/db"
	"github.com/mcp-gate/mcp-gate/internal/handlers"
	"github.com/mcp-gate/mcp-gate/internal/middleware"
)

func main() {
	cfg := config.Load()

	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

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

	// ── Handlers & Router ──────────────────────────────────────────────────
	h := handlers.New(db.New(pool))
	mux := http.NewServeMux()

	// Health (no auth required)
	mux.HandleFunc("GET /healthz", healthzHandler)

	// ── Roles
	mux.HandleFunc("GET /api/v1/roles",        h.ListRoles)
	mux.HandleFunc("POST /api/v1/roles",        h.CreateRole)
	mux.HandleFunc("GET /api/v1/roles/{id}",    h.GetRole)
	mux.HandleFunc("PUT /api/v1/roles/{id}",    h.UpdateRole)
	mux.HandleFunc("DELETE /api/v1/roles/{id}", h.DeleteRole)

	// ── Users
	mux.HandleFunc("GET /api/v1/users",                  h.ListUsers)
	mux.HandleFunc("POST /api/v1/users",                  h.CreateUser)
	mux.HandleFunc("GET /api/v1/users/{id}",              h.GetUser)
	mux.HandleFunc("PUT /api/v1/users/{id}/role",         h.UpdateUserRole)
	mux.HandleFunc("DELETE /api/v1/users/{id}",           h.DeactivateUser)

	// ── Downstream servers
	mux.HandleFunc("GET /api/v1/downstream-servers",           h.ListDownstreamServers)
	mux.HandleFunc("POST /api/v1/downstream-servers",           h.CreateDownstreamServer)
	mux.HandleFunc("GET /api/v1/downstream-servers/{id}",       h.GetDownstreamServer)
	mux.HandleFunc("PUT /api/v1/downstream-servers/{id}",       h.UpdateDownstreamServer)
	mux.HandleFunc("DELETE /api/v1/downstream-servers/{id}",    h.SetDownstreamServerActive)

	// ── Policies
	mux.HandleFunc("GET /api/v1/policies",        h.ListPolicies)
	mux.HandleFunc("POST /api/v1/policies",        h.UpsertPolicy)
	mux.HandleFunc("GET /api/v1/policies/{id}",   h.GetPolicy)
	mux.HandleFunc("DELETE /api/v1/policies/{id}", h.DeletePolicy)

	// ── Audit log (read-only)
	mux.HandleFunc("GET /api/v1/audit-events", h.ListAuditEvents)

	// ── Middleware chain: RequestID → Logger → Auth → mux
	var handler http.Handler = mux
	handler = middleware.RequireAdminToken(cfg.AdminToken)(handler)
	handler = middleware.Logger(handler)
	handler = middleware.RequestID(handler)

	addr := fmt.Sprintf(":%d", cfg.AdminAPIPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("adminapi listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("adminapi server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	slog.Info("adminapi shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("adminapi shutdown error", "err", err)
		os.Exit(1)
	}
	slog.Info("adminapi stopped")
}

func healthzHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "adminapi"}) //nolint:errcheck
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
