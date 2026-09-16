// Binary mockdownstream is a lightweight fake MCP server for local development
// and integration testing. It responds to any "tools/call" request with a
// canned employee record that includes a salary field — so redaction can be
// verified end-to-end without a real downstream service.
//
// Usage:
//
//	go run ./cmd/mockdownstream  (listens on :9090 by default)
//	MOCK_PORT=9091 go run ./cmd/mockdownstream
package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func main() {
	port := envOr("MOCK_PORT", "9090")
	addr := ":" + port

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("POST /", mcpHandler)

	slog.Info("mock downstream listening", "addr", addr)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("mock downstream error", "err", err)
		os.Exit(1)
	}
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, `{"status":"ok","service":"mockdownstream"}`)
}

// mcpHandler handles all MCP JSON-RPC requests with a canned employee record.
// The record intentionally contains a salary and ssn field so the proxy's
// field-level redaction can be tested end-to-end.
func mcpHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, nil, -32700, "parse error")
		return
	}

	slog.Info("mock downstream received", "method", req.Method)

	switch req.Method {
	case "tools/call":
		handleToolCall(w, req.ID)
	case "tools/list":
		handleToolList(w, req.ID)
	case "initialize":
		handleInitialize(w, req.ID)
	default:
		writeError(w, req.ID, -32601, "method not found: "+req.Method)
	}
}

func handleToolCall(w http.ResponseWriter, id json.RawMessage) {
	// Canned employee record — deliberately includes sensitive fields.
	record := map[string]any{
		"employee_id": "EMP-0042",
		"name":        "Alice Testuser",
		"department":  "Engineering",
		"salary":      120000,
		"ssn":         "123-45-6789",
		"email":       "alice@example.com",
		"start_date":  "2019-03-01",
	}
	text, _ := json.Marshal(record)

	result := map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": string(text)},
		},
		"isError": false,
	}
	writeSuccess(w, id, result)
}

func handleToolList(w http.ResponseWriter, id json.RawMessage) {
	tools := map[string]any{
		"tools": []map[string]any{
			{
				"name":        "get_employee_record",
				"description": "Fetch an employee record by ID.",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"employee_id": map[string]any{"type": "string"},
					},
					"required": []string{"employee_id"},
				},
			},
			{
				"name":        "list_employees",
				"description": "List all employees.",
				"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
			},
		},
	}
	writeSuccess(w, id, tools)
}

func handleInitialize(w http.ResponseWriter, id json.RawMessage) {
	writeSuccess(w, id, map[string]any{
		"protocolVersion": "2024-11-05",
		"serverInfo":      map[string]any{"name": "mock-bamboohr", "version": "1.0.0"},
		"capabilities":    map[string]any{"tools": map[string]any{}},
	})
}

func writeSuccess(w http.ResponseWriter, id json.RawMessage, result any) {
	resp := map[string]any{"jsonrpc": "2.0", "id": id, "result": result}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

func writeError(w http.ResponseWriter, id json.RawMessage, code int, msg string) {
	resp := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   map[string]any{"code": code, "message": msg},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // JSON-RPC errors still return HTTP 200
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
