// Package config loads all MCP Gate configuration from environment variables.
// Call Load() once at startup; the returned Config is safe for concurrent read access.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the resolved configuration for any MCP Gate binary.
// Each binary uses only the subset of fields relevant to it.
type Config struct {
	// ── Postgres ─────────────────────────────────────────────────────────────
	PostgresDSN string // e.g. "postgres://user:pass@localhost:5432/mcpgate?sslmode=disable"

	// ── Redis ────────────────────────────────────────────────────────────────
	RedisURL string // e.g. "redis://localhost:6379/0"

	// ── RabbitMQ ─────────────────────────────────────────────────────────────
	RabbitMQURL string // e.g. "amqp://guest:guest@localhost:5672/"

	// ── Proxy service ────────────────────────────────────────────────────────
	ProxyPort int // default 8080

	// ── Admin API service ────────────────────────────────────────────────────
	AdminAPIPort int    // default 8081
	AdminToken   string // hardcoded dev token; replaced with JWT in Phase 8

	// ── General ──────────────────────────────────────────────────────────────
	LogLevel string // "debug" | "info" | "warn" | "error"  (default "info")
}

// Load reads all environment variables and returns a populated Config.
// It panics with a descriptive message if any required variable is missing.
func Load() *Config {
	return &Config{
		PostgresDSN:  requireEnv("POSTGRES_DSN"),
		RedisURL:     requireEnv("REDIS_URL"),
		RabbitMQURL:  requireEnv("RABBITMQ_URL"),
		AdminToken:   requireEnv("ADMIN_TOKEN"),
		ProxyPort:    optionalInt("PROXY_PORT", 8080),
		AdminAPIPort: optionalInt("ADMINAPI_PORT", 8081),
		LogLevel:     optionalStr("LOG_LEVEL", "info"),
	}
}

// requireEnv returns the value of the named environment variable or panics.
func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("mcp-gate: required environment variable %q is not set", key))
	}
	return v
}

// optionalStr returns the env var value or the provided default.
func optionalStr(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// optionalInt returns the env var value parsed as int, or the provided default.
func optionalInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		panic(fmt.Sprintf("mcp-gate: environment variable %q must be an integer, got %q", key, v))
	}
	return n
}
