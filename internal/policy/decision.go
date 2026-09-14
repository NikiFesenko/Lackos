// Package policy implements MCP Gate's policy evaluation engine.
// It is the authoritative answer to: "is this user allowed to call this tool?"
package policy

// Decision is the result of evaluating a policy for a single tool call.
type Decision struct {
	// Allowed reports whether the tool call is permitted.
	// When false the proxy must deny the request immediately.
	Allowed bool

	// RedactFields lists response field names that must be stripped before
	// the response is returned to the agent.
	RedactFields []string

	// MaxCallsPerMinute is the rate-limit cap for this (user, server) pair.
	MaxCallsPerMinute int32
}

// Deny is a convenience value returned whenever a request must be blocked.
var Deny = Decision{Allowed: false}
