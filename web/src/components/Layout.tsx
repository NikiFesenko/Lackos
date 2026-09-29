// ── Sidebar navigation layout ───────────────────────────────────────────────
import { NavLink, Outlet } from 'react-router-dom'
import {
  Users,
  ShieldCheck,
  Server,
  ClipboardList,
  ScrollText,
  Key,
  LogOut,
  Shield,
} from 'lucide-react'
import { useAuth } from '../context/AuthContext'

const NAV = [
  { to: '/users',   icon: Users,         label: 'Users' },
  { to: '/roles',   icon: ShieldCheck,   label: 'Roles' },
  { to: '/servers', icon: Server,        label: 'Downstream Servers' },
  { to: '/policies',icon: ClipboardList, label: 'Policies' },
  { to: '/audit',   icon: ScrollText,    label: 'Audit Log' },
  { to: '/tokens',  icon: Key,           label: 'Tokens' },
]

export default function Layout() {
  const { logout } = useAuth()

  return (
    <div className="layout">
      <aside className="sidebar" aria-label="Main navigation">
        <div className="sidebar-logo">
          <Shield size={20} color="var(--primary)" />
          MCP<span>Gate</span>
        </div>

        <nav className="sidebar-nav">
          {NAV.map(({ to, icon: Icon, label }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                `nav-item${isActive ? ' active' : ''}`
              }
            >
              <Icon size={16} />
              {label}
            </NavLink>
          ))}
        </nav>

        <div className="sidebar-footer">
          <button
            className="nav-item btn-ghost"
            style={{ width: '100%', border: 'none', background: 'none' }}
            onClick={logout}
            aria-label="Sign out"
          >
            <LogOut size={16} />
            Sign out
          </button>
        </div>
      </aside>

      <main className="main-content" id="main-content">
        <Outlet />
      </main>
    </div>
  )
}
