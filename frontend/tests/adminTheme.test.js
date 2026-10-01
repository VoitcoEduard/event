import test from 'node:test'
import assert from 'node:assert/strict'
import { ADMIN_THEME_KEY, ADMIN_THEMES, normalizeAdminTheme, readAdminTheme } from '../src/composables/useAdminTheme.js'

test('admin theme accepts only supported values', () => {
  assert.deepEqual(ADMIN_THEMES, ['light', 'dark'])
  assert.equal(normalizeAdminTheme('light'), 'light')
  assert.equal(normalizeAdminTheme('dark'), 'dark')
})

test('admin theme defaults invalid and absent values to light', () => {
  assert.equal(normalizeAdminTheme(null), 'light')
  assert.equal(normalizeAdminTheme('system'), 'light')
  assert.equal(normalizeAdminTheme(''), 'light')
})

test('admin theme safely falls back when browser storage is unavailable', () => {
  const unavailableStorage = { getItem: () => { throw new Error('blocked') } }
  assert.equal(readAdminTheme(unavailableStorage), 'light')
  assert.equal(readAdminTheme({ getItem: key => key === ADMIN_THEME_KEY ? 'dark' : null }), 'dark')
})
