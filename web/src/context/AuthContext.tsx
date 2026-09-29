// ── Auth context ────────────────────────────────────────────────────────────
// Stores the admin Bearer token in localStorage and exposes login/logout.

import {
  createContext,
  useContext,
  useState,
  useCallback,
  type ReactNode,
} from 'react'
import {
  getStoredToken,
  setStoredToken,
  clearStoredToken,
} from '../api'

interface AuthCtx {
  token: string | null
  login: (t: string) => void
  logout: () => void
  isAuthenticated: boolean
}

const Ctx = createContext<AuthCtx | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(getStoredToken)

  const login = useCallback((t: string) => {
    setStoredToken(t)
    setToken(t)
  }, [])

  const logout = useCallback(() => {
    clearStoredToken()
    setToken(null)
  }, [])

  return (
    <Ctx.Provider value={{ token, login, logout, isAuthenticated: !!token }}>
      {children}
    </Ctx.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth(): AuthCtx {
  const ctx = useContext(Ctx)
  if (!ctx) throw new Error('useAuth must be inside AuthProvider')
  return ctx
}
