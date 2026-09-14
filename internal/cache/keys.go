// Package cache defines every Redis key pattern used by MCP Gate.
// All key construction must go through this file — never build key strings
// ad-hoc elsewhere in the codebase.
package cache

import "fmt"

const (
	// PolicyTTL is how long a cached policy lives before a fresh DB read.
	// Short enough to pick up admin changes quickly; long enough to protect
	// the DB from a thundering-herd on every request.
	PolicyTTL = 60 // seconds

	// TokenTTL is the lifetime of a cached per-user proxy token mapping.
	TokenTTL = 8 * 60 * 60 // 8 hours in seconds

	// RateLimitWindow is the sliding-window size for rate limiting in seconds.
	RateLimitWindow = 60
)

// PolicyKey returns the Redis key for a compiled policy decision.
// Format: policy:{roleID}:{serverID}:{toolName}
func PolicyKey(roleID, serverID, toolName string) string {
	return fmt.Sprintf("policy:%s:%s:%s", roleID, serverID, toolName)
}

// RateLimitKey returns the Redis key for a per-user/per-server sliding-window counter.
// Format: ratelimit:{userID}:{serverID}
func RateLimitKey(userID, serverID string) string {
	return fmt.Sprintf("ratelimit:%s:%s", userID, serverID)
}

// TokenKey returns the Redis key for a cached proxy token → userID mapping.
// Format: token:{tokenHash}
func TokenKey(tokenHash string) string {
	return fmt.Sprintf("token:%s", tokenHash)
}
