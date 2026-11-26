import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { flushPromises, mount } from '@vue/test-utils'
import i18n from '../src/i18n.js'
import { clearDialogQueue } from '../src/composables/useDialog.js'
import {
  confirmActiveUnsavedChanges,
  runBypassingUnsavedGuard,
  useUnsavedChanges
} from '../src/composables/useUnsavedChanges.js'
import { session } from '../src/session.js'
import AppDialogHost from '../src/components/AppDialogHost.vue'
import AdminLayout from '../src/components/AdminLayout.vue'
import AdminModuleEditorScreen from '../src/views/AdminModuleEditorScreen.vue'

const wrappers = []
function track(wrapper) {
  wrappers.push(wrapper)
  return wrapper
}

function makeRouter(initialPath = '/admin/events/evt-1/modules/mod-1/edit') {
  const Dummy = defineComponent({ template: '<div />' })
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/login', name: 'login', component: Dummy },
      { path: '/admin/events', name: 'admin-events', component: Dummy },
      { path: '/admin/events/:eventId', name: 'admin-event-details', component: Dummy },
      { path: '/admin/events/:eventId/modules/:moduleId/edit', name: 'admin-module-editor', component: AdminModuleEditorScreen },
      { path: '/admin/users', name: 'admin-users', component: Dummy }
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

function quizModule(name = 'Main Quiz', extraData = {}) {
  return {
    id: 'mod-1',
    name,
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
      ],
      ...extraData
    }
  }
}

const moduleFetch = (mod) => ({ ok: true, json: async () => mod })

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

describe('AdminModuleEditorScreen save guards (H2, M4)', () => {
  it('keeps the editor dirty when edits land while the save is in flight (H2)', async () => {
    const router = await makeRouter()
    let resolveModPut
    const modPut = new Promise((resolve) => { resolveModPut = resolve })

    const fetchMock = vi.fn((url, opts = {}) => {
      if (url === '/events/evt-1' && (!opts.method || opts.method === 'GET')) {
        return Promise.resolve({ ok: true, json: async () => ({ id: 'evt-1', name: 'Spring Trivia', modules: [] }) })
      }
      if (url === '/events/evt-1/modules/mod-1' && (!opts.method || opts.method === 'GET')) {
        return Promise.resolve(moduleFetch(quizModule()))
      }
      if (url === '/tracks') {
        return Promise.resolve({ ok: true, json: async () => [] })
      }
      if (url === '/events/evt-1/modules/mod-1/quiz/questions' && opts.method === 'PUT') {
        return Promise.resolve({ ok: true, json: async () => ({ message: 'ok' }) })
      }
      if (url === '/events/evt-1/modules/mod-1' && opts.method === 'PUT') {
        return modPut.then(() => ({ ok: true, json: async () => ({ message: 'ok' }) }))
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

    await wrapper.find('#module-editor-name').setValue('Renamed Quiz')
    expect(wrapper.find('.dirty-dot').exists()).toBe(true)

    await wrapper.find('.save-primary-btn').trigger('click')
    await flushPromises()
    expect(wrapper.find('.save-primary-btn').attributes('disabled')).toBeDefined()

    await wrapper.find('.main-input').setValue('Capital of Romania?')

    resolveModPut()
    await flushPromises()

    expect(wrapper.find('.dirty-dot').exists()).toBe(true)
    expect(wrapper.find('.save-primary-btn').attributes('disabled')).toBeUndefined()

    const modPutCall = fetchMock.mock.calls.find(([u, o]) => u === '/events/evt-1/modules/mod-1' && o?.method === 'PUT')
    const body = JSON.parse(modPutCall[1].body)
    expect(body.name).toBe('Renamed Quiz')
    expect(body.data.questions[0].text).toBe('Capital of France?')
  })

  it('does not corrupt the newly loaded module when navigation happens mid-save (H2)', async () => {
    const router = await makeRouter()
    let resolveModPut
    const modPut = new Promise((resolve) => { resolveModPut = resolve })

    const fetchMock = vi.fn((url, opts = {}) => {
      if (url === '/events/evt-1' && (!opts.method || opts.method === 'GET')) {
        return Promise.resolve({ ok: true, json: async () => ({ id: 'evt-1', name: 'Spring Trivia', modules: [] }) })
      }
      if (url === '/events/evt-1/modules/mod-1' && (!opts.method || opts.method === 'GET')) {
        return Promise.resolve(moduleFetch(quizModule()))
      }
      if (url === '/events/evt-1/modules/mod-2' && (!opts.method || opts.method === 'GET')) {
        return Promise.resolve(moduleFetch(quizModule('Second Module')))
      }
      if (url === '/tracks') {
        return Promise.resolve({ ok: true, json: async () => [] })
      }
      if (url === '/events/evt-1/modules/mod-1/quiz/questions' && opts.method === 'PUT') {
        return Promise.resolve({ ok: true, json: async () => ({ message: 'ok' }) })
      }
      if (url === '/events/evt-1/modules/mod-1' && opts.method === 'PUT') {
        return modPut.then(() => ({ ok: true, json: async () => ({ message: 'ok' }) }))
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

    await wrapper.find('#module-editor-name').setValue('Renamed Quiz')
    await wrapper.find('.save-primary-btn').trigger('click')
    await flushPromises()

    track(mount(AppDialogHost, { attachTo: document.body, global: { plugins: [i18n] } }))
    const nav = router.push('/admin/events/evt-1/modules/mod-2/edit')
    await flushPromises()
    expect(document.body.querySelector('.app-dialog')).not.toBeNull()
    document.body.querySelector('.app-dialog-btn--danger').click()
    await nav
    await flushPromises()

    resolveModPut()
    await flushPromises()

    expect(wrapper.find('#module-editor-name').element.value).toBe('Second Module')
  })

  it('preserves backgroundMusicUrl when /tracks fails (M4)', async () => {
    const router = await makeRouter()
    const fetchMock = vi.fn((url, opts = {}) => {
      if (url === '/events/evt-1' && (!opts.method || opts.method === 'GET')) {
        return Promise.resolve({ ok: true, json: async () => ({ id: 'evt-1', name: 'Spring Trivia', modules: [] }) })
      }
      if (url === '/events/evt-1/modules/mod-1' && (!opts.method || opts.method === 'GET')) {
        return Promise.resolve(moduleFetch(quizModule('Main Quiz', { backgroundMusicUrl: '/media/trk-1/file' })))
      }
      if (url === '/tracks') {
        return Promise.reject(new Error('network down'))
      }
      if (url === '/events/evt-1/modules/mod-1/quiz/questions' && opts.method === 'PUT') {
        return Promise.resolve({ ok: true, json: async () => ({ message: 'ok' }) })
      }
      if (url === '/events/evt-1/modules/mod-1' && opts.method === 'PUT') {
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

    await wrapper.find('.settings-collapse-trigger').trigger('click')
    const select = wrapper.find('#quiz-bg-music')
    expect(select.exists()).toBe(true)
    const fallbackOption = select.findAll('option').find(o => o.text().includes('Previously selected track'))
    expect(fallbackOption).toBeTruthy()

    await wrapper.find('.save-primary-btn').trigger('click')
    await flushPromises()

    const modPutCall = fetchMock.mock.calls.find(([u, o]) => u === '/events/evt-1/modules/mod-1' && o?.method === 'PUT')
    const body = JSON.parse(modPutCall[1].body)
    expect(body.data.backgroundMusicId).toBe('trk-1')
    expect(body.data.backgroundMusicUrl).toBe('/media/trk-1/file')
  })
})

describe('Unsaved-changes-aware sign-out (M6)', () => {
  const DirtyComponent = defineComponent({
    setup() {
      const dirty = ref(true)
      useUnsavedChanges(dirty)
      return () => h('div', { class: 'dirty-page' })
    }
  })

  function mountShell(router) {
    return track(mount(defineComponent({
      components: { AdminLayout, DirtyComponent, AppDialogHost },
      template: `
        <AdminLayout>
          <DirtyComponent />
        </AdminLayout>
        <AppDialogHost />
      `
    }), {
      attachTo: document.body,
      global: { plugins: [i18n, router] }
    }))
  }

  async function makeShellRouter() {
    const DirtyShellTestRoute = { path: '/admin/events', component: { template: '<div />' } }
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        DirtyShellTestRoute,
        { path: '/login', component: { template: '<div />' } }
      ]
    })
    await router.push('/admin/events')
    await router.isReady()
    return router
  }

  it('runs the unsaved-changes guard before signing out and keeps the session on Stay', async () => {
    const router = await makeShellRouter()
    const fetchMock = vi.fn(() => Promise.resolve({ ok: true, json: async () => ({}) }))
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = mountShell(router)
    await flushPromises()

    await wrapper.find('.admin-signout-btn').trigger('click')
    await flushPromises()

    expect(document.body.querySelector('.app-dialog')).not.toBeNull()
    expect(fetchMock.mock.calls.some(([u]) => u === '/auth/logout')).toBe(false)

    document.body.querySelector('.app-dialog-btn--secondary').click()
    await flushPromises()
    expect(fetchMock.mock.calls.some(([u]) => u === '/auth/logout')).toBe(false)
    expect(router.currentRoute.value.path).toBe('/admin/events')
  })

  it('signs out and routes to /login with a single confirmation on Leave', async () => {
    const router = await makeShellRouter()
    const fetchMock = vi.fn(() => Promise.resolve({ ok: true, json: async () => ({}) }))
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = mountShell(router)
    await flushPromises()

    await wrapper.find('.admin-signout-btn').trigger('click')
    await flushPromises()
    document.body.querySelector('.app-dialog-btn--danger').click()
    await flushPromises()

    expect(fetchMock.mock.calls.some(([u]) => u === '/auth/logout')).toBe(true)
    expect(router.currentRoute.value.path).toBe('/login')
    expect(document.body.querySelector('.app-dialog')).toBeNull()
  })

  it('clean editors sign out without any prompt, and bypassed navigation skips dirty guards', async () => {
    const router = await makeShellRouter()
    const fetchMock = vi.fn(() => Promise.resolve({ ok: true, json: async () => ({}) }))
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = mountShell(router)
    const cleanShell = track(mount(defineComponent({
      components: { AdminLayout, AppDialogHost },
      template: '<AdminLayout><div /></AdminLayout><AppDialogHost />'
    }), { attachTo: document.body, global: { plugins: [i18n, router] } }))
    wrapper.unmount()
    await flushPromises()

    await expect(confirmActiveUnsavedChanges()).resolves.toBe(true)
    expect(cleanShell.find('.admin-signout-btn').exists()).toBe(true)
    await cleanShell.find('.admin-signout-btn').trigger('click')
    await flushPromises()
    expect(fetchMock.mock.calls.some(([u]) => u === '/auth/logout')).toBe(true)
    expect(document.body.querySelector('.app-dialog')).toBeNull()

    const DirtyRoute = defineComponent({
      setup() {
        useUnsavedChanges(ref(true))
        return () => h('div')
      }
    })
    const router2 = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/editor', component: DirtyRoute },
        { path: '/login', component: { template: '<div />' } }
      ]
    })
    await router2.push('/editor')
    await router2.isReady()
    track(mount(defineComponent({ template: '<router-view />' }), { global: { plugins: [i18n, router2] } }))
    await flushPromises()

    await runBypassingUnsavedGuard(() => router2.push('/login'))
    await flushPromises()
    expect(router2.currentRoute.value.path).toBe('/login')
    expect(document.body.querySelector('.app-dialog')).toBeNull()
  })
})
