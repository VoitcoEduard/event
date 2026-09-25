import { createRouter, createWebHistory } from 'vue-router'
import MainScreen from '../views/MainScreen.vue'
import EventScreen from '../views/EventScreen.vue'
import AdminScreen from '../views/AdminScreen.vue'
import AdminEventDetailsScreen from '../views/AdminEventDetailsScreen.vue'
import DataLibraryScreen from '../views/DataLibraryScreen.vue'
import LoginScreen from '../views/LoginScreen.vue'

import QuizHostScreen from '../views/QuizHostScreen.vue'
import QuizPlayerScreen from '../views/QuizPlayerScreen.vue'
import ProjectorView from '../views/ProjectorView.vue'
import ScreenManagerScreen from '../views/ScreenManagerScreen.vue'

import TeamsAdminScreen from '../views/TeamsAdminScreen.vue'
import TeamSelectionScreen from '../views/TeamSelectionScreen.vue'
import MyTeamScreen from '../views/MyTeamScreen.vue'

import { loadSession, session } from '../session.js'

const routes = [
  { path: '/', name: 'main', component: MainScreen },
  { path: '/event', name: 'event', component: EventScreen },
  { path: '/login', name: 'login', component: LoginScreen },
	{ path: '/pending', name: 'pending', component: () => import('../views/PendingAccessScreen.vue') },
	{ path: '/admin/users', name: 'admin-users', component: () => import('../views/AdminUsersScreen.vue'), meta: { requiresOwner: true } },
  { path: '/admin', name: 'admin', component: AdminScreen, meta: { requiresAuth: true } },
  { path: '/admin/event/:id', name: 'admin-event-details', component: AdminEventDetailsScreen, meta: { requiresAuth: true } },
  { path: '/admin/event/:id/screen-manager', name: 'admin-screen-manager', component: ScreenManagerScreen, meta: { requiresAuth: true } },
  { path: '/admin/event/:id/teams', name: 'teams-admin', component: TeamsAdminScreen, meta: { requiresAuth: true } },
  { path: '/data/library', name: 'data-library', component: DataLibraryScreen, meta: { requiresAuth: true } },
  { path: '/quiz/library', name: 'quiz-library', component: () => import('../views/QuizLibraryScreen.vue'), meta: { requiresAuth: true } },
  { path: '/quiz/:eventId/:moduleId/host', name: 'quiz-host', component: QuizHostScreen, meta: { requiresAuth: true } },
  { path: '/quiz/:eventId/:moduleId/player', name: 'quiz-player', component: QuizPlayerScreen },
  { path: '/wrong-pictures/:eventId/:moduleId/host', name: 'wrong-pictures-host', component: QuizHostScreen, meta: { requiresAuth: true } },
  { path: '/wrong-pictures/:eventId/:moduleId/player', name: 'wrong-pictures-player', component: QuizPlayerScreen },
  { path: '/simple-quiz/:eventId/:moduleId/host', name: 'simple-quiz-host', component: () => import('../views/SimpleQuizHostScreen.vue'), meta: { requiresAuth: true } },
  { path: '/simple-quiz/:eventId/:moduleId/player', name: 'simple-quiz-player', component: () => import('../views/SimpleQuizPlayerScreen.vue') },
  { path: '/simple-quiz/:eventId/:moduleId/judge', name: 'simple-quiz-judge', component: () => import('../views/SimpleQuizJudgeScreen.vue') },
  { path: '/projector/:eventId', name: 'projector', component: ProjectorView },
  { path: '/event/team-selection', name: 'team-selection', component: TeamSelectionScreen },
  { path: '/event/my-team', name: 'my-team', component: MyTeamScreen }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach(async (to) => {
	if (!to.meta.requiresAuth && !to.meta.requiresOwner && to.name !== 'pending') return true
	try {
		await loadSession(true)
	} catch {
		return { name: 'login' }
	}
	if (!session.user) return { name: 'login' }
	if (session.user.status === 'pending') return to.name === 'pending' ? true : { name: 'pending' }
	if (session.user.status !== 'active') return { name: 'login', query: { error: 'disabled' } }
	if (to.meta.requiresOwner && session.user.role !== 'owner') return { name: 'admin' }
	if (to.name === 'pending') return { name: 'admin' }
	return true
})

export default router
