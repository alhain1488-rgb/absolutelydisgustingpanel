import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { useTheme } from '../theme/theme'
import { Button, Card, Field, Input } from '../components/ui'

export function Login() {
  const { login, verify2fa } = useAuth()
  const { theme, toggle } = useTheme()
  const navigate = useNavigate()

  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [code, setCode] = useState('')
  const [pending, setPending] = useState<string | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const submitCredentials = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      const res = await login(username, password)
      if (res.need_2fa && res.pending_token) {
        setPending(res.pending_token)
      } else if (res.token) {
        navigate('/')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed')
    } finally {
      setBusy(false)
    }
  }

  const submitCode = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!pending) return
    setError('')
    setBusy(true)
    try {
      await verify2fa(pending, code)
      navigate('/')
    } catch {
      setError('Invalid code')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-neutral-50 p-4 text-neutral-900 dark:bg-neutral-950 dark:text-neutral-100">
      <div className="w-full max-w-sm">
        <div className="mb-6 flex items-center justify-between">
          <h1 className="text-2xl font-semibold">Xray Panel</h1>
          <Button variant="ghost" onClick={toggle}>
            {theme === 'dark' ? '☀' : '☾'}
          </Button>
        </div>
        <Card>
          {!pending ? (
            <form onSubmit={submitCredentials} className="space-y-4">
              <Field label="Username">
                <Input value={username} onChange={(e) => setUsername(e.target.value)} autoFocus />
              </Field>
              <Field label="Password">
                <Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
              </Field>
              {error && <p className="text-sm text-red-500">{error}</p>}
              <Button type="submit" className="w-full" disabled={busy}>
                {busy ? 'Signing in…' : 'Sign in'}
              </Button>
            </form>
          ) : (
            <form onSubmit={submitCode} className="space-y-4">
              <p className="text-sm text-neutral-500">Enter the 6-digit code from your authenticator.</p>
              <Field label="2FA code">
                <Input
                  value={code}
                  onChange={(e) => setCode(e.target.value)}
                  inputMode="numeric"
                  autoFocus
                  placeholder="123456"
                />
              </Field>
              {error && <p className="text-sm text-red-500">{error}</p>}
              <Button type="submit" className="w-full" disabled={busy}>
                Verify
              </Button>
            </form>
          )}
        </Card>
      </div>
    </div>
  )
}
