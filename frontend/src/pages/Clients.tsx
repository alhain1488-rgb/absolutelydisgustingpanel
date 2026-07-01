import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { Client } from '../api/types'
import { Badge, Button, Card, Field, Input, Spinner } from '../components/ui'
import { PageHeader } from '../components/PageHeader'
import { Modal } from '../components/Modal'

export function Clients() {
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [remark, setRemark] = useState('')

  const { data: clients, isLoading } = useQuery({
    queryKey: ['clients'],
    queryFn: () => api<Client[]>('/api/clients'),
  })

  const create = useMutation({
    mutationFn: () => api<Client>('/api/clients', { method: 'POST', body: { name, remark } }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['clients'] })
      setOpen(false)
      setName('')
      setRemark('')
    },
  })

  return (
    <div className="space-y-6">
      <PageHeader title="Clients" action={<Button onClick={() => setOpen(true)}>Add client</Button>} />

      {isLoading ? (
        <Spinner />
      ) : (
        <div className="space-y-2">
          {(clients ?? []).map((c) => (
            <Card key={c.id}>
              <div className="flex items-center justify-between">
                <div>
                  <Link to={`/clients/${c.id}`} className="font-medium hover:underline">
                    {c.name}
                  </Link>
                  <p className="text-xs text-neutral-500">{c.remark || c.uuid}</p>
                </div>
                {c.enabled ? <Badge tone="green">enabled</Badge> : <Badge tone="red">disabled</Badge>}
              </div>
            </Card>
          ))}
          {clients?.length === 0 && <p className="text-sm text-neutral-500">No clients yet.</p>}
        </div>
      )}

      <Modal open={open} title="Add client" onClose={() => setOpen(false)}>
        <div className="space-y-3">
          <Field label="Name">
            <Input value={name} onChange={(e) => setName(e.target.value)} autoFocus />
          </Field>
          <Field label="Remark (optional)">
            <Input value={remark} onChange={(e) => setRemark(e.target.value)} />
          </Field>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="secondary" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button onClick={() => create.mutate()} disabled={create.isPending || !name}>
              Create
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  )
}
