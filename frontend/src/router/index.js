import { createRouter, createWebHistory } from 'vue-router'
import MainScreen from '../views/MainScreen.vue'
import EventScreen from '../views/EventScreen.vue'
import AdminScreen from '../views/AdminScreen.vue'
import AdminEventDetailsScreen from '../views/AdminEventDetailsScreen.vue'

const routes = [
  { path: '/', name: 'main', component: MainScreen },
  { path: '/event', name: 'event', component: EventScreen },
  { path: '/admin', name: 'admin', component: AdminScreen },
  { path: '/admin/event/:id', name: 'admin-event-details', component: AdminEventDetailsScreen }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
