import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import FormModule from '../src/components/FormModule.vue'

const { toastMock } = vi.hoisted(() => ({
  toastMock: {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
  },
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: k => k, locale: { value: 'en' } }),
}))

vi.mock('../src/composables/useToast', () => ({
  useToast: () => toastMock,
}))

vi.mock('../src/utils', () => ({
  API_BASE_URL: 'http://test.local/api',
  getAuthHeaders: () => ({}),
}))

let wrapper = null
let fetchMock = null

function mountForm(timeline) {
  return mount(FormModule, {
    props: {
      module: { id: 'mod-1', data: { timeline } },
      eventId: 'evt-1',
      userId: 'user-1',
    },
    global: {
      mocks: { $t: k => k },
      stubs: { AppIcon: true },
    },
  })
}

beforeEach(() => {
  toastMock.success.mockClear()
  toastMock.error.mockClear()
  toastMock.info.mockClear()

  fetchMock = vi.fn(async (url) => {
    if (String(url).endsWith('/templates')) {
      return { ok: true, json: async () => [] }
    }
    return { ok: true, json: async () => ({ id: 'resp-1' }) }
  })
  vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => {
  if (wrapper) {
    wrapper.unmount()
    wrapper = null
  }
  vi.unstubAllGlobals()
})

describe('FormModule "Other" answers, validation, and checkbox selection mode', () => {
  it('requires custom text for radio "Other" and submits resolved text instead of __other__', async () => {
    wrapper = mountForm([
      {
        id: 'step-radio',
        type: 'field',
        data: {
          label: 'Favorite color',
          fieldType: 'radio',
          allowOther: true,
          predefinedAnswers: [{ id: '1', text: 'Red', value: 'red' }],
        },
      },
    ])
    await flushPromises()

    const options = wrapper.findAll('.radio-option')
    await options[1].trigger('click') // "Other"

    // Empty custom text should fail validation
    await wrapper.find('.next-button').trigger('click')
    expect(toastMock.info).toHaveBeenCalledWith('form_module.validation.specify_other')
    expect(fetchMock).toHaveBeenCalledTimes(1) // Only initial /templates fetch

    // Whitespace-only custom text should also fail validation
    await wrapper.find('.other-input').setValue('   ')
    await wrapper.find('.next-button').trigger('click')
    expect(toastMock.info).toHaveBeenCalledTimes(2)
    expect(fetchMock).toHaveBeenCalledTimes(1)

    // Typing custom text allows submission and resolves __other__
    await wrapper.find('.other-input').setValue('  Teal  ')
    await wrapper.find('.next-button').trigger('click')
    await flushPromises()

    const postCall = fetchMock.mock.calls.find(([url]) => String(url).endsWith('/response'))
    expect(postCall).toBeTruthy()
    const body = JSON.parse(postCall[1].body)
    expect(body.data).toEqual({ 'step-radio': 'Teal' })
    expect(body.otherAnswers).toEqual({ 'step-radio': '  Teal  ' })
    expect(wrapper.emitted('complete')?.[0]).toEqual([{ 'step-radio': 'Teal' }])
  })

  it('requires custom text for dropdown "Other" and submits resolved text', async () => {
    wrapper = mountForm([
      {
        id: 'step-dropdown',
        type: 'field',
        data: {
          label: 'Country',
          fieldType: 'dropdown',
          allowOther: true,
          predefinedAnswers: [{ id: '1', text: 'USA', value: 'usa' }],
        },
      },
    ])
    await flushPromises()

    await wrapper.find('select.dropdown-select').setValue('__other__')

    await wrapper.find('.next-button').trigger('click')
    expect(toastMock.info).toHaveBeenCalledWith('form_module.validation.specify_other')
    expect(fetchMock).toHaveBeenCalledTimes(1)

    await wrapper.find('.other-input').setValue('Moldova')
    await wrapper.find('.next-button').trigger('click')
    await flushPromises()

    const postCall = fetchMock.mock.calls.find(([url]) => String(url).endsWith('/response'))
    expect(postCall).toBeTruthy()
    const body = JSON.parse(postCall[1].body)
    expect(body.data).toEqual({ 'step-dropdown': 'Moldova' })
  })

  it('enforces single-choice selection for checkboxes when allowMultiple is false or omitted', async () => {
    wrapper = mountForm([
      {
        id: 'step-cb-single',
        type: 'field',
        data: {
          label: 'Pick one',
          fieldType: 'checkbox',
          allowMultiple: false,
          allowOther: true,
          predefinedAnswers: [
            { id: '1', text: 'Alpha', value: 'alpha' },
            { id: '2', text: 'Beta', value: 'beta' },
          ],
        },
      },
    ])
    await flushPromises()

    const options = wrapper.findAll('.checkbox-option')
    await options[0].trigger('click') // Alpha
    expect(wrapper.findAll('.selection-indicator.selected')).toHaveLength(1)

    await options[1].trigger('click') // Beta replaces Alpha
    expect(wrapper.findAll('.selection-indicator.selected')).toHaveLength(1)
    expect(options[0].find('.selection-indicator').classes()).not.toContain('selected')
    expect(options[1].find('.selection-indicator').classes()).toContain('selected')

    await options[2].trigger('click') // Other replaces Beta
    expect(wrapper.findAll('.selection-indicator.selected')).toHaveLength(1)
    expect(options[1].find('.selection-indicator').classes()).not.toContain('selected')
    expect(options[2].find('.selection-indicator').classes()).toContain('selected')

    await wrapper.find('.other-input').setValue('Gamma')
    await wrapper.find('.next-button').trigger('click')
    await flushPromises()

    const postCall = fetchMock.mock.calls.find(([url]) => String(url).endsWith('/response'))
    const body = JSON.parse(postCall[1].body)
    expect(body.data).toEqual({ 'step-cb-single': ['Gamma'] })
  })

  it('allows multiple selections for checkboxes when allowMultiple is true and resolves Other', async () => {
    wrapper = mountForm([
      {
        id: 'step-cb-multi',
        type: 'field',
        data: {
          label: 'Pick many',
          fieldType: 'checkbox',
          allowMultiple: true,
          allowOther: true,
          predefinedAnswers: [
            { id: '1', text: 'Alpha', value: 'alpha' },
            { id: '2', text: 'Beta', value: 'beta' },
          ],
        },
      },
    ])
    await flushPromises()

    const options = wrapper.findAll('.checkbox-option')
    await options[0].trigger('click') // Alpha
    await options[1].trigger('click') // Beta
    await options[2].trigger('click') // Other
    expect(wrapper.findAll('.selection-indicator.selected')).toHaveLength(3)

    await wrapper.find('.other-input').setValue('Delta')
    await wrapper.find('.next-button').trigger('click')
    await flushPromises()

    const postCall = fetchMock.mock.calls.find(([url]) => String(url).endsWith('/response'))
    const body = JSON.parse(postCall[1].body)
    expect(body.data).toEqual({ 'step-cb-multi': ['alpha', 'beta', 'Delta'] })
  })
})
