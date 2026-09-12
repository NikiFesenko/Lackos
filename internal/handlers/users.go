package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/mcp-gate/mcp-gate/internal/api"
	"github.com/mcp-gate/mcp-gate/internal/db"
)

// ListUsers handles GET /api/v1/users
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.q.ListActiveUsers(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to list users")
		return
	}
	api.WriteJSON(w, http.StatusOK, users)
}

// CreateUser handles POST /api/v1/users
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		RoleID      string `json:"role_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeInvalidPayload, "invalid JSON body")
		return
	}
	switch {
	case body.Email == "":
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "email is required")
		return
	case body.DisplayName == "":
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "display_name is required")
		return
	case body.RoleID == "":
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "role_id is required")
		return
	}

	roleID, ok := parseUUID(w, body.RoleID)
	if !ok {
		return
	}

	user, err := h.q.CreateUser(r.Context(), db.CreateUserParams{
		Email:       body.Email,
		DisplayName: body.DisplayName,
		RoleID:      roleID,
	})
	if err != nil {
		if isUniqueViolation(err) {
			api.WriteError(w, http.StatusConflict, api.ErrCodeConflict, "email already registered")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to create user")
		return
	}
	api.WriteJSON(w, http.StatusCreated, user)
}

// GetUser handles GET /api/v1/users/{id}
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	user, err := h.q.GetUser(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "user not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to get user")
		return
	}
	api.WriteJSON(w, http.StatusOK, user)
}

// UpdateUserRole handles PUT /api/v1/users/{id}/role
func (h *Handler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	var body struct {
		RoleID string `json:"role_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeInvalidPayload, "invalid JSON body")
		return
	}
	roleID, ok := parseUUID(w, body.RoleID)
	if !ok {
		return
	}

	user, err := h.q.UpdateUserRole(r.Context(), db.UpdateUserRoleParams{
		ID:     id,
		RoleID: roleID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "user not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to update user role")
		return
	}
	api.WriteJSON(w, http.StatusOK, user)
}

// DeactivateUser handles DELETE /api/v1/users/{id}
// Soft-deletes by setting is_active = false — never hard-deletes a user with audit history.
func (h *Handler) DeactivateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	user, err := h.q.SetUserActive(r.Context(), db.SetUserActiveParams{
		ID:       id,
		IsActive: false,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "user not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to deactivate user")
		return
	}
	api.WriteJSON(w, http.StatusOK, user)
}
