import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { flushPromises, mount } from '@vue/test-utils'
import i18n from '../src/i18n.js'
import { clearDialogQueue } from '../src/composables/useDialog.js'
import { session } from '../src/session.js'
import AppDialogHost from '../src/components/AppDialogHost.vue'
import AddModuleModal from '../src/components/AddModuleModal.vue'
import AdminScreen from '../src/views/AdminScreen.vue'
import AdminEventDetailsScreen from '../src/views/AdminEventDetailsScreen.vue'
import DataLibraryScreen from '../src/views/DataLibraryScreen.vue'
import ScreenManagerScreen from '../src/views/ScreenManagerScreen.vue'
import AdminUsersScreen from '../src/views/AdminUsersScreen.vue'

function makeRouter(initialPath = '/admin/events') {
  const Dummy = defineComponent({ template: '<div />' })
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: Dummy },
      { path: '/login', component: Dummy },
      { path: '/admin/events', component: Dummy },
      { path: '/admin/events/:eventId', component: Dummy },
      { path: '/admin/events/:eventId/modules/:moduleId/edit', component: Dummy },
      { path: '/admin/events/:eventId/modules/:moduleId/host', component: Dummy },
      { path: '/admin/events/:eventId/projector', component: Dummy },
      { path: '/admin/events/:eventId/data', component: Dummy },
      { path: '/admin/events/:eventId/teams', component: Dummy },
      { path: '/admin/library/:tab', component: Dummy },
      { path: '/admin/users', component: Dummy }
    ]
  })
  return router.push(initialPath).then(() => router.isReady()).then(() => router)
}

const wrappers = []
function track(w) {
  wrappers.push(w)
  return w
}

beforeEach(() => {
  i18n.global.locale.value = 'en'
  sessionStorage.clear()
  clearDialogQueue()
  session.loaded = true
  session.user = { id: 'owner-1', role: 'owner', status: 'active' }
  session.csrfToken = 'csrf-123'
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

describe('AdminScreen (Phase 2)', () => {
  it('validates whitespace-only event names inline, edits an event, and deletes via useDialog confirm', async () => {
    const router = await makeRouter('/admin/events')
    const eventsList = [
      { id: 'e-1', name: 'Spring Meetup', description: 'Welcome', isActive: false, language: 'en' }
    ]
    const fetchMock = vi.fn((url, opts = {}) => {
      if (url === '/events' && (!opts.method || opts.method === 'GET')) {
        return Promise.resolve({ ok: true, json: async () => eventsList })
      }
      if (url === '/events/e-1' && opts.method === 'PUT') {
        return Promise.resolve({ ok: true, json: async () => ({ ...eventsList[0], name: 'Updated Meetup' }) })
      }
      if (url === '/events/e-1' && opts.method === 'DELETE') {
        return Promise.resolve({ ok: true, json: async () => ({}) })
      }
      return Promise.resolve({ ok: true, json: async () => ({}) })
    })
    vi.stubGlobal('fetch', fetchMock)

    track(mount(AppDialogHost, { attachTo: document.body, global: { plugins: [i18n] } }))
    const wrapper = track(mount(AdminScreen, {
      attachTo: document.body,
      global: { plugins: [i18n, router] }
    }))
    await flushPromises()

    expect(wrapper.find('.section-title').text()).toBe('Events')
    expect(wrapper.find('.back-button').exists()).toBe(false)
    expect(wrapper.find('.portal-link').attributes('href')).toBe('/')

    // Open New Event form and test whitespace-only validation
    await wrapper.find('.create-toggle-btn').trigger('click')
    await wrapper.find('#admin-event-name').setValue('   ')
    await wrapper.find('form.form-card').trigger('submit')
    await flushPromises()

    expect(wrapper.find('#admin-event-name-error').text()).toBe('Please enter an event name.')
    expect(fetchMock.mock.calls.some(([, o]) => o?.method === 'POST')).toBe(false)

    // Edit existing event populates and submits PUT
    await wrapper.find('.edit-action').trigger('click')
    await flushPromises()
    expect(wrapper.find('#admin-event-name').element.value).toBe('Spring Meetup')
    await wrapper.find('#admin-event-name').setValue('  Updated Meetup  ')
    await wrapper.find('form.form-card').trigger('submit')
    await flushPromises()

    const putCall = fetchMock.mock.calls.find(([u, o]) => u === '/events/e-1' && o?.method === 'PUT')
    expect(putCall).toBeTruthy()
    expect(JSON.parse(putCall[1].body).name).toBe('Updated Meetup')

    // Delete triggers AppDialogHost confirmation naming the event
    await wrapper.find('.delete-action').trigger('click')
    await flushPromises()
    const dialogMsg = document.body.querySelector('.app-dialog-message')?.textContent || ''
    expect(dialogMsg).toContain('Spring Meetup')
    document.body.querySelector('.app-dialog-btn--danger').click()
    await flushPromises()

    expect(fetchMock.mock.calls.some(([u, o]) => u === '/events/e-1' && o?.method === 'DELETE')).toBe(true)
  })
})

describe('AddModuleModal (Phase 3)', () => {
  it('shows 5 type cards, copies & normalizes a library template with new question IDs, and routes to module edit', async () => {
    const router = await makeRouter('/admin/events/evt-9')
    const pushSpy = vi.spyOn(router, 'push')

    const templatesPayload = [
      {
        id: 'tpl-q1',
        name: 'General Trivia',
        description: 'Fun trivia pack',
        moduleType: 'quiz',
        data: {
          backgroundMusicId: 'trk-1',
          questions: [
            { id: 'old-q-1', type: 'classic', text: 'Capital of France?', answers: ['Paris', 'Rome', '', ''], correctIndex: 0, points: 100 }
          ]
        }
      },
      {
        id: 'tpl-f1',
        name: 'Check-in',
        moduleType: 'form',
        data: { timeline: [{ id: 'old-s-1', type: 'field', data: { label: 'Name' } }] }
      }
    ]

    const fetchMock = vi.fn((url, opts = {}) => {
      if (url === '/templates') {
        return Promise.resolve({ ok: true, json: async () => templatesPayload })
      }
      if (url === '/events/evt-9/modules' && opts.method === 'POST') {
        return Promise.resolve({ ok: true, json: async () => ({ id: 'mod-new-1', type: 'quiz', name: 'Friday Trivia' }) })
      }
      return Promise.resolve({ ok: true, json: async () => ({}) })
    })
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = track(mount(AddModuleModal, {
      props: { visible: true, eventId: 'evt-9' },
      attachTo: document.body,
      global: { plugins: [i18n, router] }
    }))
    await flushPromises()

    const typeCards = document.body.querySelectorAll('.module-type-card')
    expect(typeCards.length).toBe(5)

    // Choose Quiz
    typeCards[0].click()
    await flushPromises()

    const nameInput = document.body.querySelector('#add-module-name')
    expect(nameInput.getAttribute('placeholder')).toBe('e.g. Friday Trivia')

    // Switch to Copy from Library template
    const sourceBtns = document.body.querySelectorAll('.source-option')
    sourceBtns[1].click()
    await flushPromises()

    const choices = document.body.querySelectorAll('.template-choice')
    expect(choices.length).toBe(1)
    expect(choices[0].textContent).toContain('General Trivia')
    choices[0].click()
    await flushPromises()

    // Name auto-filled from template when blank; customize it
    nameInput.value = 'Friday Trivia'
    nameInput.dispatchEvent(new Event('input', { bubbles: true }))
    await flushPromises()

    document.body.querySelector('form.step-body').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    await flushPromises()

    const postCall = fetchMock.mock.calls.find(([u, o]) => u === '/events/evt-9/modules' && o?.method === 'POST')
    expect(postCall).toBeTruthy()
    const sent = JSON.parse(postCall[1].body)
    expect(sent.type).toBe('quiz')
    expect(sent.name).toBe('Friday Trivia')
    expect(sent.data.questions[0].id).not.toBe('old-q-1')
    expect(sent.data.questions[0].explainTime).toBe(5)
    expect(wrapper.emitted('created')).toBeTruthy()
    expect(pushSpy).toHaveBeenCalledWith('/admin/events/evt-9/modules/mod-new-1/edit')
  })
})

describe('AdminEventDetailsScreen (Phase 2)', () => {
  it('renders localized module badges, Edit/Host routes, module-named delete dialog, and event start/stop toast', async () => {
    const router = await makeRouter('/admin/events/evt-1')
    const pushSpy = vi.spyOn(router, 'push')

    const eventPayload = {
      id: 'evt-1',
      name: 'Annual Gala',
      description: 'Evening party',
      pin: '445566',
      isActive: false,
      modules: [
        { id: 'm-1', name: 'Pub Quiz', type: 'quiz', isActive: false },
        { id: 'm-2', name: 'House Teams', type: 'teams', isActive: true }
      ]
    }

    const fetchMock = vi.fn((url, opts = {}) => {
      if (url === '/events/evt-1' && (!opts.method || opts.method === 'GET')) {
        return Promise.resolve({ ok: true, json: async () => eventPayload })
      }
      if (url === '/events/evt-1/status' && opts.method === 'PUT') {
        eventPayload.isActive = true
        return Promise.resolve({ ok: true, json: async () => ({}) })
      }
      if (url === '/events/evt-1/modules/m-1' && opts.method === 'DELETE') {
        return Promise.resolve({ ok: true, json: async () => ({}) })
      }
      return Promise.resolve({ ok: true, json: async () => ({}) })
    })
    vi.stubGlobal('fetch', fetchMock)

    track(mount(AppDialogHost, { attachTo: document.body, global: { plugins: [i18n] } }))
    const wrapper = track(mount(AdminEventDetailsScreen, {
      attachTo: document.body,
      global: {
        plugins: [i18n, router],
        stubs: { QRCodeVue: true }
      }
    }))
    await flushPromises()

    // Human-readable module type badges
    const badges = wrapper.findAll('.module-type-badge').map((b) => b.text().trim())
    expect(badges).toEqual(['Quiz', 'Teams'])

    // Edit & Host buttons route to canonical endpoints
    const editBtns = wrapper.findAll('.config-btn')
    expect(editBtns[0].text()).toBe('Edit')
    await editBtns[0].trigger('click')
    expect(pushSpy).toHaveBeenCalledWith('/admin/events/evt-1/modules/m-1/edit')

    const hostBtns = wrapper.findAll('.host-btn')
    expect(hostBtns[0].text()).toBe('Host')
    await hostBtns[0].trigger('click')
    expect(pushSpy).toHaveBeenCalledWith('/admin/events/evt-1/modules/m-1/host')

    await hostBtns[1].trigger('click')
    expect(pushSpy).toHaveBeenCalledWith('/admin/events/evt-1/teams?moduleId=m-2')

    // Copy Projector Link is labeled
    expect(wrapper.find('.copy-link-label').text()).toBe('Copy Projector Link')

    // Delete module opens confirmation naming the module
    await wrapper.find('.module-btn.delete-btn').trigger('click')
    await flushPromises()
    const msg = document.body.querySelector('.app-dialog-message')?.textContent || ''
    expect(msg).toContain('Pub Quiz')
    document.body.querySelector('.app-dialog-btn--danger').click()
    await flushPromises()
    expect(fetchMock.mock.calls.some(([u, o]) => u === '/events/evt-1/modules/m-1' && o?.method === 'DELETE')).toBe(true)
  })
})

describe('DataLibraryScreen, ScreenManagerScreen & AdminUsersScreen (Phase 2)', () => {
  it('DataLibraryScreen omits empty table shell when there are no rows and reads params.eventId', async () => {
    const router = await makeRouter('/admin/events/evt-5/data')
    vi.stubGlobal('fetch', vi.fn((url) => {
      if (url === '/events/evt-5') return Promise.resolve({ ok: true, json: async () => ({ id: 'evt-5', name: 'Summit', modules: [] }) })
      if (url === '/events/evt-5/participants') return Promise.resolve({ ok: true, json: async () => [] })
      if (url === '/events/evt-5/teams') return Promise.resolve({ ok: true, json: async () => [] })
      if (url === '/events/evt-5/scores') return Promise.resolve({ ok: true, json: async () => ({ participants: [], teams: [], games: [], entries: [] }) })
      return Promise.resolve({ ok: true, json: async () => ({}) })
    }))

    const wrapper = track(mount(DataLibraryScreen, {
      global: {
        plugins: [i18n, router],
        mocks: { $t: (k, p) => i18n.global.t(k, p) }
      }
    }))
    await flushPromises()

    expect(wrapper.find('h1').text()).toBe('Summit — Data')
    expect(wrapper.find('.table-shell').exists()).toBe(false)
    expect(wrapper.find('.empty-rows-note').exists()).toBe(true)
  })

  it('ScreenManagerScreen raises join URL contrast on dark Welcome slide preview', async () => {
    const router = await makeRouter('/admin/events/evt-7/projector')
    vi.stubGlobal('fetch', vi.fn((url) => {
      if (url === '/events/evt-7') return Promise.resolve({ ok: true, json: async () => ({ id: 'evt-7', name: 'Welcome Night', pin: '112233' }) })
      if (url === '/events/evt-7/slides') {
        return Promise.resolve({
          ok: true,
          json: async () => [{
            id: 's-1',
            type: 'welcome',
            name: 'Welcome',
            backgroundColor: '#1A1A1D',
            elements: [
              { id: 'default-welcome-url-evt-7', type: 'text', x: 45, y: 50, width: 45, height: 10, content: 'https://example.com/?pin=112233', fontSize: 45, color: '#3b82f6' }
            ]
          }]
        })
      }
      return Promise.resolve({ ok: true, json: async () => ({}) })
    }))

    const wrapper = track(mount(ScreenManagerScreen, {
      global: { plugins: [i18n, router], stubs: { QRCodeVue: true } }
    }))
    await flushPromises()

    const urlEl = wrapper.find('.canvas-element')
    expect(urlEl.attributes('style')).toMatch(/rgb\(147,\s*197,\s*253\)|#93c5fd/i)
  })

  it('AdminUsersScreen wraps in AdminLayout with Administrators breadcrumb', async () => {
    const router = await makeRouter('/admin/users')
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({
      ok: true,
      json: async () => ({ users: [], hasMore: false })
    })))

    const wrapper = track(mount(AdminUsersScreen, {
      global: { plugins: [i18n, router] }
    }))
    await flushPromises()

    expect(wrapper.find('.admin-breadcrumb-current').text()).toBe('Administrators')
    expect(wrapper.find('h1.users-title').text()).toBe('Administrators')
  })
})
