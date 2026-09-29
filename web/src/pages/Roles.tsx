// ── Roles page ──────────────────────────────────────────────────────────────
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../api'
import type { Role } from '../types'
import { Button, Input, Spinner, ErrorAlert, Modal } from '../components/ui'
import { Plus, Pencil, Trash2 } from 'lucide-react'

// ── Create / Edit modal ───────────────────────────────────────────────────────
function RoleModal({
  existing,
  onClose,
}: {
  existing?: Role
  onClose: () => void
}) {
  const qc = useQueryClient()
  const [name, setName]     = useState(existing?.name ?? '')
  const [desc, setDesc]     = useState(existing?.description ?? '')
  const [err, setErr]       = useState<string | null>(null)

  const save = useMutation({
    mutationFn: () =>
      existing
        ? api.updateRole(existing.id, { name, description: desc })
        : api.createRole({ name, description: desc }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['roles'] }); onClose() },
    onError: (e: Error) => setErr(e.message),
  })

  return (
    <Modal
      title={existing ? `Edit role — ${existing.name}` : 'Create Role'}
      onClose={onClose}
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>Cancel</Button>
          <Button loading={save.isPending} onClick={() => save.mutate()}>
            {existing ? 'Save' : 'Create'}
          </Button>
        </>
      }
    >
      {err && <ErrorAlert error={err} />}
      <Input label="Name" value={name} onChange={e => setName(e.target.value)} placeholder="analyst" />
      <Input label="Description" value={desc} onChange={e => setDesc(e.target.value)} placeholder="Read-only analyst role" />
    </Modal>
  )
}

// ── Main ─────────────────────────────────────────────────────────────────────
export default function Roles() {
  const qc = useQueryClient()
  const [modal, setModal] = useState<'create' | Role | null>(null)

  const roles = useQuery({ queryKey: ['roles'], queryFn: () => api.listRoles() })

  const del = useMutation({
    mutationFn: (id: string) => api.deleteRole(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['roles'] }),
  })

  return (
    <div className="page">
      <div className="page-header">
        <h1 className="page-title">Roles</h1>
        <Button onClick={() => setModal('create')}>
          <Plus size={15} /> Create Role
        </Button>
      </div>

      {roles.isLoading && <Spinner />}
      {roles.isError   && <ErrorAlert error={roles.error} />}

      {roles.data && (
        <div className="card" style={{ padding: 0 }}>
          <div className="table-wrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Description</th>
                  <th>Created</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {roles.data.map(r => (
                  <tr key={r.id}>
                    <td style={{ fontWeight: 600 }}>{r.name}</td>
                    <td style={{ color: 'var(--text-muted)' }}>{r.description}</td>
                    <td style={{ color: 'var(--text-muted)', fontSize: '0.8rem' }}>
                      {new Date(r.created_at).toLocaleDateString()}
                    </td>
                    <td style={{ display: 'flex', gap: '0.4rem' }}>
                      <Button variant="ghost" size="sm" onClick={() => setModal(r)}>
                        <Pencil size={13} />
                      </Button>
                      <Button
                        variant="danger"
                        size="sm"
                        loading={del.isPending}
                        onClick={() => del.mutate(r.id)}
                      >
                        <Trash2 size={13} />
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {modal === 'create' && <RoleModal onClose={() => setModal(null)} />}
      {modal && typeof modal === 'object' && (
        <RoleModal existing={modal} onClose={() => setModal(null)} />
      )}
    </div>
  )
}
