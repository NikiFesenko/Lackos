package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mcp-gate/mcp-gate/internal/auth"
	"github.com/mcp-gate/mcp-gate/internal/handlers"
)

func newAuthTestHandler(t *testing.T) (*handlers.Handler, *auth.TokenManager) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	tokenMgr := auth.NewTokenManager(rdb, 0)
	h := handlers.New(nil, tokenMgr)
	return h, tokenMgr
}

func TestIssueToken_Success(t *testing.T) {
	h, tokenMgr := newAuthTestHandler(t)
	userID := "00000000-0000-0000-0000-000000000001"

	body, _ := json.Marshal(map[string]string{"user_id": userID})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/token", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.IssueToken(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var res map[string]string
	err := json.NewDecoder(w.Body).Decode(&res)
	require.NoError(t, err)
	assert.Equal(t, userID, res["user_id"])
	assert.NotEmpty(t, res["token"])
	assert.Len(t, res["token"], 64)

	// Verify token works in TokenManager
	resolvedUser, err := tokenMgr.Validate(context.Background(), res["token"])
	require.NoError(t, err)
	assert.Equal(t, userID, resolvedUser)
}

func TestIssueToken_MissingUserID(t *testing.T) {
	h, _ := newAuthTestHandler(t)

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/token", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.IssueToken(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIssueToken_InvalidJSON(t *testing.T) {
	h, _ := newAuthTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/token", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	h.IssueToken(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRevokeToken_Success(t *testing.T) {
	h, tokenMgr := newAuthTestHandler(t)
	userID := "00000000-0000-0000-0000-000000000002"

	token, err := tokenMgr.Issue(context.Background(), userID)
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]string{"token": token})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/token", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.RevokeToken(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify token is no longer valid
	_, err = tokenMgr.Validate(context.Background(), token)
	assert.ErrorIs(t, err, auth.ErrTokenNotFound)
}

func TestRevokeToken_MissingToken(t *testing.T) {
	h, _ := newAuthTestHandler(t)

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/token", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.RevokeToken(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIssueToken_NoTokenManager(t *testing.T) {
	h := handlers.New(nil) // no token manager passed
	body, _ := json.Marshal(map[string]string{"user_id": "00000000-0000-0000-0000-000000000001"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/token", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.IssueToken(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
