// ── Login page ──────────────────────────────────────────────────────────────
// For Phase 7 the admin token is entered directly (Phase 8 adds real auth).
import { useState, FormEvent } from 'react'
import { useAuth } from '../context/AuthContext'
import { Button, Input, ErrorAlert } from '../components/ui'
import { Shield } from 'lucide-react'

export default function Login() {
  const { login } = useAuth()
  const [token, setToken] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!token.trim()) { setError('Token is required'); return }
    setError(null)
    setLoading(true)
    try {
      // Verify the token works by hitting /healthz with it in the Authorization header.
      const res = await fetch('/api/v1/roles', {
        headers: { Authorization: `Bearer ${token.trim()}` },
      })
      if (!res.ok) throw new Error(`Invalid token (HTTP ${res.status})`)
      login(token.trim())
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: 'var(--bg)',
    }}>
      <div className="card" style={{ width: '100%', maxWidth: 400 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.6rem', marginBottom: '1.5rem' }}>
          <Shield size={24} color="var(--primary)" />
          <span style={{ fontWeight: 700, fontSize: '1.1rem' }}>MCP Gate Admin</span>
        </div>

        <p style={{ color: 'var(--text-muted)', fontSize: '0.875rem', marginBottom: '1.5rem' }}>
          Enter your admin token to access the dashboard.
        </p>

        {error && <ErrorAlert error={error} />}

        <form onSubmit={handleSubmit}>
          <Input
            id="admin-token"
            label="Admin Token"
            type="password"
            placeholder="Enter ADMIN_TOKEN…"
            value={token}
            onChange={e => setToken(e.target.value)}
            autoComplete="current-password"
            autoFocus
          />
          <Button type="submit" loading={loading} style={{ width: '100%', marginTop: '0.5rem' }}>
            Sign in
          </Button>
        </form>

        <p style={{ marginTop: '1rem', fontSize: '0.8rem', color: 'var(--text-muted)' }}>
          Phase 8 will replace this with email + password auth.
        </p>
      </div>
    </div>
  )
}
