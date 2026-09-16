package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mcp-gate/mcp-gate/internal/auth"
	"github.com/mcp-gate/mcp-gate/internal/db"
	"github.com/mcp-gate/mcp-gate/internal/mcp"
	"github.com/mcp-gate/mcp-gate/internal/policy"
	"github.com/mcp-gate/mcp-gate/internal/ratelimit"
)

// mockQuerier provides mock DB responses for the proxy integration tests.
type mockQuerier struct {
	db.Querier
	user       db.User
	userErr    error
	downstream db.DownstreamServer
	downErr    error
	policy     db.Policy
	policyErr  error
}

func (m *mockQuerier) GetUser(_ context.Context, _ pgtype.UUID) (db.User, error) {
	if m.userErr != nil {
		return db.User{}, m.userErr
	}
	return m.user, nil
}

func (m *mockQuerier) GetDownstreamServerByName(_ context.Context, _ string) (db.DownstreamServer, error) {
	if m.downErr != nil {
		return db.DownstreamServer{}, m.downErr
	}
	return m.downstream, nil
}

func (m *mockQuerier) GetPolicyByRoleServerTool(_ context.Context, _ db.GetPolicyByRoleServerToolParams) (db.Policy, error) {
	if m.policyErr != nil {
		return db.Policy{}, m.policyErr
	}
	return m.policy, nil
}

type staticResolver map[string]string

func (s staticResolver) Resolve(ref string) (string, error) {
	if val, ok := s[ref]; ok {
		return val, nil
	}
	return "", nil
}

// setupProxyTest creates the mock downstream server, in-memory redis, mock DB, and proxy handler.
func setupProxyTest(t *testing.T, mq *mockQuerier) (*httptest.Server, *auth.TokenManager, string) {
	t.Helper()

	// 1. Mock downstream MCP server
	mockDownstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record := map[string]any{
			"employee_id": "EMP-0042",
			"name":        "Alice Testuser",
			"department":  "Engineering",
			"salary":      120000,
			"ssn":         "123-45-6789",
		}
		rawText, _ := json.Marshal(record)
		result := mcp.ToolResult{
			Content: []mcp.ContentItem{{Type: "text", Text: string(rawText)}},
			IsError: false,
		}
		rawRes, _ := json.Marshal(result)
		resp := mcp.Response{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Result:  rawRes,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(mockDownstream.Close)

	// 2. Miniredis
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	tokenMgr := auth.NewTokenManager(rdb, 0)
	policyEngine := policy.New(mq, rdb)
	limiter := ratelimit.New(rdb)
	forwarder := mcp.NewForwarder(staticResolver{"env:TEST_KEY": "dummy-key"})

	// Configure downstream server base URL
	mq.downstream.BaseUrl = mockDownstream.URL

	handler := makeProxyHandler(mq, tokenMgr, policyEngine, limiter, forwarder)
	proxyServer := httptest.NewServer(handler)
	t.Cleanup(proxyServer.Close)

	return proxyServer, tokenMgr, mockDownstream.URL
}

func testUUID(s string) pgtype.UUID {
	var id pgtype.UUID
	_ = id.Scan(s)
	return id
}

func TestProxy_FullFlow_AllowedAndRedacted(t *testing.T) {
	userIDStr := "00000000-0000-0000-0000-000000000001"
	roleIDStr := "00000000-0000-0000-0000-000000000002"
	serverIDStr := "00000000-0000-0000-0000-000000000003"

	mq := &mockQuerier{
		user: db.User{
			ID:       testUUID(userIDStr),
			Email:    "alice@example.com",
			RoleID:   testUUID(roleIDStr),
			IsActive: true,
		},
		downstream: db.DownstreamServer{
			ID:            testUUID(serverIDStr),
			Name:          "mock-bamboohr",
			AuthType:      "bearer",
			AuthSecretRef: "env:TEST_KEY",
			IsActive:      true,
		},
		policy: db.Policy{
			ID:                 testUUID("00000000-0000-0000-0000-000000000004"),
			RoleID:             testUUID(roleIDStr),
			DownstreamServerID: testUUID(serverIDStr),
			ToolName:           "get_employee_record",
			IsAllowed:          true,
			RedactFields:       []string{"salary", "ssn"},
			MaxCallsPerMinute:  60,
		},
	}

	proxyServer, tokenMgr, _ := setupProxyTest(t, mq)

	// Issue user token
	token, err := tokenMgr.Issue(context.Background(), userIDStr)
	require.NoError(t, err)

	// Build MCP request
	reqPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"server": "mock-bamboohr",
			"name":   "get_employee_record",
			"arguments": map[string]any{
				"employee_id": "EMP-0042",
			},
		},
	}
	body, _ := json.Marshal(reqPayload)

	httpReq, err := http.NewRequest(http.MethodPost, proxyServer.URL+"/", bytes.NewReader(body))
	require.NoError(t, err)
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var rpcResp mcp.Response
	err = json.NewDecoder(resp.Body).Decode(&rpcResp)
	require.NoError(t, err)
	assert.Nil(t, rpcResp.Error)
	assert.NotNil(t, rpcResp.Result)

	var result mcp.ToolResult
	err = json.Unmarshal(rpcResp.Result, &result)
	require.NoError(t, err)
	require.Len(t, result.Content, 1)

	text := result.Content[0].Text
	assert.Contains(t, text, `"salary":"[REDACTED]"`)
	assert.Contains(t, text, `"ssn":"[REDACTED]"`)
	assert.Contains(t, text, `"name":"Alice Testuser"`)
	assert.Contains(t, text, `"department":"Engineering"`)
}

func TestProxy_DenyOnMissingPolicy(t *testing.T) {
	userIDStr := "00000000-0000-0000-0000-000000000001"
	roleIDStr := "00000000-0000-0000-0000-000000000002"
	serverIDStr := "00000000-0000-0000-0000-000000000003"

	mq := &mockQuerier{
		user: db.User{
			ID:       testUUID(userIDStr),
			RoleID:   testUUID(roleIDStr),
			IsActive: true,
		},
		downstream: db.DownstreamServer{
			ID:       testUUID(serverIDStr),
			Name:     "mock-bamboohr",
			IsActive: true,
		},
		policyErr: pgx.ErrNoRows, // No policy configured
	}

	proxyServer, tokenMgr, _ := setupProxyTest(t, mq)

	token, err := tokenMgr.Issue(context.Background(), userIDStr)
	require.NoError(t, err)

	reqPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"server": "mock-bamboohr",
			"name":   "unauthorized_tool",
		},
	}
	body, _ := json.Marshal(reqPayload)

	httpReq, err := http.NewRequest(http.MethodPost, proxyServer.URL+"/", bytes.NewReader(body))
	require.NoError(t, err)
	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	var rpcResp mcp.Response
	_ = json.NewDecoder(resp.Body).Decode(&rpcResp)
	require.NotNil(t, rpcResp.Error)
	assert.Equal(t, mcp.CodeForbidden, rpcResp.Error.Code)
	assert.Contains(t, rpcResp.Error.Message, "denied by policy")
}

func TestProxy_Unauthorized_MissingToken(t *testing.T) {
	mq := &mockQuerier{}
	proxyServer, _, _ := setupProxyTest(t, mq)

	reqPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params":  map[string]any{"server": "test", "name": "tool"},
	}
	body, _ := json.Marshal(reqPayload)

	resp, err := http.Post(proxyServer.URL+"/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	var rpcResp mcp.Response
	_ = json.NewDecoder(resp.Body).Decode(&rpcResp)
	require.NotNil(t, rpcResp.Error)
	assert.Equal(t, mcp.CodeUnauthorized, rpcResp.Error.Code)
}

func TestProxy_RateLimitExceeded(t *testing.T) {
	userIDStr := "00000000-0000-0000-0000-000000000001"
	roleIDStr := "00000000-0000-0000-0000-000000000002"
	serverIDStr := "00000000-0000-0000-0000-000000000003"

	mq := &mockQuerier{
		user: db.User{
			ID:       testUUID(userIDStr),
			RoleID:   testUUID(roleIDStr),
			IsActive: true,
		},
		downstream: db.DownstreamServer{
			ID:       testUUID(serverIDStr),
			Name:     "mock-bamboohr",
			IsActive: true,
		},
		policy: db.Policy{
			ID:                 testUUID("00000000-0000-0000-0000-000000000004"),
			RoleID:             testUUID(roleIDStr),
			DownstreamServerID: testUUID(serverIDStr),
			ToolName:           "get_employee_record",
			IsAllowed:          true,
			MaxCallsPerMinute:  1, // Allow only 1 call
		},
	}

	proxyServer, tokenMgr, _ := setupProxyTest(t, mq)

	token, err := tokenMgr.Issue(context.Background(), userIDStr)
	require.NoError(t, err)

	reqPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"server": "mock-bamboohr",
			"name":   "get_employee_record",
		},
	}
	body, _ := json.Marshal(reqPayload)

	// Call 1: Success
	req1, _ := http.NewRequest(http.MethodPost, proxyServer.URL+"/", bytes.NewReader(body))
	req1.Header.Set("Authorization", "Bearer "+token)
	resp1, err := http.DefaultClient.Do(req1)
	require.NoError(t, err)
	resp1.Body.Close()
	assert.Equal(t, http.StatusOK, resp1.StatusCode)

	// Call 2: Rate limited
	req2, _ := http.NewRequest(http.MethodPost, proxyServer.URL+"/", bytes.NewReader(body))
	req2.Header.Set("Authorization", "Bearer "+token)
	resp2, err := http.DefaultClient.Do(req2)
	require.NoError(t, err)
	defer resp2.Body.Close()

	var rpcResp mcp.Response
	_ = json.NewDecoder(resp2.Body).Decode(&rpcResp)
	require.NotNil(t, rpcResp.Error)
	assert.Equal(t, mcp.CodeRateLimited, rpcResp.Error.Code)
	assert.Equal(t, "1", resp2.Header.Get("X-RateLimit-Limit"))
	assert.Equal(t, "0", resp2.Header.Get("X-RateLimit-Remaining"))
}

func TestProxy_DeactivatedUser_Denied(t *testing.T) {
	userIDStr := "00000000-0000-0000-0000-000000000001"

	mq := &mockQuerier{
		user: db.User{
			ID:       testUUID(userIDStr),
			IsActive: false, // Inactive user
		},
	}

	proxyServer, tokenMgr, _ := setupProxyTest(t, mq)

	token, err := tokenMgr.Issue(context.Background(), userIDStr)
	require.NoError(t, err)

	reqPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params":  map[string]any{"server": "srv", "name": "tool"},
	}
	body, _ := json.Marshal(reqPayload)

	req, _ := http.NewRequest(http.MethodPost, proxyServer.URL+"/", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var rpcResp mcp.Response
	_ = json.NewDecoder(resp.Body).Decode(&rpcResp)
	require.NotNil(t, rpcResp.Error)
	assert.Equal(t, mcp.CodeUnauthorized, rpcResp.Error.Code)
	assert.Contains(t, rpcResp.Error.Message, "deactivated")
}
