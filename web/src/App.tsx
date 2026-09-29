// ── App router ─────────────────────────────────────────────────────────────
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider, useAuth } from './context/AuthContext'
import Layout from './components/Layout'
import './components/components.css'

// Pages (lazy-loaded for smaller initial bundle)
import { lazy, Suspense } from 'react'
import { Spinner } from './components/ui'

const Login         = lazy(() => import('./pages/Login'))
const Users         = lazy(() => import('./pages/Users'))
const Roles         = lazy(() => import('./pages/Roles'))
const Servers       = lazy(() => import('./pages/Servers'))
const Policies      = lazy(() => import('./pages/Policies'))
const AuditLog      = lazy(() => import('./pages/AuditLog'))
const Tokens        = lazy(() => import('./pages/Tokens'))

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuth()
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" replace />
}

function AppRoutes() {
  const { isAuthenticated } = useAuth()

  return (
    <Suspense fallback={<Spinner />}>
      <Routes>
        <Route
          path="/login"
          element={isAuthenticated ? <Navigate to="/users" replace /> : <Login />}
        />
        <Route
          path="/"
          element={
            <ProtectedRoute>
              <Layout />
            </ProtectedRoute>
          }
        >
          <Route index element={<Navigate to="/users" replace />} />
          <Route path="users"   element={<Users />} />
          <Route path="roles"   element={<Roles />} />
          <Route path="servers" element={<Servers />} />
          <Route path="policies" element={<Policies />} />
          <Route path="audit"   element={<AuditLog />} />
          <Route path="tokens"  element={<Tokens />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Suspense>
  )
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <AppRoutes />
      </BrowserRouter>
    </AuthProvider>
  )
}
