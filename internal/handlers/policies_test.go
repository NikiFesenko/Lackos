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

// createTestServer is a helper that registers a downstream server and returns its UUID string.
func createTestServer(t *testing.T, h interface {
	CreateDownstreamServer(http.ResponseWriter, *http.Request)
}, name string) string {
	t.Helper()
	body := fmt.Sprintf(`{"name":%q,"base_url":"http://localhost:9000","auth_type":"api_key","auth_secret_ref":"env:TEST_KEY"}`, name)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/downstream-servers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.CreateDownstreamServer(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
	var srv db.DownstreamServer
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &srv))
	uid, _ := srv.ID.Value()
	return fmt.Sprintf("%v", uid)
}

func TestUpsertPolicy_CreateAndUpdate(t *testing.T) {
	h := newTestHandler(t)
	roleID := createTestRole(t, h, "recruiter_pol")
	serverID := createTestServer(t, h, "bamboohr-pol")

	payload := func(allowed bool, fields []string) string {
		f, _ := json.Marshal(fields)
		return fmt.Sprintf(`{
			"role_id":%q,
			"downstream_server_id":%q,
			"tool_name":"get_employee_record",
			"is_allowed":%v,
			"redact_fields":%s,
			"max_calls_per_minute":30
		}`, roleID, serverID, allowed, f)
	}

	// Create
	req := httptest.NewRequest(http.MethodPost, "/api/v1/policies", bytes.NewBufferString(payload(false, []string{})))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.UpsertPolicy(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	var p1 db.Policy
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &p1))
	assert.False(t, p1.IsAllowed)

	// Update in-place — same (role, server, tool) triple, different values
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/policies", bytes.NewBufferString(payload(true, []string{"salary", "ssn"})))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	h.UpsertPolicy(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)
	var p2 db.Policy
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &p2))
	// UUID must be stable — upsert must not create a new row
	assert.Equal(t, p1.ID, p2.ID)
	assert.True(t, p2.IsAllowed)
	assert.Equal(t, []string{"salary", "ssn"}, p2.RedactFields)
}

func TestListPolicies_ByRole(t *testing.T) {
	h := newTestHandler(t)
	roleID := createTestRole(t, h, "manager_pol")
	serverID := createTestServer(t, h, "bamboohr-list")

	tools := []string{"get_employee_record", "list_pto_requests", "get_salary_info"}
	for _, tool := range tools {
		body := fmt.Sprintf(`{"role_id":%q,"downstream_server_id":%q,"tool_name":%q,"is_allowed":true}`,
			roleID, serverID, tool)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/policies", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.UpsertPolicy(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/policies?role_id="+roleID, nil)
	rec := httptest.NewRecorder()
	h.ListPolicies(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	var policies []db.Policy
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &policies))
	assert.Len(t, policies, 3)
}

func TestUpsertPolicy_MissingFields(t *testing.T) {
	h := newTestHandler(t)
	cases := []string{
		`{"downstream_server_id":"x","tool_name":"t"}`,
		`{"role_id":"x","tool_name":"t"}`,
		`{"role_id":"x","downstream_server_id":"y"}`,
	}
	for _, body := range cases {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/policies", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.UpsertPolicy(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	}
}
