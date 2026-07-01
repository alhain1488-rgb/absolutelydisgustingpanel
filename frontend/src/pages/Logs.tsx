import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'
import type { AuditLog } from '../api/types'
import { Card, Spinner } from '../components/ui'
import { PageHeader } from '../components/PageHeader'
import { formatTime } from '../lib/format'

export function Logs() {
  const { data, isLoading } = useQuery({
    queryKey: ['logs'],
    queryFn: () => api<AuditLog[]>('/api/logs?limit=200'),
  })

  if (isLoading) return <Spinner />

  return (
    <div className="space-y-6">
      <PageHeader title="Audit log" />
      <Card className="overflow-x-auto p-0">
        <table className="w-full text-sm">
          <thead className="border-b border-neutral-200 text-left text-xs uppercase text-neutral-400 dark:border-neutral-800">
            <tr>
              <th className="px-4 py-2">Time</th>
              <th className="px-4 py-2">Action</th>
              <th className="px-4 py-2">Target</th>
              <th className="px-4 py-2">IP</th>
            </tr>
          </thead>
          <tbody>
            {(data ?? []).map((e) => (
              <tr key={e.id} className="border-b border-neutral-100 last:border-0 dark:border-neutral-800/50">
                <td className="whitespace-nowrap px-4 py-2 text-neutral-500">{formatTime(e.created_at)}</td>
                <td className="px-4 py-2 font-medium">{e.action}</td>
                <td className="px-4 py-2 text-neutral-500">
                  {e.target_type}
                  {e.target_id ? ` #${e.target_id}` : ''}
                </td>
                <td className="px-4 py-2 font-mono text-xs text-neutral-400">{e.ip}</td>
              </tr>
            ))}
            {data?.length === 0 && (
              <tr>
                <td colSpan={4} className="px-4 py-6 text-center text-neutral-500">
                  No log entries.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>
    </div>
  )
}
