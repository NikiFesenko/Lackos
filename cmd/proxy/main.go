// Binary proxy is the MCP-facing gateway.
//
// Phase 5 — full request lifecycle:
//  1. Parse MCP JSON-RPC request (POST /)
//  2. Extract & validate per-user Bearer token → userID
//  3. Load user record → roleID
//  4. Policy engine evaluation (fail-closed)
//  5. Rate limiter check (fail-closed)
//  6. Forward to downstream server
//  7. Apply field-level redaction from policy
//  8. Return response to agent
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/mcp-gate/mcp-gate/internal/auth"
	"github.com/mcp-gate/mcp-gate/internal/config"
	"github.com/mcp-gate/mcp-gate/internal/db"
	"github.com/mcp-gate/mcp-gate/internal/mcp"
	"github.com/mcp-gate/mcp-gate/internal/policy"
	"github.com/mcp-gate/mcp-gate/internal/ratelimit"
)

func main() {
	cfg := config.Load()

	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	// ── Postgres ──────────────────────────────────────────────────────────
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

	// ── Redis ─────────────────────────────────────────────────────────────
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		slog.Error("invalid REDIS_URL", "err", err)
		os.Exit(1)
	}
	rdb := redis.NewClient(opt)
	defer rdb.Close()

	// ── Subsystems ────────────────────────────────────────────────────────
	queries := db.New(pool)
	tokenMgr := auth.NewTokenManager(rdb, 0)
	policyEngine := policy.New(queries, rdb)
	limiter := ratelimit.New(rdb)
	forwarder := mcp.NewForwarder(auth.EnvSecretResolver{})

	// ── Router ────────────────────────────────────────────────────────────
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthzHandler)
	mux.HandleFunc("POST /", makeProxyHandler(queries, tokenMgr, policyEngine, limiter, forwarder))

	addr := fmt.Sprintf(":%d", cfg.ProxyPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("proxy listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("proxy server error", "err", err)
			os.Exit(1)
		}
	}()

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

// makeProxyHandler wires together the full MCP proxy request lifecycle.
func makeProxyHandler(
	queries db.Querier,
	tokenMgr *auth.TokenManager,
	policyEngine *policy.Engine,
	limiter *ratelimit.Limiter,
	forwarder *mcp.Forwarder,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx := r.Context()

		// ── Step 1: Parse the MCP JSON-RPC request ───────────────────────
		var req mcp.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeRPCError(w, nil, mcp.CodeParseError, "parse error")
			return
		}

		// ── Step 2: Authenticate — extract Bearer token ──────────────────
		rawToken, err := extractBearer(r)
		if err != nil {
			slog.InfoContext(ctx, "proxy: missing or malformed Authorization header")
			writeRPCError(w, req.ID, mcp.CodeUnauthorized, "authorization required")
			return
		}

		userID, err := tokenMgr.Validate(ctx, rawToken)
		if err != nil {
			if errors.Is(err, auth.ErrTokenNotFound) || errors.Is(err, auth.ErrTokenInvalid) {
				writeRPCError(w, req.ID, mcp.CodeUnauthorized, "invalid or expired token")
				return
			}
			slog.ErrorContext(ctx, "proxy: token validation error", "err", err)
			writeRPCError(w, req.ID, mcp.CodeInternalError, "internal error")
			return
		}

		// ── Step 3: Resolve user → role ──────────────────────────────────
		var userUUID pgtype.UUID
		if err := userUUID.Scan(userID); err != nil {
			slog.ErrorContext(ctx, "proxy: invalid userID in token", "user_id", userID)
			writeRPCError(w, req.ID, mcp.CodeInternalError, "internal error")
			return
		}

		user, err := queries.GetUser(ctx, userUUID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeRPCError(w, req.ID, mcp.CodeUnauthorized, "user not found or deactivated")
				return
			}
			slog.ErrorContext(ctx, "proxy: get user error", "err", err)
			writeRPCError(w, req.ID, mcp.CodeInternalError, "internal error")
			return
		}
		if !user.IsActive {
			writeRPCError(w, req.ID, mcp.CodeUnauthorized, "user account is deactivated")
			return
		}

		// ── Step 4: Resolve downstream server from params ────────────────
		// The proxy expects a non-standard "server" field in params to identify
		// which downstream server to route to.
		serverName, toolName, toolParams, parseErr := parseToolCall(req)
		if parseErr != nil {
			writeRPCError(w, req.ID, mcp.CodeInvalidParams, parseErr.Error())
			return
		}

		downstream, err := queries.GetDownstreamServerByName(ctx, serverName)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeRPCError(w, req.ID, mcp.CodeInvalidParams, "unknown downstream server: "+serverName)
				return
			}
			slog.ErrorContext(ctx, "proxy: get downstream server error", "err", err)
			writeRPCError(w, req.ID, mcp.CodeInternalError, "internal error")
			return
		}
		if !downstream.IsActive {
			writeRPCError(w, req.ID, mcp.CodeForbidden, "downstream server is inactive")
			return
		}

		roleID := fmt.Sprintf("%x-%x-%x-%x-%x",
			user.RoleID.Bytes[0:4], user.RoleID.Bytes[4:6], user.RoleID.Bytes[6:8],
			user.RoleID.Bytes[8:10], user.RoleID.Bytes[10:16])
		serverIDStr := fmt.Sprintf("%x-%x-%x-%x-%x",
			downstream.ID.Bytes[0:4], downstream.ID.Bytes[4:6], downstream.ID.Bytes[6:8],
			downstream.ID.Bytes[8:10], downstream.ID.Bytes[10:16])

		// ── Step 5: Policy evaluation (fail-closed) ───────────────────────
		decision, err := policyEngine.Evaluate(ctx, roleID, serverIDStr, toolName)
		if err != nil {
			slog.ErrorContext(ctx, "proxy: policy evaluation error", "err", err)
		}
		if !decision.Allowed {
			slog.InfoContext(ctx, "proxy: request denied by policy",
				"user_id", userID, "tool", toolName, "server", serverName)
			writeRPCError(w, req.ID, mcp.CodeForbidden, "tool call denied by policy")
			return
		}

		// ── Step 6: Rate limiter (fail-closed) ────────────────────────────
		rl, err := limiter.Allow(ctx, userID, serverIDStr, decision.MaxCallsPerMinute)
		if err != nil {
			slog.ErrorContext(ctx, "proxy: rate limiter error; failing closed", "err", err)
			writeRPCError(w, req.ID, mcp.CodeRateLimited, "service temporarily unavailable")
			return
		}
		if !rl.Allowed {
			slog.InfoContext(ctx, "proxy: rate limit exceeded",
				"user_id", userID, "server", serverName,
				"count", rl.Count, "limit", rl.Limit)
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.Limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", rl.Remaining()))
			writeRPCError(w, req.ID, mcp.CodeRateLimited, "rate limit exceeded")
			return
		}

		// ── Step 7: Forward to downstream server ─────────────────────────
		// Build a clean forwarded request with the tool params only (strip the
		// proxy-internal "server" field before forwarding).
		forwardReq := mcp.Request{
			JSONRPC: "2.0",
			ID:      req.ID,
			Method:  "tools/call",
			Params:  toolParams,
		}
		downstreamResp, err := forwarder.Forward(ctx,
			downstream.BaseUrl,
			downstream.AuthType,
			downstream.AuthSecretRef,
			forwardReq,
		)
		if err != nil {
			slog.ErrorContext(ctx, "proxy: downstream forward error",
				"server", serverName, "tool", toolName,
				"latency_ms", time.Since(start).Milliseconds(), "err", err)
			writeRPCError(w, req.ID, mcp.CodeUpstreamError, "downstream server error")
			return
		}

		// ── Step 8: Apply field-level redaction ───────────────────────────
		if downstreamResp.Result != nil && len(decision.RedactFields) > 0 {
			var toolResult mcp.ToolResult
			if err := json.Unmarshal(downstreamResp.Result, &toolResult); err == nil {
				redacted := mcp.RedactFields(toolResult, decision.RedactFields)
				if raw, err := json.Marshal(redacted); err == nil {
					downstreamResp.Result = raw
				}
			}
		}

		slog.InfoContext(ctx, "proxy: request completed",
			"user_id", userID, "tool", toolName, "server", serverName,
			"latency_ms", time.Since(start).Milliseconds())

		writeJSON(w, downstreamResp)
	}
}

// parseToolCall extracts the server name, tool name, and clean params from the
// proxy's extended "tools/call" params format:
//
//	{ "server": "mock-bamboohr", "name": "get_employee_record", "arguments": {...} }
func parseToolCall(req mcp.Request) (serverName, toolName string, params json.RawMessage, err error) {
	if req.Method != "tools/call" {
		return "", "", nil, fmt.Errorf("proxy only handles tools/call, got %q", req.Method)
	}
	var p struct {
		Server    string         `json:"server"`
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return "", "", nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.Server == "" {
		return "", "", nil, fmt.Errorf("params.server is required")
	}
	if p.Name == "" {
		return "", "", nil, fmt.Errorf("params.name is required")
	}
	// Build clean forwarded params without the proxy-internal "server" field.
	clean := struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments,omitempty"`
	}{Name: p.Name, Arguments: p.Arguments}
	raw, err := json.Marshal(clean)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to marshal params: %w", err)
	}
	return p.Server, p.Name, raw, nil
}

// extractBearer parses "Authorization: Bearer <token>" from the request header.
func extractBearer(r *http.Request) (string, error) {
	h := r.Header.Get("Authorization")
	token, found := strings.CutPrefix(h, "Bearer ")
	if !found || token == "" {
		return "", errors.New("missing bearer token")
	}
	return token, nil
}

func writeRPCError(w http.ResponseWriter, id json.RawMessage, code int, msg string) {
	resp := mcp.ErrorResponse(id, code, msg)
	writeJSON(w, resp)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func healthzHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, `{"status":"ok","service":"proxy"}`)
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
