import { API_BASE_URL, getAuthHeaders, getUserId } from '../utils'

export const teamsBase = (eventId, moduleId) => `${API_BASE_URL}/events/${encodeURIComponent(eventId)}/modules/${encodeURIComponent(moduleId)}/teams`
async function request(eventId, moduleId, path, body, admin = false) {
  const response = await fetch(`${teamsBase(eventId, moduleId)}/${path}`, {
    method: body === undefined ? 'GET' : 'POST',
    credentials: 'same-origin',
    headers: { ...(body === undefined ? {} : { 'Content-Type': 'application/json' }), ...(admin ? getAuthHeaders() : { 'X-Teams-Player': getUserId() }) },
    ...(body === undefined ? {} : { body: JSON.stringify(body) })
  })
  if (!response.ok) {
    const error = new Error((await response.text()).trim() || `HTTP ${response.status}`)
    error.status = response.status
    throw error
  }
  return response.json()
}
export const teamsActivity = {
  setVisibility: async (eventId, moduleId, isActive) => {
    const response = await fetch(`${API_BASE_URL}/events/${encodeURIComponent(eventId)}/modules/${encodeURIComponent(moduleId)}/status`, {
      method: 'PUT',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify({ isActive })
    })
    if (!response.ok) throw new Error((await response.text()).trim() || `HTTP ${response.status}`)
  },
  state: (e, m) => request(e, m, 'state'),
  projector: async (e, m) => {
    const response = await fetch(`${teamsBase(e, m)}/projector`)
    if (!response.ok) throw new Error(await response.text())
    return response.json()
  },
  admin: (e, m) => request(e, m, 'admin', undefined, true),
  join: (e, m, name) => request(e, m, 'join', { name }),
  choose: (e, m, teamId) => request(e, m, 'choose', { teamId }),
  action: async (e, m, body) => {
    // Keep the same request ID through ambiguous network failures. Caller may
    // retain this payload for a later retry, without issuing the award twice.
    if (['award', 'undo'].includes(body.action) && !body.requestId) body.requestId = crypto.randomUUID()
    try { return await request(e, m, 'actions', body, true) }
    catch (error) {
      if (error.status) throw error
      return request(e, m, 'actions', body, true)
    }
  }
}
