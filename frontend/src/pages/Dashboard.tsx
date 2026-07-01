import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { DashboardSummary } from '../api/types'
import { Card, Spinner, StatusDot } from '../components/ui'
import { PageHeader } from '../components/PageHeader'
import { formatTime } from '../lib/format'

export function Dashboard() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['dashboard'],
    queryFn: () => api<DashboardSummary>('/api/dashboard/summary'),
  })

  if (isLoading) return <Spinner />
  if (error || !data) return <p className="text-red-500">Failed to load dashboard.</p>

  const stats = [
    { label: 'Servers', value: `${data.servers_online}/${data.servers_total}`, sub: 'online' },
    { label: 'Clients', value: `${data.clients_enabled}/${data.clients_total}`, sub: 'enabled' },
    { label: 'Last sync', value: data.last_sync_at ? formatTime(data.last_sync_at) : '—', sub: '' },
  ]

  return (
    <div className="space-y-6">
      <PageHeader title="Dashboard" />
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        {stats.map((s) => (
          <Card key={s.label}>
            <p className="text-sm text-neutral-500">{s.label}</p>
            <p className="mt-1 text-2xl font-semibold">{s.value}</p>
            {s.sub && <p className="text-xs text-neutral-400">{s.sub}</p>}
          </Card>
        ))}
      </div>

      <div>
        <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-neutral-500">Servers</h2>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          {data.servers.map((s) => (
            <Link key={s.id} to={`/servers/${s.id}`}>
              <Card className="transition-shadow hover:shadow-md">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="font-medium">{s.name}</p>
                    <p className="text-xs text-neutral-500">{s.host}</p>
                  </div>
                  <StatusDot status={s.status} />
                </div>
                <div className="mt-3 flex gap-4 text-xs text-neutral-500">
                  {s.geo_country && <span>{s.geo_country}</span>}
                  <span>sync: {s.last_sync_at ? formatTime(s.last_sync_at) : '—'}</span>
                </div>
                {s.last_sync_error && (
                  <p className="mt-2 truncate text-xs text-red-500" title={s.last_sync_error}>
                    {s.last_sync_error}
                  </p>
                )}
              </Card>
            </Link>
          ))}
          {data.servers.length === 0 && (
            <p className="text-sm text-neutral-500">No servers yet. Add one under Servers.</p>
          )}
        </div>
      </div>
    </div>
  )
}
