// Package auth handles per-user proxy token issuance and validation.
//
// Phase 5 design:
//   - Tokens are cryptographically random 32-byte values, hex-encoded (64 chars).
//   - The token's SHA-256 hash is stored in Redis as the cache key, mapping to
//     the user's UUID. The raw token is only ever held in memory at issuance time
//     and then given to the caller — it is never stored anywhere.
//   - Validation resolves token → userID in O(1) via Redis.
//   - Token revocation: delete the Redis key.
//
// Phase 8 upgrade path: swap this module for JWT-based sessions without changing
// the proxy's call site (just the Validate function).
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/mcp-gate/mcp-gate/internal/cache"
)

// ErrTokenNotFound is returned when a token is absent or expired.
var ErrTokenNotFound = errors.New("auth: token not found or expired")

// ErrTokenInvalid is returned when the token format is wrong.
var ErrTokenInvalid = errors.New("auth: invalid token format")

// TokenManager issues and validates per-user proxy tokens.
type TokenManager struct {
	redis *redis.Client
	ttl   time.Duration
}

// NewTokenManager creates a TokenManager.
// ttl is the token lifetime; pass 0 to use the default (8 hours).
func NewTokenManager(rdb *redis.Client, ttl time.Duration) *TokenManager {
	if ttl == 0 {
		ttl = time.Duration(cache.TokenTTL) * time.Second
	}
	return &TokenManager{redis: rdb, ttl: ttl}
}

// Issue generates a new random token for the given userID, stores its hash in
// Redis, and returns the raw token string.
// The raw token is shown exactly once — it cannot be retrieved again.
func (m *TokenManager) Issue(ctx context.Context, userID string) (string, error) {
	raw, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("auth: generate token: %w", err)
	}
	key := cache.TokenKey(hashToken(raw))
	if err := m.redis.Set(ctx, key, userID, m.ttl).Err(); err != nil {
		return "", fmt.Errorf("auth: store token: %w", err)
	}
	return raw, nil
}

// Validate looks up the token and returns the associated userID.
// Returns ErrTokenNotFound if the token is missing or expired.
// Returns ErrTokenInvalid if the token is not the expected length.
func (m *TokenManager) Validate(ctx context.Context, raw string) (string, error) {
	if len(raw) != 64 {
		return "", ErrTokenInvalid
	}
	key := cache.TokenKey(hashToken(raw))
	userID, err := m.redis.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrTokenNotFound
		}
		return "", fmt.Errorf("auth: redis lookup: %w", err)
	}
	return userID, nil
}

// Revoke immediately invalidates a token.
func (m *TokenManager) Revoke(ctx context.Context, raw string) error {
	key := cache.TokenKey(hashToken(raw))
	return m.redis.Del(ctx, key).Err()
}

// ── Internal helpers ──────────────────────────────────────────────────────────

// generateToken returns a cryptographically random 32-byte token, hex-encoded.
func generateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// hashToken returns the hex-encoded SHA-256 hash of a raw token.
// Storing the hash (not the raw token) means a Redis breach doesn't expose
// usable credentials.
func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
