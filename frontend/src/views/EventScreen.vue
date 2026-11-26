<template>
  <div class="event-screen" :class="'theme-' + theme">
    <div class="responsive-wrapper">
      <div class="top-controls">
        <button @click="toggleTheme" class="theme-toggle" :title="$t('event.toggle_theme')">
          <AppIcon :name="theme === 'bold' ? 'palette' : 'settings'" :size="20" />
        </button>
        <div class="language-switcher-container">
          <select v-model="$i18n.locale" @change="changeLanguage" class="lang-select">
            <option value="en">English</option>
            <option value="ru">Русский</option>
            <option value="ro">Română</option>
          </select>
        </div>
      </div>

      <div v-if="!currentEvent" class="error-container">
        <p class="error-text">{{ $t('event.no_data') }}</p>
        <button class="back-button" @click="goToMain">{{ $t('event.go_back') }}</button>
      </div>

      <div v-else-if="selectedModule" class="module-view">
        <div class="module-header">
          <button class="back-link" @click="closeSelectedModule">
            <AppIcon class="back-arrow" name="arrowLeft" :size="18" />
            <span>{{ $t('event.back_to_event') }}</span>
          </button>
        </div>
        <FormModule
          v-if="selectedModule.type === 'form'"
          :module="selectedModule"
          :event-id="currentEvent.id"
          :user-id="userId"
          :is-onboarding="Boolean(selectedModule?.data?.mandatoryOnEntry && selectedModule?.name === 'Welcome')"
          @complete="handleModuleComplete"
        />
      </div>

      <div v-else class="content">
        <div class="user-profile">
          <div class="user-main">
            <div class="avatar">
              {{ userName ? userName.charAt(0).toUpperCase() : '?' }}
            </div>
            <div class="user-details">
              <h2 class="user-name">{{ userName || $t('event.guest') }}</h2>
            </div>
          </div>
          <div v-if="participantStatus" class="points-badge">
            <span class="points-value">{{ participantStatus.totalPoints || 0 }}</span>
              <span class="points-label">{{ $t('event.points_short') }}</span>
          </div>
        </div>

        <div class="event-header">
          <p class="connected-label">{{ $t('event.connected_to') }}</p>
          <h1 class="event-title">{{ currentEvent.name }}</h1>
        </div>

        <p v-if="registrationNotice" class="registration-notice">{{ registrationNotice }}</p>

        <div class="list-container">
          <div v-if="dynamicCards.length > 0" class="dynamic-section">
            <div
              v-for="card in dynamicCards"
              :key="card.id"
              class="dynamic-card"
              :class="card.type"
              :style="{ borderLeftColor: card.color }"
            >
              <AppIcon class="card-icon" :name="dynamicCardIcon(card)" :size="22" />
              <div class="card-body">
                <h3 class="card-title">{{ card.title }}</h3>
                <p class="card-text">{{ card.content }}</p>
              </div>
            </div>
          </div>

          <template v-if="activeModules.length > 0">
            <p class="section-title">{{ $t('event.active_activities') }}</p>
            <div
              v-for="module in activeModules"
              :key="module.id"
              class="module-card"
              @click="selectModule(module)"
            >
              <div class="module-icon-box">
                <AppIcon class="module-emoji" :name="moduleIcon(module.type)" :size="24" />
              </div>
              <div class="module-main">
                <p class="module-name">{{ module.name }}</p>
                <p class="module-call">{{ $t('event.tap_to_start') }}</p>
              </div>
              <AppIcon class="module-arrow" name="arrowRight" :size="20" />
            </div>
          </template>

          <template v-else>
            <div class="waiting-state">
              <div class="spinner"></div>
              <p class="wait-title">{{ $t('event.waiting_msg') }}</p>
              <p class="wait-sub">{{ $t('event.waiting_submsg') }}</p>
            </div>
          </template>
        </div>

        <button class="exit-button" @click="goToMain">
          {{ $t('event.leave_btn') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import FormModule from '../components/FormModule.vue'
import AppIcon from '../components/AppIcon.vue'
import { API_BASE_URL, getUserId, setUserId, newUUID, getUserName, setUserName, getParticipantSecret, setParticipantSecret } from '../utils'

const { locale } = useI18n()
const router = useRouter()
const route = useRoute()
const userId = ref(getUserId())
const userName = ref(getUserName())
const registrationNotice = ref('')

const changeLanguage = () => {
  try { localStorage.setItem('userLanguage', locale.value) } catch {  }
}

const params = new URLSearchParams(window.location.search)
const queryEventId = params.get('id') || params.get('eventId')
const readStoredEvent = () => {
  if (queryEventId) return { id: queryEventId }
  try {
    const value = JSON.parse(sessionStorage.getItem('currentEvent') || 'null')
    if (value && typeof value.id === 'string' && value.id) return value
    sessionStorage.removeItem('currentEvent')
  } catch { try { sessionStorage.removeItem('currentEvent') } catch {  } }
  return null
}
const currentEvent = ref(readStoredEvent())
const theme = ref((() => { try { return localStorage.getItem('userTheme') || 'bold' } catch { return 'bold' } })())

const toggleTheme = () => {
  theme.value = theme.value === 'bold' ? 'material' : 'bold'
  try { localStorage.setItem('userTheme', theme.value) } catch {  }
}

const moduleIcon = (type) => ({
  form: 'form',
  quiz: 'quiz',
  wrong_pictures: 'image',
  simple_quiz: 'edit',
  teams: 'teams'
}[type] || 'chart')

const dynamicCardIcon = (card) => {
  const supported = ['alert', 'info', 'message', 'link', 'play', 'check', 'chart', 'teams', 'image']
  if (supported.includes(card.icon)) return card.icon
  return ({ alert: 'alert', action: 'play', info: 'info' }[card.type] || 'info')
}

const activeModules = ref([])
const dynamicCards = ref([])
const participantStatus = ref(null)
const selectedModule = ref(null)
let pollingTimer = null
let completionTimer = null
let activeRequest = null
let joinController = null
let stopped = false
let fetchGeneration = 0

const fetchEvent = async () => {
	if (stopped || !currentEvent.value?.id) return
	if (pollingTimer) clearTimeout(pollingTimer)
	if (activeRequest) activeRequest.abort()
	const controller = new AbortController()
	activeRequest = controller
  const generation = ++fetchGeneration
  const eventId = currentEvent.value.id

  try {
    const response = await fetch(`${API_BASE_URL}/events/${eventId}?userId=${userId.value}`, { signal: controller.signal })
    if (response.ok) {
      const data = await response.json()
      if (stopped || generation !== fetchGeneration || currentEvent.value?.id !== eventId) return
      currentEvent.value = data

      participantStatus.value = data.participantStatus || null
      dynamicCards.value = data.dynamicCards || []

      if (participantStatus.value?.displayName && participantStatus.value.displayName !== userName.value) {
        setUserName(participantStatus.value.displayName)
        userName.value = participantStatus.value.displayName
      }

      if (data.modules) {
        const active = data.modules.filter(m => m.isActive)

        activeModules.value = active.filter(m => {
          if (m.data?.hideAfterCompletion) {
            try { return sessionStorage.getItem(`completed_module_${m.id}`) !== 'true' } catch { return true }
          }
          return true
        })

        const routeModuleId = route.params.moduleId
        if (routeModuleId && !selectedModule.value) {
          const moduleToSelect = data.modules.find(m => m.id === routeModuleId)
          if (moduleToSelect) {
            selectedModule.value = moduleToSelect
          }
        }

        if (!selectedModule.value && !routeModuleId) {
            const mandatoryModule = active.find(m => {
              if (!m.data?.mandatoryOnEntry) return false
              try { return sessionStorage.getItem(`completed_module_${m.id}`) !== 'true' } catch { return true }
            })
            if (mandatoryModule) {
               selectModule(mandatoryModule)
            }
        }
      }
    }
  } catch (error) {
	if (error?.name !== 'AbortError') console.error('Polling error:', error)
  } finally {
	if (activeRequest === controller) {
		activeRequest = null
		if (!stopped) pollingTimer = setTimeout(fetchEvent, 3000)
	}
  }
}

onMounted(async () => {
  if (currentEvent.value?.id) {
	joinController = new AbortController()
    try {
      const eventId = currentEvent.value.id
      const previousSecret = getParticipantSecret(eventId)
      const join = (id, secret) => fetch(`${API_BASE_URL}/events/${eventId}/participants/join`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ userId: id, displayName: userName.value, joinSecret: secret }),
		signal: joinController.signal,
      })
      let response = await join(userId.value, previousSecret)
      let replacementId = ''
      if (response.status === 409 && !previousSecret) {
        replacementId = newUUID()
        response = await join(replacementId, '')
      }
      if (!response.ok) throw new Error('Participant registration failed')
		const participant = await response.json()
		if (participant.joinSecret) setParticipantSecret(eventId, participant.joinSecret)
      if (replacementId && !stopped) {
        setUserId(replacementId)
        userId.value = replacementId
        registrationNotice.value = {
          en: 'A new player identity was created. Earlier points remain on your previous record. Ask the host for help if needed.',
          ru: 'Создан новый профиль игрока. Прежние баллы остались в старой записи. При необходимости обратитесь к ведущему.',
          ro: 'A fost creată o identitate nouă. Punctele vechi rămân în înregistrarea anterioară. Cere ajutorul gazdei dacă este nevoie.',
        }[locale.value] || 'A new player identity was created.'
      }
    } catch (error) { console.error('Participant registration failed:', error) }
    if (stopped) return
    fetchEvent()
  }
})

onUnmounted(() => {
  stopped = true
  fetchGeneration++
	if (pollingTimer) clearTimeout(pollingTimer)
	if (completionTimer) clearTimeout(completionTimer)
	if (activeRequest) activeRequest.abort()
	if (joinController) joinController.abort()
})

const clearCompletionTimer = () => {
  if (completionTimer) {
    clearTimeout(completionTimer)
    completionTimer = null
  }
}

const closeSelectedModule = () => {
  clearCompletionTimer()
  selectedModule.value = null
}

const selectModule = (module) => {
  clearCompletionTimer()
  if (module.type === 'teams') {
    router.push(`/teams/${currentEvent.value.id}/${module.id}/player`)
    return
  }
  if (module.type === 'quiz') {
    router.push(`/quiz/${currentEvent.value.id}/${module.id}/player`)
    return
  }
  if (module.type === 'wrong_pictures') {
    router.push(`/wrong-pictures/${currentEvent.value.id}/${module.id}/player`)
    return
  }
  if (module.type === 'simple_quiz') {
    router.push(`/simple-quiz/${currentEvent.value.id}/${module.id}/player`)
    return
  }
  selectedModule.value = module
}

const handleModuleComplete = async (data) => {
  if (selectedModule.value?.name === 'Welcome' && data) {
    const name = data['step_welcome_name']
    if (name) {
      try {
        const response = await fetch(`${API_BASE_URL}/events/${currentEvent.value.id}/participants/${userId.value}/name`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ displayName: name, joinSecret: getParticipantSecret(currentEvent.value.id) })
        })
        if (!response.ok) throw new Error((await response.text()).trim() || `HTTP ${response.status}`)
        setUserName(name)
        userName.value = name
      } catch (error) {
        window.alert(`Could not update your name: ${error.message}`)
        return
      }
    }
  }
	if (selectedModule.value) {
    try { sessionStorage.setItem(`completed_module_${selectedModule.value.id}`, 'true') } catch {  }
  }
	const isWelcomeOnboarding = Boolean(selectedModule.value?.data?.mandatoryOnEntry && selectedModule.value?.name === 'Welcome')
	clearCompletionTimer()
  completionTimer = setTimeout(() => {
    completionTimer = null
	if (stopped) return
    selectedModule.value = null
    fetchEvent()
  }, isWelcomeOnboarding ? 600 : 2000)
}

const goToMain = () => {
  router.push('/')
}
</script>

<style>
:root {
  --bg-primary: var(--ops-bg);
  --bg-secondary: var(--ops-surface);
  --surface-layer: var(--ops-surface-raised);
  --accent: var(--ops-blue);
  --text-main: var(--ops-text);
  --text-dim: var(--ops-text-muted);
  --border-solid: var(--ops-border);
  --gold: var(--ops-amber);
  --radius-main: var(--ops-radius-lg);
  --shadow-main: 0 8px 24px rgba(0, 0, 0, 0.2);
}

.theme-material {
  --bg-primary: var(--ops-bg);
  --bg-secondary: var(--ops-surface);
  --surface-layer: var(--ops-surface-raised);
  --accent: var(--ops-blue);
  --text-main: var(--ops-text);
  --text-dim: var(--ops-text-muted);
  --border-solid: var(--ops-border);
  --gold: var(--ops-amber);
  --radius-main: var(--ops-radius-md);
  --shadow-main: 0 2px 8px rgba(0, 0, 0, 0.4);
}
</style>

<style scoped>

* {
  font-family: var(--ops-font);
  box-sizing: border-box;
  -webkit-tap-highlight-color: transparent;
}

.event-screen {
  height: 100vh;
  width: 100vw;
  background: var(--bg-primary);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.responsive-wrapper {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--bg-primary);
  position: relative;
}

.top-controls {
  position: relative;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 20px 8px;
  flex-shrink: 0;
  z-index: 100;
}

.theme-toggle {
  background: var(--surface-layer);
  border: 1px solid var(--border-solid);
  color: #fff;
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  font-size: 18px;
  transition: all 0.2s;
}

.theme-toggle:hover {
  border-color: var(--accent);
}

.language-switcher-container {
  position: relative;
  z-index: 100;
}

.lang-select {
  background: var(--surface-layer);
  color: var(--text-dim);
  border: 1px solid var(--border-solid);
  padding: 6px 12px;
  border-radius: 10px;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
  outline: none;
  transition: all 0.2s;
}

.lang-select:hover {
  border-color: var(--accent);
  color: #fff;
}

@media (min-width: 500px) {
  .responsive-wrapper {
    max-width: 480px;
    height: calc(100% - 40px);
    max-height: 850px;
    border-radius: 32px;
    box-shadow: 0 50px 100px rgba(0, 0, 0, 0.8), 0 0 0 1px var(--border-solid);
    overflow: hidden;
  }
}

@media (min-width: 768px) {
  .responsive-wrapper {
    max-width: 680px;
    max-height: 920px;
  }
}

.content {
  width: 100%;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 12px 24px 24px;
  overflow: hidden;
}

.user-profile {
  background: var(--bg-secondary);
  border: 1px solid var(--border-solid);
  border-radius: var(--radius-main);
  padding: 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  box-shadow: var(--shadow-main);
}

.user-main {
  display: flex;
  align-items: center;
  gap: 12px;
}

.avatar {
  width: 44px;
  height: 44px;
  background: var(--accent);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: 800;
  color: #fff;
}

.user-name {
  color: #fff;
  font-size: 18px;
  font-weight: 700;
  margin: 0;
  line-height: 1.2;
}

.team-badge {
  display: inline-block;
  font-size: 11px;
  font-weight: 800;
  padding: 2px 8px;
  border-radius: 6px;
  text-transform: uppercase;
  margin-top: 4px;
  color: #fff;
}

.points-badge {
  background: rgba(251, 191, 36, 0.1);
  border: 1px solid var(--gold);
  padding: 6px 14px;
  border-radius: 14px;
  text-align: center;
}

.points-value {
  display: block;
  font-size: 20px;
  font-weight: 900;
  color: var(--gold);
  line-height: 1;
}

.points-label {
  font-size: 9px;
  font-weight: 800;
  color: var(--gold);
  letter-spacing: 1px;
}

.event-header {
  margin-bottom: 32px;
}

.connected-label {
  color: var(--text-dim);
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 1.5px;
  margin-bottom: 4px;
}

.event-title {
  color: #fff;
  font-size: 34px;
  font-weight: 900;
  margin: 0;
  letter-spacing: -1px;
  line-height: 1;
}

.registration-notice {
  margin: -12px 0 20px;
  padding: 12px 14px;
  border-radius: 10px;
  background: rgba(255, 196, 63, 0.14);
  color: var(--text);
  font-size: 13px;
  line-height: 1.4;
}

.list-container {
  flex: 1;
  overflow-y: auto;
  padding-bottom: 100px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.list-container::-webkit-scrollbar { width: 0; }

.section-title {
  color: var(--text-dim);
  font-size: 13px;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 2px;
  margin: 0;
}

.dynamic-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.dynamic-card {
  background: var(--surface-layer);
  border-radius: 12px;
  padding: 16px;
  display: flex;
  gap: 16px;
  border-left: 5px solid var(--accent);
  box-shadow: var(--shadow-main);
}

.card-icon {
  font-size: 24px;
  flex-shrink: 0;
}

.card-title {
  color: #fff;
  font-size: 16px;
  font-weight: 700;
  margin: 0 0 4px 0;
}

.card-text {
  color: var(--text-dim);
  font-size: 14px;
  margin: 0;
  line-height: 1.4;
}

.module-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-solid);
  border-radius: var(--radius-main);
  padding: 18px;
  display: flex;
  align-items: center;
  gap: 16px;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: var(--shadow-main);
}

.module-card:hover {
  border-color: var(--accent);
  transform: scale(1.02);
  background: var(--surface-layer);
}

.module-icon-box {
  width: 52px;
  height: 52px;
  background: var(--surface-layer);
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  border: 1px solid var(--border-solid);
}

.module-main {
  flex: 1;
}

.module-name {
  color: #fff;
  font-size: 18px;
  font-weight: 700;
  margin: 0 0 2px 0;
}

.module-call {
  color: var(--accent);
  font-size: 13px;
  font-weight: 700;
}

.module-arrow {
  color: var(--border-solid);
  font-size: 20px;
  font-weight: 600;
}

.module-card:hover .module-arrow {
  color: var(--accent);
  transform: translateX(4px);
}

.waiting-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 40px 0;
}

.spinner {
  width: 48px;
  height: 48px;
  border: 4px solid var(--border-solid);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 1s infinite linear;
  margin-bottom: 24px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.wait-title {
  color: #fff;
  font-size: 18px;
  font-weight: 700;
  margin: 0 0 8px 0;
}

.wait-sub {
  color: var(--text-dim);
  font-size: 14px;
  text-align: center;
}

.exit-button {
  position: absolute;
  bottom: 24px;
  left: 24px;
  right: 24px;
  background: var(--bg-primary);
  border: 1px solid #ef4444;
  color: #ef4444;
  padding: 16px;
  border-radius: 16px;
  font-size: 16px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s;
}

.exit-button:hover {
  background: rgba(239, 68, 68, 0.1);
}

.module-view {
  width: 100%;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.module-header {
  padding: 10px 20px 14px;
  border-bottom: 1px solid var(--border-solid);
  flex-shrink: 0;
}

.back-link {
  background: none;
  border: none;
  color: var(--text-dim);
  font-size: 15px;
  font-weight: 700;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
}

.back-link:hover {
  color: #fff;
}
</style>
