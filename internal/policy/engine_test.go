package policy_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mcp-gate/mcp-gate/internal/db"
	"github.com/mcp-gate/mcp-gate/internal/policy"
)

// ── Mock store ────────────────────────────────────────────────────────────────

// mockStore implements policy.Store for testing.
type mockStore struct {
	policy *db.Policy
	err    error
}

func (m *mockStore) GetPolicyByRoleServerTool(
	_ context.Context, _ db.GetPolicyByRoleServerToolParams,
) (db.Policy, error) {
	if m.err != nil {
		return db.Policy{}, m.err
	}
	if m.policy == nil {
		return db.Policy{}, pgx.ErrNoRows
	}
	return *m.policy, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// newEngine starts a miniredis server and returns the engine + the miniredis handle.
func newEngine(t *testing.T, store policy.Store) (*policy.Engine, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return policy.New(store, rdb), mr
}

// waitCacheWrite gives the async cache goroutine time to complete.
// The goroutine uses a 2-second context deadline, so 50 ms is plenty.
func waitCacheWrite() { time.Sleep(50 * time.Millisecond) }

// samplePolicy returns a policy row with the given allow flag and redact fields.
func samplePolicy(allowed bool, redact []string) *db.Policy {
	var id pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000001")
	return &db.Policy{
		ID:                id,
		ToolName:          "get_employee_record",
		IsAllowed:         allowed,
		RedactFields:      redact,
		MaxCallsPerMinute: 30,
	}
}

const (
	roleID   = "00000000-0000-0000-0000-000000000001"
	serverID = "00000000-0000-0000-0000-000000000002"
	toolName = "get_employee_record"
)

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestEvaluate_CacheHit_AllowedPolicy verifies that a cached allow decision
// is returned without hitting the store on the second call.
func TestEvaluate_CacheHit_AllowedPolicy(t *testing.T) {
	store := &mockStore{policy: samplePolicy(true, []string{"salary"})}
	eng, _ := newEngine(t, store)
	ctx := context.Background()

	// First call → cache miss → DB hit → cache written asynchronously.
	d1, err := eng.Evaluate(ctx, roleID, serverID, toolName)
	require.NoError(t, err)
	assert.True(t, d1.Allowed)
	assert.Equal(t, []string{"salary"}, d1.RedactFields)

	// Wait for the async cache write goroutine to finish, then break the store.
	waitCacheWrite()
	store.err = errors.New("db is gone")
	store.policy = nil

	// Second call must be served entirely from cache — no DB contact.
	d2, err := eng.Evaluate(ctx, roleID, serverID, toolName)
	require.NoError(t, err, "cache hit must not return an error even when DB is down")
	assert.True(t, d2.Allowed)
	assert.Equal(t, []string{"salary"}, d2.RedactFields)
}

// TestEvaluate_CacheMiss_DBHit verifies the cache-miss → DB-load → cache-store path.
func TestEvaluate_CacheMiss_DBHit(t *testing.T) {
	store := &mockStore{policy: samplePolicy(true, []string{})}
	eng, _ := newEngine(t, store)

	d, err := eng.Evaluate(context.Background(), roleID, serverID, toolName)
	require.NoError(t, err)
	assert.True(t, d.Allowed)
	assert.Equal(t, int32(30), d.MaxCallsPerMinute)
}

// TestEvaluate_FailClosed_BothRedisAndDBDown is the most important test.
// When neither Redis nor Postgres can be reached, the engine MUST deny.
func TestEvaluate_FailClosed_BothRedisAndDBDown(t *testing.T) {
	store := &mockStore{err: errors.New("connection refused")}

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	mr.Close() // kill Redis immediately after creating the client
	t.Cleanup(func() { _ = rdb.Close() })

	eng := policy.New(store, rdb)

	d, err := eng.Evaluate(context.Background(), roleID, serverID, toolName)
	assert.False(t, d.Allowed, "must fail closed when both Redis and DB are unreachable")
	_ = err // DB error is expected and propagated — we only care about Allowed=false
}

// TestEvaluate_NoPolicy_DefaultDeny verifies that a missing policy row → deny.
func TestEvaluate_NoPolicy_DefaultDeny(t *testing.T) {
	store := &mockStore{policy: nil} // ErrNoRows
	eng, _ := newEngine(t, store)

	d, err := eng.Evaluate(context.Background(), roleID, serverID, toolName)
	require.NoError(t, err)
	assert.False(t, d.Allowed, "no policy must default to deny")
}

// TestEvaluate_ExplicitDenyPolicy verifies an is_allowed=false row → deny.
func TestEvaluate_ExplicitDenyPolicy(t *testing.T) {
	store := &mockStore{policy: samplePolicy(false, []string{})}
	eng, _ := newEngine(t, store)

	d, err := eng.Evaluate(context.Background(), roleID, serverID, toolName)
	require.NoError(t, err)
	assert.False(t, d.Allowed)
}

// TestEvaluate_RedactFields verifies that redact_fields are propagated correctly.
func TestEvaluate_RedactFields(t *testing.T) {
	store := &mockStore{policy: samplePolicy(true, []string{"salary", "ssn", "bank_account"})}
	eng, _ := newEngine(t, store)

	d, err := eng.Evaluate(context.Background(), roleID, serverID, toolName)
	require.NoError(t, err)
	assert.True(t, d.Allowed)
	assert.ElementsMatch(t, []string{"salary", "ssn", "bank_account"}, d.RedactFields)
}

// TestEvaluate_DenyDecision_HasEmptyRedactFields verifies that a deny Decision
// always has a non-nil RedactFields slice, so callers never panic on range.
func TestEvaluate_DenyDecision_HasEmptyRedactFields(t *testing.T) {
	store := &mockStore{policy: nil}
	eng, _ := newEngine(t, store)

	d, err := eng.Evaluate(context.Background(), roleID, serverID, toolName)
	require.NoError(t, err)
	assert.False(t, d.Allowed)
	assert.NotNil(t, d.RedactFields, "RedactFields must never be nil, even on deny")
}

// TestInvalidatePolicy verifies that after invalidation, a changed DB value is picked up.
func TestInvalidatePolicy(t *testing.T) {
	store := &mockStore{policy: samplePolicy(true, []string{})}
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	eng := policy.New(store, rdb)
	ctx := context.Background()

	// Prime the cache with allowed=true.
	d1, err := eng.Evaluate(ctx, roleID, serverID, toolName)
	require.NoError(t, err)
	assert.True(t, d1.Allowed)

	// Wait for async cache write to complete before changing store.
	waitCacheWrite()

	// Admin changes the policy to denied.
	store.policy = samplePolicy(false, []string{})

	// Without invalidation, cache returns the old allow (stale read is intentional).
	d2, _ := eng.Evaluate(ctx, roleID, serverID, toolName)
	assert.True(t, d2.Allowed, "stale cache should still return old allow")

	// Invalidate → next call hits DB and gets the updated deny.
	eng.InvalidatePolicy(ctx, roleID, serverID, toolName)
	d3, err := eng.Evaluate(ctx, roleID, serverID, toolName)
	require.NoError(t, err)
	assert.False(t, d3.Allowed, "after invalidation, updated DB deny must be returned")
}
