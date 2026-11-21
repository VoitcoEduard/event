import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ProjectorView from '../src/views/ProjectorView.vue'

const { wsHandlers, mockQuizWS } = vi.hoisted(() => {
  const wsHandlers = new Map()
  const mockQuizWS = {
    isConnected: false,
    on: vi.fn((type, fn) => {
      if (!wsHandlers.has(type)) wsHandlers.set(type, [])
      wsHandlers.get(type).push(fn)
    }),
    off: vi.fn((type, fn) => {
      const list = wsHandlers.get(type)
      if (!list) return
      const idx = list.indexOf(fn)
      if (idx !== -1) list.splice(idx, 1)
    }),
    emit(type, payload) {
      ;(wsHandlers.get(type) || []).slice().forEach(fn => fn(payload))
    },
    send: vi.fn(),
    connect: vi.fn(() => {
      mockQuizWS.isConnected = true
    }),
    disconnect: vi.fn(() => {
      mockQuizWS.isConnected = false
    }),
    clearHandlers: vi.fn(() => wsHandlers.clear()),
  }
  return { wsHandlers, mockQuizWS }
})

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { eventId: 'e1' } }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: k => k, locale: { value: 'en' } }),
}))

vi.mock('../src/services/websocket', () => ({ quizWS: mockQuizWS }))
vi.mock('../src/services/media', () => ({ getImageUrl: id => `/media/${id}` }))
vi.mock('../src/utils', () => ({ API_BASE_URL: '' }))
vi.mock('../src/utils/imagePreload', () => ({
  preloader: {
    preloadQuestions: vi.fn(),
    preloadQuestion: vi.fn(),
    clear: vi.fn(),
  },
}))

let wrapper = null
const mountOpts = {
  global: {
    mocks: { $t: k => k, $i18n: { locale: 'en' } },
    stubs: {
      AppIcon: true,
      QRCodeVue: true,
      WrongPictureTargetCrop: true,
      SimpleQuizProjector: true,
      TeamsProjector: true,
    },
  },
}

const mockEventFetch = () => {
  vi.stubGlobal('fetch', vi.fn((url) => {
    if (url === '/events/e1') {
      return Promise.resolve({
        ok: true,
        json: async () => ({
          id: 'e1',
          pin: '1234',
          modules: [{ id: 'm1', type: 'quiz', isActive: true, data: {} }],
        }),
      })
    }
    return Promise.resolve({ ok: false })
  }))
}

beforeEach(() => {
  vi.useFakeTimers()
  wsHandlers.clear()
  mockQuizWS.isConnected = false
  mockQuizWS.on.mockClear()
  mockQuizWS.connect.mockClear()
  mockQuizWS.disconnect.mockClear()
  mockQuizWS.clearHandlers.mockClear()
  mockEventFetch()
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('ProjectorView answer counter (Bug #4)', () => {
  it('uses sync:minimal answeredCount for range/multi_select/wrong_pictures, ignores stale questionIndex, and resets on next question', async () => {
    wrapper = mount(ProjectorView, mountOpts)
    await flushPromises()

    mockQuizWS.emit('data:batch', {
      startIndex: 0,
      totalQuestions: 2,
      questions: [{ id: 'q1', type: 'range', text: 'Range Q', rangeConfig: { minValue: 0, maxValue: 100 } }],
    })
    mockQuizWS.emit('cmd:state', { state: 'question-active', currentIndex: 0, totalQuestions: 2 })
    await flushPromises()

    expect(wrapper.find('.answer-counter-widget .counter-value').text()).toBe('0')

    // Range question has no answerCounts, only answeredCount
    mockQuizWS.emit('sync:minimal', { questionIndex: 0, answeredCount: 3, answerCounts: {} })
    await flushPromises()
    expect(wrapper.find('.answer-counter-widget .counter-value').text()).toBe('3')

    // Stale sync:minimal for another question is ignored
    mockQuizWS.emit('sync:minimal', { questionIndex: 99, answeredCount: 1, answerCounts: {} })
    await flushPromises()
    expect(wrapper.find('.answer-counter-widget .counter-value').text()).toBe('3')

    // Moving to next question resets counter to 0 even if multi_select has higher option sums
    mockQuizWS.emit('data:batch', {
      startIndex: 1,
      totalQuestions: 2,
      questions: [{ id: 'q2', type: 'multi_select', text: 'Multi Q', answers: ['A', 'B', 'C', 'D'] }],
    })
    mockQuizWS.emit('cmd:state', { state: 'question-presenting', currentIndex: 1, timeLimit: 5, totalQuestions: 2 })
    await flushPromises()
    mockQuizWS.emit('cmd:state', { state: 'question-active', currentIndex: 1, totalQuestions: 2 })
    await flushPromises()
    expect(wrapper.find('.answer-counter-widget .counter-value').text()).toBe('0')

    mockQuizWS.emit('sync:minimal', {
      questionIndex: 1,
      answeredCount: 2,
      answerCounts: { '0': 2, '1': 2, '2': 1, '3': 0 },
    })
    await flushPromises()
    expect(wrapper.find('.answer-counter-widget .counter-value').text()).toBe('2')
  })
})

describe('ProjectorView reconnect stability (Bug #5)', () => {
  it('does not wipe state or reconnect every 3s during a brief WebSocket disconnect for the same active module', async () => {
    wrapper = mount(ProjectorView, mountOpts)
    await flushPromises()

    expect(mockQuizWS.connect).toHaveBeenCalledTimes(1)

    mockQuizWS.emit('data:batch', {
      startIndex: 0,
      totalQuestions: 2,
      questions: [{ id: 'q1', type: 'classic', text: 'Persisted Q1', answers: ['A', 'B'] }],
    })
    mockQuizWS.emit('cmd:state', { state: 'question-active', currentIndex: 0, totalQuestions: 2 })
    await flushPromises()
    expect(wrapper.text()).toContain('Persisted Q1')

    mockQuizWS.connect.mockClear()
    mockQuizWS.disconnect.mockClear()
    mockQuizWS.clearHandlers.mockClear()

    // Brief disconnect: quizWS handles its own reconnect; 3s poll must not wipe state
    mockQuizWS.isConnected = false
    mockQuizWS.emit('disconnect')
    await flushPromises()

    await vi.advanceTimersByTimeAsync(6500)
    await flushPromises()

    expect(mockQuizWS.disconnect).not.toHaveBeenCalled()
    expect(mockQuizWS.clearHandlers).not.toHaveBeenCalled()
    expect(mockQuizWS.connect).not.toHaveBeenCalled()

    // Once quizWS reconnects and emits cmd:state, question state is still intact
    mockQuizWS.isConnected = true
    mockQuizWS.emit('cmd:state', { state: 'question-active', currentIndex: 0, totalQuestions: 2 })
    await flushPromises()
    expect(wrapper.text()).toContain('Persisted Q1')
  })
})
