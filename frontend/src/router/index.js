import { createRouter, createWebHistory } from 'vue-router'
import MainScreen from '../views/MainScreen.vue'
import EventScreen from '../views/EventScreen.vue'
import AdminScreen from '../views/AdminScreen.vue'
import AdminEventDetailsScreen from '../views/AdminEventDetailsScreen.vue'
import DataLibraryScreen from '../views/DataLibraryScreen.vue' // Changed import
import DataViewPage from '../views/DataViewPage.vue'

import QuizHostScreen from '../views/QuizHostScreen.vue'
import QuizPlayerScreen from '../views/QuizPlayerScreen.vue'

const routes = [
  { path: '/', name: 'main', component: MainScreen },
  { path: '/event', name: 'event', component: EventScreen },
  { path: '/admin', name: 'admin', component: AdminScreen },
  { path: '/admin/event/:id', name: 'admin-event-details', component: AdminEventDetailsScreen },
  { path: '/data/library', name: 'data-library', component: DataLibraryScreen }, // New route
  { path: '/data/view', name: 'data-view', component: DataViewPage }, // Data view route
  { path: '/quiz/library', name: 'quiz-library', component: () => import('../views/QuizLibraryScreen.vue') },
  { path: '/quiz/:eventId/:moduleId/host', name: 'quiz-host', component: QuizHostScreen },
  { path: '/quiz/:eventId/:moduleId/player', name: 'quiz-player', component: QuizPlayerScreen }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
