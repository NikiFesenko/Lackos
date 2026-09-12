package handlers_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/mcp-gate/mcp-gate/internal/db"
	"github.com/mcp-gate/mcp-gate/internal/handlers"
)

// newTestHandler spins up a Postgres container, runs migrations, and returns
// a fully wired Handler ready for httptest requests.
func newTestHandler(t *testing.T) *handlers.Handler {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("mcpgate_handler_test"),
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
	t.Cleanup(func() { _ = pgContainer.Terminate(ctx) })

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return handlers.New(db.New(pool))
}
