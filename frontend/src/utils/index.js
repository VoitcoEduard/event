import { session, signOut } from '../session'

export const API_BASE_URL = import.meta.env.VITE_API_URL ?? ''

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
