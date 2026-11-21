import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import QuizModuleConfig from '../src/components/QuizModuleConfig.vue'

const { mockSuccess, mockError, mockInfo } = vi.hoisted(() => ({
  mockSuccess: vi.fn(),
  mockError: vi.fn(),
  mockInfo: vi.fn(),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: k => k }),
}))

vi.mock('../src/utils', () => ({
  API_BASE_URL: '',
  getAuthHeaders: () => ({ Authorization: 'Bearer test' }),
  normalizeTemplates: data => (Array.isArray(data) ? data : []),
}))

vi.mock('../src/composables/useToast', () => ({
  useToast: () => ({ success: mockSuccess, error: mockError, info: mockInfo }),
}))

vi.mock('../src/services/media', () => ({
  getTracks: vi.fn().mockResolvedValue([]),
  getImageUrl: id => `/media/${id}`,
}))

let wrapper = null
let fetchMock = null

function mountConfig(questions = []) {
  wrapper = mount(QuizModuleConfig, {
    props: {
      visible: true,
      eventId: 'evt-1',
      module: {
        id: 'mod-1',
        name: 'Test Quiz',
        data: { questions },
      },
    },
    global: {
      mocks: { $t: k => k },
      stubs: {
        Teleport: true,
        AppIcon: true,
        ImageUploader: true,
        CoordinatePicker: true,
        RangeComponent: true,
        ImageOptionEditor: true,
      },
    },
  })
  return wrapper
}

beforeEach(() => {
  mockSuccess.mockReset()
  mockError.mockReset()
  mockInfo.mockReset()
  fetchMock = vi.fn((url) => {
    if (url === '/templates') {
      return Promise.resolve({ ok: true, json: async () => [] })
    }
    return Promise.resolve({ ok: true, json: async () => ({}) })
  })
  vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = null
  vi.unstubAllGlobals()
})

describe('QuizModuleConfig save validation', () => {
  it('allows saving a wrong_pictures question without filling the 4 hidden text answers', async () => {
    mountConfig([
      {
        id: 'wp-1',
        type: 'wrong_pictures',
        text: 'Find the anachronisms',
        answers: ['', '', '', ''],
        imageId: 'img-wp',
        wrongPictureConfig: {
          targets: [{ id: 't1', label: 'Sneakers', xPercent: 40, yPercent: 60, radius: 8 }],
          tolerancePercent: 10,
          maxPoints: 50,
        },
      },
    ])
    await flushPromises()

    await wrapper.find('.publish-btn').trigger('click')
    await flushPromises()

    expect(mockError).not.toHaveBeenCalled()
    expect(mockSuccess).toHaveBeenCalledWith('quiz_config.saved_success')
    expect(fetchMock).toHaveBeenCalledWith(
      '/events/evt-1/modules/mod-1/quiz/questions',
      expect.objectContaining({ method: 'PUT' })
    )
    expect(fetchMock).toHaveBeenCalledWith(
      '/events/evt-1/modules/mod-1',
      expect.objectContaining({ method: 'PUT' })
    )
  })

  it('allows saving a newly added wrong_pictures question once text is provided', async () => {
    mountConfig([])
    await flushPromises()

    await wrapper.find('.add-btn').trigger('click')
    const wrongPicturesCard = wrapper
      .findAll('.type-card')
      .find(btn => btn.text().includes('quiz_config.question_editor.types.wrong_pictures'))
    expect(wrongPicturesCard).toBeTruthy()
    await wrongPicturesCard.trigger('click')

    await wrapper.find('.main-input').setValue('Spot what is wrong')
    await wrapper.find('.publish-btn').trigger('click')
    await flushPromises()

    expect(mockError).not.toHaveBeenCalled()
    expect(mockSuccess).toHaveBeenCalledWith('quiz_config.saved_success')
  })

  it('still requires question text for wrong_pictures questions', async () => {
    mountConfig([
      {
        id: 'wp-2',
        type: 'wrong_pictures',
        text: '   ',
        answers: ['', '', '', ''],
        imageId: 'img-wp',
        wrongPictureConfig: { targets: [], tolerancePercent: 10, maxPoints: 50 },
      },
    ])
    await flushPromises()

    await wrapper.find('.publish-btn').trigger('click')
    await flushPromises()

    expect(mockError).toHaveBeenCalledWith('quiz_config.validation.text_required')
    expect(mockSuccess).not.toHaveBeenCalled()
  })

  it('allows saving an image-only simple_text question without question text', async () => {
    mountConfig([
      {
        id: 'st-1',
        type: 'simple_text',
        text: '',
        imageId: 'img-only-1',
        answers: ['', '', '', ''],
        timeLimit: 0,
        explainTime: 0,
        scoringConfig: { mode: 'symmetric', maxPoints: 100, tolerance: 0, partialPercent: 0 },
      },
    ])
    await flushPromises()

    await wrapper.find('.publish-btn').trigger('click')
    await flushPromises()

    expect(mockError).not.toHaveBeenCalled()
    expect(mockSuccess).toHaveBeenCalledWith('quiz_config.saved_success')
    expect(fetchMock).toHaveBeenCalledWith(
      '/events/evt-1/modules/mod-1/quiz/questions',
      expect.objectContaining({ method: 'PUT' })
    )
  })

  it('rejects a simple_text question with neither text nor image', async () => {
    mountConfig([
      {
        id: 'st-2',
        type: 'simple_text',
        text: '   ',
        imageId: null,
        answers: ['', '', '', ''],
        scoringConfig: { mode: 'symmetric', maxPoints: 100, tolerance: 0, partialPercent: 0 },
      },
    ])
    await flushPromises()

    await wrapper.find('.publish-btn').trigger('click')
    await flushPromises()

    expect(mockError).toHaveBeenCalledWith('quiz_config.validation.simple_text_required')
    expect(mockSuccess).not.toHaveBeenCalled()
  })

  it('rejects a simple_text image-only question when maxPoints is not positive', async () => {
    mountConfig([
      {
        id: 'st-3',
        type: 'simple_text',
        text: '',
        imageId: 'img-only-2',
        answers: ['', '', '', ''],
        scoringConfig: { mode: 'symmetric', maxPoints: 0, tolerance: 0, partialPercent: 0 },
      },
    ])
    await flushPromises()

    await wrapper.find('.publish-btn').trigger('click')
    await flushPromises()

    expect(mockError).toHaveBeenCalledWith('quiz_config.validation.simple_points')
    expect(mockSuccess).not.toHaveBeenCalled()
  })

  it('still requires all 4 answer options for classic questions', async () => {
    mountConfig([
      {
        id: 'cl-1',
        type: 'classic',
        text: 'What is 2 + 2?',
        answers: ['3', '4', '', ''],
        correctIndex: 1,
      },
    ])
    await flushPromises()

    await wrapper.find('.publish-btn').trigger('click')
    await flushPromises()

    expect(mockError).toHaveBeenCalledWith('quiz_config.validation.mc_options')
    expect(mockSuccess).not.toHaveBeenCalled()
  })
})
