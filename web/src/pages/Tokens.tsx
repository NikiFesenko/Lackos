// ── Tokens page ─────────────────────────────────────────────────────────────
// Issue and revoke proxy Bearer tokens on a per-user basis.
import { useState } from 'react'
import { useQuery, useMutation } from '@tanstack/react-query'
import { api } from '../api'
import type { User } from '../types'
import { Select, Button, Spinner, ErrorAlert } from '../components/ui'
import { Copy, RefreshCw, Trash2, Check } from 'lucide-react'

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false)
  async function copy() {
    await navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }
  return (
    <Button variant="ghost" size="sm" onClick={copy} title="Copy to clipboard">
      {copied ? <Check size={13} color="var(--success)" /> : <Copy size={13} />}
    </Button>
  )
}

export default function Tokens() {
  const [selectedUserId, setSelectedUserId] = useState('')
  const [issuedToken,    setIssuedToken]    = useState<string | null>(null)
  const [issuedAt,       setIssuedAt]       = useState<string | null>(null)
  const [err,            setErr]            = useState<string | null>(null)

  const users = useQuery({ queryKey: ['users'], queryFn: () => api.listUsers() })
  const activeUsers = (users.data ?? []).filter((u: User) => u.is_active)

  const issue = useMutation({
    mutationFn: () => api.issueToken(selectedUserId),
    onSuccess: data => {
      setIssuedToken(data.token)
      setIssuedAt(data.expires_at)
      setErr(null)
    },
    onError: (e: Error) => setErr(e.message),
  })

  const revoke = useMutation({
    mutationFn: () => api.revokeToken(selectedUserId),
    onSuccess: () => { setIssuedToken(null); setIssuedAt(null); setErr(null) },
    onError: (e: Error) => setErr(e.message),
  })

  const userOpts = [
    { value: '', label: '— select user —' },
    ...activeUsers.map((u: User) => ({ value: u.id, label: `${u.name} <${u.email}>` })),
  ]

  return (
    <div className="page">
      <div className="page-header">
        <h1 className="page-title">Proxy Tokens</h1>
      </div>

      <p style={{ color: 'var(--text-muted)', marginBottom: '1.5rem', fontSize: '0.9rem' }}>
        Issue short-lived Bearer tokens that AI agents use to call the MCP proxy.
        Each user can hold at most one active token at a time. Revoking invalidates immediately.
      </p>

      {users.isLoading && <Spinner />}
      {users.isError   && <ErrorAlert error={users.error} />}

      <div className="card" style={{ maxWidth: 520 }}>
        <Select
          label="User"
          value={selectedUserId}
          onChange={id => { setSelectedUserId(id); setIssuedToken(null); setErr(null) }}
          options={userOpts}
        />

        {err && <ErrorAlert error={err} />}

        <div style={{ display: 'flex', gap: '0.75rem', marginTop: '0.5rem' }}>
          <Button
            loading={issue.isPending}
            disabled={!selectedUserId}
            onClick={() => issue.mutate()}
          >
            <RefreshCw size={14} />
            Issue Token
          </Button>
          <Button
            variant="danger"
            loading={revoke.isPending}
            disabled={!selectedUserId}
            onClick={() => revoke.mutate()}
          >
            <Trash2 size={14} />
            Revoke
          </Button>
        </div>

        {issuedToken && (
          <div style={{ marginTop: '1.25rem' }}>
            <p style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginBottom: '0.4rem' }}>
              Bearer token — copy now, it won't be shown again:
            </p>
            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: '0.5rem',
              background: 'var(--surface-2)',
              border: '1px solid var(--border)',
              borderRadius: 6,
              padding: '0.5rem 0.75rem',
            }}>
              <code style={{
                flex: 1,
                fontSize: '0.78rem',
                wordBreak: 'break-all',
                color: 'var(--primary)',
                fontFamily: 'var(--font-mono)',
              }}>
                {issuedToken}
              </code>
              <CopyButton text={issuedToken} />
            </div>
            {issuedAt && (
              <p style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginTop: '0.4rem' }}>
                Expires: {new Date(issuedAt).toLocaleString()}
              </p>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
