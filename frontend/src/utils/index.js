export const API_BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

export const getUserId = () => {
  return localStorage.getItem('userId') || 'unknown_user'
}
