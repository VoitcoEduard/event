import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import DataLibraryScreen from '../src/views/DataLibraryScreen.vue'
import i18n from '../src/i18n.js'

const mockRoute = reactive({ query: { eventId: 'evt-1' } })
const mockRouter = { push: vi.fn() }

vi.mock('vue-router', () => ({
  useRoute: () => mockRoute,
  useRouter: () => mockRouter,
  onBeforeRouteLeave: vi.fn(),
  onBeforeRouteUpdate: vi.fn(),
}))

vi.mock('../src/utils', () => ({
  API_BASE_URL: '/api',
  getAuthHeaders: () => ({}),
}))

let wrapper = null
const mountOpts = {
  global: {
    plugins: [i18n],
    stubs: {
      AppIcon: true,
      'router-link': true,
      AdminLayout: { template: '<div><slot name="actions" /><slot /></div>' },
    },
  },
}

beforeEach(() => {
  i18n.global.locale.value = 'en'
  mockRoute.query = { eventId: 'evt-1' }
  mockRouter.push.mockClear()
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  vi.unstubAllGlobals()
})

describe('DataLibraryScreen simple_quiz score handling (Bug #14)', () => {
  it('renders simple_quiz latest game score as read-only and saves manual points without calling quiz-scores', async () => {
    const fetchCalls = []
    const eventData = {
      id: 'evt-1',
      name: 'Quiz Event',
      modules: [
        { id: 'mod-sq', name: 'Simple Round', type: 'simple_quiz', data: {} },
        { id: 'mod-q', name: 'Classic Quiz', type: 'quiz', data: {} },
      ],
    }
    const participantsData = [{ id: 'p1', displayName: 'Alice' }]
    const scoresData = {
      participants: participantsData,
      teams: [],
      games: [
        { moduleId: 'mod-sq', players: [{ id: 'p1', nickname: 'Alice', score: 30 }] },
        { moduleId: 'mod-q', players: [{ id: 'p1', nickname: 'Alice', score: 50 }] },
      ],
      entries: [],
    }

    vi.stubGlobal('fetch', vi.fn((url, opts = {}) => {
      fetchCalls.push({ url, method: opts.method || 'GET', body: opts.body })
      if (url === '/api/events/evt-1') return Promise.resolve({ ok: true, status: 200, json: async () => eventData })
      if (url === '/api/events/evt-1/participants') return Promise.resolve({ ok: true, status: 200, json: async () => participantsData })
      if (url === '/api/events/evt-1/teams') return Promise.resolve({ ok: true, status: 200, json: async () => [] })
      if (url === '/api/events/evt-1/scores') return Promise.resolve({ ok: true, status: 200, json: async () => scoresData })
      if (opts.method === 'PUT') return Promise.resolve({ ok: true, status: 200, json: async () => ({ message: 'ok' }) })
      return Promise.resolve({ ok: false, status: 404, text: async () => 'Not found' })
    }))

    wrapper = mount(DataLibraryScreen, mountOpts)
    await flushPromises()

    // Select the Simple Quiz module tab
    const navButtons = wrapper.findAll('nav button')
    const simpleQuizTab = navButtons.find(b => b.text() === 'Simple Round')
    await simpleQuizTab.trigger('click')
    await flushPromises()

    // Published score is shown, but NOT as an editable game points input
    expect(wrapper.find('input[aria-label="Alice game points"]').exists()).toBe(false)
    expect(wrapper.find('tbody').text()).toContain('30')

    // Manual points column remains editable and saves cleanly without hitting quiz-scores
    const manualInput = wrapper.find('input[aria-label="Alice manual points"]')
    expect(manualInput.exists()).toBe(true)
    await manualInput.setValue('5')
    const saveBtn = wrapper.find('.actions-cell button')
    await saveBtn.trigger('click')
    await flushPromises()

    expect(fetchCalls.some(c => c.url.includes('/quiz-scores/'))).toBe(false)
    expect(fetchCalls.some(c => c.method === 'PUT' && c.url === '/api/events/evt-1/modules/mod-sq/manual-scores/players/p1')).toBe(true)

    // Classic Quiz tab still allows editing the latest game score
    const classicQuizTab = navButtons.find(b => b.text() === 'Classic Quiz')
    await classicQuizTab.trigger('click')
    await flushPromises()

    const gameInput = wrapper.find('input[aria-label="Alice game points"]')
    expect(gameInput.exists()).toBe(true)
    await gameInput.setValue('75')
    await wrapper.find('.actions-cell button').trigger('click')
    await flushPromises()

    expect(fetchCalls.some(c => c.method === 'PUT' && c.url === '/api/events/evt-1/modules/mod-q/quiz-scores/p1')).toBe(true)
  })
})
