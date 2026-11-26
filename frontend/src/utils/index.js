import { session, signOut } from '../session'

export const API_BASE_URL = import.meta.env.VITE_API_URL ?? ''

export const normalizeTemplates = (payload) => {
  if (Array.isArray(payload)) return payload
  if (payload == null) return []
  if (Array.isArray(payload?.templates)) return payload.templates
  if (Array.isArray(payload?.data)) return payload.data
  throw new TypeError('Unexpected /templates response: expected an array')
}

let temporaryUserId = ''

export const setUserId = (id) => {
  temporaryUserId = id
  try { sessionStorage.setItem('userId', id) } catch {  }
}

export const getUserId = () => {
  let userId = ''
  try { userId = sessionStorage.getItem('userId') || '' } catch {  }
  if (userId) return userId
  if (!temporaryUserId) setUserId(newUUID())
  return temporaryUserId
}

export const newUUID = () => {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) return crypto.randomUUID()
  const bytes = new Uint8Array(16)
  crypto.getRandomValues(bytes)
  bytes[6] = (bytes[6] & 0x0f) | 0x40
  bytes[8] = (bytes[8] & 0x3f) | 0x80
  const hex = Array.from(bytes, b => b.toString(16).padStart(2, '0')).join('')
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
}

const temporaryParticipantSecrets = new Map()

export const setParticipantSecret = (eventId, secret) => {
  temporaryParticipantSecrets.set(eventId, secret)
  try { sessionStorage.setItem(`participantSecret:${eventId}`, secret) } catch {  }
}

export const getParticipantSecret = (eventId) => {
  try { return sessionStorage.getItem(`participantSecret:${eventId}`) || temporaryParticipantSecrets.get(eventId) || '' }
  catch { return temporaryParticipantSecrets.get(eventId) || '' }
}

let temporaryUserName = ''

export const getUserName = () => {
  try { return sessionStorage.getItem('userName') || temporaryUserName } catch { return temporaryUserName }
}

export const setUserName = (name) => {
  temporaryUserName = name
  try { sessionStorage.setItem('userName', name) } catch {  }
}

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
