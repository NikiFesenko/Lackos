package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/mcp-gate/mcp-gate/internal/api"
	"github.com/mcp-gate/mcp-gate/internal/db"
)

// ListPolicies handles GET /api/v1/policies?role_id=<uuid>
func (h *Handler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	roleIDStr := r.URL.Query().Get("role_id")
	if roleIDStr == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "role_id query parameter is required")
		return
	}
	roleID, ok := parseUUID(w, roleIDStr)
	if !ok {
		return
	}
	policies, err := h.q.ListPoliciesByRole(r.Context(), roleID)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to list policies")
		return
	}
	api.WriteJSON(w, http.StatusOK, policies)
}

// UpsertPolicy handles POST /api/v1/policies
// Creates or updates the policy for a (role, server, tool) triple atomically.
func (h *Handler) UpsertPolicy(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RoleID             string   `json:"role_id"`
		DownstreamServerID string   `json:"downstream_server_id"`
		ToolName           string   `json:"tool_name"`
		IsAllowed          bool     `json:"is_allowed"`
		RedactFields       []string `json:"redact_fields"`
		MaxCallsPerMinute  int32    `json:"max_calls_per_minute"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeInvalidPayload, "invalid JSON body")
		return
	}
	switch {
	case body.RoleID == "":
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "role_id is required")
		return
	case body.DownstreamServerID == "":
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "downstream_server_id is required")
		return
	case body.ToolName == "":
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "tool_name is required")
		return
	}

	roleID, ok := parseUUID(w, body.RoleID)
	if !ok {
		return
	}
	serverID, ok := parseUUID(w, body.DownstreamServerID)
	if !ok {
		return
	}

	if body.RedactFields == nil {
		body.RedactFields = []string{}
	}
	if body.MaxCallsPerMinute <= 0 {
		body.MaxCallsPerMinute = 30
	}

	policy, err := h.q.UpsertPolicy(r.Context(), db.UpsertPolicyParams{
		RoleID:             roleID,
		DownstreamServerID: serverID,
		ToolName:           body.ToolName,
		IsAllowed:          body.IsAllowed,
		RedactFields:       body.RedactFields,
		MaxCallsPerMinute:  body.MaxCallsPerMinute,
	})
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to upsert policy")
		return
	}
	api.WriteJSON(w, http.StatusOK, policy)
}

// GetPolicy handles GET /api/v1/policies/{id}
func (h *Handler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	policy, err := h.q.GetPolicy(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "policy not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to get policy")
		return
	}
	api.WriteJSON(w, http.StatusOK, policy)
}

// DeletePolicy handles DELETE /api/v1/policies/{id}
func (h *Handler) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	if err := h.q.DeletePolicy(r.Context(), id); err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to delete policy")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
