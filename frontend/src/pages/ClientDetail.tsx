import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useParams } from 'react-router-dom'
import { api, getToken, subscriptionUrl } from '../api/client'
import type { Client, Inbound, Server } from '../api/types'
import { Badge, Button, Card, Spinner } from '../components/ui'
import { PageHeader } from '../components/PageHeader'

interface ServerInbounds {
  server: Server
  inbounds: Inbound[]
}

export function ClientDetail() {
  const { id } = useParams()
  const clientId = Number(id)
  const qc = useQueryClient()
  const navigate = useNavigate()

  const { data: client, isLoading } = useQuery({
    queryKey: ['client', clientId],
    queryFn: () => api<Client>(`/api/clients/${clientId}`),
  })
  const { data: grouped } = useQuery({
    queryKey: ['all-inbounds'],
    queryFn: async (): Promise<ServerInbounds[]> => {
      const servers = await api<Server[]>('/api/servers')
      return Promise.all(
        (servers ?? []).map(async (server) => ({
          server,
          inbounds: (await api<Inbound[]>(`/api/servers/${server.id}/inbounds`)) ?? [],
        })),
      )
    },
  })
  const { data: links } = useQuery({
    queryKey: ['client-links', clientId],
    queryFn: () => api<string[]>(`/api/clients/${clientId}/links`),
    enabled: !!client,
  })

  const [selected, setSelected] = useState<Set<number>>(new Set())
  useEffect(() => {
    if (client) setSelected(new Set(client.inbound_ids ?? []))
  }, [client])

  const qrUrl = useQrBlob(clientId, client?.subscription_token)

  const saveGrants = useMutation({
    mutationFn: () => api(`/api/clients/${clientId}/inbounds`, { method: 'PUT', body: { inbound_ids: [...selected] } }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['client', clientId] })
      void qc.invalidateQueries({ queryKey: ['client-links', clientId] })
    },
  })
  const toggleEnabled = useMutation({
    mutationFn: (enabled: boolean) =>
      api(`/api/clients/${clientId}/${enabled ? 'enable' : 'disable'}`, { method: 'POST' }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['client', clientId] }),
  })
  const rotate = useMutation({
    mutationFn: () => api<Client>(`/api/clients/${clientId}/rotate-token`, { method: 'POST' }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['client', clientId] }),
  })
  const remove = useMutation({
    mutationFn: () => api(`/api/clients/${clientId}`, { method: 'DELETE' }),
    onSuccess: () => navigate('/clients'),
  })

  const subUrl = useMemo(
    () => (client ? subscriptionUrl(client.subscription_token) : ''),
    [client],
  )

  if (isLoading || !client) return <Spinner />

  const toggle = (inboundId: number) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(inboundId)) next.delete(inboundId)
      else next.add(inboundId)
      return next
    })
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={client.name}
        action={
          <div className="flex gap-2">
            <Button
              variant="secondary"
              onClick={() => toggleEnabled.mutate(!client.enabled)}
              disabled={toggleEnabled.isPending}
            >
              {client.enabled ? 'Disable' : 'Enable'}
            </Button>
            <Button variant="danger" onClick={() => remove.mutate()}>
              Delete
            </Button>
          </div>
        }
      />

      <Card>
        <div className="flex flex-wrap items-center gap-3 text-sm">
          {client.enabled ? <Badge tone="green">enabled</Badge> : <Badge tone="red">disabled</Badge>}
          <span className="font-mono text-xs text-neutral-500">{client.uuid}</span>
        </div>
      </Card>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <Card>
          <h2 className="mb-3 font-medium">Subscription</h2>
          <div className="flex items-center gap-2">
            <input
              readOnly
              value={subUrl}
              className="w-full rounded-lg border border-neutral-300 bg-neutral-50 px-3 py-2 font-mono text-xs dark:border-neutral-700 dark:bg-neutral-950"
            />
            <Button variant="secondary" onClick={() => navigator.clipboard.writeText(subUrl)}>
              Copy
            </Button>
          </div>
          <div className="mt-3 flex gap-2">
            <Button variant="secondary" onClick={() => rotate.mutate()} disabled={rotate.isPending}>
              Refresh configuration
            </Button>
            <a href={`/api/clients/${clientId}/config`} onClick={downloadWithAuth(clientId)}>
              <Button variant="secondary">Download</Button>
            </a>
          </div>
        </Card>

        <Card>
          <h2 className="mb-3 font-medium">QR code</h2>
          {qrUrl ? (
            <img src={qrUrl} alt="subscription QR" className="h-40 w-40 rounded bg-white p-2" />
          ) : (
            <p className="text-xs text-neutral-500">Loading…</p>
          )}
        </Card>
      </div>

      <Card>
        <div className="mb-3 flex items-center justify-between">
          <h2 className="font-medium">Access (inbounds)</h2>
          <Button onClick={() => saveGrants.mutate()} disabled={saveGrants.isPending}>
            Save access
          </Button>
        </div>
        <div className="space-y-4">
          {(grouped ?? []).map(({ server, inbounds }) => (
            <div key={server.id}>
              <p className="mb-1.5 text-sm font-medium text-neutral-600 dark:text-neutral-300">{server.name}</p>
              <div className="space-y-1">
                {inbounds.map((inb) => (
                  <label key={inb.id} className="flex items-center gap-2 text-sm">
                    <input type="checkbox" checked={selected.has(inb.id)} onChange={() => toggle(inb.id)} />
                    <span>{inb.tag}</span>
                    <Badge>{inb.protocol}</Badge>
                    <span className="text-xs text-neutral-400">:{inb.port}</span>
                  </label>
                ))}
                {inbounds.length === 0 && <p className="text-xs text-neutral-400">No inbounds.</p>}
              </div>
            </div>
          ))}
          {grouped?.length === 0 && <p className="text-sm text-neutral-500">No servers/inbounds yet.</p>}
        </div>
      </Card>

      {links && links.length > 0 && (
        <Card>
          <h2 className="mb-3 font-medium">Connection links</h2>
          <div className="space-y-2">
            {links.map((l, i) => (
              <p key={i} className="break-all rounded bg-neutral-100 p-2 font-mono text-xs dark:bg-neutral-950">
                {l}
              </p>
            ))}
          </div>
        </Card>
      )}
    </div>
  )
}

// useQrBlob fetches the authenticated QR PNG and exposes it as an object URL.
function useQrBlob(clientId: number, token: string | undefined): string | null {
  const [url, setUrl] = useState<string | null>(null)
  useEffect(() => {
    let revoke: string | null = null
    let active = true
    void (async () => {
      const res = await fetch(`/api/clients/${clientId}/qrcode`, {
        headers: getToken() ? { Authorization: `Bearer ${getToken()}` } : {},
      })
      if (!res.ok || !active) return
      const blob = await res.blob()
      const obj = URL.createObjectURL(blob)
      revoke = obj
      if (active) setUrl(obj)
    })()
    return () => {
      active = false
      if (revoke) URL.revokeObjectURL(revoke)
    }
    // token in deps so the QR refreshes after a token rotation.
  }, [clientId, token])
  return url
}

// downloadWithAuth intercepts the anchor click to fetch the config with the
// bearer token and trigger a browser download.
function downloadWithAuth(clientId: number) {
  return (e: React.MouseEvent) => {
    e.preventDefault()
    void (async () => {
      const res = await fetch(`/api/clients/${clientId}/config`, {
        headers: getToken() ? { Authorization: `Bearer ${getToken()}` } : {},
      })
      if (!res.ok) return
      const blob = await res.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'subscription.txt'
      a.click()
      URL.revokeObjectURL(url)
    })()
  }
}
