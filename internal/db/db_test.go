// Package db integration tests.
//
// These tests spin up a real Postgres 16 container via testcontainers-go,
// run all migrations, and verify the generated sqlc queries work correctly
// against the actual schema.
//
// Run with:   go test -v -count=1 ./internal/db/...
// Requires:   Docker running locally.
package db_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/mcp-gate/mcp-gate/internal/db"
)

// testDB starts a Postgres container, runs migrations, and returns a
// connected pool + Queries handle. The container is terminated when t ends.
func testDB(t *testing.T) *db.Queries {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("mcpgate_test"),
		postgres.WithUsername("mcpgate"),
		postgres.WithPassword("testpass"),
		postgres.WithInitScripts(
			"../../migrations/000001_create_roles.up.sql",
			"../../migrations/000002_create_users.up.sql",
			"../../migrations/000003_create_downstream_servers.up.sql",
			"../../migrations/000004_create_policies.up.sql",
			"../../migrations/000005_create_audit_events.up.sql",
		),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return db.New(pool)
}

// ── Roles ──────────────────────────────────────────────────────────────────

func TestCreateAndGetRole(t *testing.T) {
	q := testDB(t)
	ctx := context.Background()

	role, err := q.CreateRole(ctx, db.CreateRoleParams{
		Name:        "test_role",
		Description: ptr("A test role"),
	})
	require.NoError(t, err)
	assert.Equal(t, "test_role", role.Name)
	assert.NotEmpty(t, role.ID)

	fetched, err := q.GetRole(ctx, role.ID)
	require.NoError(t, err)
	assert.Equal(t, role.ID, fetched.ID)
	assert.Equal(t, "test_role", fetched.Name)
}

func TestListRoles(t *testing.T) {
	q := testDB(t)
	ctx := context.Background()

	for i := range 3 {
		_, err := q.CreateRole(ctx, db.CreateRoleParams{
			Name: fmt.Sprintf("role_%d", i),
		})
		require.NoError(t, err)
	}

	roles, err := q.ListRoles(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(roles), 3)
}

// ── Users ──────────────────────────────────────────────────────────────────

func TestCreateAndGetUser(t *testing.T) {
	q := testDB(t)
	ctx := context.Background()

	role, err := q.CreateRole(ctx, db.CreateRoleParams{Name: "employee"})
	require.NoError(t, err)

	user, err := q.CreateUser(ctx, db.CreateUserParams{
		Email:       "bob@example.com",
		DisplayName: "Bob Employee",
		RoleID:      role.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, "bob@example.com", user.Email)
	assert.True(t, user.IsActive)

	byEmail, err := q.GetUserByEmail(ctx, "bob@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, byEmail.ID)
}

func TestSetUserActive_Deactivate(t *testing.T) {
	q := testDB(t)
	ctx := context.Background()

	role, err := q.CreateRole(ctx, db.CreateRoleParams{Name: "manager"})
	require.NoError(t, err)

	user, err := q.CreateUser(ctx, db.CreateUserParams{
		Email:       "carol@example.com",
		DisplayName: "Carol Manager",
		RoleID:      role.ID,
	})
	require.NoError(t, err)
	assert.True(t, user.IsActive)

	deactivated, err := q.SetUserActive(ctx, db.SetUserActiveParams{
		ID:       user.ID,
		IsActive: false,
	})
	require.NoError(t, err)
	assert.False(t, deactivated.IsActive)
}

// ── Policies ───────────────────────────────────────────────────────────────

func TestUpsertAndGetPolicy(t *testing.T) {
	q := testDB(t)
	ctx := context.Background()

	role, err := q.CreateRole(ctx, db.CreateRoleParams{Name: "recruiter"})
	require.NoError(t, err)

	srv, err := q.CreateDownstreamServer(ctx, db.CreateDownstreamServerParams{
		Name:          "bamboohr-test",
		BaseUrl:       "http://localhost:9000",
		AuthType:      "api_key",
		AuthSecretRef: "env:TEST_API_KEY",
		ToolManifest:  []byte(`[]`),
	})
	require.NoError(t, err)

	policy, err := q.UpsertPolicy(ctx, db.UpsertPolicyParams{
		RoleID:             role.ID,
		DownstreamServerID: srv.ID,
		ToolName:           "get_employee_record",
		IsAllowed:          true,
		RedactFields:       []string{"salary", "ssn"},
		MaxCallsPerMinute:  30,
	})
	require.NoError(t, err)
	assert.True(t, policy.IsAllowed)
	assert.Equal(t, []string{"salary", "ssn"}, policy.RedactFields)

	// GetPolicyByRoleServerTool is the hot-path query used by the policy engine.
	fetched, err := q.GetPolicyByRoleServerTool(ctx, db.GetPolicyByRoleServerToolParams{
		RoleID:             role.ID,
		DownstreamServerID: srv.ID,
		ToolName:           "get_employee_record",
	})
	require.NoError(t, err)
	assert.Equal(t, policy.ID, fetched.ID)
	assert.True(t, fetched.IsAllowed)
}

func TestUpsertPolicy_UpdateExisting(t *testing.T) {
	q := testDB(t)
	ctx := context.Background()

	role, err := q.CreateRole(ctx, db.CreateRoleParams{Name: "hr_admin"})
	require.NoError(t, err)

	srv, err := q.CreateDownstreamServer(ctx, db.CreateDownstreamServerParams{
		Name: "bamboohr-upsert", BaseUrl: "http://x", AuthType: "api_key",
		AuthSecretRef: "env:X", ToolManifest: []byte(`[]`),
	})
	require.NoError(t, err)

	params := db.UpsertPolicyParams{
		RoleID: role.ID, DownstreamServerID: srv.ID,
		ToolName: "get_salary_info", IsAllowed: false,
		RedactFields: []string{}, MaxCallsPerMinute: 0,
	}

	first, err := q.UpsertPolicy(ctx, params)
	require.NoError(t, err)
	assert.False(t, first.IsAllowed)

	// Upsert again with is_allowed = true — should update in-place.
	params.IsAllowed = true
	params.RedactFields = []string{"ssn"}
	second, err := q.UpsertPolicy(ctx, params)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID, "ID must not change on upsert")
	assert.True(t, second.IsAllowed)
}

// ── Audit Events ───────────────────────────────────────────────────────────

func TestCreateAuditEvent(t *testing.T) {
	q := testDB(t)
	ctx := context.Background()

	role, _ := q.CreateRole(ctx, db.CreateRoleParams{Name: "audit_role"})
	user, _ := q.CreateUser(ctx, db.CreateUserParams{
		Email: "dan@example.com", DisplayName: "Dan", RoleID: role.ID,
	})
	srv, _ := q.CreateDownstreamServer(ctx, db.CreateDownstreamServerParams{
		Name: "bamboohr-audit", BaseUrl: "http://x", AuthType: "api_key",
		AuthSecretRef: "env:X", ToolManifest: []byte(`[]`),
	})

	latency := int32(42)
	event, err := q.CreateAuditEvent(ctx, db.CreateAuditEventParams{
		UserID:               user.ID,
		DownstreamServerID:   srv.ID,
		ToolName:             "list_employees",
		InputParamsRedacted:  []byte(`{"employee_id":"123"}`),
		Outcome:              "allowed",
		ResponseSummary:      []byte(`{"count":5}`),
		LatencyMs:            &latency,
	})
	require.NoError(t, err)
	assert.Equal(t, "allowed", event.Outcome)
	assert.Equal(t, "list_employees", event.ToolName)
}

// ptr returns a pointer to the given value (helper for nullable fields).
func ptr[T any](v T) *T { return &v }
