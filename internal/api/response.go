// Package api provides shared HTTP response helpers used by all MCP Gate services.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// WriteJSON serialises v as JSON and writes it with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to write JSON response", "err", err)
	}
}

// WriteError writes a standard error envelope with the given HTTP status.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorResponse{Code: code, Message: message})
}

// ErrorResponse is the standard error envelope returned by all endpoints.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
