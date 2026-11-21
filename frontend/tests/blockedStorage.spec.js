import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'

const originalLocalStorage = Object.getOwnPropertyDescriptor(window, 'localStorage')
const originalSessionStorage = Object.getOwnPropertyDescriptor(window, 'sessionStorage')

const blockStorageGetters = () => {
  const deny = () => {
    throw new DOMException('Blocked', 'SecurityError')
  }
  Object.defineProperty(window, 'localStorage', { configurable: true, get: deny })
  Object.defineProperty(window, 'sessionStorage', { configurable: true, get: deny })
  Object.defineProperty(globalThis, 'localStorage', { configurable: true, get: deny })
  Object.defineProperty(globalThis, 'sessionStorage', { configurable: true, get: deny })
}

const restoreStorage = () => {
  if (originalLocalStorage) Object.defineProperty(window, 'localStorage', originalLocalStorage)
  if (originalSessionStorage) Object.defineProperty(window, 'sessionStorage', originalSessionStorage)
}

afterEach(() => {
  restoreStorage()
  vi.unstubAllGlobals()
  vi.resetModules()
})

describe('Blocked browser storage resilience (Bug #20)', () => {
  it('loads i18n, session, useAdminTheme, and mounts App.vue without crashing when storage getters throw SecurityError', async () => {
    vi.resetModules()
    blockStorageGetters()
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({ status: 401, ok: false })))

    const { default: i18n } = await import('../src/i18n.js')
    const { loadSession, session } = await import('../src/session.js')
    const { useAdminTheme } = await import('../src/composables/useAdminTheme.js')
    const { default: App } = await import('../src/App.vue')

    expect(i18n.global.locale.value).toBe('en')
    const { theme } = useAdminTheme()
    expect(theme.value).toBe('light')

    await expect(loadSession(true)).resolves.toBeNull()
    expect(session.loaded).toBe(true)

    const wrapper = mount(App, {
      global: {
        stubs: {
          'router-view': true,
          Toast: true,
          AdminThemeToggle: true,
        },
      },
    })
    await flushPromises()
    expect(typeof wrapper.vm.getUserId()).toBe('string')
    expect(wrapper.vm.getUserId().length).toBeGreaterThan(0)
    wrapper.unmount()
  })
})
