import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import EventScreen from '../src/views/EventScreen.vue'

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: {} }),
  useRouter: () => ({ push: vi.fn() }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: k => k, locale: { value: 'en' } }),
}))

vi.mock('../src/utils', () => ({
  API_BASE_URL: '',
  getUserId: () => 'u1',
  setUserId: vi.fn(),
  newUUID: () => 'u2',
  getUserName: () => 'Alice',
  setUserName: vi.fn(),
  getParticipantSecret: () => 'sec-1',
  setParticipantSecret: vi.fn(),
}))

let wrapper = null
const mountOpts = {
  global: {
    mocks: { $t: k => k, $i18n: { locale: 'en' } },
    stubs: {
      AppIcon: true,
      FormModule: {
        props: ['module'],
        emits: ['complete'],
        template: '<div class="form-module-stub" :data-id="module.id"><button class="complete-btn" @click="$emit(\'complete\', {})">Done</button></div>',
      },
    },
  },
}

beforeEach(() => {
  vi.useFakeTimers()
  sessionStorage.setItem('currentEvent', JSON.stringify({ id: 'e1' }))
  vi.stubGlobal('fetch', vi.fn((url) => {
    if (url.includes('/participants/join')) {
      return Promise.resolve({ ok: true, json: async () => ({ joinSecret: 'sec-1' }) })
    }
    if (url.includes('/events/e1')) {
      return Promise.resolve({
        ok: true,
        json: async () => ({
          id: 'e1',
          name: 'Demo Event',
          modules: [
            { id: 'f1', name: 'Form 1', type: 'form', isActive: true, data: {} },
            { id: 'f2', name: 'Form 2', type: 'form', isActive: true, data: {} },
          ],
          dynamicCards: [],
        }),
      })
    }
    return Promise.resolve({ ok: false })
  }))
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  sessionStorage.clear()
  localStorage.clear()
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('EventScreen module completion timer (Bug #10)', () => {
  it('clears pending completionTimer when navigating back and opening another module', async () => {
    wrapper = mount(EventScreen, mountOpts)
    await flushPromises()

    const cards = wrapper.findAll('.module-card')
    expect(cards).toHaveLength(2)

    await cards[0].trigger('click')
    await flushPromises()
    expect(wrapper.find('.form-module-stub').attributes('data-id')).toBe('f1')

    await wrapper.find('.complete-btn').trigger('click')
    await flushPromises()

    await wrapper.find('.back-link').trigger('click')
    await flushPromises()
    await wrapper.findAll('.module-card')[1].trigger('click')
    await flushPromises()
    expect(wrapper.find('.form-module-stub').attributes('data-id')).toBe('f2')

    await vi.advanceTimersByTimeAsync(2500)
    await flushPromises()
    expect(wrapper.find('.form-module-stub').exists()).toBe(true)
    expect(wrapper.find('.form-module-stub').attributes('data-id')).toBe('f2')
  })
})
