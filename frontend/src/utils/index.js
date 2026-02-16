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
