import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import QuizPlayerScreen from '../src/views/QuizPlayerScreen.vue'
import { questionCache } from '../src/utils/questionCache'

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
  useI18n: () => ({ t: (k, f) => f || k, locale: { value: 'en' } }),
}))

vi.mock('../src/services/websocket', () => ({ quizWS: mockQuizWS }))
vi.mock('../src/utils/signalQueue', () => ({ signalQueue: mockSignalQueue }))
vi.mock('../src/services/media', () => ({ getImageUrl: id => `/media/${id}` }))
vi.mock('../src/utils', () => ({
  getUserName: () => 'Alice',
  newUUID: () => 'sub-uuid-1',
}))

let wrapper = null
const mountOpts = {
  global: {
    mocks: { $t: (k, f) => f || k, $i18n: { locale: 'en' } },
    stubs: {
      AppIcon: true,
      RangeComponent: true,
      HorizontalCarousel: true,
      MapCoordinatePicker: true,
      WrongPictureAnswerReview: {
        props: ['review'],
        template: '<div class="wp-review-stub">{{ review.totalScore }}</div>',
      },
    },
  },
}

beforeEach(() => {
  vi.useFakeTimers()
  questionCache.clear()
  wsHandlers.clear()
  mockQuizWS.connect.mockClear()
  mockQuizWS.disconnect.mockClear()
  mockQuizWS.send.mockClear()
  mockSignalQueue.enqueue.mockClear()
  mockSignalQueue.clear.mockClear()
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  questionCache.clear()
  vi.useRealTimers()
})

describe('QuizPlayerScreen edge cases (Bug #9)', () => {
  it('clears timerInterval when leaving question-active so stale auto-submit cannot fire in results', async () => {
    wrapper = mount(QuizPlayerScreen, mountOpts)
    await flushPromises()

    mockQuizWS.emit('data:batch', {
      batchIndex: 0,
      totalBatches: 1,
      startIndex: 0,
      runId: 'run-1',
      questions: [{ id: 'q1', type: 'classic', text: 'Q1', answers: ['A', 'B'] }],
    })
    mockQuizWS.emit('cmd:state', { state: 'question-active', currentIndex: 0, runId: 'run-1', timeLimit: 10 })
    await flushPromises()

    mockQuizWS.emit('cmd:state', { state: 'results', currentIndex: 0, runId: 'run-1' })
    await flushPromises()
    expect(wrapper.find('.result-container').exists()).toBe(true)

    await vi.advanceTimersByTimeAsync(12000)
    await flushPromises()
    expect(wrapper.find('.result-container').exists()).toBe(true)
  })

  it('renders a non-blank QUESTION_ENDED screen when the local timer expires before results', async () => {
    wrapper = mount(QuizPlayerScreen, mountOpts)
    await flushPromises()

    mockQuizWS.emit('data:batch', {
      batchIndex: 0,
      totalBatches: 1,
      startIndex: 0,
      runId: 'run-1',
      questions: [{ id: 'q1', type: 'classic', text: 'Q1', answers: ['A', 'B'] }],
    })
    mockQuizWS.emit('cmd:state', { state: 'question-active', currentIndex: 0, runId: 'run-1', timeLimit: 5 })
    await flushPromises()

    await vi.advanceTimersByTimeAsync(5500)
    await flushPromises()

    expect(wrapper.find('.answered-container').exists()).toBe(true)
    expect(wrapper.text()).toContain("Time's up!")
  })

  it('updates wrongPictureReview totalScore when answer:ack arrives after data:reveal and results', async () => {
    wrapper = mount(QuizPlayerScreen, mountOpts)
    await flushPromises()

    mockQuizWS.emit('data:batch', {
      batchIndex: 0,
      totalBatches: 1,
      startIndex: 0,
      runId: 'run-1',
      questions: [{
        id: 'wp1',
        type: 'wrong_pictures',
        text: 'Find mistakes',
        imageId: 'img-1',
        wrongPictureConfig: { showTargetCount: true, targetCount: 1, maxPoints: 100 },
      }],
    })
    mockQuizWS.emit('cmd:state', { state: 'question-active', currentIndex: 0, runId: 'run-1', timeLimit: 20 })
    await flushPromises()

    await wrapper.find('.wrong-pictures-container .submit-selection-btn').trigger('click')
    await flushPromises()
    expect(mockSignalQueue.enqueue).toHaveBeenCalledTimes(1)

    mockQuizWS.emit('cmd:state', { state: 'results', currentIndex: 0, runId: 'run-1' })
    mockQuizWS.emit('data:reveal', {
      questionIndex: 0,
      wrongPictureConfig: {
        maxPoints: 100,
        targets: [{ id: 't1', xPercent: 20, yPercent: 20, radius: 10 }],
      },
    })
    await flushPromises()
    expect(wrapper.find('.wp-review-stub').text()).toBe('0')

    mockQuizWS.emit('answer:ack', {
      submissionId: 'sub-uuid-1',
      questionIndex: 0,
      runId: 'run-1',
      accepted: true,
      awarded: 85,
      total: 85,
      streak: 1,
    })
    await flushPromises()
    expect(wrapper.find('.wp-review-stub').text()).toBe('85')
  })

  it('does not lock out premature taps before batch arrives, clears cache on runId change, and sends lastKnownQuestionIndex -1 before Q0 arrives', async () => {
    wrapper = mount(QuizPlayerScreen, mountOpts)
    await flushPromises()

    mockQuizWS.emit('open')
    mockQuizWS.emit('open')
    expect(mockQuizWS.send).toHaveBeenCalledWith('player:reconnect', expect.objectContaining({
      lastKnownQuestionIndex: -1,
    }))

    mockQuizWS.emit('cmd:state', { state: 'question-active', currentIndex: 0, runId: 'run-1', timeLimit: 20 })
    await flushPromises()

    await wrapper.find('.answer-btn').trigger('click')
    await flushPromises()
    expect(wrapper.find('.answered-container').exists()).toBe(false)
    expect(mockSignalQueue.enqueue).not.toHaveBeenCalled()

    mockQuizWS.emit('data:batch', {
      batchIndex: 0,
      totalBatches: 1,
      startIndex: 0,
      runId: 'run-1',
      questions: [{ id: 'q1', type: 'classic', text: 'Q1', answers: ['A', 'B'] }],
    })
    await flushPromises()

    await wrapper.find('.answer-btn').trigger('click')
    await flushPromises()
    expect(wrapper.find('.answered-container').exists()).toBe(true)
    expect(mockSignalQueue.enqueue).toHaveBeenCalledTimes(1)

    mockQuizWS.emit('cmd:state', { state: 'question-presenting', currentIndex: 1, runId: 'run-2', timeLimit: 5 })
    await flushPromises()
    expect(questionCache.has(0)).toBe(false)
  })
})

describe('QuizPlayerScreen UX audit regressions (H1, L10, L11)', () => {
  it('renders currentQuestion.text on classic, range, and number active questions and shows lobby waiting explanation', async () => {
    wrapper = mount(QuizPlayerScreen, mountOpts)
    await flushPromises()

    expect(wrapper.find('.lobby-wait-msg').exists()).toBe(true)
    expect(wrapper.find('.lobby-wait-msg').text()).toBe('quiz.player.waiting_for_host')

    mockQuizWS.emit('data:batch', {
      batchIndex: 0,
      totalBatches: 1,
      startIndex: 0,
      runId: 'run-1',
      questions: [
        { id: 'q1', type: 'classic', text: 'What is the capital of France?', answers: ['Paris', 'Lyon'] },
        { id: 'q2', type: 'range', text: 'When was the moon landing?', rangeConfig: { minValue: 1950, maxValue: 2000, step: 1, unit: 'year' } },
        { id: 'q3', type: 'number', text: 'How many continents?' },
      ],
    })

    mockQuizWS.emit('cmd:state', { state: 'question-active', currentIndex: 0, runId: 'run-1', timeLimit: 20 })
    await flushPromises()
    expect(wrapper.find('.player-question-text').text()).toBe('What is the capital of France?')

    mockQuizWS.emit('cmd:state', { state: 'question-active', currentIndex: 1, runId: 'run-1', timeLimit: 20 })
    await flushPromises()
    expect(wrapper.find('.range-container .player-question-text').text()).toBe('When was the moon landing?')

    mockQuizWS.emit('cmd:state', { state: 'question-active', currentIndex: 2, runId: 'run-1', timeLimit: 20 })
    await flushPromises()
    expect(wrapper.find('.number-container .player-question-text').text()).toBe('How many continents?')
  })

  it('formats ordinal rank as 1st, 2nd, 11th in leaderboard and finished states (L11)', async () => {
    wrapper = mount(QuizPlayerScreen, mountOpts)
    await flushPromises()

    mockQuizWS.emit('state:update', {
      players: { p1: { id: 'p1', nickname: 'Alice', score: 500 } },
      rank: 1,
    })
    mockQuizWS.emit('cmd:state', { state: 'leaderboard', currentIndex: 0, runId: 'run-1' })
    await flushPromises()
    expect(wrapper.find('.rank-badge-large .rank-num').text()).toBe('1st')

    mockQuizWS.emit('state:update', {
      players: { p1: { id: 'p1', nickname: 'Alice', score: 500 } },
      rank: 2,
    })
    await flushPromises()
    expect(wrapper.find('.rank-badge-large .rank-num').text()).toBe('2nd')

    mockQuizWS.emit('state:update', {
      players: { p1: { id: 'p1', nickname: 'Alice', score: 500 } },
      rank: 11,
    })
    mockQuizWS.emit('cmd:state', { state: 'finished', currentIndex: 0, runId: 'run-1' })
    await flushPromises()
    expect(wrapper.find('.final-rank-badge .rank-num').text()).toBe('11th')
  })
})
