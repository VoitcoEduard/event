import test from 'node:test'
import assert from 'node:assert/strict'
import { session, loadSession, loadAuthMode, signOut, listAdminUsers, setAdminStatus } from '../src/session.js'

test('restores the server session and CSRF token', async () => {
  session.loaded = false
  const original = global.fetch
  global.fetch = async (url) => {
    assert.equal(url, '/auth/me')
    return new Response(JSON.stringify({ user: { id: 'user-1', role: 'owner', status: 'active' }, csrfToken: 'csrf-1', expiresAt: '2030-01-01T00:00:00Z' }), { status: 200 })
  }
  try {
    const user = await loadSession(true)
    assert.equal(user.role, 'owner')
    assert.equal(session.csrfToken, 'csrf-1')
  } finally { global.fetch = original }
})

test('expired sessions clear local state and logout sends CSRF', async () => {
  const original = global.fetch
  global.fetch = async () => new Response('', { status: 401 })
  try {
    assert.equal(await loadSession(true), null)
    assert.equal(session.csrfToken, '')
  } finally { global.fetch = original }

  session.csrfToken = 'logout-csrf'
  session.user = { id: 'user-1' }
  global.fetch = async (url, options) => {
    assert.equal(url, '/auth/logout')
    assert.equal(options.headers['X-CSRF-Token'], 'logout-csrf')
    return new Response(null, { status: 204 })
  }
  try {
    await signOut()
    assert.equal(session.user, null)
    assert.equal(session.csrfToken, '')
  } finally { global.fetch = original }
})

test('login mode is loaded from backend configuration', async () => {
  session.mode = null
  const original = global.fetch
  global.fetch = async (url) => {
    assert.equal(url, '/auth/config')
    return new Response(JSON.stringify({ mode: 'google' }), { status: 200 })
  }
  try { assert.equal(await loadAuthMode(), 'google') }
  finally { global.fetch = original }
})

test('invalid auth configuration fails without caching a mode and can be retried', async () => {
  session.mode = null
  const original = global.fetch
  try {
    for (const data of [{ message: 'Hello from Go Backend!' }, { mode: 'unknown' }, null]) {
      global.fetch = async () => new Response(JSON.stringify(data), { status: 200 })
      await assert.rejects(loadAuthMode(), /Authentication configuration unavailable/)
      assert.equal(session.mode, null)
    }
    global.fetch = async () => new Response(JSON.stringify({ mode: 'dev' }), { status: 200 })
    assert.equal(await loadAuthMode(), 'dev')
  } finally {
    global.fetch = original
    session.mode = null
  }
})

test('pending account becomes active after owner approval', async () => {
  const original = global.fetch
  let status = 'pending'
  global.fetch = async () => new Response(JSON.stringify({ user: { id: 'user-2', role: 'admin', status }, csrfToken: 'csrf-2' }), { status: 200 })
  try {
    assert.equal((await loadSession(true)).status, 'pending')
    status = 'active'
    assert.equal((await loadSession(true)).status, 'active')
    assert.equal(session.user.id, 'user-2')
  } finally { global.fetch = original }
})

test('owner management uses session CSRF and updates only the selected user', async () => {
  const original = global.fetch
  session.csrfToken = 'owner-csrf'
  global.fetch = async (url, options) => {
    if (url === '/auth/users?page=0') {
      assert.equal(options.cache, 'no-store')
      return new Response(JSON.stringify({ users: [{ id: 'user-2', status: 'pending' }], hasMore: false }), { status: 200 })
    }
    assert.equal(url, '/auth/users/user-2')
    assert.equal(options.method, 'PATCH')
    assert.equal(options.headers['X-CSRF-Token'], 'owner-csrf')
    assert.deepEqual(JSON.parse(options.body), { status: 'active' })
    return new Response(null, { status: 204 })
  }
  try {
    const result = await listAdminUsers()
    assert.equal(result.users[0].status, 'pending')
    await setAdminStatus(result.users[0].id, 'active')
  } finally { global.fetch = original }
})
