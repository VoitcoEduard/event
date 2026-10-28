import { API_BASE_URL, getAuthHeaders } from '../utils'

const base = (eventId, moduleId) => `${API_BASE_URL}/events/${eventId}/modules/${moduleId}/simple-quiz`

async function req(path, opts = {}) {
  const res = await fetch(path, opts)
  const text = await res.text()
  let data = null
  try {
    data = text ? JSON.parse(text) : null
  } catch {
    data = { raw: text }
  }
  if (!res.ok) {
    const msg = data?.message || text || `Request failed (${res.status})`
    throw new Error(typeof msg === 'string' ? msg : `Request failed (${res.status})`)
  }
  return data
}

export const simpleQuizApi = {
  launch(eventId, moduleId) {
    return req(`${base(eventId, moduleId)}/launch`, { method: 'POST', headers: { ...getAuthHeaders() } })
  },
  open(eventId, moduleId, questionIndex) {
    return req(`${base(eventId, moduleId)}/open`, {
      method: 'POST', headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify(questionIndex !== undefined ? { questionIndex } : {}),
    })
  },
  lock(eventId, moduleId) {
    return req(`${base(eventId, moduleId)}/lock`, { method: 'POST', headers: { ...getAuthHeaders() } })
  },
  next(eventId, moduleId) {
    return req(`${base(eventId, moduleId)}/next`, { method: 'POST', headers: { ...getAuthHeaders() } })
  },
  endAnswering(eventId, moduleId) {
    return req(`${base(eventId, moduleId)}/end-answering`, { method: 'POST', headers: { ...getAuthHeaders() } })
  },
  resume(eventId, moduleId) {
    return req(`${base(eventId, moduleId)}/resume`, { method: 'POST', headers: { ...getAuthHeaders() } })
  },
  reveal(eventId, moduleId, questionIndex) {
    return req(`${base(eventId, moduleId)}/reveal`, {
      method: 'POST', headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify(questionIndex !== undefined ? { questionIndex } : {}),
    })
  },
  publish(eventId, moduleId) {
    return req(`${base(eventId, moduleId)}/publish`, { method: 'POST', headers: { ...getAuthHeaders() } })
  },
  updatePublished(eventId, moduleId) {
    return req(`${base(eventId, moduleId)}/update-published`, { method: 'POST', headers: { ...getAuthHeaders() } })
  },
  complete(eventId, moduleId) {
    return req(`${base(eventId, moduleId)}/complete`, { method: 'POST', headers: { ...getAuthHeaders() } })
  },
  reset(eventId, moduleId) {
    return req(`${base(eventId, moduleId)}/reset`, { method: 'POST', headers: { ...getAuthHeaders() } })
  },
  join(eventId, moduleId, nickname, secret = '') {
    return req(`${base(eventId, moduleId)}/join`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ nickname, secret }),
    })
  },
  submitAnswer(eventId, moduleId, { playerId, secret, questionIndex, text, answerId }) {
    return req(`${base(eventId, moduleId)}/answers`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Idempotency-Key': answerId },
      body: JSON.stringify({ playerId, secret, questionIndex, text, answerId }),
    })
  },
  state(eventId, moduleId, { view = 'projector', playerId = '' } = {}) {
    const qs = new URLSearchParams({ view, ...(playerId ? { playerId } : {}) })
    return req(`${base(eventId, moduleId)}/state?${qs.toString()}`)
  },
  queue(eventId, moduleId, judgeToken, { questionIndex = -1, filter = 'pending', limit = 20, cursor = '' } = {}) {
    const qs = new URLSearchParams({
      judgeToken, filter, limit: String(limit),
      ...(questionIndex >= 0 ? { questionIndex: String(questionIndex) } : {}),
      ...(cursor ? { cursor } : {}),
    })
    return req(`${base(eventId, moduleId)}/queue?${qs.toString()}`)
  },
  grade(eventId, moduleId, { answerId, verdict, judgeToken, judgeKey = 'judge' }) {
    return req(`${base(eventId, moduleId)}/grade`, {
      method: 'PUT', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ answerId, verdict, judgeToken, judgeKey }),
    })
  },
  undo(eventId, moduleId, { judgeToken, judgeKey = 'judge' }) {
    return req(`${base(eventId, moduleId)}/undo`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ judgeToken, judgeKey }),
    })
  },
  results(eventId, moduleId, { view = 'projector', playerId = '', judgeToken = '' } = {}) {
    const qs = new URLSearchParams({ view, ...(playerId ? { playerId } : {}), ...(judgeToken ? { judgeToken } : {}) })
    return req(`${base(eventId, moduleId)}/results?${qs.toString()}`)
  },
  rotateJudgeToken(eventId, moduleId) {
    return req(`${base(eventId, moduleId)}/judge-token/rotate`, { method: 'POST', headers: { ...getAuthHeaders() } })
  },
}

export function draftKey(eventId, moduleId, questionIndex) {
  return `sq-draft:${eventId}:${moduleId}:${questionIndex}`
}

export function loadDraft(eventId, moduleId, questionIndex) {
  try {
    return localStorage.getItem(draftKey(eventId, moduleId, questionIndex)) || ''
  } catch {
    return ''
  }
}

export function saveDraft(eventId, moduleId, questionIndex, text) {
  try {
    if (!text) localStorage.removeItem(draftKey(eventId, moduleId, questionIndex))
    else localStorage.setItem(draftKey(eventId, moduleId, questionIndex), text)
  } catch { /* ignore */ }
}

export function clearDraft(eventId, moduleId, questionIndex) {
  try {
    localStorage.removeItem(draftKey(eventId, moduleId, questionIndex))
  } catch { /* ignore */ }
}

export function newAnswerId() {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) return crypto.randomUUID()
  // randomUUID is unavailable on insecure origins; still send a valid UUID.
  const bytes = new Uint8Array(16)
  if (typeof crypto !== 'undefined' && crypto.getRandomValues) crypto.getRandomValues(bytes)
  else for (let i = 0; i < bytes.length; i++) bytes[i] = Math.floor(Math.random() * 256)
  bytes[6] = (bytes[6] & 0x0f) | 0x40
  bytes[8] = (bytes[8] & 0x3f) | 0x80
  const hex = Array.from(bytes, b => b.toString(16).padStart(2, '0')).join('')
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
}

export function judgeLink(origin, eventId, moduleId, token) {
  return `${origin}/simple-quiz/${eventId}/${moduleId}/judge?token=${encodeURIComponent(token)}`
}
