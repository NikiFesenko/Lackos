// ── Users page ──────────────────────────────────────────────────────────────
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../api'
import type { User, Role } from '../types'
import {
  Button, Input, Select, Spinner, Badge, ErrorAlert, Modal,
} from '../components/ui'
import { UserPlus, Trash2 } from 'lucide-react'

// ── Add User modal ───────────────────────────────────────────────────────────
function AddUserModal({
  roles,
  onClose,
}: {
  roles: Role[]
  onClose: () => void
}) {
  const qc = useQueryClient()
  const [name, setName]     = useState('')
  const [email, setEmail]   = useState('')
  const [roleId, setRoleId] = useState(roles[0]?.id ?? '')
  const [err, setErr]       = useState<string | null>(null)

  const create = useMutation({
    mutationFn: () => api.createUser({ name, email, role_id: roleId }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['users'] })
      onClose()
    },
    onError: (e: Error) => setErr(e.message),
  })

  return (
    <Modal
      title="Add User"
      onClose={onClose}
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>Cancel</Button>
          <Button
            loading={create.isPending}
            onClick={() => create.mutate()}
          >
            Create
          </Button>
        </>
      }
    >
      {err && <ErrorAlert error={err} />}
      <Input label="Name"  value={name}  onChange={e => setName(e.target.value)}  placeholder="Alice Smith" />
      <Input label="Email" value={email} onChange={e => setEmail(e.target.value)} placeholder="alice@example.com" type="email" />
      <Select
        label="Role"
        value={roleId}
        onChange={setRoleId}
        options={roles.map(r => ({ value: r.id, label: r.name }))}
      />
    </Modal>
  )
}

// ── Change role modal ─────────────────────────────────────────────────────────
function ChangeRoleModal({
  user,
  roles,
  onClose,
}: {
  user: User
  roles: Role[]
  onClose: () => void
}) {
  const qc = useQueryClient()
  const [roleId, setRoleId] = useState(user.role_id)
  const [err, setErr]       = useState<string | null>(null)

  const update = useMutation({
    mutationFn: () => api.updateUserRole(user.id, roleId),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['users'] }); onClose() },
    onError: (e: Error) => setErr(e.message),
  })

  return (
    <Modal
      title={`Change role — ${user.name}`}
      onClose={onClose}
      footer={
        <>
          <Button variant="ghost" onClick={onClose}>Cancel</Button>
          <Button loading={update.isPending} onClick={() => update.mutate()}>Save</Button>
        </>
      }
    >
      {err && <ErrorAlert error={err} />}
      <Select
        label="Role"
        value={roleId}
        onChange={setRoleId}
        options={roles.map(r => ({ value: r.id, label: r.name }))}
      />
    </Modal>
  )
}

// ── Main ─────────────────────────────────────────────────────────────────────
export default function Users() {
  const qc = useQueryClient()
  const [showAdd, setShowAdd]               = useState(false)
  const [changeRoleUser, setChangeRoleUser] = useState<User | null>(null)

  const users  = useQuery({ queryKey: ['users'],  queryFn: () => api.listUsers() })
  const roles  = useQuery({ queryKey: ['roles'],  queryFn: () => api.listRoles() })

  const deactivate = useMutation({
    mutationFn: (id: string) => api.deactivateUser(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['users'] }),
  })

  const roleMap = Object.fromEntries((roles.data ?? []).map(r => [r.id, r.name]))

  return (
    <div className="page">
      <div className="page-header">
        <h1 className="page-title">Users</h1>
        <Button onClick={() => setShowAdd(true)}>
          <UserPlus size={15} /> Add User
        </Button>
      </div>

      {users.isLoading && <Spinner />}
      {users.isError  && <ErrorAlert error={users.error} />}

      {users.data && (
        <div className="card" style={{ padding: 0 }}>
          <div className="table-wrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Email</th>
                  <th>Role</th>
                  <th>Status</th>
                  <th>Created</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {users.data.map(u => (
                  <tr key={u.id}>
                    <td>{u.name}</td>
                    <td style={{ color: 'var(--text-muted)' }}>{u.email}</td>
                    <td>
                      <button
                        className="btn btn-ghost btn-sm"
                        onClick={() => setChangeRoleUser(u)}
                        title="Change role"
                      >
                        {roleMap[u.role_id] ?? u.role_id}
                      </button>
                    </td>
                    <td>
                      <Badge variant={u.is_active ? 'green' : 'red'}>
                        {u.is_active ? 'active' : 'inactive'}
                      </Badge>
                    </td>
                    <td style={{ color: 'var(--text-muted)', fontSize: '0.8rem' }}>
                      {new Date(u.created_at).toLocaleDateString()}
                    </td>
                    <td>
                      {u.is_active && (
                        <Button
                          variant="danger"
                          size="sm"
                          loading={deactivate.isPending}
                          onClick={() => deactivate.mutate(u.id)}
                          title="Deactivate user"
                        >
                          <Trash2 size={13} />
                        </Button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {showAdd && roles.data && (
        <AddUserModal roles={roles.data} onClose={() => setShowAdd(false)} />
      )}
      {changeRoleUser && roles.data && (
        <ChangeRoleModal
          user={changeRoleUser}
          roles={roles.data}
          onClose={() => setChangeRoleUser(null)}
        />
      )}
    </div>
  )
}
