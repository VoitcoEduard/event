export const API_BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

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
  const token = localStorage.getItem('adminToken')
  return token ? { Authorization: `Bearer ${token}` } : {}
}

export const isAuthenticated = () => {
  return !!localStorage.getItem('adminToken')
}

export const logout = (router) => {
  localStorage.removeItem('adminToken')
  if (router) router.push('/login')
}
