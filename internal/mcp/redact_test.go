package mcp_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mcp-gate/mcp-gate/internal/mcp"
)

func sampleResult(text string) mcp.ToolResult {
	return mcp.ToolResult{
		Content: []mcp.ContentItem{
			{Type: "text", Text: text},
		},
	}
}

// TestRedactFields_RemovesConfiguredFields verifies that listed fields are replaced.
func TestRedactFields_RemovesConfiguredFields(t *testing.T) {
	payload := `{"name":"Alice","salary":120000,"department":"Engineering"}`
	result := sampleResult(payload)

	out := mcp.RedactFields(result, []string{"salary"})

	require.Len(t, out.Content, 1)
	assert.Contains(t, out.Content[0].Text, `"salary":"[REDACTED]"`)
	assert.Contains(t, out.Content[0].Text, `"name":"Alice"`)
	assert.Contains(t, out.Content[0].Text, `"department":"Engineering"`)
}

// TestRedactFields_MultipleFields verifies that multiple fields are all redacted.
func TestRedactFields_MultipleFields(t *testing.T) {
	payload := `{"name":"Bob","salary":95000,"ssn":"123-45-6789","role":"hr_admin"}`
	result := sampleResult(payload)

	out := mcp.RedactFields(result, []string{"salary", "ssn"})

	text := out.Content[0].Text
	assert.Contains(t, text, `"salary":"[REDACTED]"`)
	assert.Contains(t, text, `"ssn":"[REDACTED]"`)
	assert.Contains(t, text, `"name":"Bob"`)
}

// TestRedactFields_EmptyFieldList_PassThrough verifies that an empty redact list is a no-op.
func TestRedactFields_EmptyFieldList_PassThrough(t *testing.T) {
	payload := `{"salary":50000}`
	result := sampleResult(payload)

	out := mcp.RedactFields(result, []string{})

	assert.Equal(t, payload, out.Content[0].Text)
}

// TestRedactFields_NonJSONContent_Untouched verifies plain text is not modified.
func TestRedactFields_NonJSONContent_Untouched(t *testing.T) {
	plainText := "Employee Alice works in Engineering."
	result := sampleResult(plainText)

	out := mcp.RedactFields(result, []string{"salary"})

	assert.Equal(t, plainText, out.Content[0].Text, "non-JSON content must be passed through unchanged")
}

// TestRedactFields_MissingField_NoOp verifies that redacting a non-existent field is safe.
func TestRedactFields_MissingField_NoOp(t *testing.T) {
	payload := `{"name":"Carol","department":"Finance"}`
	result := sampleResult(payload)

	out := mcp.RedactFields(result, []string{"salary"})

	assert.Contains(t, out.Content[0].Text, `"name":"Carol"`)
	assert.NotContains(t, out.Content[0].Text, "REDACTED")
}

// TestRedactFields_NonTextItem_Untouched verifies non-text content items are passed through.
func TestRedactFields_NonTextItem_Untouched(t *testing.T) {
	result := mcp.ToolResult{
		Content: []mcp.ContentItem{
			{Type: "image"},
		},
	}

	out := mcp.RedactFields(result, []string{"salary"})

	assert.Equal(t, "image", out.Content[0].Type)
}

// TestRedactFields_PreservesIsError verifies the IsError flag survives redaction.
func TestRedactFields_PreservesIsError(t *testing.T) {
	result := mcp.ToolResult{
		IsError: true,
		Content: []mcp.ContentItem{{Type: "text", Text: `{"error":"details","salary":0}`}},
	}

	out := mcp.RedactFields(result, []string{"salary"})

	assert.True(t, out.IsError)
	assert.Contains(t, out.Content[0].Text, `"salary":"[REDACTED]"`)
}
