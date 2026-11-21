import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import AdminEventDetailsScreen from '../src/views/AdminEventDetailsScreen.vue'
import en from '../src/locales/en.json'

function lookupEn(key) {
  return key.split('.').reduce((acc, part) => (acc && acc[part] !== undefined ? acc[part] : undefined), en) ?? key
}

const mockRoute = reactive({ params: { id: 'evt-1' } })
const mockRouter = { push: vi.fn() }
const mockToast = { success: vi.fn(), error: vi.fn(), info: vi.fn() }

vi.mock('vue-router', () => ({
  useRoute: () => mockRoute,
  useRouter: () => mockRouter,
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: lookupEn, locale: { value: 'en' } }),
}))

vi.mock('../src/composables/useToast', () => ({
  useToast: () => mockToast,
}))

vi.mock('../src/utils', () => ({
  API_BASE_URL: '/api',
  getAuthHeaders: () => ({}),
}))

vi.mock('../src/services/media', () => ({
  getImageUrl: id => `/media/${id}`,
  getScreensavers: vi.fn(async () => []),
}))

let wrapper = null
const mountOpts = {
  global: {
    mocks: { $t: k => k, $i18n: { locale: 'en' } },
    stubs: {
      RouterLink: true,
      QRCodeVue: true,
      AppIcon: true,
      AddModuleModal: true,
    },
  },
}

beforeEach(() => {
  sessionStorage.clear()
  mockRoute.params = { id: 'evt-1' }
  mockRouter.push.mockClear()
  mockToast.success.mockClear()
  mockToast.error.mockClear()
  mockToast.info.mockClear()
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  vi.unstubAllGlobals()
})

describe('AdminEventDetailsScreen route-driven event loading (Bug #13)', () => {
  it('loads event from route.params.id when sessionStorage is empty', async () => {
    const fetchMock = vi.fn((url) => {
      if (url === '/api/events/evt-1') {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            id: 'evt-1',
            name: 'Direct Navigation Event',
            description: 'Opened directly',
            pin: '123456',
            isActive: true,
            modules: [],
          }),
        })
      }
      if (url === '/api/events/evt-1/screensaver') {
        return Promise.resolve({ ok: true, json: async () => ({}) })
      }
      return Promise.resolve({ ok: false, status: 404 })
    })
    vi.stubGlobal('fetch', fetchMock)

    wrapper = mount(AdminEventDetailsScreen, mountOpts)
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledWith('/api/events/evt-1')
    expect(wrapper.find('.event-title').text()).toBe('Direct Navigation Event')
    expect(wrapper.find('.pin-text').text()).toBe('123456')
    expect(JSON.parse(sessionStorage.getItem('currentEvent')).id).toBe('evt-1')
  })

  it('ignores stale sessionStorage for a different event and reloads when route.params.id changes', async () => {
    sessionStorage.setItem('currentEvent', JSON.stringify({
      id: 'evt-stale',
      name: 'Stale Cached Event',
      pin: '000000',
      isActive: false,
      modules: [],
    }))

    mockRoute.params = { id: 'evt-2' }

    const eventsById = {
      'evt-2': {
        id: 'evt-2',
        name: 'Event Two',
        description: 'Second event',
        pin: '222222',
        isActive: true,
        modules: [],
      },
      'evt-3': {
        id: 'evt-3',
        name: 'Event Three',
        description: 'Third event',
        pin: '333333',
        isActive: false,
        modules: [],
      },
    }

    const fetchMock = vi.fn((url) => {
      const match = url.match(/^\/api\/events\/([^/]+)$/)
      if (match && eventsById[match[1]]) {
        return Promise.resolve({
          ok: true,
          json: async () => eventsById[match[1]],
        })
      }
      if (url.endsWith('/screensaver')) {
        return Promise.resolve({ ok: true, json: async () => ({}) })
      }
      return Promise.resolve({ ok: false, status: 404 })
    })
    vi.stubGlobal('fetch', fetchMock)

    wrapper = mount(AdminEventDetailsScreen, mountOpts)
    // Before fetch resolves, stale event from another ID must NOT be shown
    expect(wrapper.find('.event-title').text()).toBe('')

    await flushPromises()
    expect(wrapper.find('.event-title').text()).toBe('Event Two')
    expect(JSON.parse(sessionStorage.getItem('currentEvent')).id).toBe('evt-2')

    // Navigate to another event route while mounted
    mockRoute.params = { id: 'evt-3' }
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledWith('/api/events/evt-3')
    expect(wrapper.find('.event-title').text()).toBe('Event Three')
    expect(JSON.parse(sessionStorage.getItem('currentEvent')).id).toBe('evt-3')
  })

  it('handles malformed sessionStorage and 404 responses gracefully without crashing', async () => {
    sessionStorage.setItem('currentEvent', '{invalid json')
    mockRoute.params = { id: 'missing-evt' }

    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({ ok: false, status: 404 })))

    wrapper = mount(AdminEventDetailsScreen, mountOpts)
    await flushPromises()

    expect(wrapper.find('.event-state-card').exists()).toBe(true)
    expect(wrapper.find('.event-state-card').text()).toContain('Event not found')
    expect(wrapper.find('.screen-manager-btn').attributes('disabled')).toBeDefined()
    expect(wrapper.find('.data-btn').attributes('disabled')).toBeDefined()
  })
})
