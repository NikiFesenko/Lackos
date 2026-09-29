// ── Downstream Servers page ─────────────────────────────────────────────────
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../api'
import type { DownstreamServer } from '../types'
import {
  Button, Input, Select, Toggle, Spinner, Badge, ErrorAlert, Modal,
} from '../components/ui'
import { Plus, Pencil } from 'lucide-react'

const AUTH_TYPES = [
  { value: 'none',    label: 'None' },
  { value: 'bearer',  label: 'Bearer Token' },
  { value: 'api_key', label: 'API Key' },
]

// ── Server modal ─────────────────────────────────────────────────────────────
function ServerModal({
  existing,
  onClose,
}: {
  existing?: DownstreamServer
  onClose: () => void
}) {
  const qc = useQueryClient()
  const [name,      setName]      = useState(existing?.name ?? '')
  const [baseUrl,   setBaseUrl]   = useState(existing?.base_url ?? '')
  const [authType,  setAuthType]  = useState(existing?.auth_type ?? 'none')
  const [secretRef, setSecretRef] = useState(existing?.auth_secret_ref ?? '')
  const [err,       setErr]       = useState<string | null>(null)

  const save = useMutation({
    mutationFn: () => {
      const body = { name, base_url: baseUrl, auth_type: authType, auth_secret_ref: secretRef }
      return existing
        ? api.updateServer(existing.id, body)
        : api.createServer(body)
    },
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['servers'] }); onClose() },
    onError: (e: Error) => setErr(e.message),
  })

  return (
    <Modal
      title={existing ? `Edit — ${existing.name}` : 'Register Downstream Server'}
      onClose={onClose}
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>Cancel</Button>
          <Button loading={save.isPending} onClick={() => save.mutate()}>
            {existing ? 'Save' : 'Register'}
          </Button>
        </>
      }
    >
      {err && <ErrorAlert error={err} />}
      <Input label="Name" value={name} onChange={e => setName(e.target.value)} placeholder="bamboohr-mcp" />
      <Input label="Base URL" value={baseUrl} onChange={e => setBaseUrl(e.target.value)} placeholder="https://mcp.example.com" />
      <Select label="Auth Type" value={authType} onChange={setAuthType} options={AUTH_TYPES} />
      {authType !== 'none' && (
        <Input
          label="Secret ref"
          value={secretRef}
          onChange={e => setSecretRef(e.target.value)}
          placeholder="env:DOWNSTREAM_API_KEY"
        />
      )}
      <p style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginTop: '-0.5rem' }}>
        Secret refs use the <code>env:VAR_NAME</code> scheme and are resolved server-side. The value is never stored in plain text.
      </p>
    </Modal>
  )
}

// ── Main ─────────────────────────────────────────────────────────────────────
export default function Servers() {
  const qc = useQueryClient()
  const [modal, setModal] = useState<'create' | DownstreamServer | null>(null)

  const servers = useQuery({ queryKey: ['servers'], queryFn: () => api.listServers() })

  const toggle = useMutation({
    mutationFn: (id: string) => api.toggleServer(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['servers'] }),
  })

  return (
    <div className="page">
      <div className="page-header">
        <h1 className="page-title">Downstream Servers</h1>
        <Button onClick={() => setModal('create')}>
          <Plus size={15} /> Register Server
        </Button>
      </div>

      {servers.isLoading && <Spinner />}
      {servers.isError   && <ErrorAlert error={servers.error} />}

      {servers.data && (
        <div className="card" style={{ padding: 0 }}>
          <div className="table-wrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Base URL</th>
                  <th>Auth</th>
                  <th>Tools</th>
                  <th>Active</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {servers.data.map(s => (
                  <tr key={s.id}>
                    <td style={{ fontWeight: 600 }}>{s.name}</td>
                    <td style={{ color: 'var(--text-muted)', fontSize: '0.8rem' }}>
                      <code>{s.base_url}</code>
                    </td>
                    <td>
                      <Badge variant="muted">{s.auth_type}</Badge>
                    </td>
                    <td style={{ color: 'var(--text-muted)' }}>
                      {s.tool_manifest?.length ?? 0} tools
                    </td>
                    <td>
                      <Toggle
                        checked={s.is_active}
                        onChange={() => toggle.mutate(s.id)}
                        label={s.is_active ? 'on' : 'off'}
                      />
                    </td>
                    <td>
                      <Button variant="ghost" size="sm" onClick={() => setModal(s)}>
                        <Pencil size={13} />
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {modal === 'create' && <ServerModal onClose={() => setModal(null)} />}
      {modal && typeof modal === 'object' && (
        <ServerModal existing={modal} onClose={() => setModal(null)} />
      )}
    </div>
  )
}
