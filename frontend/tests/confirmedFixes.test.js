import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { runInNewContext } from 'node:vm'

test('server state can restore an active question after missing presentation', () => {
  const source = readFileSync(new URL('../src/composables/useQuizStateMachine.js', import.meta.url), 'utf8')
    .replace(/^import .*$/gm, '').replace('export function', 'function')
  const machine = runInNewContext(`${source}\nuseQuizStateMachine()`, { ref: value => ({ value }), console })
  assert.equal(machine.canTransition('question-active'), false)
  machine.transition('question-active', null, { authoritative: true })
  assert.equal(machine.state.value, 'question-active')
  machine.transition('question-presenting', null, { authoritative: true })
  assert.equal(machine.state.value, 'question-presenting')
})

test('player ID generation works without randomUUID', () => {
  const source = readFileSync(new URL('../src/utils/index.js', import.meta.url), 'utf8')
    .replace(/^import .*$/gm, '').replace("import.meta.env.VITE_API_URL", "''")
    .replaceAll('export const ', 'globalThis.')
  const saved = new Map()
  const context = { session: {}, signOut() {}, localStorage: { getItem: key => saved.get(key), setItem: (key, value) => saved.set(key, value) }, crypto: { getRandomValues: bytes => bytes.fill(42) } }
  runInNewContext(source, context)
  const id = context.getUserId()
  assert.match(id, /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/)
  assert.equal(context.getUserId(), id)
})

test('player identity and event proof survive disabled storage for the page', () => {
  const source = readFileSync(new URL('../src/utils/index.js', import.meta.url), 'utf8')
    .replace(/^import .*$/gm, '').replace("import.meta.env.VITE_API_URL", "''")
    .replaceAll('export const ', 'globalThis.')
  const context = { session: {}, signOut() {}, localStorage: {
    getItem() { throw new Error('disabled') }, setItem() { throw new Error('disabled') },
  }, crypto: { getRandomValues: bytes => bytes.fill(42) } }
  runInNewContext(source, context)
  const id = context.getUserId()
  assert.equal(context.getUserId(), id)
  context.setParticipantSecret('event', 'proof')
  assert.equal(context.getParticipantSecret('event'), 'proof')
})

test('offline answers are sent after ten seconds so the server can decide', () => {
  const source = readFileSync(new URL('../src/utils/signalQueue.js', import.meta.url), 'utf8')
    .replace('export const ', 'globalThis.')
  const context = { console, Date }
  runInNewContext(source, context)
  const sent = []
  context.signalQueue.setSender(answer => sent.push(answer))
  context.signalQueue.onDisconnect()
  context.signalQueue.enqueue({ questionIndex: 0, timestamp: Date.now() - 30_000 })
  context.signalQueue.onReconnect()
  assert.equal(sent.length, 1)
  assert.equal(sent[0].questionIndex, 0)
})
