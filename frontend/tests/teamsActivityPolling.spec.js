import { it, expect, vi, afterEach } from 'vitest'
import { defineComponent } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import { useTeamsActivity } from '../src/composables/useTeamsActivity'
import { teamsActivity } from '../src/services/teamsActivity'
let wrapper
const sources = []
class TestEventSource {
  constructor() { this.listeners = {}; this.closed = false; sources.push(this) }
  addEventListener(name, callback) { this.listeners[name] = callback }
  close() { this.closed = true }
}
afterEach(() => { wrapper?.unmount(); wrapper = null; sources.length = 0; vi.unstubAllGlobals() })
function setup() {
  vi.stubGlobal('localStorage', { getItem: () => 'device', setItem: () => {} })
  vi.stubGlobal('EventSource', TestEventSource)
}
it('an update arriving during an in-flight fetch triggers a fresh read, and streams close on unmount', async () => {
  setup()
  let completeFirst
  const first = new Promise(resolve => { completeFirst = resolve })
  const fetch = vi.fn().mockReturnValueOnce(first).mockResolvedValue(new Response(JSON.stringify({ initialized: true, name: 'Updated', revision: 2 })))
  vi.stubGlobal('fetch', fetch)
  wrapper = mount(defineComponent({ setup: () => useTeamsActivity('e', 'm'), template: '<p>{{ state?.name }}</p>' }))
  sources[0].onmessage({ data: '{"revision":2}' })
  completeFirst(new Response(JSON.stringify({ initialized: true, name: 'Old', revision: 1 })))
  await flushPromises()
  expect(fetch).toHaveBeenCalledTimes(2)
  expect(wrapper.text()).toBe('Updated')
  const source = sources[0]; wrapper.unmount(); wrapper = null
  expect(source.closed).toBe(true)
})
it('a lost award response retries with the original request ID', async () => {
  setup()
  vi.stubGlobal('crypto', { getRandomValues: bytes => { bytes.fill(7); return bytes } })
  const fetch = vi.fn().mockRejectedValueOnce(new TypeError('Disconnected')).mockResolvedValueOnce(new Response(JSON.stringify({ revision: 2 })))
  vi.stubGlobal('fetch', fetch)
  await teamsActivity.action('e', 'm', { action: 'award', points: 20, teamId: 'blue' })
  expect(fetch).toHaveBeenCalledTimes(2)
  const first = JSON.parse(fetch.mock.calls[0][1].body), second = JSON.parse(fetch.mock.calls[1][1].body)
  expect(first.requestId).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/); expect(second).toEqual(first)
})
