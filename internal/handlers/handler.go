package handlers

import (
	"github.com/mcp-gate/mcp-gate/internal/auth"
	"github.com/mcp-gate/mcp-gate/internal/db"
)

// Handler holds shared dependencies for all HTTP handlers.
// Construct one with New() and register its methods on a mux.
type Handler struct {
	q        db.Querier
	tokenMgr *auth.TokenManager
}

// New creates a Handler backed by the given Querier and optional TokenManager.
// tokenMgr may be omitted or nil — token endpoints will return an error if called without it.
func New(q db.Querier, tokenMgr ...*auth.TokenManager) *Handler {
	h := &Handler{q: q}
	if len(tokenMgr) > 0 {
		h.tokenMgr = tokenMgr[0]
	}
	return h
}
