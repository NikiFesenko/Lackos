// Package handlers contains all HTTP request handlers for the admin API.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mcp-gate/mcp-gate/internal/api"
	"github.com/mcp-gate/mcp-gate/internal/db"
)

// ── Roles ─────────────────────────────────────────────────────────────────────

// ListRoles handles GET /api/v1/roles
func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.q.ListRoles(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to list roles")
		return
	}
	api.WriteJSON(w, http.StatusOK, roles)
}

// CreateRole handles POST /api/v1/roles
func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeInvalidPayload, "invalid JSON body")
		return
	}
	if body.Name == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "name is required")
		return
	}

	role, err := h.q.CreateRole(r.Context(), db.CreateRoleParams{
		Name:        body.Name,
		Description: body.Description,
	})
	if err != nil {
		if isUniqueViolation(err) {
			api.WriteError(w, http.StatusConflict, api.ErrCodeConflict, "role name already exists")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to create role")
		return
	}
	api.WriteJSON(w, http.StatusCreated, role)
}

// GetRole handles GET /api/v1/roles/{id}
func (h *Handler) GetRole(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	role, err := h.q.GetRole(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "role not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to get role")
		return
	}
	api.WriteJSON(w, http.StatusOK, role)
}

// UpdateRole handles PUT /api/v1/roles/{id}
func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	var body struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeInvalidPayload, "invalid JSON body")
		return
	}
	if body.Name == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "name is required")
		return
	}

	role, err := h.q.UpdateRole(r.Context(), db.UpdateRoleParams{
		ID:          id,
		Name:        body.Name,
		Description: body.Description,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "role not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to update role")
		return
	}
	api.WriteJSON(w, http.StatusOK, role)
}

// DeleteRole handles DELETE /api/v1/roles/{id}
func (h *Handler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	if err := h.q.DeleteRole(r.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "role not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to delete role")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Shared helpers ─────────────────────────────────────────────────────────────

// parseUUID parses a UUID string from a path segment.
// It writes a 400 error and returns false on failure.
func parseUUID(w http.ResponseWriter, s string) (pgtype.UUID, bool) {
	var id pgtype.UUID
	if err := id.Scan(s); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeInvalidUUID, "invalid UUID format")
		return pgtype.UUID{}, false
	}
	return id, true
}

// isUniqueViolation returns true when the error is a Postgres unique-constraint violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return containsAny(err.Error(), "23505", "unique constraint", "duplicate key")
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
