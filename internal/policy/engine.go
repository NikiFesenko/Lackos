package policy

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"

	"github.com/mcp-gate/mcp-gate/internal/cache"
	"github.com/mcp-gate/mcp-gate/internal/db"
)

// Store is the minimal DB interface the policy engine requires.
// Using a narrow interface keeps the engine independently testable.
type Store interface {
	GetPolicyByRoleServerTool(ctx context.Context, arg db.GetPolicyByRoleServerToolParams) (db.Policy, error)
}

// Engine evaluates policy decisions for incoming proxy requests.
// It caches results in Redis with a short TTL; on any uncertainty it denies.
type Engine struct {
	store Store
	redis *redis.Client
}

// New creates a policy Engine.
func New(store Store, rdb *redis.Client) *Engine {
	return &Engine{store: store, redis: rdb}
}

// cachedPolicy is the JSON-serialisable form stored in Redis.
type cachedPolicy struct {
	Allowed           bool     `json:"allowed"`
	RedactFields      []string `json:"redact_fields"`
	MaxCallsPerMinute int32    `json:"max_calls_per_minute"`
}

// Evaluate returns a Decision for the given (roleID, serverID, toolName) triple.
//
// Fail-closed contract:
//   - If both Redis AND Postgres are unreachable → deny and log.
//   - If no policy row exists for this triple → deny (default-deny).
//   - Never return Allowed=true unless a positive policy row confirms it.
func (e *Engine) Evaluate(
	ctx context.Context,
	roleID, serverID, toolName string,
) (Decision, error) {
	key := cache.PolicyKey(roleID, serverID, toolName)

	// ── 1. Try Redis cache ────────────────────────────────────────────────
	cached, err := e.redis.Get(ctx, key).Bytes()
	if err == nil {
		// Cache hit — deserialize and return.
		var cp cachedPolicy
		if jsonErr := json.Unmarshal(cached, &cp); jsonErr == nil {
			return Decision{
				Allowed:           cp.Allowed,
				RedactFields:      cp.RedactFields,
				MaxCallsPerMinute: cp.MaxCallsPerMinute,
			}, nil
		}
		// Corrupt cache entry — fall through to DB.
		slog.WarnContext(ctx, "corrupt policy cache entry; falling back to DB", "key", key)
	} else if !errors.Is(err, redis.Nil) {
		// Redis error (not a simple miss) — log and fall through to DB.
		slog.WarnContext(ctx, "redis error on policy lookup; falling back to DB",
			"key", key, "err", err)
	}

	// ── 2. Cache miss (or Redis error) → query Postgres ──────────────────
	var roleUUID, serverUUID pgtype.UUID
	if scanErr := roleUUID.Scan(roleID); scanErr != nil {
		slog.ErrorContext(ctx, "invalid roleID UUID", "role_id", roleID, "err", scanErr)
		return Deny, scanErr
	}
	if scanErr := serverUUID.Scan(serverID); scanErr != nil {
		slog.ErrorContext(ctx, "invalid serverID UUID", "server_id", serverID, "err", scanErr)
		return Deny, scanErr
	}

	row, dbErr := e.store.GetPolicyByRoleServerTool(ctx, db.GetPolicyByRoleServerToolParams{
		RoleID:             roleUUID,
		DownstreamServerID: serverUUID,
		ToolName:           toolName,
	})

	if dbErr != nil {
		if errors.Is(dbErr, pgx.ErrNoRows) {
			// No policy configured for this triple → default deny.
			slog.InfoContext(ctx, "no policy found; denying",
				"role_id", roleID, "server_id", serverID, "tool", toolName)
			// Cache the deny so we don't hammer the DB for unknown tool calls.
			e.cacheDecision(ctx, key, cachedPolicy{Allowed: false, RedactFields: []string{}})
			return Deny, nil
		}
		// DB unreachable — FAIL CLOSED. This is intentional and critical.
		slog.ErrorContext(ctx, "DB unreachable during policy eval; failing closed",
			"role_id", roleID, "server_id", serverID, "tool", toolName, "err", dbErr)
		return Deny, dbErr
	}

	// ── 3. Got a DB row — cache it and return ────────────────────────────
	cp := cachedPolicy{
		Allowed:           row.IsAllowed,
		RedactFields:      row.RedactFields,
		MaxCallsPerMinute: row.MaxCallsPerMinute,
	}
	e.cacheDecision(ctx, key, cp)

	return Decision{
		Allowed:           cp.Allowed,
		RedactFields:      cp.RedactFields,
		MaxCallsPerMinute: cp.MaxCallsPerMinute,
	}, nil
}

// cacheDecision stores a policy decision in Redis.
// Failures are logged but not fatal — the next request will do another DB round-trip.
func (e *Engine) cacheDecision(ctx context.Context, key string, cp cachedPolicy) {
	data, err := json.Marshal(cp)
	if err != nil {
		slog.WarnContext(ctx, "failed to marshal policy for cache", "key", key, "err", err)
		return
	}
	if err := e.redis.Set(ctx, key, data, time.Duration(cache.PolicyTTL)*time.Second).Err(); err != nil {
		slog.WarnContext(ctx, "failed to write policy to cache", "key", key, "err", err)
	}
}

// InvalidatePolicy removes a cached policy entry so the next request gets a fresh DB read.
// Must be called from the admin API whenever a policy row is created or updated.
func (e *Engine) InvalidatePolicy(ctx context.Context, roleID, serverID, toolName string) {
	key := cache.PolicyKey(roleID, serverID, toolName)
	if err := e.redis.Del(ctx, key).Err(); err != nil {
		slog.WarnContext(ctx, "failed to invalidate policy cache", "key", key, "err", err)
	}
}
