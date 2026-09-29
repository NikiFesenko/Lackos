// ── Reusable UI primitives ──────────────────────────────────────────────────
import { type InputHTMLAttributes, type ButtonHTMLAttributes, type ReactNode, useState, useRef, KeyboardEvent } from 'react'
import { X } from 'lucide-react'
import { clsx } from 'clsx'

// ── Button ───────────────────────────────────────────────────────────────────
type BtnVariant = 'primary' | 'danger' | 'ghost'
interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: BtnVariant
  size?: 'sm' | 'md'
  loading?: boolean
  children: ReactNode
}

export function Button({
  variant = 'primary',
  size = 'md',
  loading,
  children,
  className,
  disabled,
  ...rest
}: ButtonProps) {
  return (
    <button
      className={clsx('btn', `btn-${variant}`, size === 'sm' && 'btn-sm', className)}
      disabled={disabled ?? loading}
      aria-busy={loading}
      {...rest}
    >
      {loading ? <span className="spinner" style={{ width: 14, height: 14, borderWidth: 2 }} /> : null}
      {children}
    </button>
  )
}

// ── Input ────────────────────────────────────────────────────────────────────
interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string
  error?: string
}
export function Input({ label, error, className, id, ...rest }: InputProps) {
  return (
    <div className="form-group">
      {label && <label className="form-label" htmlFor={id}>{label}</label>}
      <input id={id} className={clsx('form-input', className)} {...rest} />
      {error && <span style={{ fontSize: '0.8rem', color: 'var(--danger)' }}>{error}</span>}
    </div>
  )
}

// ── Select ───────────────────────────────────────────────────────────────────
interface SelectProps {
  label?: string
  value: string
  onChange: (v: string) => void
  options: { value: string; label: string }[]
  id?: string
}
export function Select({ label, value, onChange, options, id }: SelectProps) {
  return (
    <div className="form-group">
      {label && <label className="form-label" htmlFor={id}>{label}</label>}
      <select
        id={id}
        className="form-select"
        value={value}
        onChange={e => onChange(e.target.value)}
      >
        {options.map(o => (
          <option key={o.value} value={o.value}>{o.label}</option>
        ))}
      </select>
    </div>
  )
}

// ── Spinner ───────────────────────────────────────────────────────────────────
export function Spinner({ center = true }: { center?: boolean }) {
  return center
    ? <div className="spinner-center"><div className="spinner" /></div>
    : <div className="spinner" />
}

// ── Badge ─────────────────────────────────────────────────────────────────────
type BadgeVariant = 'green' | 'red' | 'blue' | 'warn' | 'muted'
export function Badge({ children, variant = 'muted' }: { children: ReactNode; variant?: BadgeVariant }) {
  return <span className={`badge badge-${variant}`}>{children}</span>
}

export function OutcomeBadge({ outcome }: { outcome: string }) {
  const map: Record<string, BadgeVariant> = {
    allowed:      'green',
    denied:       'red',
    rate_limited: 'warn',
    error:        'red',
  }
  return <Badge variant={map[outcome] ?? 'muted'}>{outcome}</Badge>
}

// ── Toggle ────────────────────────────────────────────────────────────────────
export function Toggle({
  checked,
  onChange,
  label,
}: {
  checked: boolean
  onChange: (v: boolean) => void
  label?: string
}) {
  const id = useRef(`tgl-${Math.random().toString(36).slice(2)}`).current
  return (
    <label style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', cursor: 'pointer' }}>
      <span className="toggle">
        <input
          id={id}
          type="checkbox"
          checked={checked}
          onChange={e => onChange(e.target.checked)}
        />
        <span className="toggle-slider" />
      </span>
      {label && <span style={{ fontSize: '0.85rem', color: 'var(--text-muted)' }}>{label}</span>}
    </label>
  )
}

// ── Chip input ─────────────────────────────────────────────────────────────
export function ChipInput({
  label,
  value,
  onChange,
  placeholder = 'Type and press Enter',
}: {
  label?: string
  value: string[]
  onChange: (v: string[]) => void
  placeholder?: string
}) {
  const [draft, setDraft] = useState('')

  function addChip() {
    const trimmed = draft.trim()
    if (trimmed && !value.includes(trimmed)) {
      onChange([...value, trimmed])
    }
    setDraft('')
  }

  function onKey(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter' || e.key === ',') { e.preventDefault(); addChip() }
    if (e.key === 'Backspace' && draft === '' && value.length) {
      onChange(value.slice(0, -1))
    }
  }

  return (
    <div className="form-group">
      {label && <span className="form-label">{label}</span>}
      <div className="chip-wrap">
        {value.map(chip => (
          <span key={chip} className="chip">
            {chip}
            <button type="button" onClick={() => onChange(value.filter(c => c !== chip))} aria-label={`Remove ${chip}`}>
              <X size={12} />
            </button>
          </span>
        ))}
        <input
          className="chip-input"
          value={draft}
          onChange={e => setDraft(e.target.value)}
          onKeyDown={onKey}
          onBlur={addChip}
          placeholder={value.length === 0 ? placeholder : ''}
        />
      </div>
    </div>
  )
}

// ── Modal ─────────────────────────────────────────────────────────────────────
export function Modal({
  title,
  children,
  onClose,
  footer,
}: {
  title: string
  children: ReactNode
  onClose: () => void
  footer?: ReactNode
}) {
  return (
    <div
      className="modal-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
      onClick={e => { if (e.target === e.currentTarget) onClose() }}
    >
      <div className="modal">
        <h2 id="modal-title">{title}</h2>
        {children}
        {footer && <div className="modal-footer">{footer}</div>}
      </div>
    </div>
  )
}

// ── Error alert ───────────────────────────────────────────────────────────────
export function ErrorAlert({ error }: { error: unknown }) {
  const msg = error instanceof Error ? error.message : String(error)
  return <div className="alert alert-error" role="alert">{msg}</div>
}
