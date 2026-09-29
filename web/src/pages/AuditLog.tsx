// ── Audit Log page ──────────────────────────────────────────────────────────
import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../api'
import type { AuditEvent } from '../types'
import { Spinner, ErrorAlert, OutcomeBadge, Select, Input, Button } from '../components/ui'
import { ChevronDown, ChevronRight } from 'lucide-react'

const OUTCOMES = [
  { value: '',             label: 'All outcomes' },
  { value: 'allowed',      label: 'Allowed' },
  { value: 'denied',       label: 'Denied' },
  { value: 'rate_limited', label: 'Rate Limited' },
  { value: 'error',        label: 'Error' },
]

const PER_PAGE = 50

// ── Expandable row ────────────────────────────────────────────────────────────
function EventRow({ ev }: { ev: AuditEvent }) {
  const [open, setOpen] = useState(false)

  return (
    <>
      <tr
        style={{ cursor: 'pointer' }}
        onClick={() => setOpen(o => !o)}
        aria-expanded={open}
      >
        <td>
          {open ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
        </td>
        <td style={{ fontSize: '0.8rem', color: 'var(--text-muted)', whiteSpace: 'nowrap' }}>
          {new Date(ev.created_at).toLocaleString()}
        </td>
        <td style={{ fontFamily: 'var(--font-mono)', fontSize: '0.8rem' }}>{ev.tool_name}</td>
        <td><OutcomeBadge outcome={ev.outcome} /></td>
        <td style={{ color: 'var(--text-muted)', fontSize: '0.8rem' }}>{ev.latency_ms} ms</td>
        <td style={{ color: 'var(--text-muted)', fontSize: '0.8rem' }}>{ev.user_id.slice(0, 8)}…</td>
      </tr>
      {open && (
        <tr>
          <td colSpan={6} style={{ background: 'var(--surface-2)', padding: '0.75rem 1rem' }}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' }}>
              <div>
                <p style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginBottom: '0.25rem' }}>
                  Input (redacted)
                </p>
                <pre style={{
                  background: 'var(--bg)', borderRadius: 6, padding: '0.6rem',
                  fontSize: '0.8rem', overflow: 'auto', color: 'var(--text)',
                }}>
                  {JSON.stringify(ev.input_params_redacted, null, 2)}
                </pre>
              </div>
              <div>
                <p style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginBottom: '0.25rem' }}>
                  Response summary
                </p>
                <pre style={{
                  background: 'var(--bg)', borderRadius: 6, padding: '0.6rem',
                  fontSize: '0.8rem', overflow: 'auto', color: 'var(--text)',
                }}>
                  {ev.response_summary || '—'}
                </pre>
              </div>
            </div>
          </td>
        </tr>
      )}
    </>
  )
}

// ── Main ─────────────────────────────────────────────────────────────────────
export default function AuditLog() {
  const [outcome, setOutcome] = useState('')
  const [from,    setFrom]    = useState('')
  const [to,      setTo]      = useState('')
  const [page,    setPage]    = useState(1)

  const events = useQuery({
    queryKey: ['audit', outcome, from, to, page],
    queryFn:  () => api.listAuditEvents({
      outcome:  outcome || undefined,
      from:     from    || undefined,
      to:       to      || undefined,
      page,
      per_page: PER_PAGE,
    }),
  })

  return (
    <div className="page">
      <div className="page-header">
        <h1 className="page-title">Audit Log</h1>
      </div>

      {/* Filters */}
      <div style={{ display: 'flex', gap: '1rem', marginBottom: '1.25rem', flexWrap: 'wrap', alignItems: 'flex-end' }}>
        <div style={{ width: 180 }}>
          <Select label="Outcome" value={outcome} onChange={v => { setOutcome(v); setPage(1) }} options={OUTCOMES} />
        </div>
        <Input
          label="From"
          type="date"
          value={from}
          onChange={e => { setFrom(e.target.value); setPage(1) }}
          style={{ width: 160 }}
        />
        <Input
          label="To"
          type="date"
          value={to}
          onChange={e => { setTo(e.target.value); setPage(1) }}
          style={{ width: 160 }}
        />
        <Button
          variant="ghost"
          onClick={() => { setOutcome(''); setFrom(''); setTo(''); setPage(1) }}
        >
          Reset
        </Button>
      </div>

      {events.isLoading && <Spinner />}
      {events.isError   && <ErrorAlert error={events.error} />}

      {events.data && (
        <>
          <div className="card" style={{ padding: 0 }}>
            <div className="table-wrap">
              <table className="tbl">
                <thead>
                  <tr>
                    <th style={{ width: 24 }}></th>
                    <th>Time</th>
                    <th>Tool</th>
                    <th>Outcome</th>
                    <th>Latency</th>
                    <th>User</th>
                  </tr>
                </thead>
                <tbody>
                  {events.data.length === 0 ? (
                    <tr>
                      <td colSpan={6} style={{ textAlign: 'center', color: 'var(--text-muted)', padding: '2rem' }}>
                        No events found.
                      </td>
                    </tr>
                  ) : (
                    events.data.map(ev => <EventRow key={ev.id} ev={ev} />)
                  )}
                </tbody>
              </table>
            </div>
          </div>

          {/* Pagination */}
          <div style={{ display: 'flex', gap: '0.5rem', marginTop: '1rem', justifyContent: 'flex-end' }}>
            <Button
              variant="ghost"
              size="sm"
              disabled={page === 1}
              onClick={() => setPage(p => p - 1)}
            >
              ← Prev
            </Button>
            <span style={{ alignSelf: 'center', fontSize: '0.85rem', color: 'var(--text-muted)' }}>
              Page {page}
            </span>
            <Button
              variant="ghost"
              size="sm"
              disabled={(events.data?.length ?? 0) < PER_PAGE}
              onClick={() => setPage(p => p + 1)}
            >
              Next →
            </Button>
          </div>
        </>
      )}
    </div>
  )
}
