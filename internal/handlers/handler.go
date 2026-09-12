package handlers

import "github.com/mcp-gate/mcp-gate/internal/db"

// Handler holds shared dependencies for all HTTP handlers.
// Construct one with New() and register its methods on a mux.
type Handler struct {
	q db.Querier
}

// New creates a Handler backed by the given Querier.
func New(q db.Querier) *Handler {
	return &Handler{q: q}
}
