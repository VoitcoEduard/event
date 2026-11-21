import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import QuizPlayerScreen from '../src/views/QuizPlayerScreen.vue'

const { wsHandlers, mockQuizWS, mockSignalQueue } = vi.hoisted(() => {
  const wsHandlers = new Map()
  const mockQuizWS = {
    userId: 'p1',
    on: vi.fn((type, fn) => {
      if (!wsHandlers.has(type)) wsHandlers.set(type, [])
      wsHandlers.get(type).push(fn)
    }),
    emit(type, payload) {
      ;(wsHandlers.get(type) || []).slice().forEach(fn => fn(payload))
    },
    send: vi.fn(),
    connect: vi.fn(),
    disconnect: vi.fn(),
    clearHandlers: vi.fn(() => wsHandlers.clear()),
  }
  const mockSignalQueue = {
    enqueue: vi.fn(),
    clear: vi.fn(),
    settle: vi.fn(),
  }
  return { wsHandlers, mockQuizWS, mockSignalQueue }
})

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { eventId: 'e1', moduleId: 'm1' } }),
  useRouter: () => ({ push: vi.fn() }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: k => k, locale: { value: 'en' } }),
}))

vi.mock('../src/services/websocket', () => ({ quizWS: mockQuizWS }))
vi.mock('../src/utils/signalQueue', () => ({ signalQueue: mockSignalQueue }))
vi.mock('../src/services/media', () => ({ getImageUrl: id => `/media/${id}` }))
vi.mock('../src/utils', () => ({
  getUserName: () => 'Alice',
  newUUID: () => '00000000-0000-4000-8000-000000000001',
}))

let wrapper = null
const mountOpts = {
  global: {
    mocks: { $t: k => k, $i18n: { locale: 'en' } },
    stubs: {
      AppIcon: true,
      RangeComponent: true,
      HorizontalCarousel: true,
      MapCoordinatePicker: true,
      WrongPictureAnswerReview: true,
    },
  },
}

beforeEach(() => {
  wsHandlers.clear()
  mockQuizWS.connect.mockClear()
  mockQuizWS.disconnect.mockClear()
  mockQuizWS.send.mockClear()
  mockSignalQueue.clear.mockClear()
})

afterEach(() => {
  if (wrapper) {
    wrapper.unmount()
    wrapper = null
  }
})

describe('QuizPlayerScreen reset handling', () => {
  it('leaves lobby joined on normal lobby cmd:state without reset flag', async () => {
    wrapper = mount(QuizPlayerScreen, mountOpts)
    await flushPromises()

    expect(mockQuizWS.connect).toHaveBeenCalledTimes(1)
    mockQuizWS.emit('cmd:state', { state: 'lobby', currentIndex: 0, runId: '' })
    await flushPromises()

    expect(mockQuizWS.disconnect).not.toHaveBeenCalled()
    expect(wrapper.find('.lobby-container').exists()).toBe(true)
  })

  it('disconnects and returns to join screen when cmd:state carries reset: true', async () => {
    wrapper = mount(QuizPlayerScreen, mountOpts)
    await flushPromises()

    mockQuizWS.emit('cmd:state', { state: 'question-active', currentIndex: 0, runId: 'run-1', timeLimit: 20 })
    await flushPromises()
    expect(wrapper.find('.lobby-container').exists()).toBe(false)

    mockQuizWS.emit('cmd:state', { state: 'lobby', currentIndex: 0, runId: '', reset: true })
    await flushPromises()

    expect(mockQuizWS.disconnect).toHaveBeenCalledTimes(1)
    expect(wrapper.find('.name-entry-container').exists()).toBe(true)
    expect(wrapper.find('.lobby-container').exists()).toBe(false)
  })
})
