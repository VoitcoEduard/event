import { reactive } from 'vue'

export const session = reactive({ loaded: false, user: null, csrfToken: '', expiresAt: null, mode: null })

export async function loadSession(force = false) {
  if (typeof localStorage !== 'undefined') localStorage.removeItem('adminToken')
  if (session.loaded && !force) return session.user
  const response = await fetch('/auth/me', { credentials: 'same-origin', cache: 'no-store' })
  session.loaded = true
  if (response.status === 401) {
    session.user = null
    session.csrfToken = ''
    session.expiresAt = null
    return null
  }
  if (!response.ok) {
    session.user = null
    session.csrfToken = ''
    session.expiresAt = null
    throw new Error('Session unavailable')
  }
  const data = await response.json()
  session.user = data.user
  session.csrfToken = data.csrfToken
  session.expiresAt = data.expiresAt
  return session.user
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
