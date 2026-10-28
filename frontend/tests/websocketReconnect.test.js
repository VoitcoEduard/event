import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { runInNewContext, createContext } from 'node:vm'

function setup() {
  const sent = [], sockets = [], timers = new Map()
  let nextTimer = 0
  class WebSocket {
    static OPEN = 1
    constructor() { this.readyState = WebSocket.OPEN; sockets.push(this) }
    send(raw) { sent.push(JSON.parse(raw)) }
    close() { this.readyState = 3 }
  }
  const context = createContext({
    WebSocket, Date, console, API_BASE_URL: '', getUserId: () => 'player',
    window: { location: { protocol: 'http:', host: 'localhost' } },
    setTimeout: fn => { const id = ++nextTimer; timers.set(id, fn); return id },
    clearTimeout: id => timers.delete(id),
  })
  for (const path of ['../src/utils/signalQueue.js', '../src/services/websocket.js']) {
    const source = readFileSync(new URL(path, import.meta.url), 'utf8')
      .replace(/^import .*$/gm, '').replaceAll('export const ', 'globalThis.')
    runInNewContext(source, context)
  }
  return { ...context, sockets, sent, timers }
}

test('queued answers are sent only after joining on reconnect', () => {
  const { quizWS, signalQueue, sockets, sent } = setup()
  quizWS.connect('m', 'Player')
  sockets[0].onopen()
  sockets[0].onclose({ code: 1006 })
  signalQueue.enqueue({ questionIndex: 0, scoreEarned: 100, timestamp: Date.now() })
  quizWS.connect('m', 'Player')
  sockets[1].onopen()
  assert.deepEqual(sent.map(m => m.type), ['player:join', 'player:join', 'player:signal'])
  assert.equal(signalQueue.queue.length, 0)
})

test('disconnect cancels pending retries and discards queued answers', () => {
  const { quizWS, signalQueue, sockets, timers } = setup()
  quizWS.connect('m', 'Player')
  sockets[0].onopen()
  sockets[0].onclose({ code: 1006 })
  signalQueue.enqueue({ timestamp: Date.now() })
  const retry = [...timers.values()][0]
  quizWS.disconnect()
  assert.equal(timers.size, 0)
  assert.equal(signalQueue.queue.length, 0)
  retry() // Also guard against a callback that was already dispatched.
  assert.equal(sockets.length, 1)
})

test('a pending retry cannot replace a newer connection', () => {
  const { quizWS, sockets, timers } = setup()
  quizWS.connect('old', 'Player')
  sockets[0].onopen()
  sockets[0].onclose({ code: 1006 })
  const retry = [...timers.values()][0]
  quizWS.connect('new', 'Player')
  retry()
  assert.equal(sockets.length, 2)
  assert.equal(quizWS.moduleId, 'new')
})
