import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { Server, ServerInput } from '../api/types'
import { Button, Card, Field, Input, Select, Spinner, StatusDot } from '../components/ui'
import { PageHeader } from '../components/PageHeader'
import { Modal } from '../components/Modal'
import { formatTime } from '../lib/format'

const empty: ServerInput = {
  name: '',
  host: '',
  ssh_port: 22,
  ssh_user: 'root',
  ssh_auth_method: 'key',
  ssh_secret: '',
  xray_config_path: '/usr/local/etc/xray/config.json',
  xray_service_name: 'xray',
}

export function Servers() {
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const [form, setForm] = useState<ServerInput>(empty)
  const [error, setError] = useState('')

  const { data: servers, isLoading } = useQuery({
    queryKey: ['servers'],
    queryFn: () => api<Server[]>('/api/servers'),
  })

  const create = useMutation({
    mutationFn: (input: ServerInput) => api<Server>('/api/servers', { method: 'POST', body: input }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['servers'] })
      setOpen(false)
      setForm(empty)
      setError('')
    },
    onError: (e) => setError(e instanceof Error ? e.message : 'Failed'),
  })

  const check = useMutation({
    mutationFn: (id: number) => api<Server>(`/api/servers/${id}/check`, { method: 'POST' }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['servers'] }),
  })

  const set = (patch: Partial<ServerInput>) => setForm((f) => ({ ...f, ...patch }))

  return (
    <div className="space-y-6">
      <PageHeader title="Servers" action={<Button onClick={() => setOpen(true)}>Add server</Button>} />

      {isLoading ? (
        <Spinner />
      ) : (
        <div className="space-y-3">
          {(servers ?? []).map((s) => (
            <Card key={s.id}>
              <div className="flex items-center justify-between gap-4">
                <div className="min-w-0">
                  <Link to={`/servers/${s.id}`} className="font-medium hover:underline">
                    {s.name}
                  </Link>
                  <p className="truncate text-xs text-neutral-500">
                    {s.ssh_user}@{s.host}:{s.ssh_port} · {s.geo_country || 'geo ?'}
                  </p>
                </div>
                <div className="flex items-center gap-4">
                  <StatusDot status={s.status} />
                  <span className="hidden text-xs text-neutral-400 sm:inline">
                    checked {formatTime(s.last_check_at)}
                  </span>
                  <Button variant="secondary" onClick={() => check.mutate(s.id)} disabled={check.isPending}>
                    Check
                  </Button>
                </div>
              </div>
            </Card>
          ))}
          {servers?.length === 0 && <p className="text-sm text-neutral-500">No servers yet.</p>}
        </div>
      )}

      <Modal open={open} title="Add server" onClose={() => setOpen(false)}>
        <div className="space-y-3">
          <Field label="Name">
            <Input value={form.name} onChange={(e) => set({ name: e.target.value })} />
          </Field>
          <div className="grid grid-cols-3 gap-3">
            <div className="col-span-2">
              <Field label="Host">
                <Input value={form.host} onChange={(e) => set({ host: e.target.value })} />
              </Field>
            </div>
            <Field label="SSH port">
              <Input
                type="number"
                value={form.ssh_port}
                onChange={(e) => set({ ssh_port: Number(e.target.value) })}
              />
            </Field>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <Field label="SSH user">
              <Input value={form.ssh_user} onChange={(e) => set({ ssh_user: e.target.value })} />
            </Field>
            <Field label="Auth method">
              <Select
                value={form.ssh_auth_method}
                onChange={(e) => set({ ssh_auth_method: e.target.value as 'key' | 'password' })}
              >
                <option value="key">Private key</option>
                <option value="password">Password</option>
              </Select>
            </Field>
          </div>
          <Field label={form.ssh_auth_method === 'key' ? 'Private key (PEM)' : 'Password'}>
            <textarea
              className="w-full rounded-lg border border-neutral-300 bg-white px-3 py-2 font-mono text-xs dark:border-neutral-700 dark:bg-neutral-900"
              rows={form.ssh_auth_method === 'key' ? 4 : 1}
              value={form.ssh_secret}
              onChange={(e) => set({ ssh_secret: e.target.value })}
            />
          </Field>
          {error && <p className="text-sm text-red-500">{error}</p>}
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="secondary" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button onClick={() => create.mutate(form)} disabled={create.isPending}>
              Create
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  )
}
