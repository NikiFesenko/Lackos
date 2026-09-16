package mcp

import (
	"encoding/json"
	"log/slog"
)

const redactedPlaceholder = "[REDACTED]"

// RedactFields removes sensitive fields from a JSON-RPC tool-call result.
// It walks the ToolResult.Content items and, for any ContentItem of type "text"
// whose decoded JSON contains a key in the fields list, replaces the value
// with the "[REDACTED]" sentinel.
//
// If the content is not valid JSON (e.g. plain text), the item is left untouched —
// redaction only applies to structured JSON content.
//
// This runs in the proxy hot path: the result is returned directly to the agent.
func RedactFields(result ToolResult, fields []string) ToolResult {
	if len(fields) == 0 {
		return result
	}
	fieldSet := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		fieldSet[f] = struct{}{}
	}

	out := ToolResult{IsError: result.IsError, Content: make([]ContentItem, len(result.Content))}
	for i, item := range result.Content {
		if item.Type != "text" || item.Text == "" {
			out.Content[i] = item
			continue
		}
		redacted, err := redactJSON([]byte(item.Text), fieldSet)
		if err != nil {
			// Not valid JSON — leave as-is and log at debug level.
			slog.Debug("redact: content item is not JSON, skipping field redaction", "err", err)
			out.Content[i] = item
			continue
		}
		out.Content[i] = ContentItem{Type: "text", Text: string(redacted)}
	}
	return out
}

// redactJSON replaces the values of the named keys anywhere in the JSON
// document (top-level only) with "[REDACTED]".
func redactJSON(data []byte, fields map[string]struct{}) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	for f := range fields {
		if _, ok := doc[f]; ok {
			doc[f] = redactedPlaceholder
		}
	}
	return json.Marshal(doc)
}
