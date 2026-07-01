import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../api/client'
import { useAuth } from '../auth/AuthContext'
import { useTheme } from '../theme/theme'
import type { TotpSetup } from '../api/types'
import { Button, Card, Field, Input, Spinner } from '../components/ui'
import { PageHeader } from '../components/PageHeader'

export function Settings() {
  const qc = useQueryClient()
  const { admin, refresh } = useAuth()
  const { theme, toggle } = useTheme()

  const { data: settings, isLoading } = useQuery({
    queryKey: ['settings'],
    queryFn: () => api<Record<string, string>>('/api/settings'),
  })

  const [domain, setDomain] = useState('')
  const [subBase, setSubBase] = useState('')
  useEffect(() => {
    if (settings) {
      setDomain(settings.domain ?? '')
      setSubBase(settings.sub_base_url ?? '')
    }
  }, [settings])

  const save = useMutation({
    mutationFn: () => api('/api/settings', { method: 'PUT', body: { domain, sub_base_url: subBase } }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['settings'] }),
  })

  // 2FA enrollment.
  const [setup, setSetup] = useState<TotpSetup | null>(null)
  const [code, setCode] = useState('')
  const [twoFaMsg, setTwoFaMsg] = useState('')
  const beginSetup = useMutation({
    mutationFn: () => api<TotpSetup>('/api/auth/2fa/setup', { method: 'POST' }),
    onSuccess: (d) => setSetup(d),
  })
  const enable = useMutation({
    mutationFn: () => api('/api/auth/2fa/enable', { method: 'POST', body: { code } }),
    onSuccess: async () => {
      setTwoFaMsg('Two-factor authentication enabled.')
      setSetup(null)
      setCode('')
      await refresh()
    },
    onError: () => setTwoFaMsg('Invalid code, try again.'),
  })

  if (isLoading) return <Spinner />

  return (
    <div className="space-y-6">
      <PageHeader title="Settings" />

      <Card>
        <h2 className="mb-3 font-medium">General</h2>
        <div className="space-y-3">
          <Field label="Panel domain">
            <Input value={domain} onChange={(e) => setDomain(e.target.value)} placeholder="panel.example.com" />
          </Field>
          <Field label="Subscription base URL">
            <Input value={subBase} onChange={(e) => setSubBase(e.target.value)} placeholder="https://panel.example.com" />
          </Field>
          <div className="flex justify-end">
            <Button onClick={() => save.mutate()} disabled={save.isPending}>
              Save
            </Button>
          </div>
        </div>
      </Card>

      <Card>
        <h2 className="mb-3 font-medium">Appearance</h2>
        <div className="flex items-center justify-between">
          <span className="text-sm text-neutral-500">Theme</span>
          <Button variant="secondary" onClick={toggle}>
            {theme === 'dark' ? '☀ Switch to light' : '☾ Switch to dark'}
          </Button>
        </div>
      </Card>

      <Card>
        <h2 className="mb-3 font-medium">Two-factor authentication</h2>
        {admin?.totp_enabled ? (
          <p className="text-sm text-emerald-600 dark:text-emerald-400">2FA is enabled for your account.</p>
        ) : setup ? (
          <div className="space-y-3">
            <img src={setup.qr} alt="TOTP QR" className="h-40 w-40 rounded bg-white p-2" />
            <p className="break-all font-mono text-xs text-neutral-500">{setup.otpauth_url}</p>
            <Field label="Enter code to confirm">
              <Input value={code} onChange={(e) => setCode(e.target.value)} inputMode="numeric" placeholder="123456" />
            </Field>
            <Button onClick={() => enable.mutate()} disabled={enable.isPending}>
              Enable 2FA
            </Button>
            {twoFaMsg && <p className="text-sm text-red-500">{twoFaMsg}</p>}
          </div>
        ) : (
          <div className="space-y-2">
            <p className="text-sm text-neutral-500">Protect your account with a TOTP authenticator app.</p>
            <Button variant="secondary" onClick={() => beginSetup.mutate()} disabled={beginSetup.isPending}>
              Set up 2FA
            </Button>
            {twoFaMsg && <p className="text-sm text-emerald-500">{twoFaMsg}</p>}
          </div>
        )}
      </Card>
    </div>
  )
}
