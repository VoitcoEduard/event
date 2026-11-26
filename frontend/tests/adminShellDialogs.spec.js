import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, nextTick, ref } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { flushPromises, mount } from '@vue/test-utils'
import i18n from '../src/i18n.js'
import { session } from '../src/session.js'
import { setAdminTheme, useAdminTheme } from '../src/composables/useAdminTheme.js'
import { clearDialogQueue, useDialog } from '../src/composables/useDialog.js'
import { useUnsavedChanges } from '../src/composables/useUnsavedChanges.js'
import AdminLayout from '../src/components/AdminLayout.vue'
import AdminThemeToggle from '../src/components/AdminThemeToggle.vue'
import AppDialogHost from '../src/components/AppDialogHost.vue'
import App from '../src/App.vue'

function makeTestRouter(initialPath = '/admin/events') {
  const Dummy = defineComponent({ template: '<div class="dummy-page">Dummy</div>' })
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: Dummy },
      { path: '/login', component: Dummy },
      { path: '/admin/events', component: Dummy },
      { path: '/admin/event/:id', component: Dummy },
      { path: '/admin/library/quizzes', component: Dummy },
      { path: '/admin/users', component: Dummy }
    ]
  })
  return { router, initialPath }
}

let wrappers = []

function track(wrapper) {
  wrappers.push(wrapper)
  return wrapper
}

beforeEach(() => {
  localStorage.clear()
  sessionStorage.clear()
  i18n.global.locale.value = 'en'
  setAdminTheme('light')
  clearDialogQueue()
  session.loaded = true
  session.user = { id: 'u-1', role: 'admin', status: 'active' }
  session.csrfToken = 'csrf-test'
  vi.restoreAllMocks()
})

afterEach(() => {
  clearDialogQueue()
  while (wrappers.length) {
    wrappers.pop().unmount()
  }
  document.body.innerHTML = ''
})

describe('AdminThemeToggle', () => {
  it('supports floating default and inline variant', async () => {
    const floating = track(mount(AdminThemeToggle, { global: { plugins: [i18n] } }))
    expect(floating.find('.admin-theme-control').classes()).not.toContain('admin-theme-control--inline')

    const inline = track(mount(AdminThemeToggle, {
      props: { variant: 'inline' },
      global: { plugins: [i18n] }
    }))
    const control = inline.find('.admin-theme-control')
    expect(control.classes()).toContain('admin-theme-control--inline')

    const { theme } = useAdminTheme()
    expect(theme.value).toBe('light')
    await inline.findAll('.admin-theme-option')[1].trigger('click')
    expect(theme.value).toBe('dark')
  })
})

describe('AdminLayout', () => {
  it('renders brand link, nav links with active state, owner-only Users link, utility cluster, and breadcrumbs/actions', async () => {
    const { router } = makeTestRouter()
    await router.push('/admin/events')
    await router.isReady()

    const wrapper = track(mount(AdminLayout, {
      props: {
        breadcrumbs: [
          { label: 'Events', to: '/admin/events' },
          { label: 'Spring Quiz' }
        ]
      },
      slots: {
        actions: '<button type="button" class="test-page-action">Save</button>',
        default: '<main class="test-body">Page Body</main>'
      },
      global: { plugins: [i18n, router] }
    }))

    expect(wrapper.find('.admin-brand').attributes('href')).toBe('/admin/events')

    const navLinks = wrapper.findAll('.admin-nav-link')
    expect(navLinks.map(l => l.attributes('href'))).toEqual(['/admin/events', '/admin/library/quizzes'])
    expect(navLinks[0].classes()).toContain('active')
    expect(navLinks[0].attributes('aria-current')).toBe('page')
    expect(navLinks[1].classes()).not.toContain('active')

    session.user = { id: 'u-owner', role: 'owner', status: 'active' }
    await nextTick()
    const ownerLinks = wrapper.findAll('.admin-nav-link')
    expect(ownerLinks.map(l => l.attributes('href'))).toEqual([
      '/admin/events',
      '/admin/library/quizzes',
      '/admin/users'
    ])

    await router.push('/admin/library/quizzes')
    await flushPromises()
    const updatedLinks = wrapper.findAll('.admin-nav-link')
    expect(updatedLinks[0].classes()).not.toContain('active')
    expect(updatedLinks[1].classes()).toContain('active')

    expect(wrapper.find('.admin-theme-control').classes()).toContain('admin-theme-control--inline')
    expect(wrapper.find('.test-page-action').text()).toBe('Save')
    expect(wrapper.find('.test-body').text()).toBe('Page Body')
    expect(wrapper.find('.admin-breadcrumb-link').attributes('href')).toBe('/admin/events')
    expect(wrapper.find('.admin-breadcrumb-current').attributes('aria-current')).toBe('page')
  })

  it('syncs language select with i18n locale and localStorage, toggles mobile menu, and signs out', async () => {
    const { router } = makeTestRouter()
    await router.push('/admin/events')
    await router.isReady()

    const fetchSpy = vi.fn().mockResolvedValue({ ok: true, status: 200 })
    vi.stubGlobal('fetch', fetchSpy)

    const wrapper = track(mount(AdminLayout, {
      global: { plugins: [i18n, router] }
    }))

    const select = wrapper.find('.admin-lang-select')
    await select.setValue('ro')
    expect(i18n.global.locale.value).toBe('ro')
    expect(localStorage.getItem('userLanguage')).toBe('ro')

    const toggle = wrapper.find('.admin-menu-toggle')
    expect(toggle.attributes('aria-expanded')).toBe('false')
    await toggle.trigger('click')
    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(wrapper.find('#admin-shell-nav-panel').classes()).toContain('is-open')

    await wrapper.find('.admin-signout-btn').trigger('click')
    await flushPromises()
    expect(fetchSpy).toHaveBeenCalledWith('/auth/logout', expect.objectContaining({ method: 'POST' }))
    expect(router.currentRoute.value.path).toBe('/login')
  })

  it('closes the mobile drawer when tapping the link for the current page (L7b)', async () => {
    const { router } = makeTestRouter()
    await router.push('/admin/events')
    await router.isReady()

    const wrapper = track(mount(AdminLayout, { global: { plugins: [i18n, router] } }))

    const toggle = wrapper.find('.admin-menu-toggle')
    await toggle.trigger('click')
    expect(wrapper.find('#admin-shell-nav-panel').classes()).toContain('is-open')

    await wrapper.find('.admin-brand').trigger('click')
    expect(wrapper.find('#admin-shell-nav-panel').classes()).not.toContain('is-open')

    await toggle.trigger('click')
    const eventsLink = wrapper.findAll('.admin-nav-link')[0]
    expect(eventsLink.attributes('href')).toBe('/admin/events')
    await eventsLink.trigger('click')
    expect(wrapper.find('#admin-shell-nav-panel').classes()).not.toContain('is-open')
    expect(router.currentRoute.value.path).toBe('/admin/events')
  })
})

describe('AppDialogHost & useDialog', () => {
  it('handles confirm resolve/cancel, destructive role, Escape key, Enter submit, and focus restoration', async () => {
    const triggerBtn = document.createElement('button')
    triggerBtn.textContent = 'Trigger'
    document.body.appendChild(triggerBtn)
    triggerBtn.focus()

    track(mount(AppDialogHost, {
      attachTo: document.body,
      global: { plugins: [i18n] }
    }))

    const { confirm } = useDialog()
    const promise1 = confirm({
      title: 'Delete event?',
      message: 'This cannot be undone.',
      confirmLabel: 'Delete',
      cancelLabel: 'Keep',
      destructive: true
    })
    await flushPromises()

    const dialogEl = document.body.querySelector('.app-dialog')
    expect(dialogEl).not.toBeNull()
    expect(dialogEl.getAttribute('role')).toBe('alertdialog')
    expect(dialogEl.querySelector('.app-dialog-btn--danger')?.textContent.trim()).toBe('Delete')

    dialogEl.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await expect(promise1).resolves.toBe(false)
    await flushPromises()
    expect(document.body.querySelector('.app-dialog')).toBeNull()
    expect(document.activeElement).toBe(triggerBtn)

    const promise2 = confirm({ title: 'Proceed?', message: 'Confirm action' })
    await flushPromises()
    const dialogEl2 = document.body.querySelector('.app-dialog')
    expect(dialogEl2.getAttribute('role')).toBe('dialog')
    dialogEl2.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    await expect(promise2).resolves.toBe(true)
  })

  it('handles prompt defaultValue, form submission, cancellation, and queueing', async () => {
    track(mount(AppDialogHost, {
      attachTo: document.body,
      global: { plugins: [i18n] }
    }))

    const { prompt, confirm } = useDialog()
    const p1 = prompt({
      title: 'Template name',
      label: 'Name',
      defaultValue: 'Quiz 1',
      placeholder: 'Enter name'
    })
    const p2 = confirm({ title: 'Second queued' })
    await flushPromises()

    const input = document.body.querySelector('.app-dialog-input')
    expect(input).not.toBeNull()
    expect(input.value).toBe('Quiz 1')
    expect(document.activeElement).toBe(input)

    input.value = 'Updated Quiz'
    input.dispatchEvent(new Event('input', { bubbles: true }))
    await nextTick()
    document.body.querySelector('form.app-dialog-body').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))

    await expect(p1).resolves.toBe('Updated Quiz')
    await flushPromises()

    const secondTitle = document.body.querySelector('.app-dialog-title')?.textContent.trim()
    expect(secondTitle).toBe('Second queued')
    document.body.querySelector('.app-dialog-btn--secondary').click()
    await expect(p2).resolves.toBe(false)
  })

  it('mounts AppDialogHost and omits global fixed AdminThemeToggle in App.vue', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({ status: 401, ok: false })))
    const { router } = makeTestRouter('/')
    await router.push('/')
    await router.isReady()

    const appWrapper = track(mount(App, {
      global: {
        plugins: [i18n, router],
        stubs: { Toast: true }
      }
    }))
    await flushPromises()
    expect(appWrapper.findComponent(AdminThemeToggle).exists()).toBe(false)
    expect(appWrapper.findComponent(AppDialogHost).exists()).toBe(true)
  })
})

describe('useUnsavedChanges', () => {
  it('attaches beforeunload only when dirty, cleans up on unmount, and guards route leave with useDialog', async () => {
    track(mount(AppDialogHost, {
      attachTo: document.body,
      global: { plugins: [i18n] }
    }))

    const addSpy = vi.spyOn(window, 'addEventListener')
    const removeSpy = vi.spyOn(window, 'removeEventListener')
    const isDirty = ref(false)

    const EditorView = defineComponent({
      setup() {
        useUnsavedChanges(isDirty)
        return () => h('div', { class: 'editor-view' }, 'Editor')
      }
    })
    const OtherView = defineComponent({
      template: '<div class="other-view">Other</div>'
    })

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/editor', component: EditorView },
        { path: '/other', component: OtherView }
      ]
    })
    await router.push('/editor')
    await router.isReady()

    const Root = defineComponent({ template: '<router-view />' })
    const wrapper = track(mount(Root, {
      global: { plugins: [i18n, router] }
    }))

    expect(addSpy.mock.calls.filter(([evt]) => evt === 'beforeunload')).toHaveLength(0)

    isDirty.value = true
    await nextTick()
    expect(addSpy.mock.calls.filter(([evt]) => evt === 'beforeunload')).toHaveLength(1)

    const unloadEvent = new Event('beforeunload', { cancelable: true })
    window.dispatchEvent(unloadEvent)
    expect(unloadEvent.defaultPrevented).toBe(true)

    const navCancel = router.push('/other')
    await flushPromises()
    expect(document.body.querySelector('.app-dialog')).not.toBeNull()
    document.body.querySelector('.app-dialog-btn--secondary').click()
    await navCancel
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/editor')

    const navConfirm = router.push('/other')
    await flushPromises()
    document.body.querySelector('.app-dialog-btn--danger').click()
    await navConfirm
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/other')
    expect(removeSpy.mock.calls.some(([evt]) => evt === 'beforeunload')).toBe(true)

    wrapper.unmount()
  })
})
