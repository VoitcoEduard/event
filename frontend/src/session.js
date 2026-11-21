import { reactive } from 'vue'
import { safeLocalRemove } from './utils/storage.js'

export const session = reactive({ loaded: false, user: null, csrfToken: '', expiresAt: null, mode: null })
let requestVersion = 0
let pendingLoad = null

export function loadSession(force = false) {
  safeLocalRemove('adminToken')
  if (session.loaded && !force) return Promise.resolve(session.user)
  if (pendingLoad && !force) return pendingLoad
  const version = ++requestVersion
  const load = (async () => {
    const response = await fetch('/auth/me', { credentials: 'same-origin', cache: 'no-store' })
    if (response.status === 401) {
      if (version === requestVersion) {
        session.loaded = true
        session.user = null
        session.csrfToken = ''
        session.expiresAt = null
      }
      return session.user
    }
    if (!response.ok) throw new Error('Session unavailable')
    const data = await response.json()
    if (!data?.user || typeof data.user !== 'object') throw new Error('Session unavailable')
    if (version === requestVersion) {
      session.loaded = true
      session.user = data.user
      session.csrfToken = data.csrfToken
      session.expiresAt = data.expiresAt
    }
    return session.user
  })()
  pendingLoad = load
  void load.finally(() => { if (pendingLoad === load) pendingLoad = null }).catch(() => {})
  return load
}

export async function loadAuthMode() {
  if (session.mode) return session.mode
  const response = await fetch('/auth/config', { cache: 'no-store' })
  if (!response.ok) throw new Error('Authentication configuration unavailable')
  const data = await response.json()
  if (data?.mode !== 'dev' && data?.mode !== 'google') {
    throw new Error('Authentication configuration unavailable')
  }
  session.mode = data.mode
  return session.mode
}

export async function signOut() {
  const response = await fetch('/auth/logout', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'X-CSRF-Token': session.csrfToken }
  })
  if (!response.ok && response.status !== 401) throw new Error('Logout failed')
  ++requestVersion
  session.loaded = true
  session.user = null
  session.csrfToken = ''
  session.expiresAt = null
}

export async function listAdminUsers(page = 0) {
  const response = await fetch(`/auth/users?page=${page}`, { credentials: 'same-origin', cache: 'no-store' })
  if (!response.ok) throw new Error('Users unavailable')
  return response.json()
}

export async function setAdminStatus(id, status) {
  const response = await fetch(`/auth/users/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': session.csrfToken },
    body: JSON.stringify({ status })
  })
  if (!response.ok) throw new Error('User update failed')
}
