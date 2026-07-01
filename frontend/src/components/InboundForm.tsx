import type { Dispatch, SetStateAction } from 'react'
import type { Protocol } from '../api/types'
import type { InboundFormState } from '../lib/inbound'
import { Field, Input, Select } from './ui'

export function InboundForm({
  form,
  setForm,
}: {
  form: InboundFormState
  setForm: Dispatch<SetStateAction<InboundFormState>>
}) {
  const set = (patch: Partial<InboundFormState>) => setForm((f) => ({ ...f, ...patch }))
  const isVless = form.protocol === 'vless'
  const isSS = form.protocol === 'shadowsocks'

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-2 gap-3">
        <Field label="Tag">
          <Input value={form.tag} onChange={(e) => set({ tag: e.target.value })} placeholder="reality-in" />
        </Field>
        <Field label="Port">
          <Input type="number" value={form.port} onChange={(e) => set({ port: Number(e.target.value) })} />
        </Field>
      </div>

      <Field label="Protocol">
        <Select
          value={form.protocol}
          onChange={(e) => {
            const protocol = e.target.value as Protocol
            // Shadowsocks has no transport/security choices in this form.
            set({ protocol, security: protocol === 'shadowsocks' ? 'none' : form.security })
          }}
        >
          <option value="vless">VLESS</option>
          <option value="vmess">VMess</option>
          <option value="trojan">Trojan</option>
          <option value="shadowsocks">Shadowsocks</option>
        </Select>
      </Field>

      {!isSS && (
        <div className="grid grid-cols-2 gap-3">
          <Field label="Network">
            <Select value={form.network} onChange={(e) => set({ network: e.target.value as InboundFormState['network'] })}>
              <option value="tcp">tcp</option>
              <option value="ws">ws</option>
              <option value="grpc">grpc</option>
            </Select>
          </Field>
          <Field label="Security">
            <Select
              value={form.security}
              onChange={(e) => set({ security: e.target.value as InboundFormState['security'] })}
            >
              <option value="none">none</option>
              <option value="tls">tls</option>
              {isVless && <option value="reality">reality</option>}
            </Select>
          </Field>
        </div>
      )}

      {isVless && form.security === 'reality' && (
        <>
          <Field label="Flow">
            <Input value={form.flow} onChange={(e) => set({ flow: e.target.value })} />
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Dest (SNI target)">
              <Input value={form.dest} onChange={(e) => set({ dest: e.target.value })} />
            </Field>
            <Field label="Server names (comma-sep)">
              <Input value={form.serverNames} onChange={(e) => set({ serverNames: e.target.value })} />
            </Field>
          </div>
          <p className="text-xs text-neutral-500">REALITY keys and shortId are generated on the backend.</p>
        </>
      )}

      {form.security === 'tls' && (
        <Field label="TLS server name">
          <Input value={form.tlsServerName} onChange={(e) => set({ tlsServerName: e.target.value })} />
        </Field>
      )}

      {form.network === 'ws' && (
        <Field label="WebSocket path">
          <Input value={form.wsPath} onChange={(e) => set({ wsPath: e.target.value })} />
        </Field>
      )}
      {form.network === 'grpc' && (
        <Field label="gRPC service name">
          <Input value={form.grpcServiceName} onChange={(e) => set({ grpcServiceName: e.target.value })} />
        </Field>
      )}

      {isSS && <p className="text-xs text-neutral-500">Method (aes-256-gcm) and password are generated on the backend.</p>}
    </div>
  )
}
