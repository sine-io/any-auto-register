export const API = import.meta.env.VITE_PY_API_BASE || '/api'
export const API_BASE = API
export const GO_API = import.meta.env.VITE_GO_API_BASE || API

const GO_PATTERNS: Record<string, RegExp[]> = {
  GET: [
    /^\/platforms$/,
    /^\/config$/,
    /^\/proxies$/,
    /^\/accounts(?:\?.*)?$/,
    /^\/accounts\/stats$/,
    /^\/solver\/status$/,
    /^\/integrations\/services$/,
    /^\/tasks(?:\?.*)?$/,
    /^\/tasks\/logs(?:\?.*)?$/,
    /^\/tasks\/[^/]+$/,
    /^\/tasks\/[^/]+\/logs\/stream(?:\?.*)?$/,
  ],
  POST: [
    /^\/tasks\/register$/,
    /^\/proxies$/,
    /^\/proxies\/bulk$/,
    /^\/proxies\/check$/,
    /^\/solver\/restart$/,
    /^\/integrations\/services\/start-all$/,
    /^\/integrations\/services\/stop-all$/,
    /^\/integrations\/services\/[^/]+\/start$/,
    /^\/integrations\/services\/[^/]+\/install$/,
    /^\/integrations\/services\/[^/]+\/stop$/,
    /^\/integrations\/backfill$/,
  ],
  PUT: [
    /^\/config$/,
  ],
  PATCH: [
    /^\/proxies\/[^/]+\/toggle$/,
  ],
  DELETE: [
    /^\/proxies\/[^/]+$/,
  ],
}

export function getApiBase(path: string, method = 'GET') {
  const normalizedMethod = method.toUpperCase()
  const patterns = GO_PATTERNS[normalizedMethod] || []
  return patterns.some((pattern) => pattern.test(path)) ? GO_API : API
}

export async function apiFetch(path: string, opts?: RequestInit) {
  const base = getApiBase(path, opts?.method || 'GET')
  const res = await fetch(base + path, {
    headers: { 'Content-Type': 'application/json' },
    ...opts,
  })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}
