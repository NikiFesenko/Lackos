package auth

import (
	"fmt"
	"os"
	"strings"
)

// EnvSecretResolver resolves auth_secret_ref strings of the form "env:VAR_NAME"
// by reading the named environment variable at runtime.
//
// This is the Phase 5 implementation. Phase 8 will add a Vault resolver behind
// the same SecretResolver interface used by the MCP Forwarder.
type EnvSecretResolver struct{}

// Resolve parses the ref string and returns the resolved secret value.
// Supported formats:
//   - "env:VAR_NAME" → os.Getenv("VAR_NAME")
//
// Returns an error if the format is unrecognised or the env var is empty.
func (r EnvSecretResolver) Resolve(ref string) (string, error) {
	if after, ok := strings.CutPrefix(ref, "env:"); ok {
		val := os.Getenv(after)
		if val == "" {
			return "", fmt.Errorf("auth: env var %q referenced by secret ref is not set", after)
		}
		return val, nil
	}
	return "", fmt.Errorf("auth: unrecognised secret ref format %q (expected \"env:VAR_NAME\")", ref)
}
