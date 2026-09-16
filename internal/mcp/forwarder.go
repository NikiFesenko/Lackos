package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Forwarder sends MCP JSON-RPC requests to a downstream MCP server and
// returns the raw JSON-RPC response. It resolves the downstream credential
// from the auth_secret_ref reference string.
type Forwarder struct {
	client   *http.Client
	resolver SecretResolver
}

// SecretResolver resolves an auth_secret_ref string (e.g. "env:BAMBOOHR_API_KEY")
// to its actual value at runtime.
type SecretResolver interface {
	Resolve(ref string) (string, error)
}

// NewForwarder creates a Forwarder with a sensible HTTP client timeout.
func NewForwarder(resolver SecretResolver) *Forwarder {
	return &Forwarder{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		resolver: resolver,
	}
}

// Forward sends req to the downstream server identified by baseURL and
// authSecretRef, returning the parsed JSON-RPC Response.
//
// The downstream server is expected to speak JSON-RPC 2.0 over HTTP POST.
// auth_type="bearer" (the only supported type in Phase 5) appends the resolved
// secret as an Authorization: Bearer header.
func (f *Forwarder) Forward(
	ctx context.Context,
	baseURL string,
	authType string,
	authSecretRef string,
	req Request,
) (Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return Response{}, fmt.Errorf("mcp: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, bytes.NewReader(body))
	if err != nil {
		return Response{}, fmt.Errorf("mcp: build http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Attach credential resolved from auth_secret_ref.
	if authSecretRef != "" {
		secret, resolveErr := f.resolver.Resolve(authSecretRef)
		if resolveErr != nil {
			return Response{}, fmt.Errorf("mcp: resolve auth secret: %w", resolveErr)
		}
		switch authType {
		case "bearer", "":
			httpReq.Header.Set("Authorization", "Bearer "+secret)
		case "api_key":
			httpReq.Header.Set("X-Api-Key", secret)
		default:
			slog.WarnContext(ctx, "unknown auth_type; skipping credential attachment",
				"auth_type", authType)
		}
	}

	resp, err := f.client.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("mcp: downstream http error: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10 MiB cap
	if err != nil {
		return Response{}, fmt.Errorf("mcp: read downstream response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Response{}, fmt.Errorf("mcp: downstream HTTP %d: %s", resp.StatusCode, raw)
	}

	var rpcResp Response
	if err := json.Unmarshal(raw, &rpcResp); err != nil {
		return Response{}, fmt.Errorf("mcp: parse downstream response: %w", err)
	}
	return rpcResp, nil
}
