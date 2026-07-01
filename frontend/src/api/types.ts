// Types mirroring the backend REST API (see docs/openapi.yaml).

export interface Admin {
  id: number
  username: string
  totp_enabled: boolean
}

export interface LoginResponse {
  token?: string
  need_2fa: boolean
  pending_token?: string
}

export interface TotpSetup {
  otpauth_url: string
  qr: string
}

export type ServerStatus = 'unknown' | 'online' | 'offline' | 'error'

export interface Server {
  id: number
  name: string
  host: string
  ssh_port: number
  ssh_user: string
  ssh_auth_method: 'key' | 'password'
  xray_config_path: string
  xray_service_name: string
  ip: string
  geo_country: string
  geo_city: string
  geo_asn: string
  status: ServerStatus
  last_check_at?: string
  last_sync_at?: string
  last_sync_error: string
  created_at: string
  updated_at: string
}

export interface ServerInput {
  name: string
  host: string
  ssh_port?: number
  ssh_user: string
  ssh_auth_method: 'key' | 'password'
  ssh_secret?: string
  ssh_passphrase?: string
  xray_config_path?: string
  xray_service_name?: string
}

export interface ServerStats {
  cpu_percent: number
  mem_percent: number
  mem_total_mb: number
  mem_used_mb: number
  disk_percent: number
  disk_total_gb: number
  disk_used_gb: number
  uptime_seconds: number
}

export type Protocol = 'vless' | 'vmess' | 'trojan' | 'shadowsocks'

export interface Inbound {
  id: number
  server_id: number
  tag: string
  protocol: Protocol
  listen: string
  port: number
  settings_json: Record<string, unknown>
  stream_settings_json: Record<string, unknown>
  sniffing_json: Record<string, unknown>
  remark: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface InboundInput {
  tag: string
  protocol: Protocol
  listen?: string
  port: number
  remark?: string
  enabled?: boolean
  settings_json?: Record<string, unknown>
  stream_settings_json?: Record<string, unknown>
  sniffing_json?: Record<string, unknown>
}

export interface Client {
  id: number
  name: string
  uuid: string
  password: string
  subscription_token: string
  enabled: boolean
  remark: string
  inbound_ids: number[]
  created_at: string
  updated_at: string
}

export interface DashboardSummary {
  clients_total: number
  clients_enabled: number
  servers_total: number
  servers_online: number
  last_sync_at?: string
  servers: Server[]
}

export interface AuditLog {
  id: number
  admin_id: number
  action: string
  target_type: string
  target_id: number
  detail_json: Record<string, unknown>
  ip: string
  user_agent: string
  created_at: string
}
