import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { api, getToken, setToken } from '../api/client'
import type { Admin, LoginResponse } from '../api/types'

interface AuthCtx {
  admin: Admin | null
  loading: boolean
  login: (username: string, password: string) => Promise<LoginResponse>
  verify2fa: (pendingToken: string, code: string) => Promise<void>
  logout: () => Promise<void>
  refresh: () => Promise<void>
}

const Ctx = createContext<AuthCtx | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [admin, setAdmin] = useState<Admin | null>(null)
  const [loading, setLoading] = useState(true)

  const refresh = async () => {
    if (!getToken()) {
      setAdmin(null)
      setLoading(false)
      return
    }
    try {
      const me = await api<Admin>('/api/auth/me')
      setAdmin(me)
    } catch {
      setAdmin(null)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void refresh()
  }, [])

  const login = async (username: string, password: string): Promise<LoginResponse> => {
    const res = await api<LoginResponse>('/api/auth/login', {
      method: 'POST',
      body: { username, password },
    })
    if (res.token) {
      setToken(res.token)
      await refresh()
    }
    return res
  }

  const verify2fa = async (pendingToken: string, code: string) => {
    const res = await api<{ token: string }>('/api/auth/2fa/verify', {
      method: 'POST',
      body: { code },
      bearer: pendingToken,
    })
    setToken(res.token)
    await refresh()
  }

  const logout = async () => {
    try {
      await api('/api/auth/logout', { method: 'POST' })
    } catch {
      // ignore network errors on logout
    }
    setToken(null)
    setAdmin(null)
  }

  return (
    <Ctx.Provider value={{ admin, loading, login, verify2fa, logout, refresh }}>
      {children}
    </Ctx.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth(): AuthCtx {
  const ctx = useContext(Ctx)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
