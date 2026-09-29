// ── Policies page ───────────────────────────────────────────────────────────
// Matrix view: select role + server → see/edit tool-level policies.
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../api'
import type { Policy } from '../types'
import {
  Select, Toggle, ChipInput, Spinner, ErrorAlert, Button,
} from '../components/ui'
import { Save } from 'lucide-react'

// ── Policy row editor ─────────────────────────────────────────────────────────
function PolicyRow({
  tool,
  policy,
  roleId,
  serverId,
}: {
  tool: string
  policy?: Policy
  roleId: string
  serverId: string
}) {
  const qc = useQueryClient()
  const [allowed,  setAllowed]  = useState(policy?.is_allowed ?? false)
  const [redact,   setRedact]   = useState<string[]>(policy?.redact_fields ?? [])
  const [rpmStr,   setRpmStr]   = useState(String(policy?.rate_limit_rpm ?? 60))
  const [saved,    setSaved]    = useState(false)

  const upsert = useMutation({
    mutationFn: () =>
      api.upsertPolicy({
        role_id:        roleId,
        server_id:      serverId,
        tool_name:      tool,
        is_allowed:     allowed,
        redact_fields:  redact,
        rate_limit_rpm: Number(rpmStr) || 60,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['policies', roleId, serverId] })
      setSaved(true)
      setTimeout(() => setSaved(false), 1500)
    },
  })

  return (
    <tr>
      <td style={{ fontFamily: 'var(--font-mono)', fontSize: '0.85rem' }}>{tool}</td>
      <td>
        <Toggle checked={allowed} onChange={setAllowed} label={allowed ? 'allow' : 'deny'} />
      </td>
      <td style={{ minWidth: 220 }}>
        <ChipInput value={redact} onChange={setRedact} placeholder="field name…" />
      </td>
      <td>
        <input
          className="form-input"
          style={{ width: 80 }}
          type="number"
          min={0}
          value={rpmStr}
          onChange={e => setRpmStr(e.target.value)}
        />
      </td>
      <td>
        <Button
          size="sm"
          variant={saved ? 'ghost' : 'primary'}
          loading={upsert.isPending}
          onClick={() => upsert.mutate()}
        >
          <Save size={13} />
          {saved ? 'Saved!' : 'Save'}
        </Button>
      </td>
    </tr>
  )
}

// ── Main ─────────────────────────────────────────────────────────────────────
export default function Policies() {
  const [roleId,   setRoleId]   = useState('')
  const [serverId, setServerId] = useState('')

  const roles   = useQuery({ queryKey: ['roles'],   queryFn: () => api.listRoles() })
  const servers = useQuery({ queryKey: ['servers'], queryFn: () => api.listServers() })

  const policies = useQuery({
    queryKey: ['policies', roleId, serverId],
    queryFn:  () => api.listPolicies({ role_id: roleId, server_id: serverId }),
    enabled:  !!roleId && !!serverId,
  })

  const selectedServer = servers.data?.find(s => s.id === serverId)
  const tools          = selectedServer?.tool_manifest ?? []
  const policyMap      = Object.fromEntries((policies.data ?? []).map(p => [p.tool_name, p]))

  const roleOpts   = [{ value: '', label: '— select role —' },   ...(roles.data   ?? []).map(r => ({ value: r.id, label: r.name }))]
  const serverOpts = [{ value: '', label: '— select server —' }, ...(servers.data ?? []).map(s => ({ value: s.id, label: s.name }))]

  return (
    <div className="page">
      <div className="page-header">
        <h1 className="page-title">Policies</h1>
      </div>

      <div style={{ display: 'flex', gap: '1rem', marginBottom: '1.25rem' }}>
        <div style={{ flex: 1 }}>
          <Select label="Role" value={roleId} onChange={setRoleId} options={roleOpts} />
        </div>
        <div style={{ flex: 1 }}>
          <Select label="Downstream Server" value={serverId} onChange={setServerId} options={serverOpts} />
        </div>
      </div>

      {!roleId || !serverId ? (
        <p style={{ color: 'var(--text-muted)' }}>Select a role and a server to view its policy matrix.</p>
      ) : tools.length === 0 ? (
        <p style={{ color: 'var(--text-muted)' }}>
          This server has no tools in its manifest yet.
          Register tools by updating the server's <code>tool_manifest</code>.
        </p>
      ) : (
        <>
          {policies.isLoading && <Spinner />}
          {policies.isError   && <ErrorAlert error={policies.error} />}
          {!policies.isLoading && (
            <div className="card" style={{ padding: 0 }}>
              <div className="table-wrap">
                <table className="tbl">
                  <thead>
                    <tr>
                      <th>Tool</th>
                      <th>Allow / Deny</th>
                      <th>Redact fields</th>
                      <th>Rate limit (rpm)</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {tools.map(tool => (
                      <PolicyRow
                        key={tool}
                        tool={tool}
                        policy={policyMap[tool]}
                        roleId={roleId}
                        serverId={serverId}
                      />
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  )
}
