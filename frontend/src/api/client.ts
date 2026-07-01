// Thin typed fetch wrapper around the panel REST API. The bearer token is held
// in memory and mirrored to localStorage so a reload keeps the session.

const TOKEN_KEY = 'xray_panel_token'

let token: string | null = localStorage.getItem(TOKEN_KEY)

export function getToken(): string | null {
  return token
}

export function setToken(value: string | null): void {
  token = value
  if (value) {
    localStorage.setItem(TOKEN_KEY, value)
  } else {
    localStorage.removeItem(TOKEN_KEY)
  }
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

interface RequestOptions {
  method?: string
  body?: unknown
  bearer?: string // override the stored token (used for the 2FA pending token)
  raw?: boolean // return the Response instead of parsed JSON
}

export async function api<T = unknown>(path: string, opts: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = {}
  const auth = opts.bearer ?? token
  if (auth) headers['Authorization'] = `Bearer ${auth}`
  if (opts.body !== undefined) headers['Content-Type'] = 'application/json'

  const res = await fetch(path, {
    method: opts.method ?? 'GET',
    headers,
    body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
  })

  if (opts.raw) return res as unknown as T

  if (res.status === 401) {
    setToken(null)
  }

  if (!res.ok) {
    let msg = `HTTP ${res.status}`
    try {
      const data = await res.json()
      if (data?.error) msg = data.error
    } catch {
      // non-JSON error body
    }
    throw new ApiError(res.status, msg)
  }

  if (res.status === 204) return undefined as T
  const text = await res.text()
  if (!text) return undefined as T
  return JSON.parse(text) as T
}

export function subscriptionUrl(token: string): string {
  return `${window.location.origin}/sub/${token}`
}
