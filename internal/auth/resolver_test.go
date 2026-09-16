package auth_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mcp-gate/mcp-gate/internal/auth"
)

func TestEnvSecretResolver_ResolveSuccess(t *testing.T) {
	t.Setenv("TEST_DOWNSTREAM_KEY", "secret-key-12345")

	resolver := auth.EnvSecretResolver{}
	val, err := resolver.Resolve("env:TEST_DOWNSTREAM_KEY")
	require.NoError(t, err)
	assert.Equal(t, "secret-key-12345", val)
}

func TestEnvSecretResolver_UnsetEnvVar(t *testing.T) {
	_ = os.Unsetenv("TEST_UNSET_KEY_9999")

	resolver := auth.EnvSecretResolver{}
	_, err := resolver.Resolve("env:TEST_UNSET_KEY_9999")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not set")
}

func TestEnvSecretResolver_InvalidFormat(t *testing.T) {
	resolver := auth.EnvSecretResolver{}
	_, err := resolver.Resolve("vault:secret/data/key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unrecognised secret ref format")
}
