import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Inbound, Server, ServerStats } from '../api/types'
import { Badge, Button, Card, Spinner, StatusDot } from '../components/ui'
import { PageHeader } from '../components/PageHeader'
import { Modal } from '../components/Modal'
import { formatTime, formatUptime } from '../lib/format'
import { InboundForm } from '../components/InboundForm'
import { buildInboundInput, emptyInboundForm, type InboundFormState } from '../lib/inbound'

export function ServerDetail() {
  const { id } = useParams()
  const serverId = Number(id)
  const qc = useQueryClient()
  const [stats, setStats] = useState<ServerStats | null>(null)
  const [statsErr, setStatsErr] = useState('')
  const [open, setOpen] = useState(false)
  const [form, setForm] = useState<InboundFormState>(emptyInboundForm())
  const [formErr, setFormErr] = useState('')

  const { data: server, isLoading } = useQuery({
    queryKey: ['server', serverId],
    queryFn: () => api<Server>(`/api/servers/${serverId}`),
  })
  const { data: inbounds } = useQuery({
    queryKey: ['inbounds', serverId],
    queryFn: () => api<Inbound[]>(`/api/servers/${serverId}/inbounds`),
  })

  const check = useMutation({
    mutationFn: () => api<Server>(`/api/servers/${serverId}/check`, { method: 'POST' }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['server', serverId] }),
  })
  const restart = useMutation({
    mutationFn: () => api(`/api/servers/${serverId}/restart-xray`, { method: 'POST' }),
  })
  const loadStats = useMutation({
    mutationFn: () => api<ServerStats>(`/api/servers/${serverId}/stats`),
    onSuccess: (d) => {
      setStats(d)
      setStatsErr('')
    },
    onError: (e) => setStatsErr(e instanceof Error ? e.message : 'Failed'),
  })
  const createInbound = useMutation({
    mutationFn: () => api<Inbound>(`/api/servers/${serverId}/inbounds`, { method: 'POST', body: buildInboundInput(form) }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['inbounds', serverId] })
      setOpen(false)
      setForm(emptyInboundForm())
      setFormErr('')
    },
    onError: (e) => setFormErr(e instanceof Error ? e.message : 'Failed'),
  })
  const deleteInbound = useMutation({
    mutationFn: (inboundId: number) => api(`/api/inbounds/${inboundId}`, { method: 'DELETE' }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['inbounds', serverId] }),
  })

  if (isLoading || !server) return <Spinner />

  return (
    <div className="space-y-6">
      <PageHeader
        title={server.name}
        action={
          <div className="flex gap-2">
            <Button variant="secondary" onClick={() => check.mutate()} disabled={check.isPending}>
              Check
            </Button>
            <Button variant="secondary" onClick={() => restart.mutate()} disabled={restart.isPending}>
              Restart Xray
            </Button>
          </div>
        }
      />

      <Card>
        <div className="grid grid-cols-2 gap-4 text-sm sm:grid-cols-4">
          <Info label="Status">
            <StatusDot status={server.status} />
          </Info>
          <Info label="Address">{server.host}</Info>
          <Info label="IP / Geo">{server.ip ? `${server.ip} ${server.geo_country}` : '—'}</Info>
          <Info label="Last sync">{formatTime(server.last_sync_at)}</Info>
        </div>
        {server.last_sync_error && <p className="mt-3 text-xs text-red-500">Sync error: {server.last_sync_error}</p>}
        {restart.isSuccess && <p className="mt-3 text-xs text-emerald-500">Restart command sent.</p>}
        {restart.isError && <p className="mt-3 text-xs text-red-500">Restart failed (server unreachable?).</p>}
      </Card>

      <Card>
        <div className="mb-3 flex items-center justify-between">
          <h2 className="font-medium">Resource metrics</h2>
          <Button variant="secondary" onClick={() => loadStats.mutate()} disabled={loadStats.isPending}>
            Load
          </Button>
        </div>
        {statsErr && <p className="text-xs text-red-500">{statsErr}</p>}
        {stats ? (
          <div className="grid grid-cols-2 gap-4 text-sm sm:grid-cols-4">
            <Info label="CPU">{stats.cpu_percent}%</Info>
            <Info label="RAM">
              {stats.mem_percent}% ({stats.mem_used_mb}/{stats.mem_total_mb} MB)
            </Info>
            <Info label="Disk">
              {stats.disk_percent}% ({stats.disk_used_gb}/{stats.disk_total_gb} GB)
            </Info>
            <Info label="Uptime">{formatUptime(stats.uptime_seconds)}</Info>
          </div>
        ) : (
          <p className="text-xs text-neutral-500">Metrics are read over SSH on demand.</p>
        )}
      </Card>

      <div>
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-neutral-500">Inbounds</h2>
          <Button onClick={() => setOpen(true)}>Add inbound</Button>
        </div>
        <div className="space-y-2">
          {(inbounds ?? []).map((inb) => (
            <Card key={inb.id}>
              <div className="flex items-center justify-between gap-4">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{inb.tag}</span>
                    <Badge>{inb.protocol}</Badge>
                    {inb.enabled ? <Badge tone="green">enabled</Badge> : <Badge tone="red">disabled</Badge>}
                  </div>
                  <p className="text-xs text-neutral-500">
                    {inb.listen}:{inb.port}
                  </p>
                </div>
                <Button variant="danger" onClick={() => deleteInbound.mutate(inb.id)}>
                  Delete
                </Button>
              </div>
            </Card>
          ))}
          {inbounds?.length === 0 && <p className="text-sm text-neutral-500">No inbounds yet.</p>}
        </div>
      </div>

      <Modal open={open} title="Add inbound" onClose={() => setOpen(false)}>
        <InboundForm form={form} setForm={setForm} />
        {formErr && <p className="mt-2 text-sm text-red-500">{formErr}</p>}
        <div className="mt-4 flex justify-end gap-2">
          <Button variant="secondary" onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button onClick={() => createInbound.mutate()} disabled={createInbound.isPending}>
            Create
          </Button>
        </div>
      </Modal>
    </div>
  )
}

function Info({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <p className="text-xs text-neutral-400">{label}</p>
      <div className="mt-0.5">{children}</div>
    </div>
  )
}
