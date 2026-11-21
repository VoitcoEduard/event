import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ScreenManagerScreen from '../src/views/ScreenManagerScreen.vue'

const { toastMock, routerMock } = vi.hoisted(() => ({
  toastMock: { success: vi.fn(), error: vi.fn() },
  routerMock: { push: vi.fn() },
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: 'evt-42' } }),
  useRouter: () => routerMock,
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: k => k }),
}))

vi.mock('../src/composables/useToast', () => ({
  useToast: () => toastMock,
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
    mocks: { $t: k => k },
    stubs: {
      AppIcon: true,
      QRCodeVue: {
        name: 'QRCodeVue',
        props: ['value', 'size', 'level'],
        template: '<div class="qr-stub" :data-value="value" />',
      },
    },
  },
}

const sampleSlide = {
  id: 'slide-1',
  type: 'welcome',
  name: 'Welcome',
  backgroundColor: '#000000',
  backgroundImage: '',
  elements: [
    { id: 'el-qr', type: 'qr', x: 10, y: 10, width: 20, height: 20 },
  ],
}

beforeEach(() => {
  vi.useFakeTimers()
  toastMock.success.mockClear()
  toastMock.error.mockClear()
  routerMock.push.mockClear()
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('ScreenManagerScreen fixes (Bug #16)', () => {
  it('tracks projector popup state reactively when closed manually and never deadlocks', async () => {
    vi.stubGlobal('fetch', vi.fn((url) => {
      if (url === '/api/events/evt-42') {
        return Promise.resolve({ ok: true, json: async () => ({ id: 'evt-42', pin: '654321' }) })
      }
      if (url === '/api/events/evt-42/slides') {
        return Promise.resolve({ ok: true, json: async () => [structuredClone(sampleSlide)] })
      }
      return Promise.resolve({ ok: true, json: async () => ({}) })
    }))

    const popup1 = { closed: false, focus: vi.fn(), close: vi.fn(() => { popup1.closed = true }) }
    const popup2 = { closed: false, focus: vi.fn(), close: vi.fn(() => { popup2.closed = true }) }
    const openSpy = vi.fn()
      .mockReturnValueOnce(popup1)
      .mockReturnValueOnce(popup2)
    vi.stubGlobal('open', openSpy)

    wrapper = mount(ScreenManagerScreen, mountOpts)
    await flushPromises()

    const btn = wrapper.find('.launch-btn')
    expect(btn.text()).toBe('screen_manager.open_projector')

    // Open projector
    await btn.trigger('click')
    expect(openSpy).toHaveBeenCalledTimes(1)
    expect(btn.text()).toBe('admin.details.close_projector')

    // Simulate user closing the popup window manually
    popup1.closed = true
    vi.advanceTimersByTime(600)
    await flushPromises()

    expect(btn.text()).toBe('screen_manager.open_projector')

    // Re-open works cleanly
    await btn.trigger('click')
    expect(openSpy).toHaveBeenCalledTimes(2)
    expect(btn.text()).toBe('admin.details.close_projector')

    // Even if user clicks Close right after popup closed before poll fires, state clears cleanly
    popup2.closed = true
    await btn.trigger('click')
    expect(btn.text()).toBe('screen_manager.open_projector')
  })

  it('surfaces non-2xx HTTP errors and aborts pushToScreen if saveSlides fails', async () => {
    const fetchMock = vi.fn((url, opts = {}) => {
      if (url === '/api/events/evt-42') {
        return Promise.resolve({ ok: true, json: async () => ({ id: 'evt-42', pin: '654321' }) })
      }
      if (url === '/api/events/evt-42/slides' && (!opts.method || opts.method === 'GET')) {
        return Promise.resolve({
          ok: true,
          json: async () => [{ ...structuredClone(sampleSlide), id: 'slide-custom', type: 'custom' }],
        })
      }
      if (url === '/api/events/evt-42/slides' && opts.method === 'PUT') {
        return Promise.resolve({ ok: false, status: 500 })
      }
      if (url === '/api/templates' && opts.method === 'POST') {
        return Promise.resolve({ ok: false, status: 500 })
      }
      return Promise.resolve({ ok: true, json: async () => ({}) })
    })
    vi.stubGlobal('fetch', fetchMock)
    vi.stubGlobal('prompt', vi.fn(() => 'My Template'))

    wrapper = mount(ScreenManagerScreen, mountOpts)
    await flushPromises()

    // Pushing when saveSlides fails must abort before calling /screen-state
    await wrapper.find('.push-btn').trigger('click')
    await flushPromises()

    expect(toastMock.error).toHaveBeenCalledWith('screen_manager.editor.save_failed')
    expect(toastMock.success).not.toHaveBeenCalled()
    expect(fetchMock.mock.calls.some(([u]) => u === '/api/events/evt-42/screen-state')).toBe(false)

    // Saving as template when server returns non-2xx must surface error
    toastMock.error.mockClear()
    const saveTplBtn = wrapper.findAll('button').find(b => b.text().includes('screen_manager.editor.save_template'))
    expect(saveTplBtn).toBeTruthy()
    await saveTplBtn.trigger('click')
    await flushPromises()
    expect(toastMock.error).toHaveBeenCalledWith('screen_manager.editor.template_save_failed')
    expect(toastMock.success).not.toHaveBeenCalled()
  })

  it('surfaces error when fetchSlides or screen-state returns non-2xx', async () => {
    let failGet = true
    vi.stubGlobal('fetch', vi.fn((url, opts = {}) => {
      if (url === '/api/events/evt-42') {
        return Promise.resolve({ ok: true, json: async () => ({ id: 'evt-42', pin: '999888' }) })
      }
      if (url === '/api/events/evt-42/slides' && (!opts.method || opts.method === 'GET')) {
        if (failGet) return Promise.resolve({ ok: false, status: 500 })
        return Promise.resolve({ ok: true, json: async () => [structuredClone(sampleSlide)] })
      }
      if (url === '/api/events/evt-42/slides' && opts.method === 'PUT') {
        return Promise.resolve({ ok: true, json: async () => ({}) })
      }
      if (url === '/api/events/evt-42/screen-state' && opts.method === 'PUT') {
        return Promise.resolve({ ok: false, status: 500 })
      }
      return Promise.resolve({ ok: true, json: async () => ({}) })
    }))

    wrapper = mount(ScreenManagerScreen, mountOpts)
    await flushPromises()
    expect(toastMock.error).toHaveBeenCalledWith('screen_manager.editor.load_failed')

    wrapper.unmount()
    toastMock.error.mockClear()
    toastMock.success.mockClear()
    failGet = false

    wrapper = mount(ScreenManagerScreen, mountOpts)
    await flushPromises()

    await wrapper.find('.push-btn').trigger('click')
    await flushPromises()
    expect(toastMock.error).toHaveBeenCalledWith('screen_manager.editor.push_failed')
  })

  it('generates welcome slide QR code using /?pin=<eventPin> instead of /?eventId=', async () => {
    vi.stubGlobal('fetch', vi.fn((url) => {
      if (url === '/api/events/evt-42') {
        return Promise.resolve({ ok: true, json: async () => ({ id: 'evt-42', pin: '789012' }) })
      }
      if (url === '/api/events/evt-42/slides') {
        return Promise.resolve({ ok: true, json: async () => [structuredClone(sampleSlide)] })
      }
      return Promise.resolve({ ok: true, json: async () => ({}) })
    }))

    wrapper = mount(ScreenManagerScreen, mountOpts)
    await flushPromises()

    const qrStub = wrapper.find('.qr-stub')
    expect(qrStub.exists()).toBe(true)
    expect(qrStub.attributes('data-value')).toBe(`${window.location.origin}/?pin=789012`)
  })
})
