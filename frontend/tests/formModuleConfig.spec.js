import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import FormModuleConfig from '../src/components/FormModuleConfig.vue'

const { toastMock } = vi.hoisted(() => ({
  toastMock: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: k => k, locale: { value: 'en' } }),
}))

vi.mock('../src/composables/useToast', () => ({
  useToast: () => toastMock,
}))

vi.mock('../src/utils', () => ({
  API_BASE_URL: '/api',
  getAuthHeaders: () => ({}),
  normalizeTemplates: d => d || [],
}))

let wrapper = null

beforeEach(() => {
  toastMock.success.mockClear()
  toastMock.error.mockClear()
  toastMock.info.mockClear()
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  vi.unstubAllGlobals()
})

describe('FormModuleConfig metadata preservation (Bug #15)', () => {
  it('preserves manualScoreTarget and existing metadata when publishing form changes', async () => {
    const fetchMock = vi.fn(async () => ({ ok: true, json: async () => ({ message: 'ok' }) }))
    vi.stubGlobal('fetch', fetchMock)

    wrapper = mount(FormModuleConfig, {
      props: {
        visible: true,
        eventId: 'evt-1',
        module: {
          id: 'mod-form-1',
          name: 'Feedback Form',
          type: 'form',
          data: {
            manualScoreTarget: 'teams',
            timeline: [{ id: 'step-1', type: 'field', data: { label: 'Name', fieldType: 'text' } }],
            mandatoryOnEntry: false,
            hideAfterCompletion: false,
          },
        },
      },
      global: {
        mocks: { $t: k => k },
        stubs: {
          Teleport: true,
          FormEditor: true,
          AppIcon: true,
        },
      },
    })
    await flushPromises()

    await wrapper.find('.publish-btn').trigger('click')
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, options] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/events/evt-1/modules/mod-form-1/publish')
    expect(options.method).toBe('PUT')
    const payload = JSON.parse(options.body)
    expect(payload.data.manualScoreTarget).toBe('teams')
    expect(payload.data.timeline).toHaveLength(1)
    expect(payload.data.mandatoryOnEntry).toBe(false)
    expect(payload.data.hideAfterCompletion).toBe(false)
  })
})
