// ── Shared domain types ─────────────────────────────────────────────────────
// Mirror the JSON shapes returned by the Go adminapi.

export interface Role {
  id: string
  name: string
  description: string
  created_at: string
  updated_at: string
}

export interface User {
  id: string
  name: string
  email: string
  role_id: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface DownstreamServer {
  id: string
  name: string
  base_url: string
  auth_type: string      // "bearer" | "api_key" | "none"
  auth_secret_ref: string
  is_active: boolean
  tool_manifest: string[]
  created_at: string
  updated_at: string
}

export interface Policy {
  id: string
  role_id: string
  server_id: string
  tool_name: string
  is_allowed: boolean
  redact_fields: string[]
  rate_limit_rpm: number
  created_at: string
  updated_at: string
}

export interface AuditEvent {
  id: string
  user_id: string
  server_id: string
  tool_name: string
  outcome: string        // "allowed" | "denied" | "rate_limited" | "error"
  input_params_redacted: Record<string, unknown> | null
  response_summary: string
  latency_ms: number
  created_at: string
}

export interface TokenResponse {
  token: string
  expires_at: string
}

// ── Pagination envelope ─────────────────────────────────────────────────────
export interface Page<T> {
  items: T[]
  total: number
  page: number
  per_page: number
}
