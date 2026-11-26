import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import i18n from '../src/i18n.js'
import { useToast } from '../src/composables/useToast.js'
import {
  extractImageFromClipboardData,
  hasActiveBlockingModal,
  isEditableElement,
  readImageFromClipboard
} from '../src/utils/clipboard.js'

vi.mock('../src/services/media.js', () => ({
  uploadImage: vi.fn(),
  getImageUrl: vi.fn((id) => (id ? `/media/${id}/file` : null))
}))

import { uploadImage } from '../src/services/media.js'
import ImageUploader from '../src/components/ImageUploader.vue'
import QuizEditor from '../src/components/QuizEditor.vue'
import QuizQuestionEditor from '../src/components/QuizQuestionEditor.vue'
import ImageOptionEditor from '../src/components/ImageOptionEditor.vue'
import WrongPicturesEditor from '../src/components/WrongPicturesEditor.vue'

const wrappers = []
function track(wrapper) {
  wrappers.push(wrapper)
  return wrapper
}

function clearToasts() {
  const { toasts, remove } = useToast()
  for (const t of [...toasts.value]) {
    remove(t.id)
  }
}

function makePasteEvent({ items = [], files = [], text = '' } = {}) {
  const event = new Event('paste', { bubbles: true, cancelable: true })
  const clipboardData = {
    items,
    files,
    getData: (type) => (type === 'text/plain' ? text : '')
  }
  Object.defineProperty(event, 'clipboardData', {
    value: clipboardData,
    configurable: true
  })
  return event
}

function makeImageItem(type = 'image/png', blob = new Blob(['img-bytes'], { type })) {
  return {
    kind: 'file',
    type,
    getAsFile: () => blob
  }
}

beforeEach(() => {
  i18n.global.locale.value = 'en'
  clearToasts()
  vi.clearAllMocks()
  uploadImage.mockResolvedValue({ id: 'img-uploaded-1' })
})

afterEach(() => {
  while (wrappers.length) {
    wrappers.pop().unmount()
  }
  document.body.innerHTML = ''
  clearToasts()
  vi.unstubAllGlobals()
})

describe('clipboard image utilities', () => {
  it('extracts image from clipboardData.items with preferred MIME order and skips SVG', () => {
    const svgItem = makeImageItem('image/svg+xml', new Blob(['<svg/>'], { type: 'image/svg+xml' }))
    expect(extractImageFromClipboardData({ items: [svgItem] })).toBeNull()

    const pngBlob = new Blob(['png'], { type: 'image/png' })
    const file = extractImageFromClipboardData({
      items: [svgItem, makeImageItem('image/png', pngBlob)]
    })
    expect(file).toBeInstanceOf(File)
    expect(file.type).toBe('image/png')
    expect(file.name).toMatch(/^pasted-image-\d+\.png$/)
  })

  it('extracts image from clipboardData.files fallback and respects ignoreWhenTextPresent', () => {
    const jpgFile = new File(['jpg'], 'photo.jpg', { type: 'image/jpeg' })
    const extracted = extractImageFromClipboardData({ files: [jpgFile] })
    expect(extracted).toBe(jpgFile)

    const ignored = extractImageFromClipboardData(
      {
        items: [makeImageItem('image/png')],
        getData: (type) => (type === 'text/plain' ? 'Copied cell text' : '')
      },
      { ignoreWhenTextPresent: true }
    )
    expect(ignored).toBeNull()
  })

  it('hasActiveBlockingModal detects every blocking overlay flavor (L4)', () => {
    // Nothing open: no blocking modal.
    expect(hasActiveBlockingModal(null)).toBe(false)

    const target = document.createElement('div')
    target.className = 'editor-input'
    document.body.appendChild(target)

    // AppDialogHost overlay (app dialogs replacing window.confirm).
    const appOverlay = document.createElement('div')
    appOverlay.className = 'app-dialog-overlay'
    document.body.appendChild(appOverlay)
    expect(hasActiveBlockingModal(target)).toBe(true)
    appOverlay.remove()

    // Generic modal overlay (QuestionTypeSelector / AddModuleModal).
    const modalOverlay = document.createElement('div')
    modalOverlay.className = 'modal-overlay'
    document.body.appendChild(modalOverlay)
    expect(hasActiveBlockingModal(target)).toBe(true)
    modalOverlay.remove()

    // ARIA dialog semantics alone must also block.
    const ariaDialog = document.createElement('div')
    ariaDialog.setAttribute('role', 'dialog')
    ariaDialog.setAttribute('aria-modal', 'true')
    document.body.appendChild(ariaDialog)
    expect(hasActiveBlockingModal(target)).toBe(true)
    ariaDialog.remove()

    target.remove()
  })

  it('readImageFromClipboard handles ok, empty, unsupported, and denied states', async () => {
    vi.stubGlobal('navigator', {})
    expect((await readImageFromClipboard()).status).toBe('unsupported')

    vi.stubGlobal('navigator', {
      clipboard: {
        read: vi.fn(async () => [
          {
            types: ['text/plain', 'image/webp'],
            getType: async () => new Blob(['webp'], { type: 'image/webp' })
          }
        ])
      }
    })
    const okRes = await readImageFromClipboard()
    expect(okRes.status).toBe('ok')
    expect(okRes.file).toBeInstanceOf(File)
    expect(okRes.file.type).toBe('image/webp')

    navigator.clipboard.read.mockResolvedValueOnce([
      { types: ['text/plain'], getType: async () => new Blob(['hi']) }
    ])
    expect((await readImageFromClipboard()).status).toBe('empty')

    const deniedErr = new Error('Denied')
    deniedErr.name = 'NotAllowedError'
    navigator.clipboard.read.mockRejectedValueOnce(deniedErr)
    expect((await readImageFromClipboard()).status).toBe('denied')
  })

  it('isEditableElement distinguishes text inputs/textareas from non-text inputs', () => {
    const textInput = document.createElement('input')
    textInput.type = 'text'
    const radioInput = document.createElement('input')
    radioInput.type = 'radio'
    const textarea = document.createElement('textarea')
    const button = document.createElement('button')

    expect(isEditableElement(textInput)).toBe(true)
    expect(isEditableElement(textarea)).toBe(true)
    expect(isEditableElement(radioInput)).toBe(false)
    expect(isEditableElement(button)).toBe(false)
  })
})

describe('ImageUploader clipboard paste', () => {
  it('uploads image from navigator.clipboard.read when clicking Paste from clipboard', async () => {
    vi.stubGlobal('navigator', {
      clipboard: {
        read: vi.fn(async () => [
          {
            types: ['image/png'],
            getType: async () => new Blob(['png-data'], { type: 'image/png' })
          }
        ])
      }
    })

    const wrapper = track(mount(ImageUploader, {
      props: { modelValue: null, eventId: 'evt-1', moduleId: 'mod-1' },
      global: { plugins: [i18n], stubs: { AppIcon: true } }
    }))

    await wrapper.find('.paste-btn').trigger('click')
    await flushPromises()

    expect(uploadImage).toHaveBeenCalledTimes(1)
    expect(uploadImage.mock.calls[0][0]).toBeInstanceOf(File)
    expect(uploadImage.mock.calls[0][1]).toBe('evt-1')
    expect(uploadImage.mock.calls[0][2]).toBe('mod-1')
    expect(wrapper.emitted('update:modelValue')).toEqual([['img-uploaded-1']])

    const { toasts } = useToast()
    expect(toasts.value.some((t) => t.type === 'success')).toBe(true)
  })

  it('shows warning toast when clipboard has no image', async () => {
    vi.stubGlobal('navigator', {
      clipboard: {
        read: vi.fn(async () => [{ types: ['text/plain'], getType: async () => new Blob(['abc']) }])
      }
    })

    const wrapper = track(mount(ImageUploader, {
      props: { modelValue: null, eventId: 'evt-1', moduleId: 'mod-1' },
      global: { plugins: [i18n], stubs: { AppIcon: true } }
    }))

    await wrapper.find('.paste-btn').trigger('click')
    await flushPromises()

    expect(uploadImage).not.toHaveBeenCalled()
    const { toasts } = useToast()
    expect(toasts.value.some((t) => t.type === 'warning')).toBe(true)
  })

  it('falls back to Ctrl+V awaiting-paste mode when clipboard.read is unavailable or denied, then uploads on paste event', async () => {
    vi.stubGlobal('navigator', {})

    const wrapper = track(mount(ImageUploader, {
      attachTo: document.body,
      props: { modelValue: null, eventId: 'evt-1', moduleId: 'mod-1' },
      global: { plugins: [i18n], stubs: { AppIcon: true } }
    }))

    await wrapper.find('.paste-btn').trigger('click')
    await flushPromises()

    expect(wrapper.find('.upload-zone').classes()).toContain('awaiting-paste')

    const pasteEvent = makePasteEvent({ items: [makeImageItem('image/png')] })
    const stopSpy = vi.spyOn(pasteEvent, 'stopPropagation')
    wrapper.element.dispatchEvent(pasteEvent)
    await flushPromises()

    expect(pasteEvent.defaultPrevented).toBe(true)
    expect(stopSpy).toHaveBeenCalled()
    expect(uploadImage).toHaveBeenCalledTimes(1)
    expect(wrapper.emitted('update:modelValue')).toEqual([['img-uploaded-1']])
    expect(wrapper.find('.upload-zone').classes()).not.toContain('awaiting-paste')
  })

  it('does not hijack plain text paste inside alt-text input', async () => {
    const wrapper = track(mount(ImageUploader, {
      props: { modelValue: 'existing-img', altText: '', eventId: 'evt-1', moduleId: 'mod-1' },
      global: { plugins: [i18n], stubs: { AppIcon: true } }
    }))

    const altInput = wrapper.find('.alt-input')
    const textWithImage = makePasteEvent({
      items: [makeImageItem('image/png')],
      text: 'Alt text description'
    })
    altInput.element.dispatchEvent(textWithImage)
    await flushPromises()

    expect(textWithImage.defaultPrevented).toBe(false)
    expect(uploadImage).not.toHaveBeenCalled()
  })

  it('replaces image from clipboard in preview mode and hidePreview compact mode', async () => {
    vi.stubGlobal('navigator', {
      clipboard: {
        read: vi.fn(async () => [
          {
            types: ['image/jpeg'],
            getType: async () => new Blob(['jpeg-data'], { type: 'image/jpeg' })
          }
        ])
      }
    })
    uploadImage.mockResolvedValueOnce({ id: 'img-replaced-1' }).mockResolvedValueOnce({ id: 'img-replaced-2' })

    const previewWrapper = track(mount(ImageUploader, {
      props: { modelValue: 'old-1', eventId: 'evt-1', moduleId: 'mod-1' },
      global: { plugins: [i18n], stubs: { AppIcon: true } }
    }))
    await previewWrapper.find('.paste-overlay-btn').trigger('click')
    await flushPromises()
    expect(previewWrapper.emitted('update:modelValue')).toEqual([['img-replaced-1']])

    const compactWrapper = track(mount(ImageUploader, {
      props: { modelValue: 'old-2', eventId: 'evt-1', moduleId: 'mod-1', hidePreview: true },
      global: { plugins: [i18n], stubs: { AppIcon: true } }
    }))
    await compactWrapper.find('.paste-compact-btn').trigger('click')
    await flushPromises()
    expect(compactWrapper.emitted('update:modelValue')).toEqual([['img-replaced-2']])
  })

  it('ignores paste button and paste events when disabled', async () => {
    vi.stubGlobal('navigator', {
      clipboard: {
        read: vi.fn(async () => [
          { types: ['image/png'], getType: async () => new Blob(['png'], { type: 'image/png' }) }
        ])
      }
    })

    const wrapper = track(mount(ImageUploader, {
      props: { modelValue: null, eventId: 'evt-1', moduleId: 'mod-1', disabled: true },
      global: { plugins: [i18n], stubs: { AppIcon: true } }
    }))

    await wrapper.find('.paste-btn').trigger('click')
    const pasteEvent = makePasteEvent({ items: [makeImageItem('image/png')] })
    wrapper.element.dispatchEvent(pasteEvent)
    await flushPromises()

    expect(uploadImage).not.toHaveBeenCalled()
    expect(pasteEvent.defaultPrevented).toBe(false)
  })
})

describe('Editor clipboard paste routing', () => {
  it('routes window paste only to the active question in QuizEditor', async () => {
    uploadImage.mockResolvedValue({ id: 'img-q2' })

    const wrapper = track(mount(QuizEditor, {
      attachTo: document.body,
      props: {
        activeIndex: 1,
        eventId: 'evt-1',
        moduleId: 'mod-1',
        questions: [
          { id: 'q1', type: 'classic', text: 'First', answers: ['A', 'B', 'C', 'D'], correctIndex: 0, scoringConfig: { maxPoints: 50 } },
          { id: 'q2', type: 'classic', text: 'Second', answers: ['A', 'B', 'C', 'D'], correctIndex: 0, scoringConfig: { maxPoints: 50 } }
        ]
      },
      global: {
        plugins: [i18n],
        stubs: { Teleport: true, AppIcon: true, CoordinatePicker: true, RangeComponent: true }
      }
    }))

    const pasteEvent = makePasteEvent({ items: [makeImageItem('image/png')] })
    window.dispatchEvent(pasteEvent)
    await flushPromises()

    expect(uploadImage).toHaveBeenCalledTimes(1)
    const emitted = wrapper.emitted('update-question')
    expect(emitted).toBeTruthy()
    expect(emitted[emitted.length - 1][0]).toBe(1)
    expect(emitted[emitted.length - 1][1].imageId).toBe('img-q2')
  })

  it('uploads image when pasting while focused in question text input if clipboard has no plain text', async () => {
    uploadImage.mockResolvedValue({ id: 'img-from-input' })

    const wrapper = track(mount(QuizQuestionEditor, {
      attachTo: document.body,
      props: {
        index: 0,
        eventId: 'evt-1',
        moduleId: 'mod-1',
        question: { id: 'q1', type: 'classic', text: 'Question', answers: ['A', 'B', 'C', 'D'], correctIndex: 0, scoringConfig: { maxPoints: 50 } }
      },
      global: { plugins: [i18n], stubs: { AppIcon: true } }
    }))

    const textInput = wrapper.find('.main-input')
    const imageOnlyPaste = makePasteEvent({ items: [makeImageItem('image/png')] })
    textInput.element.dispatchEvent(imageOnlyPaste)
    await flushPromises()

    expect(uploadImage).toHaveBeenCalledTimes(1)
    expect(imageOnlyPaste.defaultPrevented).toBe(true)

    uploadImage.mockClear()
    const textWithImagePaste = makePasteEvent({
      items: [makeImageItem('image/png')],
      text: 'Pasted question text'
    })
    textInput.element.dispatchEvent(textWithImagePaste)
    await flushPromises()

    expect(uploadImage).not.toHaveBeenCalled()
    expect(textWithImagePaste.defaultPrevented).toBe(false)
  })

  it('routes paste inside ImageOptionEditor option card only to that option', async () => {
    uploadImage.mockResolvedValue({ id: 'img-opt-2' })

    const wrapper = track(mount(ImageOptionEditor, {
      attachTo: document.body,
      props: {
        eventId: 'evt-1',
        moduleId: 'mod-1',
        correctIndex: 0,
        modelValue: [
          { id: '1', imageId: '', label: '' },
          { id: '2', imageId: '', label: '' }
        ]
      },
      global: { plugins: [i18n], stubs: { AppIcon: true } }
    }))

    const cards = wrapper.findAll('.image-option-card')
    const pasteEvent = makePasteEvent({ items: [makeImageItem('image/png')] })
    cards[1].element.dispatchEvent(pasteEvent)
    await flushPromises()

    expect(uploadImage).toHaveBeenCalledTimes(1)
    expect(wrapper.props('modelValue')[1].imageId).toBe('img-opt-2')
    expect(wrapper.props('modelValue')[0].imageId).toBe('')
  })

  it('routes window paste to active round in WrongPicturesEditor', async () => {
    uploadImage.mockResolvedValue({ id: 'img-round-1' })

    const wrapper = track(mount(WrongPicturesEditor, {
      attachTo: document.body,
      props: {
        activeIndex: 0,
        eventId: 'evt-1',
        moduleId: 'mod-1',
        rounds: [
          { id: 'r1', type: 'wrong_pictures', text: 'Find bugs', imageId: null, wrongPictureConfig: { targets: [], maxPoints: 50, tolerancePercent: 5 } }
        ]
      },
      global: {
        plugins: [i18n],
        stubs: { AppIcon: true, CoordinatePicker: true }
      }
    }))

    const pasteEvent = makePasteEvent({ items: [makeImageItem('image/png')] })
    window.dispatchEvent(pasteEvent)
    await flushPromises()

    expect(uploadImage).toHaveBeenCalledTimes(1)
    const emitted = wrapper.emitted('update:rounds')
    expect(emitted).toBeTruthy()
    expect(emitted[emitted.length - 1][0][0].imageId).toBe('img-round-1')
  })

  it('blocks window paste in QuizEditor when an open modal dialog is present', async () => {
    const wrapper = track(mount(QuizEditor, {
      attachTo: document.body,
      props: {
        activeIndex: 0,
        eventId: 'evt-1',
        moduleId: 'mod-1',
        questions: [
          { id: 'q1', type: 'classic', text: 'First', answers: ['A', 'B', 'C', 'D'], correctIndex: 0, scoringConfig: { maxPoints: 50 } }
        ]
      },
      global: {
        plugins: [i18n],
        stubs: { Teleport: true, AppIcon: true, CoordinatePicker: true, RangeComponent: true }
      }
    }))

    const blockingDialog = document.createElement('dialog')
    blockingDialog.setAttribute('open', '')
    document.body.appendChild(blockingDialog)

    const pasteEvent = makePasteEvent({ items: [makeImageItem('image/png')] })
    window.dispatchEvent(pasteEvent)
    await flushPromises()

    expect(uploadImage).not.toHaveBeenCalled()
    expect(wrapper.emitted('update-question')).toBeFalsy()
  })
})
