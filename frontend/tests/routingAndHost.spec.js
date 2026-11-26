import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createMemoryHistory } from 'vue-router'
import { createI18n } from 'vue-i18n'
import en from '../src/locales/en.json'
import ru from '../src/locales/ru.json'
import ro from '../src/locales/ro.json'

const { mockSessionState, mockLoadSession, mockQuizWS, mockSimpleQuizApi, toastMock } = vi.hoisted(() => {
  const wsHandlers = new Map()
  const mockSessionState = { loaded: true, user: { id: 'u1', role: 'admin', status: 'active' }, csrfToken: 'csrf', expiresAt: null, mode: 'dev' }
  const mockLoadSession = vi.fn(async () => mockSessionState.user)
  const mockQuizWS = {
    on: vi.fn((type, fn) => {
      if (!wsHandlers.has(type)) wsHandlers.set(type, [])
      wsHandlers.get(type).push(fn)
    }),
    off: vi.fn(),
    emit(type, payload) {
      ;(wsHandlers.get(type) || []).slice().forEach(fn => fn(payload))
    },
    send: vi.fn(),
    connect: vi.fn(),
    disconnect: vi.fn(),
    clearHandlers: vi.fn(() => wsHandlers.clear()),
  }
  const mockSimpleQuizApi = {
    state: vi.fn(async () => ({
      status: 'lobby',
      currentIndex: 0,
      total: 1,
      players: 0,
      playerList: [],
      questions: [{ text: 'Q1' }],
      progress: [],
    })),
    launch: vi.fn(),
    open: vi.fn(),
    lock: vi.fn(),
    next: vi.fn(),
    reveal: vi.fn(),
    endAnswering: vi.fn(),
    publish: vi.fn(),
    updatePublished: vi.fn(),
    complete: vi.fn(),
    resume: vi.fn(),
    linkParticipant: vi.fn(),
  }
  const toastMock = { success: vi.fn(), error: vi.fn(), info: vi.fn() }
  return { mockSessionState, mockLoadSession, mockQuizWS, mockSimpleQuizApi, toastMock }
})

vi.mock('../src/session.js', () => ({
  session: mockSessionState,
  loadSession: mockLoadSession,
  signOut: vi.fn(),
}))

vi.mock('../src/services/websocket', () => ({ quizWS: mockQuizWS }))
vi.mock('../src/services/simpleQuiz', () => ({ simpleQuizApi: mockSimpleQuizApi }))
vi.mock('../src/services/media', () => ({ getImageUrl: id => `/media/${id}` }))
vi.mock('../src/composables/useToast', () => ({ useToast: () => toastMock }))
vi.mock('../src/utils', () => ({
  API_BASE_URL: '',
  getAuthHeaders: () => ({}),
  getUserId: () => 'u1',
}))
vi.mock('../src/utils/imagePreload', () => ({
  preloader: {
    preloadQuestions: vi.fn(),
    preloadQuestion: vi.fn(),
    clear: vi.fn(),
  },
}))

import { createAppRouter, sanitizeAdminReturnTo } from '../src/router/index.js'
import AdminModuleHostScreen from '../src/views/AdminModuleHostScreen.vue'
import QuizHostScreen from '../src/views/QuizHostScreen.vue'
import SimpleQuizHostScreen from '../src/views/SimpleQuizHostScreen.vue'
import MainScreen from '../src/views/MainScreen.vue'

function makeI18n() {
  return createI18n({
    legacy: false,
    locale: 'en',
    messages: { en, ru, ro },
  })
}

let wrapper = null

beforeEach(() => {
  sessionStorage.clear()
  mockSessionState.loaded = true
  mockSessionState.user = { id: 'u1', role: 'admin', status: 'active' }
  mockLoadSession.mockClear()
  mockQuizWS.on.mockClear()
  mockQuizWS.connect.mockClear()
  mockQuizWS.disconnect.mockClear()
  mockQuizWS.clearHandlers.mockClear()
  mockSimpleQuizApi.state.mockClear()
  toastMock.success.mockClear()
  toastMock.error.mockClear()
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  vi.unstubAllGlobals()
})

describe('Phase 2 canonical routing and legacy redirects', () => {
  it('redirects legacy admin, library, data, teams, projector, and host routes to canonical /admin/... paths', async () => {
    const router = createAppRouter(createMemoryHistory())

    await router.push('/admin')
    expect(router.currentRoute.value.fullPath).toBe('/admin/events')

    await router.push('/admin/event/evt-1')
    expect(router.currentRoute.value.fullPath).toBe('/admin/events/evt-1')

    await router.push('/admin/event/evt-1/screen-manager')
    expect(router.currentRoute.value.fullPath).toBe('/admin/events/evt-1/projector')

    await router.push('/admin/event/evt-1/teams?moduleId=teams-9')
    expect(router.currentRoute.value.fullPath).toBe('/admin/events/evt-1/teams?moduleId=teams-9')

    await router.push('/data/library?eventId=evt-1')
    expect(router.currentRoute.value.fullPath).toBe('/admin/events/evt-1/data')

    await router.push('/data/library')
    expect(router.currentRoute.value.fullPath).toBe('/admin/events')

    await router.push('/admin/library')
    expect(router.currentRoute.value.fullPath).toBe('/admin/library/quizzes')

    await router.push('/quiz/library')
    expect(router.currentRoute.value.fullPath).toBe('/admin/library/quizzes')

    await router.push('/quiz/evt-1/mod-1/host')
    expect(router.currentRoute.value.fullPath).toBe('/admin/events/evt-1/modules/mod-1/host')

    await router.push('/wrong-pictures/evt-1/mod-2/host')
    expect(router.currentRoute.value.fullPath).toBe('/admin/events/evt-1/modules/mod-2/host')

    await router.push('/simple-quiz/evt-1/mod-3/host')
    expect(router.currentRoute.value.fullPath).toBe('/admin/events/evt-1/modules/mod-3/host')
  })

  it('keeps public participant, projector, and judge routes unchanged', async () => {
    const router = createAppRouter(createMemoryHistory())

    for (const path of [
      '/',
      '/event',
      '/projector/evt-1',
      '/quiz/evt-1/mod-1/player',
      '/wrong-pictures/evt-1/mod-1/player',
      '/simple-quiz/evt-1/mod-1/player',
      '/simple-quiz/evt-1/mod-1/judge',
      '/teams/evt-1/mod-1/player',
    ]) {
      await router.push(path)
      expect(router.currentRoute.value.fullPath).toBe(path)
    }
  })

  it('validates ?returnTo=<path> as an internal /admin/... path and strips unsafe targets', async () => {
    const router = createAppRouter(createMemoryHistory())

    expect(sanitizeAdminReturnTo('/admin/events/evt-1', router)).toBe('/admin/events/evt-1')
    expect(sanitizeAdminReturnTo('/admin/event/evt-1', router)).toBe('/admin/events/evt-1')
    expect(sanitizeAdminReturnTo('https://evil.example/admin', router)).toBeNull()
    expect(sanitizeAdminReturnTo('//evil.example/admin/events', router)).toBeNull()
    expect(sanitizeAdminReturnTo('/event', router)).toBeNull()
    expect(sanitizeAdminReturnTo('/admin\\events', router)).toBeNull()

    await router.push('/admin/library/quizzes?returnTo=/admin/events/evt-1')
    expect(router.currentRoute.value.query.returnTo).toBe('/admin/events/evt-1')

    await router.push('/admin/library/quizzes?returnTo=/admin/event/evt-1')
    expect(router.currentRoute.value.query.returnTo).toBe('/admin/events/evt-1')

    await router.push('/admin/library/quizzes?returnTo=https://evil.example')
    expect(router.currentRoute.value.fullPath).toBe('/admin/library/quizzes')
  })

  it('remembers unauthenticated target and resumes canonical route via takeReturnPath after login', async () => {
    const router = createAppRouter(createMemoryHistory())
    mockSessionState.user = null

    await router.push('/admin/events/evt-9/modules/mod-4/host')
    expect(router.currentRoute.value.name).toBe('login')
    expect(sessionStorage.getItem('adminReturnPath')).toBe('/admin/events/evt-9/modules/mod-4/host')

    mockSessionState.user = { id: 'u1', role: 'admin', status: 'active' }
    await router.push('/admin')
    expect(router.currentRoute.value.fullPath).toBe('/admin/events/evt-9/modules/mod-4/host')
  })
})

describe('AdminModuleHostScreen', () => {
  it('renders QuizHostScreen for quiz and wrong_pictures modules', async () => {
    vi.stubGlobal('fetch', vi.fn((url) => {
      if (url === '/events/e1/modules/m1') {
        return Promise.resolve({ ok: true, json: async () => ({ id: 'm1', name: 'Trivia', type: 'quiz', data: { questions: [] } }) })
      }
      if (url === '/events/e1') {
        return Promise.resolve({ ok: true, json: async () => ({ id: 'e1', name: 'Spring Gala', pin: '123456', modules: [] }) })
      }
      return Promise.resolve({ ok: false })
    }))

    const router = createAppRouter(createMemoryHistory())
    await router.push('/admin/events/e1/modules/m1/host')
    await router.isReady()

    wrapper = mount(AdminModuleHostScreen, {
      global: {
        plugins: [router, makeI18n()],
        stubs: { AppIcon: true },
      },
    })
    await flushPromises()

    expect(wrapper.findComponent(QuizHostScreen).exists()).toBe(true)
    expect(wrapper.find('.host-topbar-context').text()).toBe('Spring Gala / Trivia')
  })

  it('renders SimpleQuizHostScreen for simple_quiz modules', async () => {
    vi.stubGlobal('fetch', vi.fn((url) => {
      if (url === '/events/e1/modules/m2') {
        return Promise.resolve({ ok: true, json: async () => ({ id: 'm2', name: 'Open Q', type: 'simple_quiz', data: {} }) })
      }
      if (url === '/events/e1') {
        return Promise.resolve({ ok: true, json: async () => ({ id: 'e1', name: 'Spring Gala', pin: '123456' }) })
      }
      if (url === '/events/e1/participants') {
        return Promise.resolve({ ok: true, json: async () => [] })
      }
      return Promise.resolve({ ok: false })
    }))

    const router = createAppRouter(createMemoryHistory())
    await router.push('/admin/events/e1/modules/m2/host')
    await router.isReady()

    wrapper = mount(AdminModuleHostScreen, {
      global: {
        plugins: [router, makeI18n()],
        stubs: { AppIcon: true },
      },
    })
    await flushPromises()

    expect(wrapper.findComponent(SimpleQuizHostScreen).exists()).toBe(true)
  })

  it('redirects teams modules to /admin/events/:eventId/teams?moduleId=:moduleId', async () => {
    vi.stubGlobal('fetch', vi.fn((url) => {
      if (url === '/events/e1/modules/m3') {
        return Promise.resolve({ ok: true, json: async () => ({ id: 'm3', name: 'Relay', type: 'teams' }) })
      }
      if (url === '/events/e1') {
        return Promise.resolve({ ok: true, json: async () => ({ id: 'e1', name: 'Spring Gala', pin: '123456' }) })
      }
      return Promise.resolve({ ok: false })
    }))

    const router = createAppRouter(createMemoryHistory())
    await router.push('/admin/events/e1/modules/m3/host')
    await router.isReady()

    wrapper = mount(AdminModuleHostScreen, {
      global: {
        plugins: [router, makeI18n()],
        stubs: { AppIcon: true, TeamsAdminScreen: true },
      },
    })
    await flushPromises()

    expect(router.currentRoute.value.fullPath).toBe('/admin/events/e1/teams?moduleId=m3')
  })
})

describe('Host screens UX upgrades', () => {
  it('QuizHostScreen renders QRCodeVue encoding origin/?pin=PIN, Back to Event link, inline theme toggle, empty-player hint, and localized labels', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({
      ok: true,
      json: async () => ({
        pin: '654321',
        modules: [{
          id: 'm1',
          data: {
            questions: [{ text: 'Q1', type: 'classic', answers: ['A', 'B', 'C', 'D'], correctIndex: 0 }],
          },
        }],
      }),
    })))

    const router = createAppRouter(createMemoryHistory())
    await router.push('/admin/events/e1/modules/m1/host')
    await router.isReady()

    wrapper = mount(QuizHostScreen, {
      global: {
        plugins: [router, makeI18n()],
        stubs: { AppIcon: true, WrongPictureTargetCrop: true },
      },
    })
    await flushPromises()

    expect(wrapper.find('.qr-placeholder').exists()).toBe(false)
    const qrSvg = wrapper.find('.qr-container svg')
    expect(qrSvg.exists()).toBe(true)
    expect(qrSvg.attributes('aria-label')).toContain('654321')

    const backLink = wrapper.find('.host-back-link')
    expect(backLink.exists()).toBe(true)
    expect(backLink.attributes('href')).toBe('/admin/events/e1')
    expect(backLink.text()).toContain(en.host.back_to_event)

    expect(wrapper.find('.admin-theme-control--inline').exists()).toBe(true)
    expect(wrapper.find('.start-btn').attributes('disabled')).toBeDefined()
    expect(wrapper.find('.lobby-start-hint').text()).toBe(en.host.waiting_for_players)

    mockQuizWS.emit('state:update', { players: { p1: { id: 'p1', nickname: 'Ada', score: 0 } } })
    await flushPromises()
    expect(wrapper.find('.lobby-start-hint').exists()).toBe(false)

    mockQuizWS.emit('cmd:state', { state: 'question-active', currentIndex: 0 })
    await flushPromises()
    expect(wrapper.find('.counter-label').text()).toBe(en.quiz.host.answers_label)
    expect(wrapper.find('.host-controls .control-btn').text()).toBe(en.quiz.host.skip_timer)

    mockQuizWS.emit('cmd:state', { state: 'results', currentIndex: 0 })
    await flushPromises()
    expect(wrapper.find('.results-mode-title').text()).toBe('Question 1 Results')
  })

  it('SimpleQuizHostScreen renders top-left Back to Event link and inline AdminThemeToggle', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({ ok: true, json: async () => ({ pin: '654321', data: {} }) })))

    const router = createAppRouter(createMemoryHistory())
    await router.push('/admin/events/e1/modules/m2/host')
    await router.isReady()

    wrapper = mount(SimpleQuizHostScreen, {
      global: {
        plugins: [router, makeI18n()],
        stubs: { AppIcon: true },
      },
    })
    await flushPromises()

    const backLink = wrapper.find('.host-back-link')
    expect(backLink.exists()).toBe(true)
    expect(backLink.attributes('href')).toBe('/admin/events/e1')
    expect(backLink.text()).toContain(en.host.back_to_event)
    expect(wrapper.find('.admin-theme-control--inline').exists()).toBe(true)
  })
})

describe('MainScreen & locale copy', () => {
  it('replaces Minimalist Design placeholder across all locales and navigates to /admin/events', async () => {
    for (const messages of [en, ru, ro]) {
      expect(messages.main.subtitle).toBeTruthy()
      expect(messages.main.subtitle.toLowerCase()).not.toContain('minimalist')
      expect(messages.main.subtitle.toLowerCase()).not.toContain('минималистичный')
    }

    const router = createAppRouter(createMemoryHistory())
    await router.push('/')
    await router.isReady()

    wrapper = mount(MainScreen, {
      global: {
        plugins: [router, makeI18n()],
        stubs: { AppIcon: true },
      },
    })
    await flushPromises()

    expect(wrapper.find('.subtitle').text()).toBe(en.main.subtitle)
    await wrapper.find('.admin-link').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.name).toBe('admin-events')
  })
})
