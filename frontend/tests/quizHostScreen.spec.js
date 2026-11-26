import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import QuizHostScreen from '../src/views/QuizHostScreen.vue'

const { wsHandlers, mockQuizWS, toastMock } = vi.hoisted(() => {
  const wsHandlers = new Map()
  const toastMock = { success: vi.fn(), error: vi.fn() }
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
    send: vi.fn(),
    connect: vi.fn(),
    disconnect: vi.fn(),
    clearHandlers: vi.fn(() => wsHandlers.clear()),
  }
  return { wsHandlers, mockQuizWS, toastMock }
})

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { eventId: 'e1', moduleId: 'm1' } }),
  useRouter: () => ({ push: vi.fn() }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: k => k, locale: { value: 'en' } }),
}))

vi.mock('../src/services/websocket', () => ({ quizWS: mockQuizWS }))
vi.mock('../src/services/media', () => ({ getImageUrl: id => `/media/${id}` }))
vi.mock('../src/composables/useToast', () => ({ useToast: () => toastMock }))
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
    mocks: { $t: k => k },
    stubs: { AppIcon: true, WrongPictureTargetCrop: true },
  },
}

beforeEach(() => {
  vi.useFakeTimers()
  wsHandlers.clear()
  mockQuizWS.on.mockClear()
  mockQuizWS.off.mockClear()
  mockQuizWS.send.mockClear()
  mockQuizWS.connect.mockClear()
  mockQuizWS.disconnect.mockClear()
  mockQuizWS.clearHandlers.mockClear()
  toastMock.success.mockClear()
  toastMock.error.mockClear()
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('QuizHostScreen question-presenting transitions', () => {
  it('updates explainTimeRemaining, resets answer counts/signals, and restarts background audio without ReferenceError', async () => {
    vi.stubGlobal('fetch', vi.fn((url) => {
      if (url === '/events/e1') {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            pin: '4321',
            modules: [
              {
                id: 'm1',
                data: {
                  backgroundMusicUrl: '/audio/theme.mp3',
                  audioOutput: 'admin',
                  questions: [
                    {
                      text: 'Question 1',
                      type: 'multiple_choice',
                      timeLimit: 20,
                      answers: ['A', 'B', 'C', 'D'],
                      correctIndex: 0,
                    },
                    {
                      text: 'Question 2',
                      type: 'multiple_choice',
                      imageId: 'img-2',
                      timeLimit: 15,
                      answers: ['E', 'F', 'G', 'H'],
                      correctIndex: 1,
                    },
                  ],
                },
              },
            ],
          }),
        })
      }
      return Promise.resolve({ ok: false })
    }))

    wrapper = mount(QuizHostScreen, mountOpts)
    await flushPromises()

    const audioEl = wrapper.find('#bg-music-player').element
    let isPaused = true
    Object.defineProperty(audioEl, 'paused', {
      configurable: true,
      get: () => isPaused,
    })
    const playSpy = vi.spyOn(audioEl, 'play').mockImplementation(() => {
      isPaused = false
      return Promise.resolve()
    })
    const pauseSpy = vi.spyOn(audioEl, 'pause').mockImplementation(() => {
      isPaused = true
    })

    // Transition from lobby -> question-presenting (Q1)
    expect(() => {
      mockQuizWS.emit('cmd:state', { state: 'question-presenting', currentIndex: 0, timeLimit: 4 })
    }).not.toThrow()
    await flushPromises()

    expect(wrapper.find('.timer').text()).toBe('4')
    expect(playSpy).toHaveBeenCalledTimes(1)
    expect(audioEl.currentTime).toBe(0)

    vi.advanceTimersByTime(1000)
    await flushPromises()
    expect(wrapper.find('.timer').text()).toBe('3')

    // Move to active question and record answers + signals
    mockQuizWS.emit('cmd:state', { state: 'question-active', currentIndex: 0 })
    await flushPromises()

    mockQuizWS.emit('sync:minimal', {
      questionIndex: 0,
      answeredCount: 5,
      answerCounts: { 0: 3, 1: 2 },
    })
    await flushPromises()
    expect(wrapper.find('.counter-value').text()).toBe('5')

    // Move to results (pauses audio) and simulate playback position advancing
    mockQuizWS.emit('cmd:state', { state: 'results', currentIndex: 0 })
    await flushPromises()
    expect(pauseSpy).toHaveBeenCalled()
    audioEl.currentTime = 12

    // Transition to next question's question-presenting state
    expect(() => {
      mockQuizWS.emit('cmd:state', { state: 'question-presenting', currentIndex: 1, timeLimit: 3 })
    }).not.toThrow()
    await flushPromises()

    expect(wrapper.find('.timer').text()).toBe('3')
    expect(wrapper.find('.counter-value').text()).toBe('0')
    expect(audioEl.currentTime).toBe(0)
    expect(playSpy).toHaveBeenCalledTimes(2)

    // Let explain countdown finish to zero
    vi.advanceTimersByTime(3000)
    await flushPromises()
    expect(wrapper.find('.timer').text()).toBe('quiz.player.get_ready')

    // Verify answerCounts were reset (going to results without new sync:minimal shows 0s)
    mockQuizWS.emit('cmd:state', { state: 'results', currentIndex: 1 })
    await flushPromises()
    const counts = wrapper.findAll('.dist-count').map(el => el.text())
    expect(counts).toEqual(['0', '0', '0', '0'])
  })

  it('clears displayedScores when a fresh quiz starts at question 0', async () => {
    vi.stubGlobal('fetch', vi.fn((url) => {
      if (url === '/events/e1') {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            pin: '4321',
            modules: [
              {
                id: 'm1',
                data: {
                  questions: [
                    {
                      text: 'Question 1',
                      type: 'classic',
                      timeLimit: 20,
                      answers: ['A', 'B', 'C', 'D'],
                      correctIndex: 0,
                    },
                  ],
                },
              },
            ],
          }),
        })
      }
      return Promise.resolve({ ok: false })
    }))

    wrapper = mount(QuizHostScreen, mountOpts)
    await flushPromises()

    mockQuizWS.emit('state:update', {
      players: { p1: { id: 'p1', nickname: 'Alice', score: 250 } },
    })
    await flushPromises()

    // Restart quiz at question 0 with zeroed player score
    mockQuizWS.emit('cmd:state', { state: 'question-presenting', currentIndex: 0, timeLimit: 5 })
    mockQuizWS.emit('state:update', {
      players: { p1: { id: 'p1', nickname: 'Alice', score: 0 } },
    })
    mockQuizWS.emit('cmd:state', { state: 'leaderboard', currentIndex: 0 })
    await flushPromises()

    expect(wrapper.find('.lb-score-value').text()).toBe('0')
    expect(wrapper.find('.lb-diff-anim').exists()).toBe(false)
  })

  it('copies projector link via execCommand fallback when navigator.clipboard is unavailable (HTTP LAN)', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({
      ok: true,
      json: async () => ({ pin: '4321', modules: [{ id: 'm1', data: { questions: [] } }] }),
    })))

    const originalClipboard = Object.getOwnPropertyDescriptor(navigator, 'clipboard')
    const originalExecCommand = Object.getOwnPropertyDescriptor(document, 'execCommand')
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: undefined })
    const execSpy = vi.fn(() => true)
    Object.defineProperty(document, 'execCommand', { configurable: true, value: execSpy })

    try {
      wrapper = mount(QuizHostScreen, mountOpts)
      await flushPromises()

      await wrapper.find('.copy-link-btn').trigger('click')
      await flushPromises()

      expect(execSpy).toHaveBeenCalledWith('copy')
      expect(toastMock.success).toHaveBeenCalledWith('quiz.host.toast.link_copied')
      expect(toastMock.error).not.toHaveBeenCalled()
    } finally {
      if (originalClipboard) Object.defineProperty(navigator, 'clipboard', originalClipboard)
      else delete navigator.clipboard
      if (originalExecCommand) Object.defineProperty(document, 'execCommand', originalExecCommand)
      else delete document.execCommand
    }
  })

  it('shows error toast and does not show success when both clipboard API and fallback fail', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({
      ok: true,
      json: async () => ({ pin: '4321', modules: [{ id: 'm1', data: { questions: [] } }] }),
    })))

    const originalClipboard = Object.getOwnPropertyDescriptor(navigator, 'clipboard')
    const originalExecCommand = Object.getOwnPropertyDescriptor(document, 'execCommand')
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: vi.fn(() => Promise.reject(new Error('Permission denied'))) },
    })
    Object.defineProperty(document, 'execCommand', { configurable: true, value: vi.fn(() => false) })

    try {
      wrapper = mount(QuizHostScreen, mountOpts)
      await flushPromises()

      await wrapper.find('.copy-link-btn').trigger('click')
      await flushPromises()

      expect(toastMock.success).not.toHaveBeenCalled()
      expect(toastMock.error).toHaveBeenCalledTimes(1)
    } finally {
      if (originalClipboard) Object.defineProperty(navigator, 'clipboard', originalClipboard)
      else delete navigator.clipboard
      if (originalExecCommand) Object.defineProperty(document, 'execCommand', originalExecCommand)
      else delete document.execCommand
    }
  })
})

describe('QuizHostScreen distribution chart on text-only questions (Bug #7)', () => {
  it('renders .distribution-overlay-container in results state even without an image', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({
      ok: true,
      json: async () => ({
        pin: '4321',
        modules: [{
          id: 'm1',
          data: {
            questions: [{
              text: 'Text-only question',
              type: 'classic',
              answers: ['A', 'B', 'C', 'D'],
              correctIndex: 1,
            }],
          },
        }],
      }),
    })))

    wrapper = mount(QuizHostScreen, mountOpts)
    await flushPromises()

    mockQuizWS.emit('cmd:state', { state: 'results', currentIndex: 0 })
    mockQuizWS.emit('sync:minimal', { questionIndex: 0, answeredCount: 5, answerCounts: { '0': 1, '1': 3, '2': 1, '3': 0 } })
    await flushPromises()

    expect(wrapper.find('.distribution-overlay-container').exists()).toBe(true)
    expect(wrapper.findAll('.dist-count').map(n => n.text())).toEqual(['1', '3', '1', '0'])
  })
})

describe('QuizHostScreen range marker with maxValue 0 (Bug #8)', () => {
  it('computes correctMarkerPercentage accurately when maxValue is 0', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({
      ok: true,
      json: async () => ({
        pin: '4321',
        modules: [{
          id: 'm1',
          data: {
            questions: [{
              text: 'Celsius below zero',
              type: 'range',
              rangeConfig: { minValue: -100, maxValue: 0, step: 1, unit: '°C', decimalPlaces: 0 },
              correctValue: -25,
            }],
          },
        }],
      }),
    })))

    wrapper = mount(QuizHostScreen, mountOpts)
    await flushPromises()

    mockQuizWS.emit('cmd:state', { state: 'results', currentIndex: 0 })
    await flushPromises()

    expect(wrapper.find('.srt-marker').attributes('style')).toContain('bottom: 75%')
  })

  it('transitions cleanly from leaderboard to a true_false question and starts its timer even if question-presenting was skipped', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({
      ok: true,
      json: async () => ({
        pin: '4321',
        modules: [{
          id: 'm1',
          data: {
            questions: [
              { text: 'Q1', type: 'classic', timeLimit: 20, answers: ['A', 'B', 'C', 'D'], correctIndex: 0 },
              { text: 'Sky is blue?', type: 'true_false', timeLimit: 15, answers: ['True', 'False', '', ''], correctIndex: 0 },
            ],
          },
        }],
      }),
    })))

    wrapper = mount(QuizHostScreen, mountOpts)
    await flushPromises()

    // Simulate Q1 active -> leaderboard -> Q2 true_false presenting -> active
    mockQuizWS.emit('cmd:state', { state: 'question-active', currentIndex: 0 })
    await flushPromises()

    mockQuizWS.emit('cmd:state', { state: 'leaderboard', currentIndex: 0 })
    await flushPromises()

    await wrapper.find('.host-controls .control-btn.primary').trigger('click')
    expect(mockQuizWS.send).toHaveBeenCalledWith('host:next_question', { moduleId: 'm1' })

    mockQuizWS.emit('cmd:state', { state: 'question-presenting', currentIndex: 1, timeLimit: 3 })
    await flushPromises()
    expect(wrapper.find('.question-counter').text()).toBe('2 / 2')
    expect(wrapper.find('.question-text').text()).toBe('Sky is blue?')
    expect(wrapper.find('.timer').text()).toBe('3')

    mockQuizWS.emit('cmd:state', { state: 'question-active', currentIndex: 1 })
    await flushPromises()
    expect(wrapper.find('.timer').text()).toBe('15')
  })
})

describe('QuizHostScreen cmd:error surfacing (M1)', () => {
  it('shows error toasts for quiz_finishing and start_failed cmd errors, ignores unmapped codes', async () => {
    vi.stubGlobal('fetch', vi.fn(() =>
      Promise.resolve({
        ok: true,
        json: async () => ({
          pin: '4321',
          modules: [{
            id: 'm1',
            data: {
              questions: [
                { text: 'Question 1', type: 'multiple_choice', timeLimit: 20, answers: ['A', 'B'], correctIndex: 0 }
              ]
            }
          }]
        })
      })
    ))

    wrapper = mount(QuizHostScreen, mountOpts)
    await flushPromises()

    mockQuizWS.emit('cmd:error', { code: 'quiz_finishing', message: 'Quiz results are being saved. Wait a moment and try again.' })
    await flushPromises()
    expect(toastMock.error).toHaveBeenCalledWith('Quiz results are being saved. Wait a moment and try again.')

    toastMock.error.mockClear()
    mockQuizWS.emit('cmd:error', { code: 'start_failed', message: 'Cannot start quiz: no questions' })
    await flushPromises()
    expect(toastMock.error).toHaveBeenCalledWith('Cannot start quiz: no questions')

    // Unmapped codes without a message stay silent.
    toastMock.error.mockClear()
    mockQuizWS.emit('cmd:error', { code: 'something_else' })
    mockQuizWS.emit('cmd:error', { code: 'start_failed' })
    await flushPromises()
    expect(toastMock.error).not.toHaveBeenCalled()
  })
})

describe('QuizHostScreen UX audit regressions (L3, L4)', () => {
  it('passes players.length to quiz.host.players and formats word units with a space (1969 year) vs symbol units (50%)', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({
      ok: true,
      json: async () => ({
        pin: '4321',
        modules: [{
          id: 'm1',
          data: {
            questions: [
              {
                text: 'Moon landing year',
                type: 'range',
                rangeConfig: { minValue: 1950, maxValue: 2000, step: 1, unit: 'year', decimalPlaces: 0 },
                correctValue: 1969,
              },
              {
                text: 'Humidity',
                type: 'range',
                rangeConfig: { minValue: 0, maxValue: 100, step: 1, unit: '%', decimalPlaces: 0 },
                correctValue: 50,
              },
            ],
          },
        }],
      }),
    })))

    const tCalls = []
    wrapper = mount(QuizHostScreen, {
      global: {
        mocks: {
          $t: (k, arg) => {
            tCalls.push([k, arg])
            return k
          },
        },
        stubs: { AppIcon: true, WrongPictureTargetCrop: true, AdminThemeToggle: true },
      },
    })
    await flushPromises()

    mockQuizWS.emit('state:update', { players: { p1: { id: 'p1', nickname: 'Alpha', score: 0 } } })
    await flushPromises()
    expect(tCalls.some(([k, arg]) => k === 'quiz.host.players' && arg === 1)).toBe(true)

    mockQuizWS.emit('cmd:state', { state: 'question-active', currentIndex: 0 })
    await flushPromises()
    expect(wrapper.find('.sbs-label-max').text()).toBe('2000 year')
    expect(wrapper.find('.sbs-label-min').text()).toBe('1950 year')

    mockQuizWS.emit('cmd:state', { state: 'results', currentIndex: 0 })
    await flushPromises()
    expect(wrapper.find('.srt-value').text()).toBe('1969 year')

    mockQuizWS.emit('cmd:state', { state: 'results', currentIndex: 1 })
    await flushPromises()
    expect(wrapper.find('.srt-value').text()).toBe('50%')
  })
})
