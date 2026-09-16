// Package mcp defines the JSON-RPC 2.0 message types used by the
// Model Context Protocol (MCP). The proxy uses these to parse requests from
// agents and to forward/return responses from downstream MCP servers.
package mcp

import "encoding/json"

// ── JSON-RPC 2.0 core ─────────────────────────────────────────────────────────

// Request is a JSON-RPC 2.0 request object.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"` // can be string, number, or null
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response object. Exactly one of Result or Error
// will be populated per the spec.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError is the JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Standard JSON-RPC 2.0 error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603

	// MCP-specific application error codes (range: -32000 to -32099).
	CodeUnauthorized   = -32001
	CodeForbidden      = -32002
	CodeRateLimited    = -32003
	CodeUpstreamError  = -32004
)

// ── MCP tool call ─────────────────────────────────────────────────────────────

// ToolCallParams is the params object for a "tools/call" request.
type ToolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// ToolResult is the result object in a successful "tools/call" response.
type ToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ContentItem represents a single piece of content in a tool result.
type ContentItem struct {
	Type string `json:"type"` // "text", "image", "resource"
	Text string `json:"text,omitempty"`
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// ErrorResponse builds a well-formed JSON-RPC 2.0 error response.
func ErrorResponse(id json.RawMessage, code int, message string) Response {
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: message},
	}
}

// SuccessResponse builds a well-formed JSON-RPC 2.0 success response.
func SuccessResponse(id json.RawMessage, result any) (Response, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return Response{}, err
	}
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  raw,
	}, nil
}
