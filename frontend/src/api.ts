import type { ApiEnvelope } from './types'

function tokenStorageKey() {
  const app = new URLSearchParams(location.search).get('app') || 'passenger'
  return `ride-hailing-token:${app}`
}

export function getAccessToken() {
  return localStorage.getItem(tokenStorageKey()) || ''
}

export function hasAccessToken() {
  return getAccessToken() !== ''
}

export function setAccessToken(token: string) {
  localStorage.setItem(tokenStorageKey(), token)
}

export function clearAccessToken() {
  localStorage.removeItem(tokenStorageKey())
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const headers = new Headers(options?.headers || {})
  headers.set('Content-Type', 'application/json')
  const token = getAccessToken()
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  const response = await fetch(path, {
    ...options,
    headers
  })
  const payload = await response.json() as ApiEnvelope<T>
  if (!response.ok) {
    throw new Error(payload.error || '请求失败')
  }
  return payload.data
}

export function apiGet<T>(path: string): Promise<T> {
  return request<T>(path)
}

export function apiPost<T>(path: string, body?: unknown): Promise<T> {
  return request<T>(path, { method: 'POST', body: JSON.stringify(body || {}) })
}
