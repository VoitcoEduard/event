import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import i18n from '../src/i18n.js'
import { clearDialogQueue } from '../src/composables/useDialog.js'
import { confirmActiveUnsavedChanges } from '../src/composables/useUnsavedChanges.js'
import { parsePoints } from '../src/utils/eventScores.js'
import AppDialogHost from '../src/components/AppDialogHost.vue'
import QuizQuestionEditor from '../src/components/QuizQuestionEditor.vue'
import AddModuleModal from '../src/components/AddModuleModal.vue'
import AdminEventDetailsScreen from '../src/views/AdminEventDetailsScreen.vue'
import AdminScreen from '../src/views/AdminScreen.vue'
import DataLibraryScreen from '../src/views/DataLibraryScreen.vue'

const mockRoute = reactive({ params: { id: 'evt-1' }, query: {} })
const mockToast = { success: vi.fn(), error: vi.fn(), info: vi.fn() }
const mockRouter = { push: vi.fn() }

vi.mock('vue-router', () => ({
  useRoute: () => mockRoute,
  useRouter: () => mockRouter,
  RouterLink: { template: '<a><slot /></a>' },
  onBeforeRouteLeave: vi.fn(),
  onBeforeRouteUpdate: vi.fn()
}))

vi.mock('../src/composables/useToast', () => ({
  useToast: () => mockToast
}))

let wrappers = []
function track(wrapper) {
  wrappers.push(wrapper)
  return wrapper
}

beforeEach(() => {
  i18n.global.locale.value = 'en'
  mockRoute.params = { id: 'evt-1' }
  mockRoute.query = {}
  localStorage.clear()
  sessionStorage.clear()
  clearDialogQueue()
  mockToast.success.mockClear()
  mockToast.error.mockClear()
  mockToast.info.mockClear()
  mockRouter.push.mockClear()
  vi.restoreAllMocks()
})

afterEach(async () => {
  clearDialogQueue()
  while (wrappers.length) {
    wrappers.pop().unmount()
  }
  document.body.innerHTML = ''
  vi.unstubAllGlobals()
})

describe('Wrong Pictures default tolerance (L5)', () => {
  it('defaults tolerancePercent to 5 for wrong_pictures questions without config', async () => {
    const wrapper = track(mount(QuizQuestionEditor, {
      attachTo: document.body,
      props: {
        index: 0,
        eventId: 'evt-1',
        moduleId: 'mod-1',
        question: { id: 'q-wp', type: 'wrong_pictures', text: 'Find the differences', imageId: 'img-1' }
      },
      global: { plugins: [i18n], stubs: { AppIcon: true, QuestionTypeSelector: true, ImageUploader: true, CoordinatePicker: true } }
    }))
    await flushPromises()

    const toleranceInput = wrapper.find('.wrong-pictures-config input[type="number"]')
    expect(toleranceInput.exists()).toBe(true)
    expect(toleranceInput.element.value).toBe('5')
  })
})

describe('AddModuleModal template refresh (M5)', () => {
  it('refetches /templates on each open so new templates appear', async () => {
    const templateList = [{ id: 'tpl-1', name: 'Geography Quiz', moduleType: 'quiz', data: { questions: [] } }]
    const fetchMock = vi.fn((url) => {
      if (url === '/templates') {
        return Promise.resolve({ ok: true, json: async () => [...templateList] })
      }
      return Promise.resolve({ ok: true, json: async () => ({}) })
    })
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = track(mount(AddModuleModal, {
      attachTo: document.body,
      props: { visible: true, eventId: 'evt-1' },
      global: {
        plugins: [i18n],
        stubs: { AppIcon: true, QuestionTypeSelector: true, Teleport: true, ImageUploader: true, CoordinatePicker: true }
      }
    }))
    await flushPromises()

    await wrapper.find('.module-type-card').trigger('click')
    await flushPromises()

    const templateOption = () => wrapper.findAll('.source-option')[1]
    await templateOption().trigger('click')
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(wrapper.findAll('.template-choice-name').map(el => el.text())).toEqual(['Geography Quiz'])

    templateList.push({ id: 'tpl-2', name: 'History Quiz', moduleType: 'quiz', data: { questions: [] } })
    await wrapper.setProps({ visible: false })
    await flushPromises()
    await wrapper.setProps({ visible: true })
    await flushPromises()

    await wrapper.find('.module-type-card').trigger('click')
    await flushPromises()

    await templateOption().trigger('click')
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledTimes(2)
    const names = wrapper.findAll('.template-choice').map(el => el.text())
    expect(names.some(n => n.includes('History Quiz'))).toBe(true)
  })
})

describe('AdminEventDetailsScreen race handling (L2, L7c)', () => {
  const eventWithActiveQuiz = {
    id: 'evt-1',
    name: 'Race Event',
    description: '',
    pin: '123456',
    isActive: false,
    modules: [
      { id: 'mod-q', name: 'Wrong Pictures game', type: 'wrong_pictures', isActive: true }
    ]
  }

  function mountDetails(fetchMock) {
    vi.stubGlobal('fetch', fetchMock)
    return track(mount(AdminEventDetailsScreen, {
      attachTo: document.body,
      global: {
        plugins: [i18n],
        stubs: { AppIcon: true, AddModuleModal: true, QRCodeVue: true }
      }
    }))
  }

  it('aborts module deactivation when the quiz reset fails (L7c)', async () => {
    const fetchMock = vi.fn((url, opts = {}) => {
      if (url === '/events/evt-1') return Promise.resolve({ ok: true, json: async () => eventWithActiveQuiz })
      if (url === '/events/evt-1/screensaver') return Promise.resolve({ ok: true, json: async () => ({}) })
      if (url === '/events/evt-1/modules/mod-q/quiz/reset' && opts.method === 'POST') {
        return Promise.resolve({ ok: false, status: 409, text: async () => '' })
      }
      return Promise.resolve({ ok: true, json: async () => ({}) })
    })

    const wrapper = mountDetails(fetchMock)
    await flushPromises()

    const stopButtons = wrapper.findAll('.module-btn').filter(b => b.text().trim() === i18n.global.t('admin.details.stop'))
    expect(stopButtons.length).toBe(1)
    await stopButtons[0].trigger('click')
    await flushPromises()

    expect(mockToast.error).toHaveBeenCalled()
    const putStatus = fetchMock.mock.calls.some(([u, o]) => o?.method === 'PUT' && String(u).endsWith('/modules/mod-q/status'))
    expect(putStatus).toBe(false)
  })

  it('ignores a stale event-status response after the user switched events (L2)', async () => {
    let resolveStatus
    const eventsById = {
      'evt-1': { ...eventWithActiveQuiz, isActive: false, modules: [] },
      'evt-2': { id: 'evt-2', name: 'Second Event', description: '', pin: '654321', isActive: false, modules: [] }
    }
    const fetchMock = vi.fn((url, opts = {}) => {
      const match = url.match(/^\/events\/(evt-\d+)$/)
      if (match && (!opts.method || opts.method === 'GET')) {
        return Promise.resolve({ ok: true, json: async () => eventsById[match[1]] })
      }
      if (url === '/events/evt-1/screensaver') return Promise.resolve({ ok: true, json: async () => ({}) })
      if (url === '/events/evt-2/screensaver') return Promise.resolve({ ok: true, json: async () => ({}) })
      if (url === '/events/evt-1/status' && opts.method === 'PUT') {
        return new Promise((resolve) => { resolveStatus = resolve })
      }
      return Promise.resolve({ ok: true, json: async () => ({}) })
    })
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = mountDetails(fetchMock)
    await flushPromises()

    const startButton = wrapper.find('.start-button')
    expect(startButton.exists()).toBe(true)
    await startButton.trigger('click')
    await flushPromises()
    expect(resolveStatus).toBeDefined()

    mockRoute.params = { id: 'evt-2' }
    await flushPromises()
    await flushPromises()

    resolveStatus({ ok: true, json: async () => ({ message: 'ok' }) })
    await flushPromises()

    expect(mockToast.success).not.toHaveBeenCalledWith(
      expect.anything(),
      'event_details.toast_event_started'
    )
    expect(wrapper.find('.start-button').exists()).toBe(true)
    expect(fetchMock.mock.calls.filter(([u, o]) => u === '/events/evt-2' && (!o?.method || o.method === 'GET')).length).toBeGreaterThan(0)
  })
})

describe('AdminScreen dirty form guard (L1)', () => {
  it('guards the create/edit event form through confirmActiveUnsavedChanges', async () => {
    const fetchMock = vi.fn((url) => {
      if (url === '/events') return Promise.resolve({ ok: true, json: async () => [] })
      return Promise.resolve({ ok: true, json: async () => ({}) })
    })
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = track(mount(AdminScreen, {
      attachTo: document.body,
      global: {
        plugins: [i18n],
        stubs: {
          AppIcon: true,
          RouterLink: { template: '<a><slot /></a>' },
          AdminLayout: { template: '<slot name="actions" /><slot />' }
        }
      }
    }))
    track(mount(AppDialogHost, { attachTo: document.body, global: { plugins: [i18n] } }))
    await flushPromises()

    await wrapper.find('.create-toggle-btn').trigger('click')
    await flushPromises()
    await expect(confirmActiveUnsavedChanges()).resolves.toBe(true)

    await wrapper.find('#admin-event-name').setValue('Draft Event')
    const guard = confirmActiveUnsavedChanges()
    await flushPromises()
    expect(document.body.querySelector('.app-dialog')).not.toBeNull()
    document.body.querySelector('.app-dialog-btn--secondary').click()
    await expect(guard).resolves.toBe(false)

    await wrapper.find('.form-close-btn').trigger('click')
    await flushPromises()
    await expect(confirmActiveUnsavedChanges()).resolves.toBe(true)
  })
})

describe('eventScores parsePoints error codes (L3c)', () => {
  it('tags thrown errors with stable codes for i18n mapping', () => {
    expect(() => parsePoints(' ')).toThrowError(expect.objectContaining({ code: 'points_required' }))
    try {
      parsePoints('2.5', true)
      expect.unreachable('parsePoints should throw for fractional game scores')
    } catch (err) {
      expect(err.code).toBe('points_integer')
    }
    try {
      parsePoints('abc', true)
      expect.unreachable('parsePoints should throw for non-numeric input')
    } catch (err) {
      expect(err.code).toBe('points_invalid')
    }
    try {
      parsePoints('abc')
      expect.unreachable('parsePoints should throw for non-numeric input')
    } catch (err) {
      expect(err.code).toBe('points_invalid')
    }
    expect(parsePoints('12')).toBe(12)
  })
})

describe('DataLibraryScreen localized points errors (L3c)', () => {
  const eventData = {
    id: 'evt-1',
    name: 'Localized Event',
    modules: [{ id: 'mod-q', name: 'Classic Quiz', type: 'quiz', data: {} }]
  }
  const scoresData = {
    participants: [{ id: 'p1', displayName: 'Alice' }],
    teams: [],
    games: [{ moduleId: 'mod-q', players: [{ id: 'p1', nickname: 'Alice', score: 50 }] }],
    entries: []
  }

  it('renders the localized invalid-points message in ru', async () => {
    i18n.global.locale.value = 'ru'
    mockRoute.query = { eventId: 'evt-1' }
    const fetchMock = vi.fn((url, opts = {}) => {
      if (url === '/events/evt-1') return Promise.resolve({ ok: true, json: async () => eventData })
      if (url === '/events/evt-1/participants') return Promise.resolve({ ok: true, json: async () => scoresData.participants })
      if (url === '/events/evt-1/teams') return Promise.resolve({ ok: true, json: async () => [] })
      if (url === '/events/evt-1/scores') return Promise.resolve({ ok: true, json: async () => scoresData })
      if (opts.method === 'PUT') return Promise.resolve({ ok: true, status: 200, json: async () => ({ message: 'ok' }) })
      return Promise.resolve({ ok: false, status: 404, text: async () => 'Not found' })
    })
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = track(mount(DataLibraryScreen, {
      attachTo: document.body,
      global: {
        plugins: [i18n],
        stubs: {
          AppIcon: true,
          RouterLink: { template: '<a><slot /></a>' },
          AdminLayout: { template: '<slot name="actions" /><slot />' }
        }
      }
    }))
    await flushPromises()

    await wrapper.findAll('nav button').find(b => b.text().trim().includes('Classic Quiz')).trigger('click')
    await flushPromises()

    const gameInput = wrapper.find('input[type="number"][step="1"]')
    expect(gameInput.exists()).toBe(true)
    await gameInput.setValue('2.5')
    const saveBtn = wrapper.find('.actions-cell button')
    await saveBtn.trigger('click')
    await flushPromises()

    const alert = wrapper.find('p.error')
    expect(alert.exists()).toBe(true)
    expect(alert.text()).toContain('Баллы за игру должны быть целыми числами')
    expect(alert.text()).not.toContain('Game scores must be whole numbers')
  })
})

describe('add_module plural item counts (L3b)', () => {
  it('renders pluralized item counts for en and ru', () => {
    const t = i18n.global.t
    i18n.global.locale.value = 'en'
    expect(t('add_module.items_count_questions', { count: 1 }, 1)).toBe('1 question')
    expect(t('add_module.items_count_questions', { count: 2 }, 2)).toBe('2 questions')

    i18n.global.locale.value = 'ru'
    expect(t('add_module.items_count_rounds', { count: 1 }, 1)).toBe('1 раунд')
    expect(t('add_module.items_count_rounds', { count: 2 }, 2)).toBe('Раундов: 2')

    i18n.global.locale.value = 'en'
  })
})
