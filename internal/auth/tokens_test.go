package auth_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mcp-gate/mcp-gate/internal/auth"
)

func newTokenManager(t *testing.T) (*auth.TokenManager, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return auth.NewTokenManager(rdb, 0), mr
}

// TestIssueAndValidate verifies the basic issue → validate round-trip.
func TestIssueAndValidate(t *testing.T) {
	mgr, _ := newTokenManager(t)
	ctx := context.Background()
	userID := "user-00000001"

	token, err := mgr.Issue(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, token, 64, "token must be 64 hex chars (32 bytes)")

	got, err := mgr.Validate(ctx, token)
	require.NoError(t, err)
	assert.Equal(t, userID, got)
}

// TestValidate_UnknownToken returns ErrTokenNotFound for a non-existent token.
func TestValidate_UnknownToken(t *testing.T) {
	mgr, _ := newTokenManager(t)
	// 64 hex chars that were never issued.
	fake := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	_, err := mgr.Validate(context.Background(), fake)
	assert.ErrorIs(t, err, auth.ErrTokenNotFound)
}

// TestValidate_InvalidFormat returns ErrTokenInvalid for short tokens.
func TestValidate_InvalidFormat(t *testing.T) {
	mgr, _ := newTokenManager(t)
	_, err := mgr.Validate(context.Background(), "tooshort")
	assert.ErrorIs(t, err, auth.ErrTokenInvalid)
}

// TestRevoke verifies that a revoked token can no longer be validated.
func TestRevoke(t *testing.T) {
	mgr, _ := newTokenManager(t)
	ctx := context.Background()

	token, err := mgr.Issue(ctx, "user-00000002")
	require.NoError(t, err)

	// Valid before revocation.
	_, err = mgr.Validate(ctx, token)
	require.NoError(t, err)

	// Revoke.
	require.NoError(t, mgr.Revoke(ctx, token))

	// Invalid after revocation.
	_, err = mgr.Validate(ctx, token)
	assert.ErrorIs(t, err, auth.ErrTokenNotFound)
}

// TestIssue_DifferentTokensPerCall verifies that each Issue call produces a unique token.
func TestIssue_DifferentTokensPerCall(t *testing.T) {
	mgr, _ := newTokenManager(t)
	ctx := context.Background()

	t1, err := mgr.Issue(ctx, "user-00000003")
	require.NoError(t, err)
	t2, err := mgr.Issue(ctx, "user-00000003")
	require.NoError(t, err)

	assert.NotEqual(t, t1, t2, "each issued token must be unique")
}
