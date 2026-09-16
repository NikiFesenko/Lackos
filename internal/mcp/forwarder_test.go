package mcp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mcp-gate/mcp-gate/internal/mcp"
)

type mockResolver struct {
	secrets map[string]string
	err     error
}

func (m *mockResolver) Resolve(ref string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if val, ok := m.secrets[ref]; ok {
		return val, nil
	}
	return "", errors.New("secret not found")
}

func TestForwarder_ForwardSuccess_BearerAuth(t *testing.T) {
	var capturedAuth string
	var capturedBody mcp.Request

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)

		res := map[string]any{"status": "ok"}
		rawRes, _ := json.Marshal(res)
		resp := mcp.Response{
			JSONRPC: "2.0",
			ID:      capturedBody.ID,
			Result:  rawRes,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	resolver := &mockResolver{secrets: map[string]string{"env:TEST_SECRET": "token-xyz"}}
	f := mcp.NewForwarder(resolver)

	req := mcp.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`"123"`),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"get_info"}`),
	}

	resp, err := f.Forward(context.Background(), ts.URL, "bearer", "env:TEST_SECRET", req)
	require.NoError(t, err)
	assert.Equal(t, "Bearer token-xyz", capturedAuth)
	assert.Equal(t, "tools/call", capturedBody.Method)
	assert.NotNil(t, resp.Result)
}

func TestForwarder_ForwardSuccess_ApiKeyAuth(t *testing.T) {
	var capturedKey string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedKey = r.Header.Get("X-Api-Key")
		resp := mcp.Response{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`"1"`),
			Result:  json.RawMessage(`{}`),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	resolver := &mockResolver{secrets: map[string]string{"env:API_KEY": "apikey-123"}}
	f := mcp.NewForwarder(resolver)

	req := mcp.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`"1"`),
		Method:  "tools/call",
	}

	_, err := f.Forward(context.Background(), ts.URL, "api_key", "env:API_KEY", req)
	require.NoError(t, err)
	assert.Equal(t, "apikey-123", capturedKey)
}

func TestForwarder_SecretResolutionError(t *testing.T) {
	resolver := &mockResolver{err: errors.New("resolution failed")}
	f := mcp.NewForwarder(resolver)

	req := mcp.Request{JSONRPC: "2.0", ID: json.RawMessage(`"1"`), Method: "tools/call"}
	_, err := f.Forward(context.Background(), "http://localhost:9999", "bearer", "env:BAD", req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resolve auth secret")
}

func TestForwarder_DownstreamHttpError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server down", http.StatusInternalServerError)
	}))
	defer ts.Close()

	f := mcp.NewForwarder(&mockResolver{})
	req := mcp.Request{JSONRPC: "2.0", ID: json.RawMessage(`"1"`), Method: "tools/call"}

	_, err := f.Forward(context.Background(), ts.URL, "none", "", req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "downstream HTTP 500")
}
