// ── API client ──────────────────────────────────────────────────────────────
// All requests go through this module so auth headers are applied centrally.

const BASE = import.meta.env.VITE_API_BASE_URL ?? ''  // empty → use Vite proxy

// ── Token storage ───────────────────────────────────────────────────────────
const TOKEN_KEY = 'mcp_admin_token'

export function getStoredToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}
export function setStoredToken(t: string): void {
  localStorage.setItem(TOKEN_KEY, t)
}
export function clearStoredToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

// ── Core fetch wrapper ──────────────────────────────────────────────────────
export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const token = getStoredToken()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  if (token) headers['Authorization'] = `Bearer ${token}`

  const res = await fetch(`${BASE}${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  if (!res.ok) {
    let msg = res.statusText
    try {
      const err = await res.json()
      msg = err.error ?? err.message ?? msg
    } catch {
      // non-JSON error body — keep statusText
    }
    throw new ApiError(res.status, msg)
  }

  // 204 No Content
  if (res.status === 204) return undefined as unknown as T
  return res.json() as Promise<T>
}

const get  = <T>(path: string)                => request<T>('GET',    path)
const post = <T>(path: string, body: unknown) => request<T>('POST',   path, body)
const put  = <T>(path: string, body: unknown) => request<T>('PUT',    path, body)
const del  = <T>(path: string)                => request<T>('DELETE', path)

// ── Roles ───────────────────────────────────────────────────────────────────
import type { Role, User, DownstreamServer, Policy, AuditEvent, TokenResponse } from './types'

export const api = {
  // ── Roles
  listRoles: () => get<Role[]>('/api/v1/roles'),
  createRole: (b: Omit<Role, 'id' | 'created_at' | 'updated_at'>) =>
    post<Role>('/api/v1/roles', b),
  getRole:    (id: string) => get<Role>(`/api/v1/roles/${id}`),
  updateRole: (id: string, b: Partial<Role>) =>
    put<Role>(`/api/v1/roles/${id}`, b),
  deleteRole: (id: string) => del<void>(`/api/v1/roles/${id}`),

  // ── Users
  listUsers: (params?: { page?: number; per_page?: number }) => {
    const q = new URLSearchParams()
    if (params?.page)     q.set('page',     String(params.page))
    if (params?.per_page) q.set('per_page', String(params.per_page))
    const qs = q.toString() ? `?${q}` : ''
    return get<User[]>(`/api/v1/users${qs}`)
  },
  createUser: (b: Omit<User, 'id' | 'is_active' | 'created_at' | 'updated_at'>) =>
    post<User>('/api/v1/users', b),
  getUser:    (id: string) => get<User>(`/api/v1/users/${id}`),
  updateUserRole: (id: string, roleId: string) =>
    put<User>(`/api/v1/users/${id}/role`, { role_id: roleId }),
  deactivateUser: (id: string) => del<void>(`/api/v1/users/${id}`),

  // ── Downstream Servers
  listServers: () => get<DownstreamServer[]>('/api/v1/downstream-servers'),
  createServer: (b: Omit<DownstreamServer, 'id' | 'is_active' | 'tool_manifest' | 'created_at' | 'updated_at'>) =>
    post<DownstreamServer>('/api/v1/downstream-servers', b),
  getServer:    (id: string) => get<DownstreamServer>(`/api/v1/downstream-servers/${id}`),
  updateServer: (id: string, b: Partial<DownstreamServer>) =>
    put<DownstreamServer>(`/api/v1/downstream-servers/${id}`, b),
  toggleServer: (id: string) => del<void>(`/api/v1/downstream-servers/${id}`),

  // ── Policies
  listPolicies: (params?: { role_id?: string; server_id?: string }) => {
    const q = new URLSearchParams()
    if (params?.role_id)   q.set('role_id',   params.role_id)
    if (params?.server_id) q.set('server_id', params.server_id)
    const qs = q.toString() ? `?${q}` : ''
    return get<Policy[]>(`/api/v1/policies${qs}`)
  },
  upsertPolicy: (b: Omit<Policy, 'id' | 'created_at' | 'updated_at'>) =>
    post<Policy>('/api/v1/policies', b),
  getPolicy:    (id: string) => get<Policy>(`/api/v1/policies/${id}`),
  deletePolicy: (id: string) => del<void>(`/api/v1/policies/${id}`),

  // ── Audit Events
  listAuditEvents: (params?: {
    user_id?: string
    server_id?: string
    outcome?: string
    from?: string
    to?: string
    page?: number
    per_page?: number
  }) => {
    const q = new URLSearchParams()
    if (params?.user_id)   q.set('user_id',   params.user_id)
    if (params?.server_id) q.set('server_id', params.server_id)
    if (params?.outcome)   q.set('outcome',   params.outcome)
    if (params?.from)      q.set('from',      params.from)
    if (params?.to)        q.set('to',        params.to)
    if (params?.page)      q.set('page',      String(params.page))
    if (params?.per_page)  q.set('per_page',  String(params.per_page))
    const qs = q.toString() ? `?${q}` : ''
    return get<AuditEvent[]>(`/api/v1/audit-events${qs}`)
  },

  // ── Auth Tokens (proxy tokens, not admin auth)
  issueToken: (userId: string) =>
    post<TokenResponse>('/api/v1/auth/token', { user_id: userId }),
  revokeToken: (userId: string) =>
    del<void>(`/api/v1/auth/token?user_id=${userId}`),
}
