import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { flushPromises, mount } from '@vue/test-utils'
import i18n from '../src/i18n.js'
import { clearDialogQueue } from '../src/composables/useDialog.js'
import { session } from '../src/session.js'
import AppDialogHost from '../src/components/AppDialogHost.vue'
import AdminModuleEditorScreen from '../src/views/AdminModuleEditorScreen.vue'

function makeRouter(initialPath = '/admin/events/evt-1/modules/mod-1/edit') {
  const Dummy = defineComponent({ template: '<div />' })
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/admin/events', name: 'admin-events', component: Dummy },
      { path: '/admin/events/:eventId', name: 'admin-event-details', component: Dummy },
      { path: '/admin/events/:eventId/modules/:moduleId/edit', name: 'admin-module-editor', component: AdminModuleEditorScreen },
      { path: '/admin/events/:eventId/modules/:moduleId/host', name: 'admin-module-host', component: Dummy },
      { path: '/admin/events/:eventId/teams', name: 'teams-admin', component: Dummy },
      { path: '/admin/library/:tab', name: 'admin-library', component: Dummy },
      { path: '/admin/users', name: 'admin-users', component: Dummy },
      { path: '/login', name: 'login', component: Dummy }
    ]
  })
  return router.push(initialPath).then(() => router.isReady()).then(() => router)
}

function mountEditor(router, stubs = {}) {
  const Root = defineComponent({ template: '<router-view />' })
  return track(mount(Root, {
    attachTo: document.body,
    global: {
      plugins: [i18n, router],
      stubs
    }
  }))
}

const wrappers = []
function track(wrapper) {
  wrappers.push(wrapper)
  return wrapper
}

beforeEach(() => {
  i18n.global.locale.value = 'en'
  clearDialogQueue()
  session.loaded = true
  session.user = { id: 'owner-1', role: 'owner', status: 'active' }
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

describe('AdminModuleEditorScreen', () => {
  it('loads, edits, validates, and saves a quiz module with outline, pageFlow, and audio settings', async () => {
    const router = await makeRouter('/admin/events/evt-1/modules/mod-q1/edit')
    const scrollSpy = vi.fn()
    Element.prototype.scrollIntoView = scrollSpy

    const fetchMock = vi.fn((url, opts = {}) => {
      if (url === '/events/evt-1' && (!opts.method || opts.method === 'GET')) {
        return Promise.resolve({
          ok: true,
          json: async () => ({ id: 'evt-1', name: 'Spring Trivia', modules: [] })
        })
      }
      if (url === '/events/evt-1/modules/mod-q1' && (!opts.method || opts.method === 'GET')) {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            id: 'mod-q1',
            name: 'Main Quiz',
            type: 'quiz',
            data: {
              backgroundMusicId: 'trk-1',
              audioOutput: 'projector',
              questions: [
                {
                  id: 'q-1',
                  type: 'classic',
                  text: 'Capital of France?',
                  answers: ['Paris', 'Rome', 'Berlin', 'Madrid'],
                  correctIndex: 0,
                  scoringConfig: { maxPoints: 50 }
                }
              ]
            }
          })
        })
      }
      if (url === '/tracks') {
        return Promise.resolve({
          ok: true,
          json: async () => [{ id: 'trk-1', name: 'Upbeat Loop', url: '/media/trk-1/file' }]
        })
      }
      if (url === '/events/evt-1/modules/mod-q1/quiz/questions' && opts.method === 'PUT') {
        return Promise.resolve({ ok: true, json: async () => ({ message: 'ok' }) })
      }
      if (url === '/events/evt-1/modules/mod-q1' && opts.method === 'PUT') {
        return Promise.resolve({ ok: true, json: async () => ({ message: 'ok' }) })
      }
      return Promise.resolve({ ok: true, json: async () => ({}) })
    })
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = mountEditor(router, {
      ImageUploader: true,
      CoordinatePicker: true,
      RangeComponent: true,
      ImageOptionEditor: true
    })
    await flushPromises()

    // Breadcrumbs: Events / Spring Trivia / Main Quiz
    const crumbs = wrapper.findAll('.admin-breadcrumb-item').map(el => el.text().trim())
    expect(crumbs).toEqual(['Events', 'Spring Trivia', 'Main Quiz'])

    // Type badge and labeled Load from Library button with book icon
    expect(wrapper.find('.module-type-badge').text().trim()).toBe('Quiz')
    const libraryBtn = wrapper.find('.load-library-btn')
    expect(libraryBtn.exists()).toBe(true)
    expect(libraryBtn.text()).toContain('Load from Library')
    expect(libraryBtn.find('.app-icon--book').exists()).toBe(true)

    // QuizEditor renders in pageFlow with outline
    expect(wrapper.find('.quiz-editor').classes()).toContain('quiz-editor--page-flow')
    expect(wrapper.find('.quiz-outline-sidebar').exists()).toBe(true)

    // Clean initially (no dirty dot)
    expect(wrapper.find('.dirty-dot').exists()).toBe(false)

    // Trigger validation failure by clearing question text
    await wrapper.find('.main-input').setValue('')
    await flushPromises()
    expect(wrapper.find('.dirty-dot').exists()).toBe(true)

    await wrapper.find('.save-primary-btn').trigger('click')
    await flushPromises()

    expect(wrapper.find('.validation-summary-banner').exists()).toBe(true)
    expect(wrapper.find('.main-input').classes()).toContain('has-field-error')
    expect(scrollSpy).toHaveBeenCalled()
    expect(fetchMock.mock.calls.some(([, o]) => o?.method === 'PUT')).toBe(false)

    // Fix validation error, change module name and audio output, and save
    await wrapper.find('.main-input').setValue('Capital of France?')
    await wrapper.find('#module-editor-name').setValue('Round 1 Quiz')
    await wrapper.find('.settings-collapse-trigger').trigger('click')
    await wrapper.find('#quiz-audio-output').setValue('both')
    await wrapper.find('.save-primary-btn').trigger('click')
    await flushPromises()

    const qPut = fetchMock.mock.calls.find(([u, o]) => u === '/events/evt-1/modules/mod-q1/quiz/questions' && o?.method === 'PUT')
    const modPut = fetchMock.mock.calls.find(([u, o]) => u === '/events/evt-1/modules/mod-q1' && o?.method === 'PUT')
    expect(qPut).toBeTruthy()
    expect(modPut).toBeTruthy()

    const modBody = JSON.parse(modPut[1].body)
    expect(modBody.name).toBe('Round 1 Quiz')
    expect(modBody.data.backgroundMusicId).toBe('trk-1')
    expect(modBody.data.backgroundMusicUrl).toBe('/media/trk-1/file')
    expect(modBody.data.audioOutput).toBe('both')
    expect(wrapper.find('.dirty-dot').exists()).toBe(false)
  })

  it('locks question type to simple_text for simple_quiz and renders allowReveal + judge link', async () => {
    const router = await makeRouter('/admin/events/evt-2/modules/mod-sq1/edit')
    const writeTextMock = vi.fn().mockResolvedValue()
    Object.assign(navigator, { clipboard: { writeText: writeTextMock } })

    const fetchMock = vi.fn((url, opts = {}) => {
      if (url === '/events/evt-2') {
        return Promise.resolve({ ok: true, json: async () => ({ id: 'evt-2', name: 'Open Mic' }) })
      }
      if (url === '/events/evt-2/modules/mod-sq1' && (!opts.method || opts.method === 'GET')) {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            id: 'mod-sq1',
            name: 'Text Round',
            type: 'simple_quiz',
            data: {
              allowAnswerReveal: true,
              judgeToken: 'secret-token-9',
              questions: [
                { id: 'sq-1', type: 'classic', text: 'Name a planet', scoringConfig: { maxPoints: 100 } }
              ]
            }
          })
        })
      }
      if (url === '/events/evt-2/modules/mod-sq1' && opts.method === 'PUT') {
        return Promise.resolve({ ok: true, json: async () => ({ message: 'ok' }) })
      }
      return Promise.resolve({ ok: true, json: async () => ({}) })
    })
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = mountEditor(router, { ImageUploader: true })
    await flushPromises()

    // Locked badge is a non-clickable span, and "+ Other Type" is absent
    const badge = wrapper.find('.question-type-tag')
    expect(badge.element.tagName).toBe('SPAN')
    expect(badge.classes()).toContain('question-type-tag--locked')
    expect(wrapper.find('.add-other-btn').exists()).toBe(false)

    // Adding a question adds a simple_text question directly
    await wrapper.find('.add-classic-btn').trigger('click')
    await flushPromises()
    expect(wrapper.findAll('.question-editor')).toHaveLength(2)

    // Fill second question text, toggle allowReveal off, copy judge link, and save
    const textInputs = wrapper.findAll('.text-group .main-input')
    await textInputs[1].setValue('Explain gravity')
    await wrapper.find('.simple-quiz-settings input[type="checkbox"]').setValue(false)
    await wrapper.find('.copy-judge-btn').trigger('click')
    expect(writeTextMock).toHaveBeenCalledWith(
      expect.stringContaining('/simple-quiz/evt-2/mod-sq1/judge?token=secret-token-9')
    )

    await wrapper.find('.save-primary-btn').trigger('click')
    await flushPromises()

    const putCall = fetchMock.mock.calls.find(([u, o]) => u === '/events/evt-2/modules/mod-sq1' && o?.method === 'PUT')
    expect(putCall).toBeTruthy()
    const payload = JSON.parse(putCall[1].body)
    expect(payload.data.allowAnswerReveal).toBe(false)
    expect(payload.data.questions).toHaveLength(2)
    expect(payload.data.questions.every(q => q.type === 'simple_text')).toBe(true)
  })

  it('validates and saves wrong_pictures rounds via WrongPicturesEditor', async () => {
    const router = await makeRouter('/admin/events/evt-3/modules/mod-wp1/edit')
    Element.prototype.scrollIntoView = vi.fn()

    const fetchMock = vi.fn((url, opts = {}) => {
      if (url === '/events/evt-3') {
        return Promise.resolve({ ok: true, json: async () => ({ id: 'evt-3', name: 'Visual Night' }) })
      }
      if (url === '/events/evt-3/modules/mod-wp1' && (!opts.method || opts.method === 'GET')) {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            id: 'mod-wp1',
            name: 'Spot Mistakes',
            type: 'wrong_pictures',
            data: {
              questions: [
                {
                  id: 'wp-1',
                  type: 'wrong_pictures',
                  text: '',
                  imageId: null,
                  wrongPictureConfig: { targets: [], maxPoints: 50, tolerancePercent: 5, showTargetCount: false }
                }
              ]
            }
          })
        })
      }
      if (url === '/tracks') {
        return Promise.resolve({ ok: true, json: async () => [] })
      }
      return Promise.resolve({ ok: true, json: async () => ({}) })
    })
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = mountEditor(router, { ImageUploader: true, CoordinatePicker: true })
    await flushPromises()

    expect(wrapper.find('.wrong-pictures-editor').classes()).toContain('wrong-pictures-editor--page-flow')
    await wrapper.find('.save-primary-btn').trigger('click')
    await flushPromises()

    expect(wrapper.find('.round-card').classes()).toContain('has-errors')
    expect(wrapper.find('.image-required-warning').classes()).toContain('has-field-error')
    expect(fetchMock.mock.calls.some(([, o]) => o?.method === 'PUT')).toBe(false)
  })

  it('edits and publishes form modules while preserving metadata and handles template Save as Template without window.prompt', async () => {
    const router = await makeRouter('/admin/events/evt-4/modules/mod-f1/edit')
    const promptSpy = vi.spyOn(window, 'prompt')
    const confirmSpy = vi.spyOn(window, 'confirm')

    const fetchMock = vi.fn((url, opts = {}) => {
      if (url === '/events/evt-4') {
        return Promise.resolve({ ok: true, json: async () => ({ id: 'evt-4', name: 'Conference' }) })
      }
      if (url === '/events/evt-4/modules/mod-f1' && (!opts.method || opts.method === 'GET')) {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            id: 'mod-f1',
            name: 'Check-in',
            type: 'form',
            data: {
              manualScoreTarget: 'teams',
              mandatoryOnEntry: false,
              hideAfterCompletion: false,
              timeline: [{ id: 's-1', type: 'field', data: { label: 'Your Name', fieldType: 'text' } }]
            }
          })
        })
      }
      if (url === '/templates' && (!opts.method || opts.method === 'GET')) {
        return Promise.resolve({
          ok: true,
          json: async () => [
            { id: 'tpl-f1', name: 'Arrival Survey', moduleType: 'form', data: { timeline: [{ id: 'old', type: 'message', data: { content: 'Hi' } }] } },
            { id: 'tpl-q1', name: 'Other Quiz', moduleType: 'quiz', data: { questions: [] } }
          ]
        })
      }
      return Promise.resolve({ ok: true, json: async () => ({}) })
    })
    vi.stubGlobal('fetch', fetchMock)

    track(mount(AppDialogHost, { attachTo: document.body, global: { plugins: [i18n] } }))
    const wrapper = mountEditor(router)
    await flushPromises()

    expect(wrapper.find('.form-editor').classes()).toContain('form-editor--page-flow')

    // Load from Library filters to form templates and uses useDialog().confirm when replacing
    await wrapper.find('.load-library-btn').trigger('click')
    await flushPromises()
    const templateBtns = document.body.querySelectorAll('.library-modal .template-item')
    expect(templateBtns).toHaveLength(1)
    expect(templateBtns[0].textContent).toContain('Arrival Survey')
    templateBtns[0].click()
    await flushPromises()

    expect(confirmSpy).not.toHaveBeenCalled()
    expect(document.body.querySelector('.app-dialog-title')?.textContent).toContain('Replace current content?')
    document.body.querySelector('.app-dialog-btn--danger').click()
    await flushPromises()

    // Save as Template uses useDialog().prompt with "Save Template" button
    await wrapper.find('.save-template-btn').trigger('click')
    await flushPromises()
    expect(promptSpy).not.toHaveBeenCalled()
    const dialogConfirmText = document.body.querySelector('.app-dialog-btn--primary')?.textContent.trim()
    expect(dialogConfirmText).toBe('Save Template')
    const inputEl = document.body.querySelector('.app-dialog-input')
    inputEl.value = 'My Form Template'
    inputEl.dispatchEvent(new Event('input', { bubbles: true }))
    document.body.querySelector('form.app-dialog-body').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    await flushPromises()

    const tplPost = fetchMock.mock.calls.find(([u, o]) => u === '/templates' && o?.method === 'POST')
    expect(tplPost).toBeTruthy()
    expect(JSON.parse(tplPost[1].body)).toMatchObject({ name: 'My Form Template', moduleType: 'form' })

    // Toggle settings and publish form preserving manualScoreTarget
    const checkboxes = wrapper.findAll('.editor-settings-card input[type="checkbox"]')
    await checkboxes[0].setValue(true)
    await checkboxes[1].setValue(true)
    await wrapper.find('.save-primary-btn').trigger('click')
    await flushPromises()

    const pubPut = fetchMock.mock.calls.find(([u, o]) => u === '/events/evt-4/modules/mod-f1/publish' && o?.method === 'PUT')
    expect(pubPut).toBeTruthy()
    const pubPayload = JSON.parse(pubPut[1].body)
    expect(pubPayload.data.manualScoreTarget).toBe('teams')
    expect(pubPayload.data.mandatoryOnEntry).toBe(true)
    expect(pubPayload.data.hideAfterCompletion).toBe(true)
  })

  it('guards unsaved changes on Teams Manage Teams / Host navigation', async () => {
    const router = await makeRouter('/admin/events/evt-5/modules/mod-t2/edit')
    const pushSpy = vi.spyOn(router, 'push')

    const fetchMock = vi.fn((url, opts = {}) => {
      if (url === '/events/evt-5') {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            id: 'evt-5',
            name: 'Team Cup',
            modules: [
              { id: 'mod-t1', name: 'Morning Teams', type: 'teams' },
              { id: 'mod-t2', name: 'Afternoon Teams', type: 'teams' }
            ]
          })
        })
      }
      if (url === '/events/evt-5/modules/mod-t2' && (!opts.method || opts.method === 'GET')) {
        return Promise.resolve({
          ok: true,
          json: async () => ({
            id: 'mod-t2',
            name: 'Afternoon Teams',
            type: 'teams',
            data: { teamsConfig: { sourceModuleId: '', method: 'random', teamCount: 4, targetSize: 0, capacity: 0, presets: [1, 5] } }
          })
        })
      }
      if (url === '/events/evt-5/modules/mod-t2/teams/admin') {
        return Promise.resolve({ ok: true, json: async () => ({ activity: null }) })
      }
      return Promise.resolve({ ok: true, json: async () => ({}) })
    })
    vi.stubGlobal('fetch', fetchMock)

    track(mount(AppDialogHost, { attachTo: document.body, global: { plugins: [i18n] } }))
    const wrapper = mountEditor(router)
    await flushPromises()

    // Prior teams module appears in source options
    const sourceOptions = wrapper.findAll('.teams-config-fields select')[0].findAll('option').map(o => o.text())
    expect(sourceOptions).toContain('Morning Teams')

    // Make editor dirty
    await wrapper.find('#module-editor-name').setValue('Updated Teams')
    await flushPromises()
    expect(wrapper.find('.dirty-dot').exists()).toBe(true)

    // Attempt to click Manage Teams -> triggers unsaved changes dialog
    await wrapper.find('.manage-teams-btn').trigger('click')
    await flushPromises()
    expect(document.body.querySelector('.app-dialog-title')?.textContent).toContain('Unsaved changes')

    // Stay cancels navigation
    document.body.querySelector('.app-dialog-btn--secondary').click()
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/admin/events/evt-5/modules/mod-t2/edit')

    // Click Host shortcut -> confirm Leave -> navigates to teams admin
    await wrapper.find('.host-shortcut-btn').trigger('click')
    await flushPromises()
    document.body.querySelector('.app-dialog-btn--danger').click()
    await flushPromises()
    expect(pushSpy).toHaveBeenCalledWith({
      path: '/admin/events/evt-5/teams',
      query: { moduleId: 'mod-t2' }
    })

  })
})
