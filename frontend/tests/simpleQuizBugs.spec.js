import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import SimpleQuizJudgeScreen from '../src/views/SimpleQuizJudgeScreen.vue'
import SimpleQuizPlayerScreen from '../src/views/SimpleQuizPlayerScreen.vue'
import SimpleQuizHostScreen from '../src/views/SimpleQuizHostScreen.vue'
import SimpleQuizProjector from '../src/components/SimpleQuizProjector.vue'

const { wsHandlers, mockQuizWS, mockSimpleQuizApi, mockToastSuccess } = vi.hoisted(() => {
  const wsHandlers = new Map()
  const mockQuizWS = {
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
    connect: vi.fn(),
    disconnect: vi.fn(),
    clearHandlers: vi.fn(() => wsHandlers.clear()),
  }
  const mockSimpleQuizApi = {
    state: vi.fn(),
    queue: vi.fn(),
    grade: vi.fn(),
    undo: vi.fn(),
    join: vi.fn(),
    submitAnswer: vi.fn(),
    results: vi.fn(),
    open: vi.fn(),
    lock: vi.fn(),
    next: vi.fn(),
    reveal: vi.fn(),
    endAnswering: vi.fn(),
    publish: vi.fn(),
    updatePublished: vi.fn(),
    complete: vi.fn(),
    resume: vi.fn(),
    launch: vi.fn(),
    linkParticipant: vi.fn(),
  }
  const mockToastSuccess = vi.fn()
  return { wsHandlers, mockQuizWS, mockSimpleQuizApi, mockToastSuccess }
})

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { eventId: 'e1', moduleId: 'm1' }, query: { token: 'judge-tok' } }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: k => k, locale: { value: 'en' } }),
}))

vi.mock('../src/services/websocket', () => ({ quizWS: mockQuizWS }))
vi.mock('../src/services/simpleQuiz', () => ({
  simpleQuizApi: mockSimpleQuizApi,
  loadDraft: () => '',
  saveDraft: vi.fn(),
  clearDraft: vi.fn(),
  newAnswerId: () => '11111111-1111-4111-8111-111111111111',
}))
vi.mock('../src/services/media', () => ({ getImageUrl: id => `/media/${id}` }))
vi.mock('../src/composables/useToast', () => ({ useToast: () => ({ success: mockToastSuccess }) }))
vi.mock('../src/utils', () => ({
  API_BASE_URL: '',
  getAuthHeaders: () => ({}),
  getUserId: () => 'u1',
  getParticipantSecret: () => '',
  newUUID: () => 'judge-1',
}))

let wrapper = null
const mountOpts = {
  global: {
    mocks: { $t: k => k, $i18n: { locale: 'en' } },
    stubs: { AppIcon: true },
  },
}

beforeEach(() => {
  vi.useFakeTimers()
  wsHandlers.clear()
  Object.values(mockSimpleQuizApi).forEach(fn => fn.mockReset())
  mockQuizWS.on.mockClear()
  mockQuizWS.off.mockClear()
  mockQuizWS.connect.mockClear()
  mockQuizWS.disconnect.mockClear()
  mockQuizWS.clearHandlers.mockClear()
  localStorage.clear()
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('SimpleQuiz bug and race fixes', () => {
  it('SimpleQuizJudgeScreen keeps judgeNotes across WS updates, paginates >1 page, and preserves active answer on refresh', async () => {
    const qMeta = [{ index: 0, text: 'Capital?', points: 50, judgeNotes: 'Accept Pariz', correctAnswerText: 'Paris' }]
    mockSimpleQuizApi.queue
      .mockResolvedValueOnce({
        answers: [{ answerId: 'a1', questionIndex: 0, text: 'First', verdict: 'pending' }],
        nextCursor: 'a1',
        progress: [],
        gradingComplete: false,
        questions: qMeta,
        total: 1,
      })
      .mockResolvedValueOnce({
        answers: [{ answerId: 'a2', questionIndex: 0, text: 'Second', verdict: 'pending' }],
        nextCursor: '',
        progress: [],
        gradingComplete: false,
        questions: qMeta,
        total: 1,
      })
      .mockResolvedValueOnce({
        answers: [{ answerId: 'a2', questionIndex: 0, text: 'Second', verdict: 'pending' }],
        nextCursor: '',
        progress: [],
        gradingComplete: false,
        questions: qMeta,
        total: 1,
      })

    wrapper = mount(SimpleQuizJudgeScreen, mountOpts)
    await flushPromises()

    expect(mockSimpleQuizApi.state).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Accept Pariz')
    expect(wrapper.text()).toContain('Paris')
    expect(wrapper.text()).toContain('1 / 2')

    await wrapper.find('.judge-nav button:last-child').trigger('click')
    expect(wrapper.find('.judge-answer').text()).toBe('Second')

    mockQuizWS.emit('simple:state', {})
    await flushPromises()

    expect(mockSimpleQuizApi.state).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Accept Pariz')
    expect(wrapper.find('.judge-answer').text()).toBe('Second')
  })

  it('SimpleQuizPlayerScreen preserves submitted state on currentIndex change and in-flight refresh', async () => {
    localStorage.setItem('sq-identity:e1:m1', JSON.stringify({ playerId: 'p1', secret: 'sec1', nickname: 'Ana' }))
    mockSimpleQuizApi.join.mockResolvedValue({ playerId: 'p1', secret: 'sec1', nickname: 'Ana', reconnected: true })

    // Initial state loads at question 1 (currentIndex changed from default 0 -> 1) already submitted
    mockSimpleQuizApi.state.mockResolvedValueOnce({
      status: 'question_open',
      currentIndex: 1,
      total: 3,
      question: { text: 'Q2', points: 50 },
      submitted: true,
    })

    wrapper = mount(SimpleQuizPlayerScreen, mountOpts)
    await flushPromises()

    expect(wrapper.find('.sq-locked').exists()).toBe(true)
    expect(wrapper.find('.sq-answer-box').exists()).toBe(false)

    // Advance to unsubmitted Q3
    mockSimpleQuizApi.state.mockResolvedValueOnce({
      status: 'question_open',
      currentIndex: 2,
      total: 3,
      question: { text: 'Q3', points: 50 },
      submitted: false,
    })
    mockQuizWS.emit('simple:state', {})
    await flushPromises()
    expect(wrapper.find('.sq-answer-box').exists()).toBe(true)

    // Start an in-flight refresh that reports submitted: false while player submits
    let resolveStaleRefresh
    mockSimpleQuizApi.state.mockReturnValueOnce(new Promise(resolve => { resolveStaleRefresh = resolve }))
    mockQuizWS.emit('simple:state', {})

    mockSimpleQuizApi.submitAnswer.mockResolvedValueOnce({ answerId: 'ans-1' })
    await wrapper.find('.sq-textarea').setValue('My answer')
    await wrapper.find('.sq-submit').trigger('click')
    await flushPromises()
    expect(wrapper.find('.sq-locked').exists()).toBe(true)

    // Stale refresh resolves afterwards; must NOT revert submitted back to false
    resolveStaleRefresh({ status: 'question_open', currentIndex: 2, total: 3, question: { text: 'Q3', points: 50 }, submitted: false })
    await flushPromises()
    expect(wrapper.find('.sq-locked').exists()).toBe(true)

    // Session reset -> inactive returns player to Join screen and clears stored identity
    mockSimpleQuizApi.state.mockResolvedValueOnce({ status: 'inactive' })
    mockQuizWS.emit('simple:state', {})
    await flushPromises()
    expect(wrapper.find('.name-entry-container').exists()).toBe(true)
    expect(localStorage.getItem('sq-identity:e1:m1')).toBeNull()
  })

  it('SimpleQuizHostScreen restores judgeToken on mount, avoids double-fetching projector state, and guards double-clicks', async () => {
    vi.stubGlobal('fetch', vi.fn((url) => {
      if (url === '/events/e1') return Promise.resolve({ ok: true, json: async () => ({ pin: '9988' }) })
      if (url === '/events/e1/participants') return Promise.resolve({ ok: true, json: async () => ([]) })
      if (url === '/events/e1/modules/m1') return Promise.resolve({ ok: true, json: async () => ({ data: { judgeToken: 'restored-tok' } }) })
      return Promise.resolve({ ok: false })
    }))
    mockSimpleQuizApi.state.mockResolvedValue({
      status: 'lobby',
      currentIndex: 0,
      total: 1,
      players: 0,
      playerList: [],
      questions: [{ text: 'Q1' }],
      progress: [],
    })

    wrapper = mount(SimpleQuizHostScreen, mountOpts)
    await flushPromises()

    expect(mockSimpleQuizApi.state).toHaveBeenCalledTimes(1)
    expect(mockSimpleQuizApi.state).toHaveBeenCalledWith('e1', 'm1', { view: 'host' })
    expect(wrapper.text()).toContain('restored-tok')

    let resolveOpen
    mockSimpleQuizApi.open.mockReturnValueOnce(new Promise(resolve => { resolveOpen = resolve }))
    const openBtn = wrapper.find('.lobby-footer .start-btn')
    await openBtn.trigger('click')
    await openBtn.trigger('click')
    expect(mockSimpleQuizApi.open).toHaveBeenCalledTimes(1)
    resolveOpen({})
    await flushPromises()
  })

  it('SimpleQuizProjector unregisters WS listener on unmount, ignores stale responses, and renders paused state', async () => {
    let resolveFirst
    mockSimpleQuizApi.state
      .mockReturnValueOnce(new Promise(resolve => { resolveFirst = resolve }))
      .mockResolvedValueOnce({
        status: 'paused',
        currentIndex: 0,
        total: 2,
        players: 3,
        answeredCount: 1,
        questions: [{ text: 'Latest Q' }],
      })

    wrapper = mount(SimpleQuizProjector, {
      ...mountOpts,
      props: { eventId: 'e1', moduleId: 'm1', pin: '1234', visible: true },
    })

    mockQuizWS.emit('simple:state', {})
    await flushPromises()
    expect(wrapper.text()).toContain('simple_quiz.host.paused')

    resolveFirst({ status: 'lobby', currentIndex: 0, total: 2, players: 0, questions: [] })
    await flushPromises()
    expect(wrapper.text()).toContain('simple_quiz.host.paused')

    wrapper.unmount()
    wrapper = null
    expect(mockQuizWS.off).toHaveBeenCalledWith('simple:state', expect.any(Function))
    expect(wsHandlers.get('simple:state') || []).toHaveLength(0)
  })
})
