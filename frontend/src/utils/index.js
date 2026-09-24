import { session, signOut } from '../session'

export const API_BASE_URL = import.meta.env.VITE_API_URL ?? ''

// Some API responses can be empty or wrapped, so normalize list payloads before
// components call array methods on them.
export const normalizeTemplates = (payload) => {
  if (Array.isArray(payload)) return payload
  if (payload == null) return []
  if (Array.isArray(payload?.templates)) return payload.templates
  if (Array.isArray(payload?.data)) return payload.data
  throw new TypeError('Unexpected /templates response: expected an array')
}

export const getUserId = () => {
  let userId = localStorage.getItem('userId')
  if (!userId) {
    userId = crypto.randomUUID()
    localStorage.setItem('userId', userId)
  }
  return userId
}

export const getUserName = () => {
  return localStorage.getItem('userName') || ''
}

export const setUserName = (name) => {
  localStorage.setItem('userName', name)
}

// --- Auth helpers ---

export const getAuthHeaders = () => {
	return session.csrfToken ? { 'X-CSRF-Token': session.csrfToken } : {}
}

export const isAuthenticated = () => {
	return session.user?.status === 'active'
}

export const logout = async (router) => {
	await signOut()
	if (router) router.push('/login')
}
