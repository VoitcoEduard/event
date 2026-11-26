import { createRouter, createWebHistory } from 'vue-router'
import MainScreen from '../views/MainScreen.vue'
import EventScreen from '../views/EventScreen.vue'
import AdminScreen from '../views/AdminScreen.vue'
import AdminEventDetailsScreen from '../views/AdminEventDetailsScreen.vue'
import DataLibraryScreen from '../views/DataLibraryScreen.vue'
import LoginScreen from '../views/LoginScreen.vue'

import QuizPlayerScreen from '../views/QuizPlayerScreen.vue'
import ProjectorView from '../views/ProjectorView.vue'
import ScreenManagerScreen from '../views/ScreenManagerScreen.vue'

import TeamsAdminScreen from '../views/TeamsAdminScreen.vue'
import TeamsActivityPicker from '../views/TeamsActivityPicker.vue'
import TeamsPlayerScreen from '../views/TeamsPlayerScreen.vue'

import { loadSession, session } from '../session.js'

const returnKey = 'adminReturnPath'

export const routes = [
  { path: '/', name: 'main', component: MainScreen },
  { path: '/event', name: 'event', component: EventScreen },
  { path: '/login', name: 'login', component: LoginScreen, meta: { adminAppearance: true } },
  { path: '/pending', name: 'pending', component: () => import('../views/PendingAccessScreen.vue'), meta: { adminAppearance: true } },
  { path: '/quiz/:eventId/:moduleId/player', name: 'quiz-player', component: QuizPlayerScreen },
  { path: '/wrong-pictures/:eventId/:moduleId/player', name: 'wrong-pictures-player', component: QuizPlayerScreen },
  { path: '/simple-quiz/:eventId/:moduleId/player', name: 'simple-quiz-player', component: () => import('../views/SimpleQuizPlayerScreen.vue') },
  { path: '/simple-quiz/:eventId/:moduleId/judge', name: 'simple-quiz-judge', component: () => import('../views/SimpleQuizJudgeScreen.vue'), meta: { adminAppearance: true } },
  { path: '/projector/:eventId', name: 'projector', component: ProjectorView },
  { path: '/teams/:eventId/:moduleId/player', name: 'teams-player', component: TeamsPlayerScreen },
  { path: '/event/team-selection', name: 'team-selection', component: TeamsActivityPicker },
  { path: '/event/my-team', name: 'my-team', component: TeamsActivityPicker },

  { path: '/admin/events', name: 'admin-events', component: AdminScreen, meta: { requiresAuth: true, adminAppearance: true } },
  { path: '/admin/events/:eventId', name: 'admin-event-details', component: AdminEventDetailsScreen, meta: { requiresAuth: true, adminAppearance: true } },
  { path: '/admin/events/:eventId/modules/:moduleId/edit', name: 'admin-module-editor', component: () => import('../views/AdminModuleEditorScreen.vue'), meta: { requiresAuth: true, adminAppearance: true } },
  { path: '/admin/events/:eventId/modules/:moduleId/host', name: 'admin-module-host', component: () => import('../views/AdminModuleHostScreen.vue'), meta: { requiresAuth: true, adminAppearance: true } },
  { path: '/admin/events/:eventId/projector', name: 'admin-screen-manager', component: ScreenManagerScreen, meta: { requiresAuth: true, adminAppearance: true } },
  { path: '/admin/events/:eventId/teams', name: 'teams-admin', component: TeamsAdminScreen, meta: { requiresAuth: true, adminAppearance: true } },
  { path: '/admin/events/:eventId/data', name: 'data-library', component: DataLibraryScreen, meta: { requiresAuth: true, adminAppearance: true } },
  { path: '/admin/library/:tab', name: 'admin-library', component: () => import('../views/QuizLibraryScreen.vue'), meta: { requiresAuth: true, adminAppearance: true } },
  { path: '/admin/library/:tab/new', name: 'admin-template-new', component: () => import('../views/AdminTemplateEditorScreen.vue'), meta: { requiresAuth: true, adminAppearance: true } },
  { path: '/admin/library/:tab/:templateId/edit', name: 'admin-template-edit', component: () => import('../views/AdminTemplateEditorScreen.vue'), meta: { requiresAuth: true, adminAppearance: true } },
  { path: '/admin/users', name: 'admin-users', component: () => import('../views/AdminUsersScreen.vue'), meta: { requiresOwner: true, adminAppearance: true } },

  { path: '/admin', redirect: to => ({ name: 'admin-events', query: to.query, hash: to.hash }) },
  { path: '/admin/event/:id', redirect: to => ({ name: 'admin-event-details', params: { eventId: to.params.id }, query: to.query, hash: to.hash }) },
  { path: '/admin/event/:id/screen-manager', redirect: to => ({ name: 'admin-screen-manager', params: { eventId: to.params.id }, query: to.query, hash: to.hash }) },
  { path: '/admin/event/:id/teams', redirect: to => ({ name: 'teams-admin', params: { eventId: to.params.id }, query: to.query, hash: to.hash }) },
  {
    path: '/data/library',
    redirect: to => {
      const eventId = typeof to.query.eventId === 'string' ? to.query.eventId.trim() : ''
      if (eventId) {
        const { eventId: _dropped, ...restQuery } = to.query
        return { name: 'data-library', params: { eventId }, query: restQuery, hash: to.hash }
      }
      return { name: 'admin-events' }
    }
  },
  { path: '/admin/library', redirect: to => ({ name: 'admin-library', params: { tab: 'quizzes' }, query: to.query, hash: to.hash }) },
  { path: '/quiz/library', redirect: to => ({ name: 'admin-library', params: { tab: 'quizzes' }, query: to.query, hash: to.hash }) },
  { path: '/quiz/:eventId/:moduleId/host', redirect: to => ({ name: 'admin-module-host', params: { eventId: to.params.eventId, moduleId: to.params.moduleId }, query: to.query, hash: to.hash }) },
  { path: '/wrong-pictures/:eventId/:moduleId/host', redirect: to => ({ name: 'admin-module-host', params: { eventId: to.params.eventId, moduleId: to.params.moduleId }, query: to.query, hash: to.hash }) },
  { path: '/simple-quiz/:eventId/:moduleId/host', redirect: to => ({ name: 'admin-module-host', params: { eventId: to.params.eventId, moduleId: to.params.moduleId }, query: to.query, hash: to.hash }) }
]

const hasUnsafePathChars = (value) => (
  !value ||
  !value.startsWith('/') ||
  value.startsWith('//') ||
  value.includes('\\') ||
  /[\u0000-\u001f\u007f]/.test(value)
)

const isAdminPath = (pathname) => pathname === '/admin' || pathname.startsWith('/admin/')

const resolveFollowingRedirects = (rawPath, targetRouter) => {
  let resolved = targetRouter.resolve(rawPath)
  for (let depth = 0; depth < 5; depth++) {
    const lastRecord = resolved.matched[resolved.matched.length - 1]
    if (!lastRecord?.redirect) break
    const nextLocation = typeof lastRecord.redirect === 'function'
      ? lastRecord.redirect(resolved)
      : lastRecord.redirect
    resolved = targetRouter.resolve(nextLocation)
  }
  return resolved
}

export function sanitizeAdminReturnTo(raw, targetRouter = router) {
  if (typeof raw !== 'string') return null
  const trimmed = raw.trim()
  if (hasUnsafePathChars(trimmed) || !isAdminPath(trimmed)) return null

  try {
    const parsed = new URL(trimmed, 'http://localhost')
    if (parsed.origin !== 'http://localhost' || !isAdminPath(parsed.pathname)) return null
    const candidate = `${parsed.pathname}${parsed.search}${parsed.hash}`
    const resolved = resolveFollowingRedirects(candidate, targetRouter)
    if (!resolved.matched.some(record => record.meta.requiresAuth || record.meta.requiresOwner)) {
      return null
    }
    if (resolved.matched.some(record => record.meta.requiresOwner) && session.user?.role !== 'owner') {
      return null
    }
    return resolved.fullPath
  } catch {
    return null
  }
}

const rememberReturnPath = (to) => {
  if (to.name !== 'pending' && to.name !== 'login') {
    try { sessionStorage.setItem(returnKey, to.fullPath) } catch {  }
  }
}

const takeReturnPath = (targetRouter = router) => {
  try {
    const path = sessionStorage.getItem(returnKey)
    sessionStorage.removeItem(returnKey)
    if (hasUnsafePathChars(path)) return null
    const destination = resolveFollowingRedirects(path, targetRouter)
    if (destination.matched.some(record => record.meta.requiresOwner) && session.user?.role !== 'owner') return null
    return destination.matched.some(record => record.meta.requiresAuth || record.meta.requiresOwner)
      ? destination.fullPath
      : null
  } catch {
    return null
  }
}

export function attachAdminGuards(targetRouter) {
  targetRouter.beforeEach(async (to) => {
    if (!to.meta.requiresAuth && !to.meta.requiresOwner && to.name !== 'pending') return true

    try {
      await loadSession(true)
    } catch {
      rememberReturnPath(to)
      return { name: 'login' }
    }

    if (!session.user) {
      rememberReturnPath(to)
      return { name: 'login' }
    }
    if (session.user.status === 'pending') {
      return to.name === 'pending' ? true : { name: 'pending' }
    }
    if (session.user.status !== 'active') {
      return { name: 'login', query: { error: 'disabled' } }
    }
    if (to.meta.requiresOwner && session.user.role !== 'owner') {
      return { name: 'admin-events' }
    }

    if (to.name === 'admin-events' || to.name === 'pending') {
      const destination = takeReturnPath(targetRouter)
      if (destination && destination !== to.fullPath) return destination
    }
    if (to.name === 'pending') return { name: 'admin-events' }

    if (to.path.startsWith('/admin/library') && Object.prototype.hasOwnProperty.call(to.query, 'returnTo')) {
      const rawReturnTo = Array.isArray(to.query.returnTo) ? to.query.returnTo[0] : to.query.returnTo
      const validReturnTo = sanitizeAdminReturnTo(rawReturnTo, targetRouter)
      if (!validReturnTo) {
        const { returnTo: _invalid, ...restQuery } = to.query
        return { path: to.path, query: restQuery, hash: to.hash, replace: true }
      }
      if (validReturnTo !== rawReturnTo) {
        return { path: to.path, query: { ...to.query, returnTo: validReturnTo }, hash: to.hash, replace: true }
      }
    }

    return true
  })

  return targetRouter
}

export function createAppRouter(history = createWebHistory()) {
  const instance = createRouter({ history, routes })
  return attachAdminGuards(instance)
}

const router = createAppRouter()

export default router
