import { createRouter, createWebHistory } from 'vue-router'
import MainScreen from '../views/MainScreen.vue'
import EventScreen from '../views/EventScreen.vue'
import AdminScreen from '../views/AdminScreen.vue'
import AdminEventDetailsScreen from '../views/AdminEventDetailsScreen.vue'
import DataLibraryScreen from '../views/DataLibraryScreen.vue'
import LoginScreen from '../views/LoginScreen.vue'

import QuizHostScreen from '../views/QuizHostScreen.vue'
import QuizPlayerScreen from '../views/QuizPlayerScreen.vue'
import BigScreen from '../views/BigScreen.vue'
import ScreenManagerScreen from '../views/ScreenManagerScreen.vue'

import { isAuthenticated } from '../utils/index.js'

const routes = [
  { path: '/', name: 'main', component: MainScreen },
  { path: '/event', name: 'event', component: EventScreen },
  { path: '/login', name: 'login', component: LoginScreen },
  { path: '/admin', name: 'admin', component: AdminScreen, meta: { requiresAuth: true } },
  { path: '/admin/event/:id', name: 'admin-event-details', component: AdminEventDetailsScreen, meta: { requiresAuth: true } },
  { path: '/admin/event/:id/screen-manager', name: 'admin-screen-manager', component: ScreenManagerScreen, meta: { requiresAuth: true } },
  { path: '/data/library', name: 'data-library', component: DataLibraryScreen, meta: { requiresAuth: true } },
  { path: '/quiz/library', name: 'quiz-library', component: () => import('../views/QuizLibraryScreen.vue'), meta: { requiresAuth: true } },
  { path: '/quiz/:eventId/:moduleId/host', name: 'quiz-host', component: QuizHostScreen, meta: { requiresAuth: true } },
  { path: '/quiz/:eventId/:moduleId/player', name: 'quiz-player', component: QuizPlayerScreen },
  { path: '/wrong-pictures/:eventId/:moduleId/host', name: 'wrong-pictures-host', component: QuizHostScreen, meta: { requiresAuth: true } },
  { path: '/wrong-pictures/:eventId/:moduleId/player', name: 'wrong-pictures-player', component: QuizPlayerScreen },
  { path: '/bigscreen/:eventId/:moduleId', name: 'bigscreen', component: BigScreen },
  { path: '/bigscreen/:eventId', name: 'bigscreen-screensaver', component: BigScreen }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, _from, next) => {
  if (to.meta.requiresAuth && !isAuthenticated()) {
    next({ name: 'login' })
  } else {
    next()
  }
})

export default router
