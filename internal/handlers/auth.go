package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mcp-gate/mcp-gate/internal/api"
	"github.com/mcp-gate/mcp-gate/internal/auth"
)

// IssueToken handles POST /api/v1/auth/token
// Admins call this to generate a per-user proxy token that agents present to
// the proxy in the Authorization: Bearer header.
func (h *Handler) IssueToken(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeInvalidPayload, "invalid JSON body")
		return
	}
	if body.UserID == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "user_id is required")
		return
	}
	if h.tokenMgr == nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "token manager not configured")
		return
	}

	token, err := h.tokenMgr.Issue(r.Context(), body.UserID)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to issue token")
		return
	}

	api.WriteJSON(w, http.StatusCreated, map[string]string{
		"token":   token,
		"user_id": body.UserID,
		"note":    "Store this token securely — it cannot be retrieved again.",
	})
}

// RevokeToken handles DELETE /api/v1/auth/token
func (h *Handler) RevokeToken(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeInvalidPayload, "invalid JSON body")
		return
	}
	if body.Token == "" {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "token is required")
		return
	}
	if h.tokenMgr == nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "token manager not configured")
		return
	}

	if err := h.tokenMgr.Revoke(r.Context(), body.Token); err != nil {
		if errors.Is(err, auth.ErrTokenInvalid) {
			api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid token format")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to revoke token")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
