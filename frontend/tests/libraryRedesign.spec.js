import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { flushPromises, mount } from '@vue/test-utils'
import i18n from '../src/i18n.js'
import { clearDialogQueue } from '../src/composables/useDialog.js'
import { session } from '../src/session.js'
import AppDialogHost from '../src/components/AppDialogHost.vue'
import Toast from '../src/components/Toast.vue'
import QuizLibraryScreen from '../src/views/QuizLibraryScreen.vue'
import AdminTemplateEditorScreen from '../src/views/AdminTemplateEditorScreen.vue'
import QuizEditor from '../src/components/QuizEditor.vue'
import FormEditor from '../src/components/FormEditor.vue'
import WrongPicturesEditor from '../src/components/WrongPicturesEditor.vue'

function makeRouter(initialPath = '/admin/library/quizzes') {
  const Dummy = defineComponent({ template: '<div />' })
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/event', component: Dummy },
      { path: '/admin/events', component: Dummy, meta: { requiresAuth: true } },
      { path: '/admin/events/:eventId', component: Dummy, meta: { requiresAuth: true } },
      { path: '/admin/library/:tab', name: 'admin-library', component: QuizLibraryScreen, meta: { requiresAuth: true } },
      { path: '/admin/library/:tab/new', name: 'admin-template-new', component: AdminTemplateEditorScreen, meta: { requiresAuth: true } },
      { path: '/admin/library/:tab/:templateId/edit', name: 'admin-template-edit', component: AdminTemplateEditorScreen, meta: { requiresAuth: true } },
      { path: '/admin/users', component: Dummy, meta: { requiresOwner: true } }
    ]
  })
  return router.push(initialPath).then(() => router.isReady()).then(() => router)
}

const wrappers = []
function track(wrapper) {
  wrappers.push(wrapper)
  return wrapper
}

function mountWithShell(Component, router) {
  track(mount(AppDialogHost, { attachTo: document.body, global: { plugins: [i18n] } }))
  track(mount(Toast, { attachTo: document.body, global: { plugins: [i18n] } }))
  return track(mount(Component, {
    attachTo: document.body,
    global: { plugins: [i18n, router] }
  }))
}

beforeEach(() => {
  i18n.global.locale.value = 'en'
  sessionStorage.clear()
  clearDialogQueue()
  session.loaded = true
  session.user = { id: 'admin-1', role: 'admin', status: 'active' }
  session.csrfToken = 'csrf-token'
  vi.restoreAllMocks()
})

afterEach(() => {
  clearDialogQueue()
  while (wrappers.length) {
    wrappers.pop().unmount()
  }
  document.body.innerHTML = ''
  vi.unstubAllGlobals()
})

describe('QuizLibraryScreen redesign (Phase 5 Part 2)', () => {
  it('syncs activeTab with deep-linked :tab param, uses contextual headings and distinct icons, and supports tab navigation', async () => {
    const fetchMock = vi.fn((url) => {
      if (url === '/templates') {
        return Promise.resolve({
          ok: true,
          json: async () => [
            { id: 'q-1', name: 'One Q Quiz', moduleType: 'quiz', data: { questions: [{ id: '1', text: 'Q1' }] } },
            { id: 'q-2', name: 'Two Q Quiz', moduleType: 'quiz', data: { questions: [{ id: '1' }, { id: '2' }] } },
            { id: 'f-1', name: 'One Step Form', moduleType: 'form', data: { timeline: [{ id: 's1' }] } },
            { id: 'f-2', name: 'Two Step Form', moduleType: 'form', data: { timeline: [{ id: 's1' }, { id: 's2' }] } },
            { id: 'wp-1', name: 'One Round WP', moduleType: 'wrong_pictures', data: { questions: [{ id: 'r1' }] } },
            { id: 'wp-2', name: 'Two Round WP', moduleType: 'wrong_pictures', data: { questions: [{ id: 'r1' }, { id: 'r2' }] } }
          ]
        })
      }
      if (url === '/screensavers' || url === '/tracks') {
        return Promise.resolve({ ok: true, json: async () => [] })
      }
      return Promise.resolve({ ok: true, json: async () => ({}) })
    })
    vi.stubGlobal('fetch', fetchMock)

    const router = await makeRouter('/admin/library/wrong-pictures')
    const wrapper = mountWithShell(QuizLibraryScreen, router)
    await flushPromises()

    expect(wrapper.find('#library-tab-wrong-pictures').attributes('aria-selected')).toBe('true')
    expect(wrapper.find('.header-heading').text()).toBe('Templates')

    // Distinct icons for Screensavers vs Wrong Pictures
    expect(wrapper.find('#library-tab-wrong-pictures svg').classes()).toContain('app-icon--search')
    expect(wrapper.find('#library-tab-screensavers svg').classes()).toContain('app-icon--image')

    // Pluralization on Wrong Pictures cards
    const wpCounts = wrapper.findAll('.question-count').map(n => n.text())
    expect(wpCounts).toEqual(['1 Round', '2 Rounds'])

    // Switch to Quizzes and verify pluralization
    await wrapper.find('#library-tab-quizzes').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/admin/library/quizzes')
    expect(wrapper.findAll('.question-count').map(n => n.text())).toEqual(['1 Question', '2 Questions'])

    // Switch to Forms and verify pluralization
    await wrapper.find('#library-tab-forms').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/admin/library/forms')
    expect(wrapper.findAll('.question-count').map(n => n.text())).toEqual(['1 Step', '2 Steps'])

    // Switch to Screensavers -> Media Assets heading + empty CTA
    await wrapper.find('#library-tab-screensavers').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/admin/library/screensavers')
    expect(wrapper.find('.header-heading').text()).toBe('Media Assets')
    expect(wrapper.find('.empty-hint').exists()).toBe(true)
    expect(wrapper.find('.empty-cta-btn').text()).toContain('Upload Screensaver')

    // Switch to Music -> Media Assets heading + empty CTA
    await wrapper.find('#library-tab-music').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/admin/library/music')
    expect(wrapper.find('.header-heading').text()).toBe('Media Assets')
    expect(wrapper.find('.empty-hint').exists()).toBe(true)
    expect(wrapper.find('.empty-cta-btn').text()).toContain('Upload Music')
  })

  it('validates ?returnTo= as an internal /admin/... path and preserves it when navigating to new/edit template routes', async () => {
    vi.stubGlobal('fetch', vi.fn(async (url) => {
      if (url === '/templates') {
        return {
          ok: true,
          json: async () => [{ id: 'tpl-42', name: 'Event Trivia', moduleType: 'quiz', data: { questions: [] } }]
        }
      }
      return { ok: true, json: async () => ({}) }
    }))

    // Valid internal returnTo
    const router = await makeRouter('/admin/library/quizzes?returnTo=%2Fadmin%2Fevents%2Fevt-7')
    const wrapper = mountWithShell(QuizLibraryScreen, router)
    await flushPromises()

    const backLink = wrapper.find('.back-link')
    expect(backLink.exists()).toBe(true)
    expect(backLink.attributes('href')).toBe('/admin/events/evt-7')

    // Clicking Edit Template navigates to edit page preserving returnTo
    await wrapper.find('.edit-btn').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/admin/library/quizzes/tpl-42/edit')
    expect(router.currentRoute.value.query.returnTo).toBe('/admin/events/evt-7')

    // Invalid external returnTo is rejected
    const badRouter = await makeRouter('/admin/library/quizzes?returnTo=https%3A%2F%2Fevil.example%2Fphish')
    const badWrapper = mountWithShell(QuizLibraryScreen, badRouter)
    await flushPromises()
    expect(badWrapper.find('.back-link').exists()).toBe(false)

    // Clicking New Template navigates to /new
    await badWrapper.find('.create-btn').trigger('click')
    await flushPromises()
    expect(badRouter.currentRoute.value.path).toBe('/admin/library/quizzes/new')
  })

  it('renders empty-state hints and inline CTAs on template tabs, shows Export toast, uses useDialog().prompt for media upload, and sanitizes wrong_pictures JSON import', async () => {
    const postCalls = []
    vi.stubGlobal('fetch', vi.fn(async (url, opts = {}) => {
      if (url === '/templates' && (!opts.method || opts.method === 'GET')) {
        return {
          ok: true,
          json: async () => [
            { id: 'q-export', name: 'Exportable Quiz', moduleType: 'quiz', data: { questions: [] } }
          ]
        }
      }
      if (url === '/templates' && opts.method === 'POST') {
        postCalls.push(JSON.parse(opts.body))
        return { ok: true, status: 201, json: async () => ({ id: 'imported-wp-1' }) }
      }
      if (url === '/tracks' && (!opts.method || opts.method === 'GET')) {
        return { ok: true, json: async () => [] }
      }
      if (url === '/tracks' && opts.method === 'POST') {
        postCalls.push({ trackFormData: opts.body })
        return { ok: true, json: async () => ({ id: 'trk-1', name: 'Lobby Jazz' }) }
      }
      return { ok: true, json: async () => ({}) }
    }))

    vi.stubGlobal('prompt', vi.fn(() => {
      throw new Error('window.prompt must not be called')
    }))

    const createObjectURL = vi.fn(() => 'blob:test')
    const revokeObjectURL = vi.fn()
    vi.stubGlobal('URL', class extends URL {
      static createObjectURL = createObjectURL
      static revokeObjectURL = revokeObjectURL
    })

    const router = await makeRouter('/admin/library/wrong-pictures')
    const wrapper = mountWithShell(QuizLibraryScreen, router)
    await flushPromises()

    // Empty state on wrong-pictures has hint + CTA
    expect(wrapper.find('.empty-hint').text()).toBeTruthy()
    expect(wrapper.find('.empty-cta-btn').exists()).toBe(true)

    // Export toast feedback on Quizzes tab
    await wrapper.find('#library-tab-quizzes').trigger('click')
    await flushPromises()
    await wrapper.find('.export-btn').trigger('click')
    await flushPromises()
    expect(document.body.textContent).toContain('Exported "Exportable Quiz"')

    // Import wrong_pictures JSON -> confirms via AppDialogHost & sanitizes rounds
    const jsonPayload = JSON.stringify({
      name: 'Spot Mistake Pack',
      description: 'Visual rounds',
      moduleType: 'wrong_pictures',
      data: {
        questions: [
          { id: 'r1', text: 'Find 2 errors', imageId: 'img-1', wrongPictureConfig: { targets: [{ x: 10, y: 20, radius: 5 }] } }
        ]
      }
    })
    const file = new File([jsonPayload], 'wp.json', { type: 'application/json' })
    const fileInputEl = wrapper.find('input[accept=".json"]').element
    Object.defineProperty(fileInputEl, 'files', { value: [file], configurable: true })
    await wrapper.find('input[accept=".json"]').trigger('change')
    await vi.waitFor(() => {
      expect(document.body.querySelector('.app-dialog-message')?.textContent).toContain('Spot Mistake Pack')
    })
    document.body.querySelector('.app-dialog-btn--primary').click()
    await flushPromises()

    const imported = postCalls.find(c => c.moduleType === 'wrong_pictures')
    expect(imported).toBeTruthy()
    expect(imported.data.questions[0].type).toBe('wrong_pictures')
    expect(imported.data.questions[0].wrongPictureConfig.maxPoints).toBe(50)
    expect(imported.data.questions[0].wrongPictureConfig.tolerancePercent).toBe(5)

    // Music upload uses useDialog().prompt with media label
    await wrapper.find('#library-tab-music').trigger('click')
    await flushPromises()
    const audioFile = new File(['audio'], 'lobby-jazz.mp3', { type: 'audio/mpeg' })
    const trackInputEl = wrapper.find('input[accept="audio/mpeg,audio/wav,audio/ogg"]').element
    Object.defineProperty(trackInputEl, 'files', { value: [audioFile], configurable: true })
    await wrapper.find('input[accept="audio/mpeg,audio/wav,audio/ogg"]').trigger('change')
    await flushPromises()

    expect(document.body.querySelector('.app-dialog-label')?.textContent).toBe('Track Name')
    const promptInput = document.body.querySelector('.app-dialog-input')
    expect(promptInput.value).toBe('lobby-jazz')
    promptInput.value = 'Lobby Jazz'
    promptInput.dispatchEvent(new Event('input'))
    document.body.querySelector('.app-dialog-btn--primary').click()
    await flushPromises()

    const trackUpload = postCalls.find(c => c.trackFormData)
    expect(trackUpload).toBeTruthy()
    expect(trackUpload.trackFormData.get('name')).toBe('Lobby Jazz')
  })
})

describe('AdminTemplateEditorScreen (Phase 5 Part 2)', () => {
  it('mounts QuizEditor for quiz, FormEditor for form, and WrongPicturesEditor for wrong_pictures with labeled inputs and breadcrumbs', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => ({
      ok: true,
      json: async () => [
        {
          id: 'wp-10',
          name: 'Historical Errors',
          description: 'Find anachronisms',
          moduleType: 'wrong_pictures',
          data: {
            questions: [
              {
                id: 'r-1',
                type: 'wrong_pictures',
                text: 'Round 1',
                imageId: 'img-10',
                wrongPictureConfig: { targets: [{ x: 25, y: 30, radius: 8 }], maxPoints: 50, tolerancePercent: 5 }
              }
            ]
          }
        }
      ]
    })))

    // 1. Quiz new
    const quizRouter = await makeRouter('/admin/library/quizzes/new')
    const quizWrapper = mountWithShell(AdminTemplateEditorScreen, quizRouter)
    await flushPromises()
    expect(quizWrapper.findComponent(QuizEditor).exists()).toBe(true)
    expect(quizWrapper.findComponent(QuizEditor).props('showOutline')).toBe(true)
    expect(quizWrapper.findComponent(QuizEditor).props('pageFlow')).toBe(true)
    expect(quizWrapper.find('label[for="template-name-input"]').text()).toBe('Template Name')
    expect(quizWrapper.find('label[for="template-description-input"]').text()).toBe('Description')
    expect(quizWrapper.find('.admin-breadcrumb-list').text()).toContain('Library')
    expect(quizWrapper.find('.admin-breadcrumb-list').text()).toContain('Quizzes')
    expect(quizWrapper.find('.admin-breadcrumb-list').text()).toContain('New Template')

    // 2. Form new
    const formRouter = await makeRouter('/admin/library/forms/new')
    const formWrapper = mountWithShell(AdminTemplateEditorScreen, formRouter)
    await flushPromises()
    expect(formWrapper.findComponent(FormEditor).exists()).toBe(true)
    expect(formWrapper.findComponent(FormEditor).props('pageFlow')).toBe(true)

    // 3. Wrong Pictures edit -> mounts WrongPicturesEditor, NOT QuizEditor
    const wpRouter = await makeRouter('/admin/library/wrong-pictures/wp-10/edit')
    const wpWrapper = mountWithShell(AdminTemplateEditorScreen, wpRouter)
    await flushPromises()
    expect(wpWrapper.findComponent(WrongPicturesEditor).exists()).toBe(true)
    expect(wpWrapper.findComponent(QuizEditor).exists()).toBe(false)
    expect(wpWrapper.findComponent(WrongPicturesEditor).props('pageFlow')).toBe(true)
    expect(wpWrapper.find('#template-name-input').element.value).toBe('Historical Errors')
    expect(wpWrapper.find('.admin-breadcrumb-list').text()).toContain('Historical Errors')
  })

  it('guards only when actually dirty and never nags when nothing was edited', async () => {
    vi.stubGlobal('fetch', vi.fn(async (url) => {
      if (url === '/templates') {
        return {
          ok: true,
          json: async () => [
            {
              id: 'wp-11',
              name: 'Clean Template',
              description: '',
              moduleType: 'wrong_pictures',
              data: { questions: [] }
            }
          ]
        }
      }
      return { ok: true, json: async () => [] }
    }))

    const AppShell = defineComponent({ template: '<router-view />' })
    const router = await makeRouter('/admin/library/wrong-pictures/wp-11/edit')
    const wrapper = mountWithShell(AppShell, router)
    await flushPromises()

    // Navigate away without editing -> no unsaved-changes dialog
    await router.push('/admin/library/wrong-pictures')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/admin/library/wrong-pictures')
    expect(document.body.querySelector('.app-dialog')).toBeNull()

    // Navigate back into edit, edit name, then try leaving -> prompts unsaved-changes confirm
    await router.push('/admin/library/wrong-pictures/wp-11/edit')
    await flushPromises()
    await wrapper.find('#template-name-input').setValue('Edited Name')
    await flushPromises()

    const navPromise = router.push('/admin/library/wrong-pictures')
    await flushPromises()
    expect(document.body.querySelector('.app-dialog-title')?.textContent).toBe('Unsaved changes')

    // Cancel stay
    document.body.querySelector('.app-dialog-btn--secondary').click()
    await navPromise
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/admin/library/wrong-pictures/wp-11/edit')

    // Confirm leave
    const navLeavePromise = router.push('/admin/library/wrong-pictures')
    await flushPromises()
    document.body.querySelector('.app-dialog-btn--danger').click()
    await navLeavePromise
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/admin/library/wrong-pictures')
  })

  it('allows saving empty draft with save_empty_confirm, flags invalid questions/rounds inline, and saves valid wrong_pictures template', async () => {
    const requests = []
    vi.stubGlobal('fetch', vi.fn(async (url, opts = {}) => {
      if (url === '/templates' && (!opts.method || opts.method === 'GET')) {
        return {
          ok: true,
          json: async () => [
            {
              id: 'wp-20',
              name: 'Draft WP',
              description: '',
              moduleType: 'wrong_pictures',
              data: {
                questions: [
                  { id: 'r1', type: 'wrong_pictures', text: '', imageId: null, wrongPictureConfig: { targets: [], maxPoints: 50, tolerancePercent: 5 } }
                ]
              }
            }
          ]
        }
      }
      if (url === '/templates' && opts.method === 'POST') {
        requests.push({ method: 'POST', body: JSON.parse(opts.body) })
        return { ok: true, status: 201, json: async () => ({ id: 'new-tpl-1' }) }
      }
      if (url === '/templates/wp-20' && opts.method === 'PUT') {
        requests.push({ method: 'PUT', body: JSON.parse(opts.body) })
        return { ok: true, json: async () => ({ message: 'Template updated' }) }
      }
      return { ok: true, json: async () => ({}) }
    }))

    // 1. Empty draft confirmation on /new
    const newRouter = await makeRouter('/admin/library/quizzes/new')
    const newWrapper = mountWithShell(AdminTemplateEditorScreen, newRouter)
    await flushPromises()

    await newWrapper.find('#template-name-input').setValue('Empty Draft')
    await newWrapper.find('.save-btn').trigger('click')
    await flushPromises()

    expect(document.body.querySelector('.app-dialog-message')?.textContent).toContain('This template has no questions. Save it anyway?')
    document.body.querySelector('.app-dialog-btn--primary').click()
    await flushPromises()

    expect(requests.some(r => r.method === 'POST' && r.body.name === 'Empty Draft')).toBe(true)
    expect(newRouter.currentRoute.value.path).toBe('/admin/library/quizzes')

    // 2. Inline validation on incomplete wrong_pictures round blocks save
    const editRouter = await makeRouter('/admin/library/wrong-pictures/wp-20/edit')
    const editWrapper = mountWithShell(AdminTemplateEditorScreen, editRouter)
    await flushPromises()

    await editWrapper.find('.save-btn').trigger('click')
    await flushPromises()

    expect(requests.some(r => r.method === 'PUT')).toBe(false)
    expect(editWrapper.find('.validation-banner').exists()).toBe(true)
    expect(editWrapper.findAll('.field-error').length).toBeGreaterThan(0)
  })
})
