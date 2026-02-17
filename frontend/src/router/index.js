import { createRouter, createWebHistory } from 'vue-router'
import MainScreen from '../views/MainScreen.vue'
import EventScreen from '../views/EventScreen.vue'
import AdminScreen from '../views/AdminScreen.vue'
import AdminEventDetailsScreen from '../views/AdminEventDetailsScreen.vue'
import DataLibraryScreen from '../views/DataLibraryScreen.vue'

import QuizHostScreen from '../views/QuizHostScreen.vue'
import QuizPlayerScreen from '../views/QuizPlayerScreen.vue'
import BigScreen from '../views/BigScreen.vue'

const routes = [
  { path: '/', name: 'main', component: MainScreen },
  { path: '/event', name: 'event', component: EventScreen },
  { path: '/admin', name: 'admin', component: AdminScreen },
  { path: '/admin/event/:id', name: 'admin-event-details', component: AdminEventDetailsScreen },
  { path: '/data/library', name: 'data-library', component: DataLibraryScreen },
  { path: '/quiz/library', name: 'quiz-library', component: () => import('../views/QuizLibraryScreen.vue') },
  { path: '/quiz/:eventId/:moduleId/host', name: 'quiz-host', component: QuizHostScreen },
  { path: '/quiz/:eventId/:moduleId/player', name: 'quiz-player', component: QuizPlayerScreen },
  { path: '/bigscreen/:eventId/:moduleId', name: 'bigscreen', component: BigScreen }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
