package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mcp-gate/mcp-gate/internal/db"
)

// createTestRole is a helper that creates a role and returns its string UUID.
func createTestRole(t *testing.T, h interface {
	CreateRole(http.ResponseWriter, *http.Request)
}, name string) string {
	t.Helper()
	body := fmt.Sprintf(`{"name":%q}`, name)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/roles", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.CreateRole(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
	var role db.Role
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &role))
	// pgtype.UUID → string
	uid, err := role.ID.Value()
	require.NoError(t, err)
	return fmt.Sprintf("%v", uid)
}

func TestCreateUser_Success(t *testing.T) {
	h := newTestHandler(t)
	roleID := createTestRole(t, h, "employee")

	body := fmt.Sprintf(`{"email":"alice@example.com","display_name":"Alice","role_id":%q}`, roleID)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.CreateUser(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var user db.User
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &user))
	assert.Equal(t, "alice@example.com", user.Email)
	assert.True(t, user.IsActive)
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	h := newTestHandler(t)
	roleID := createTestRole(t, h, "manager")

	body := fmt.Sprintf(`{"email":"bob@example.com","display_name":"Bob","role_id":%q}`, roleID)
	for i := range 2 {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.CreateUser(rec, req)
		if i == 0 {
			assert.Equal(t, http.StatusCreated, rec.Code)
		} else {
			assert.Equal(t, http.StatusConflict, rec.Code)
		}
	}
}

func TestCreateUser_MissingFields(t *testing.T) {
	h := newTestHandler(t)

	cases := []struct {
		name string
		body string
	}{
		{"no email", `{"display_name":"X","role_id":"00000000-0000-0000-0000-000000000001"}`},
		{"no display_name", `{"email":"x@x.com","role_id":"00000000-0000-0000-0000-000000000001"}`},
		{"no role_id", `{"email":"x@x.com","display_name":"X"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.CreateUser(rec, req)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestDeactivateUser(t *testing.T) {
	h := newTestHandler(t)
	roleID := createTestRole(t, h, "recruiter")

	// Create user
	body := fmt.Sprintf(`{"email":"carol@example.com","display_name":"Carol","role_id":%q}`, roleID)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.CreateUser(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var user db.User
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &user))
	uid, _ := user.ID.Value()
	userID := fmt.Sprintf("%v", uid)

	// Deactivate
	req2 := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+userID, nil)
	req2.SetPathValue("id", userID)
	rec2 := httptest.NewRecorder()
	h.DeactivateUser(rec2, req2)

	assert.Equal(t, http.StatusOK, rec2.Code)
	var updated db.User
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &updated))
	assert.False(t, updated.IsActive)
}
