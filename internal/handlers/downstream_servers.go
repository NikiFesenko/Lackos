package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/mcp-gate/mcp-gate/internal/api"
	"github.com/mcp-gate/mcp-gate/internal/db"
)

// ListDownstreamServers handles GET /api/v1/downstream-servers
func (h *Handler) ListDownstreamServers(w http.ResponseWriter, r *http.Request) {
	servers, err := h.q.ListDownstreamServers(r.Context())
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to list servers")
		return
	}
	api.WriteJSON(w, http.StatusOK, servers)
}

// CreateDownstreamServer handles POST /api/v1/downstream-servers
func (h *Handler) CreateDownstreamServer(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name          string `json:"name"`
		BaseURL       string `json:"base_url"`
		AuthType      string `json:"auth_type"`
		AuthSecretRef string `json:"auth_secret_ref"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeInvalidPayload, "invalid JSON body")
		return
	}
	switch {
	case body.Name == "":
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "name is required")
		return
	case body.BaseURL == "":
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "base_url is required")
		return
	case body.AuthType != "api_key" && body.AuthType != "oauth2":
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "auth_type must be 'api_key' or 'oauth2'")
		return
	case body.AuthSecretRef == "":
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "auth_secret_ref is required")
		return
	}

	srv, err := h.q.CreateDownstreamServer(r.Context(), db.CreateDownstreamServerParams{
		Name:          body.Name,
		BaseUrl:       body.BaseURL,
		AuthType:      body.AuthType,
		AuthSecretRef: body.AuthSecretRef,
		ToolManifest:  []byte(`[]`),
	})
	if err != nil {
		if isUniqueViolation(err) {
			api.WriteError(w, http.StatusConflict, api.ErrCodeConflict, "server name already exists")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to create server")
		return
	}
	api.WriteJSON(w, http.StatusCreated, srv)
}

// GetDownstreamServer handles GET /api/v1/downstream-servers/{id}
func (h *Handler) GetDownstreamServer(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	srv, err := h.q.GetDownstreamServer(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "server not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to get server")
		return
	}
	api.WriteJSON(w, http.StatusOK, srv)
}

// UpdateDownstreamServer handles PUT /api/v1/downstream-servers/{id}
func (h *Handler) UpdateDownstreamServer(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	var body struct {
		Name          string `json:"name"`
		BaseURL       string `json:"base_url"`
		AuthType      string `json:"auth_type"`
		AuthSecretRef string `json:"auth_secret_ref"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, api.ErrCodeInvalidPayload, "invalid JSON body")
		return
	}

	srv, err := h.q.UpdateDownstreamServer(r.Context(), db.UpdateDownstreamServerParams{
		ID:            id,
		Name:          body.Name,
		BaseUrl:       body.BaseURL,
		AuthType:      body.AuthType,
		AuthSecretRef: body.AuthSecretRef,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "server not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to update server")
		return
	}
	api.WriteJSON(w, http.StatusOK, srv)
}

// SetDownstreamServerActive handles DELETE /api/v1/downstream-servers/{id}
// Soft-deactivates the server; never hard-deletes since policies reference it.
func (h *Handler) SetDownstreamServerActive(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	srv, err := h.q.SetDownstreamServerActive(r.Context(), db.SetDownstreamServerActiveParams{
		ID:       id,
		IsActive: false,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "server not found")
			return
		}
		api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "failed to deactivate server")
		return
	}
	api.WriteJSON(w, http.StatusOK, srv)
}
