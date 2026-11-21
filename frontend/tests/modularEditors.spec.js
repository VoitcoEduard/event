import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import i18n from '../src/i18n.js'
import { clearDialogQueue } from '../src/composables/useDialog.js'
import AppDialogHost from '../src/components/AppDialogHost.vue'
import QuizEditor from '../src/components/QuizEditor.vue'
import QuizQuestionEditor from '../src/components/QuizQuestionEditor.vue'
import QuestionTypeSelector from '../src/components/QuestionTypeSelector.vue'
import WrongPicturesEditor from '../src/components/WrongPicturesEditor.vue'
import FormEditor from '../src/components/FormEditor.vue'

let wrappers = []

function track(wrapper) {
  wrappers.push(wrapper)
  return wrapper
}

beforeEach(() => {
  i18n.global.locale.value = 'en'
  clearDialogQueue()
})

afterEach(() => {
  clearDialogQueue()
  while (wrappers.length) {
    wrappers.pop().unmount()
  }
  document.body.innerHTML = ''
  vi.restoreAllMocks()
})

describe('QuestionTypeSelector', () => {
  it('displays concise 1-line descriptions directly inside each type card without hover footer', () => {
    const wrapper = track(mount(QuestionTypeSelector, {
      props: { visible: true },
      global: {
        plugins: [i18n],
        stubs: { Teleport: true, AppIcon: true }
      }
    }))

    const cards = wrapper.findAll('button.type-card')
    expect(cards.length).toBe(8)
    for (const card of cards) {
      const desc = card.find('.type-desc')
      expect(desc.exists()).toBe(true)
      expect(desc.text().trim().length).toBeGreaterThan(0)
    }
    expect(wrapper.find('.description-area').exists()).toBe(false)
  })
})

describe('QuizEditor', () => {
  it('provides quick-add + Add Classic and + Other Type toolbar', async () => {
    const wrapper = track(mount(QuizEditor, {
      props: { questions: [] },
      global: {
        plugins: [i18n],
        stubs: { Teleport: true, AppIcon: true, ImageUploader: true, CoordinatePicker: true, RangeComponent: true, ImageOptionEditor: true }
      }
    }))

    await wrapper.find('.add-classic-btn').trigger('click')
    expect(wrapper.emitted('add-question')).toEqual([['classic']])
    expect(wrapper.find('.modal-container').exists()).toBe(false)

    await wrapper.find('.add-other-btn').trigger('click')
    expect(wrapper.findAll('.type-card').length).toBe(8)
  })

  it('supports lockedType for direct add and non-clickable type badge', async () => {
    const wrapper = track(mount(QuizEditor, {
      props: {
        lockedType: 'simple_text',
        questions: [
          { id: 'q1', type: 'simple_text', text: 'Explain photosynthesis', scoringConfig: { maxPoints: 100 } }
        ]
      },
      global: {
        plugins: [i18n],
        stubs: { Teleport: true, AppIcon: true, ImageUploader: true }
      }
    }))

    expect(wrapper.find('.add-other-btn').exists()).toBe(false)
    await wrapper.find('.add-classic-btn').trigger('click')
    expect(wrapper.emitted('add-question')).toEqual([['simple_text']])

    const badge = wrapper.find('.question-type-tag')
    expect(badge.element.tagName).toBe('SPAN')
    expect(badge.classes()).toContain('question-type-tag--locked')
  })

  it('renders question outline with error dots, tooltips, and scrolls to selected question', async () => {
    const scrollSpy = vi.fn()
    Element.prototype.scrollIntoView = scrollSpy

    const wrapper = track(mount(QuizEditor, {
      props: {
        showOutline: true,
        activeIndex: 0,
        errorsByIndex: {
          1: { text: { index: 1, field: 'text', messageKey: 'quiz_config.validation.text_required', params: {} } }
        },
        questions: [
          { id: 'q1', type: 'classic', text: 'Capital of France?', answers: ['Paris', 'Lyon', 'Nice', 'Marseille'], correctIndex: 0, scoringConfig: { maxPoints: 50 } },
          { id: 'q2', type: 'range', text: '', rangeConfig: { minValue: 0, maxValue: 100, step: 1 }, scoringConfig: { maxPoints: 100 }, correctValue: 50 }
        ]
      },
      global: {
        plugins: [i18n],
        stubs: { Teleport: true, AppIcon: true, ImageUploader: true, CoordinatePicker: true, RangeComponent: true, ImageOptionEditor: true }
      }
    }))

    const outlineItems = wrapper.findAll('.outline-item')
    expect(outlineItems).toHaveLength(2)
    expect(outlineItems[0].classes()).toContain('is-active')
    expect(outlineItems[0].find('.outline-error-dot').exists()).toBe(false)
    expect(outlineItems[1].classes()).toContain('has-error')
    expect(outlineItems[1].find('.outline-error-dot').exists()).toBe(true)

    const moveButtons = wrapper.findAll('.question-sort-controls .move-btn')
    expect(moveButtons[0].attributes('title')).toBe('Move question up')
    expect(moveButtons[1].attributes('title')).toBe('Move question down')

    await outlineItems[1].find('.outline-select-btn').trigger('click')
    await flushPromises()
    expect(wrapper.emitted('update:activeIndex')).toEqual([[1]])
    expect(wrapper.emitted('select-question')).toEqual([[1]])
    expect(scrollSpy).toHaveBeenCalled()
  })
})

describe('QuizQuestionEditor', () => {
  it('clears localized True/False answers when switching away from true_false in non-English locales', async () => {
    i18n.global.locale.value = 'ro'
    const wrapper = track(mount(QuizQuestionEditor, {
      props: {
        index: 0,
        question: {
          id: 'tf-1',
          type: 'true_false',
          text: 'Earth is round',
          answers: ['Adevărat', 'Fals', '', ''],
          correctIndex: 0,
          scoringConfig: { maxPoints: 50 }
        }
      },
      global: {
        plugins: [i18n],
        stubs: { AppIcon: true, ImageUploader: true }
      }
    }))

    await wrapper.setProps({
      question: {
        id: 'tf-1',
        type: 'classic',
        text: 'Earth is round',
        answers: ['Adevărat', 'Fals', '', ''],
        correctIndex: 0,
        scoringConfig: { maxPoints: 50 }
      }
    })
    await flushPromises()

    const emitted = wrapper.emitted('update')
    const latest = emitted[emitted.length - 1][0]
    expect(latest.answers).toEqual(['', '', '', ''])
  })

  it('shows visible Correct label, min=1 on points, and inline error rings on failing fields', () => {
    const wrapper = track(mount(QuizQuestionEditor, {
      props: {
        index: 0,
        errors: {
          points: { index: 0, field: 'points', messageKey: 'quiz_config.validation.points_positive', params: {} },
          text: { index: 0, field: 'text', messageKey: 'quiz_config.validation.text_required', params: {} },
          correctIndices: { index: 0, field: 'correctIndices', messageKey: 'quiz_config.validation.multi_select_min_correct', params: { min: 2 } }
        },
        question: {
          id: 'ms-1',
          type: 'multi_select',
          text: '',
          answers: ['A', 'B', '', ''],
          correctIndices: [0],
          scoringConfig: { maxPoints: 0 }
        }
      },
      global: {
        plugins: [i18n],
        stubs: { AppIcon: true, ImageUploader: true }
      }
    }))

    expect(wrapper.find('.question-editor').classes()).toContain('has-errors')
    const pointsInput = wrapper.find('input[type="number"][min="1"]')
    expect(pointsInput.exists()).toBe(true)
    expect(pointsInput.classes()).toContain('has-field-error')
    expect(wrapper.find('.main-input').classes()).toContain('has-field-error')
    expect(wrapper.find('.answers-grid').classes()).toContain('has-field-error')

    const correctLabels = wrapper.findAll('.correct-label')
    expect(correctLabels.length).toBeGreaterThanOrEqual(2)
    expect(correctLabels[0].text()).toBe('Correct')
  })
})

describe('WrongPicturesEditor', () => {
  it('adds rounds, renders inline errors, and uses useDialog().confirm for round removal', async () => {
    track(mount(AppDialogHost, {
      attachTo: document.body,
      global: { plugins: [i18n] }
    }))

    const wrapper = track(mount(WrongPicturesEditor, {
      props: {
        rounds: [
          {
            id: 'wp-1',
            type: 'wrong_pictures',
            text: '',
            timeLimit: 60,
            imageId: null,
            wrongPictureConfig: { targets: [], maxPoints: 0, tolerancePercent: 5, showTargetCount: true }
          }
        ],
        errorsByIndex: {
          0: {
            text: { index: 0, field: 'text', messageKey: 'quiz_config.validation.wp_title_required', params: {} },
            imageId: { index: 0, field: 'imageId', messageKey: 'quiz_config.validation.wp_image_required', params: {} }
          }
        }
      },
      global: {
        plugins: [i18n],
        stubs: { AppIcon: true, ImageUploader: true, CoordinatePicker: true }
      }
    }))

    expect(wrapper.find('.round-header h3').text()).toBe('Round 1')
    expect(wrapper.find('.round-card').classes()).toContain('has-errors')
    expect(wrapper.find('.image-required-warning').classes()).toContain('has-field-error')

    await wrapper.find('.add-round-btn').trigger('click')
    const emittedAdd = wrapper.emitted('update:rounds')
    expect(emittedAdd[0][0]).toHaveLength(2)

    await wrapper.setProps({ rounds: emittedAdd[0][0] })
    await wrapper.findAll('.remove-btn')[1].trigger('click')
    await flushPromises()

    const dialogTitle = document.body.querySelector('.app-dialog-title')?.textContent.trim()
    expect(dialogTitle).toBe('Remove round?')
    document.body.querySelector('.app-dialog-btn--danger').click()
    await flushPromises()

    const emittedAfterRemove = wrapper.emitted('update:rounds')
    expect(emittedAfterRemove[emittedAfterRemove.length - 1][0]).toHaveLength(1)
  })
})

describe('FormEditor', () => {
  it('uses useDialog().confirm with step-deletion copy and supports pageFlow', async () => {
    track(mount(AppDialogHost, {
      attachTo: document.body,
      global: { plugins: [i18n] }
    }))

    const wrapper = track(mount(FormEditor, {
      props: {
        pageFlow: true,
        timeline: [
          { id: 's1', type: 'field', data: { label: 'Name', fieldType: 'text' } }
        ]
      },
      global: {
        plugins: [i18n],
        stubs: { AppIcon: true }
      }
    }))

    expect(wrapper.find('.form-editor').classes()).toContain('form-editor--page-flow')

    await wrapper.find('.delete-btn').trigger('click')
    await flushPromises()

    const message = document.body.querySelector('.app-dialog-message')?.textContent.trim()
    expect(message).toBe('Are you sure you want to delete this step from the form?')
    document.body.querySelector('.app-dialog-btn--danger').click()
    await flushPromises()

    expect(wrapper.emitted('update:timeline')).toEqual([[[]]])
  })
})
