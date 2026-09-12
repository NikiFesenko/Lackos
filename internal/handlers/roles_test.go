package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mcp-gate/mcp-gate/internal/db"
)

func TestListRoles_Empty(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/roles", nil)
	rec := httptest.NewRecorder()
	h.ListRoles(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var roles []db.Role
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &roles))
	assert.Empty(t, roles)
}

func TestCreateRole_Success(t *testing.T) {
	h := newTestHandler(t)
	body := `{"name":"hr_admin","description":"Full HR admin"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/roles", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.CreateRole(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var role db.Role
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &role))
	assert.Equal(t, "hr_admin", role.Name)
	assert.NotEmpty(t, role.ID)
}

func TestCreateRole_MissingName(t *testing.T) {
	h := newTestHandler(t)
	body := `{"description":"no name"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/roles", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.CreateRole(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateRole_DuplicateName(t *testing.T) {
	h := newTestHandler(t)
	body := `{"name":"unique_role"}`

	for i := range 2 {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/roles", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.CreateRole(rec, req)
		if i == 0 {
			assert.Equal(t, http.StatusCreated, rec.Code)
		} else {
			assert.Equal(t, http.StatusConflict, rec.Code)
		}
	}
}

func TestGetRole_NotFound(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/roles/00000000-0000-0000-0000-000000000099", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000099")
	rec := httptest.NewRecorder()
	h.GetRole(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetRole_InvalidUUID(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/roles/not-a-uuid", nil)
	req.SetPathValue("id", "not-a-uuid")
	rec := httptest.NewRecorder()
	h.GetRole(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
