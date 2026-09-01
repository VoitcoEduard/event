<template>
  <div class="event-screen brutalist">
    <div class="responsive-wrapper">
      <!-- Language -->
      <div class="top-controls">
        <div class="language-switcher-container">
          <select v-model="$i18n.locale" @change="changeLanguage" class="lang-select b-select">
            <option value="en">EN</option>
            <option value="ru">RU</option>
            <option value="ro">RO</option>
          </select>
        </div>
      </div>

      <!-- Error State -->
      <div v-if="!currentEvent" class="error-container">
        <p class="error-text">{{ $t('event.no_data') }}</p>
        <button class="back-button b-btn" @click="goToMain">{{ $t('event.go_back') }}</button>
      </div>

      <!-- Module View -->
      <div v-else-if="selectedModule" class="module-view">
        <div class="module-header">
          <button class="back-link" @click="selectedModule = null">
            <ArrowLeft :size="16" :stroke-width="2.5" />
            <span>{{ $t('event.back_to_event') }}</span>
          </button>
        </div>
        <FormModule
          v-if="selectedModule.type === 'form'"
          :module="selectedModule"
          :event-id="currentEvent.id"
          :user-id="userId"
          @complete="handleModuleComplete"
        />
      </div>

      <!-- Home Screen Content -->
      <div v-else class="content">
        <!-- User Profile Header -->
        <div class="user-profile b-panel">
          <div class="user-main">
            <div class="avatar b-mono">
              {{ userName ? userName.charAt(0).toUpperCase() : '?' }}
            </div>
            <div class="user-details">
              <h2 class="user-name">{{ userName || $t('event.guest') }}</h2>
            </div>
          </div>
          <div v-if="participantStatus" class="points-badge b-badge b-badge-lime">
            <span class="points-value">{{ participantStatus.totalPoints || 0 }}</span>
            <span class="points-label">PTS</span>
          </div>
        </div>

        <!-- Event Name -->
        <div class="event-header">
          <p class="connected-label b-label">{{ $t('event.connected_to') }}</p>
          <h1 class="event-title">{{ currentEvent.name }}</h1>
        </div>

        <div class="list-container">
          <!-- Dynamic Cards (Polling Content) -->
          <div v-if="dynamicCards.length > 0" class="dynamic-section">
            <div
              v-for="card in dynamicCards"
              :key="card.id"
              class="dynamic-card"
              :class="card.type"
              :style="{ borderLeftColor: card.color }"
            >
              <div class="card-icon">
                <component :is="moduleIcon('poll')" :size="20" :stroke-width="2.5" />
              </div>
              <div class="card-body">
                <h3 class="card-title">{{ card.title }}</h3>
                <p class="card-text">{{ card.content }}</p>
              </div>
            </div>
          </div>

          <!-- Active Modules -->
          <template v-if="activeModules.length > 0">
            <p class="section-title b-label">{{ $t('event.active_activities') }}</p>
            <div
              v-for="module in activeModules"
              :key="module.id"
              class="module-card b-card"
              @click="selectModule(module)"
            >
              <div class="module-icon-box b-icon-box" :class="'accent-' + (module.type === 'form' ? 'cyan' : module.type === 'quiz' ? 'fuchsia' : module.type === 'wrong_pictures' ? 'amber' : 'lime')">
                <component :is="moduleIcon(module.type)" :size="24" :stroke-width="2.5" />
              </div>
              <div class="module-main">
                <p class="module-name">{{ module.name }}</p>
                <p class="module-call b-mono">{{ $t('event.tap_to_start') }}</p>
              </div>
              <div class="module-arrow">
                <ChevronRight :size="20" :stroke-width="2.5" />
              </div>
            </div>
          </template>

          <template v-else>
            <div class="waiting-state">
              <div class="loader-blocks b-loader"><span></span><span></span><span></span><span></span><span></span></div>
              <p class="wait-title">{{ $t('event.waiting_msg') }}</p>
              <p class="wait-sub">{{ $t('event.waiting_submsg') }}</p>
            </div>
          </template>
        </div>

        <button class="exit-button b-btn" @click="goToMain">
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
import { FileText, Gamepad2, Image, BarChart3, ChevronRight, ArrowLeft } from 'lucide-vue-next'
import { API_BASE_URL, getUserId, getUserName, setUserName } from '../utils'

const { locale } = useI18n()
const router = useRouter()
const route = useRoute()
const userId = ref(getUserId())
const userName = ref(getUserName())

const changeLanguage = () => {
  localStorage.setItem('userLanguage', locale.value)
}

// Check for eventId in query params first (for teams/quiz direct routes)
const params = new URLSearchParams(window.location.search)
const queryEventId = params.get('id') || params.get('eventId')
const storedEvent = queryEventId ? null : sessionStorage.getItem('currentEvent')
const currentEvent = ref(storedEvent ? JSON.parse(storedEvent) : (queryEventId ? { id: queryEventId } : null))

const moduleIcon = (type) => {
  if (type === 'form') return FileText
  if (type === 'quiz') return Gamepad2
  if (type === 'wrong_pictures') return Image
  return BarChart3
}

const activeModules = ref([])
const dynamicCards = ref([])
const participantStatus = ref(null)
const selectedModule = ref(null)
let pollingInterval = null

const fetchEvent = async () => {
  if (!currentEvent.value?.id) return

  try {
    const response = await fetch(`${API_BASE_URL}/events/${currentEvent.value.id}?userId=${userId.value}`)
    if (response.ok) {
      const data = await response.json()
      currentEvent.value = data

      // Update status and cards
      participantStatus.value = data.participantStatus || null
      dynamicCards.value = data.dynamicCards || []

      // Update name if changed on backend
      if (participantStatus.value?.displayName && participantStatus.value.displayName !== userName.value) {
        setUserName(participantStatus.value.displayName)
        userName.value = participantStatus.value.displayName
      }

      if (data.modules) {
        const active = data.modules.filter(m => m.isActive)

        // Filter out completed modules that should be hidden
        activeModules.value = active.filter(m => {
          if (m.data?.hideAfterCompletion) {
            return localStorage.getItem(`completed_module_${m.id}`) !== 'true'
          }
          return true
        })

        // Check if we need to auto-select a module from route params (e.g., /teams/:eventId/:moduleId/player)
        const routeModuleId = route.params.moduleId
        if (routeModuleId && !selectedModule.value) {
          const moduleToSelect = data.modules.find(m => m.id === routeModuleId)
          if (moduleToSelect) {
            selectedModule.value = moduleToSelect
          }
        }

        // Auto-redirect to mandatory uncompleted module if not already in a module and no route module
        if (!selectedModule.value && !routeModuleId) {
            const mandatoryModule = active.find(m => m.data?.mandatoryOnEntry && localStorage.getItem(`completed_module_${m.id}`) !== 'true')
            if (mandatoryModule) {
               selectModule(mandatoryModule)
            }
        }
      }
    }
  } catch (error) {
    console.error('Polling error:', error)
  }
}

onMounted(() => {
  if (currentEvent.value?.id) {
    fetchEvent() // Initial fetch
    pollingInterval = setInterval(fetchEvent, 3000)
  }
})

onUnmounted(() => {
  if (pollingInterval) {
    clearInterval(pollingInterval)
  }
})

const selectModule = (module) => {
  if (module.type === 'quiz') {
    router.push(`/quiz/${currentEvent.value.id}/${module.id}/player`)
    return
  }
  if (module.type === 'wrong_pictures') {
    router.push(`/wrong-pictures/${currentEvent.value.id}/${module.id}/player`)
    return
  }
  selectedModule.value = module
}

const handleModuleComplete = (data) => {
  if (selectedModule.value) {
    localStorage.setItem(`completed_module_${selectedModule.value.id}`, 'true')
  }

  if (selectedModule.value?.name === 'Welcome' && data) {
    const name = data['step_welcome_name']
    if (name) {
      setUserName(name)
      userName.value = name
      // Sync name to server
      fetch(`${API_BASE_URL}/events/${currentEvent.value.id}/participants/${userId.value}/name`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ displayName: name })
      }).catch(e => console.error('Failed to sync name:', e))
    }
  }
  setTimeout(() => {
    selectedModule.value = null
    fetchEvent() // Refresh immediately
  }, 2000)
}

const goToMain = () => {
  router.push('/')
}
</script>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&family=JetBrains+Mono:wght@400;700&display=swap');

.event-screen {
  height: 100vh;
  height: 100dvh;
  width: 100vw;
  background: var(--b-bg);
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
  background: var(--b-bg);
  position: relative;
}

.top-controls {
  position: absolute;
  top: 20px;
  right: 20px;
  display: flex;
  align-items: center;
  z-index: 100;
}

.lang-select {
  position: relative;
  z-index: 100;
}

@media (min-width: 500px) {
  .responsive-wrapper {
    max-width: 480px;
    height: 100%;
    max-height: 850px;
    border: var(--b-border-w) solid var(--b-border);
  }
}

.content {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 24px;
  padding-top: 64px;
  overflow: hidden;
}

/* ── User Profile ── */
.user-profile {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.user-main {
  display: flex;
  align-items: center;
  gap: 12px;
}

.avatar {
  width: 44px;
  height: 44px;
  background: var(--b-bg);
  border: var(--b-border-w) solid var(--b-accent-cyan);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: 700;
  color: var(--b-accent-cyan);
}

.user-name {
  color: var(--b-text);
  font-size: 18px;
  font-weight: 800;
  margin: 0;
  line-height: 1.2;
}

.points-badge {
  padding: 8px 14px;
  flex-direction: column;
  gap: 0;
}

.points-value {
  font-size: 16px;
  line-height: 1.2;
}

.points-label {
  font-size: 9px;
}

/* ── Event Header ── */
.event-header {
  margin-bottom: 32px;
}

.connected-label {
  margin-bottom: 6px;
}

.event-title {
  color: var(--b-text);
  font-size: 32px;
  font-weight: 900;
  margin: 0;
  letter-spacing: -1px;
  line-height: 1;
  text-transform: uppercase;
}

/* ── List Container ── */
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
  margin: 0;
}

/* ── Dynamic Cards ── */
.dynamic-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.dynamic-card {
  background: var(--b-surface);
  border: var(--b-border-w) solid var(--b-border);
  border-left: 4px solid var(--b-accent-cyan);
  padding: 16px;
  display: flex;
  gap: 16px;
}

.card-icon {
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border: var(--b-border-w) solid var(--b-border);
  color: var(--b-text-dim);
}

.card-title {
  color: var(--b-text);
  font-size: 15px;
  font-weight: 800;
  margin: 0 0 4px 0;
  text-transform: uppercase;
}

.card-text {
  color: var(--b-text-dim);
  font-size: 13px;
  margin: 0;
  line-height: 1.4;
}

/* ── Module Cards ── */
.module-card {
  padding: 18px;
  display: flex;
  align-items: center;
  gap: 16px;
  cursor: pointer;
  transition: transform 0.1s ease, box-shadow 0.1s ease;
  box-shadow: var(--b-offset) var(--b-offset) 0 0 var(--b-border);
}

.module-card:hover {
  transform: translate(-2px, -2px);
  box-shadow: 6px 6px 0 0 var(--b-accent-cyan);
}

.module-card:active {
  transform: translate(2px, 2px);
  box-shadow: 1px 1px 0 0 var(--b-accent-cyan);
}

.module-icon-box.accent-cyan    { color: var(--b-accent-cyan);    border-color: var(--b-accent-cyan); }
.module-icon-box.accent-fuchsia { color: var(--b-accent-fuchsia); border-color: var(--b-accent-fuchsia); }
.module-icon-box.accent-amber   { color: var(--b-accent-amber);   border-color: var(--b-accent-amber); }
.module-icon-box.accent-lime    { color: var(--b-accent-lime);    border-color: var(--b-accent-lime); }

.module-main {
  flex: 1;
  min-width: 0;
}

.module-name {
  color: var(--b-text);
  font-size: 17px;
  font-weight: 800;
  margin: 0 0 4px 0;
}

.module-call {
  color: var(--b-text-dim);
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.15em;
  margin: 0;
}

.module-arrow {
  color: var(--b-text-faint);
  flex-shrink: 0;
}

.module-card:hover .module-arrow {
  color: var(--b-accent-cyan);
  transform: translateX(4px);
}

/* ── Waiting State ── */
.waiting-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 40px 0;
}

.loader-blocks {
  margin-bottom: 24px;
}

.wait-title {
  color: var(--b-text);
  font-size: 16px;
  font-weight: 800;
  margin: 0 0 8px 0;
  text-transform: uppercase;
}

.wait-sub {
  color: var(--b-text-dim);
  font-size: 13px;
  text-align: center;
}

/* ── Buttons ── */
.exit-button {
  position: absolute;
  bottom: 24px;
  left: 24px;
  right: 24px;
  background: var(--b-bg);
  border: var(--b-border-w) solid var(--b-accent-rose);
  color: var(--b-accent-rose);
  box-shadow: var(--b-offset) var(--b-offset) 0 0 var(--b-accent-rose);
}

.exit-button:hover:not(:disabled) {
  box-shadow: 6px 6px 0 0 var(--b-accent-rose);
}

.exit-button:active:not(:disabled) {
  box-shadow: 1px 1px 0 0 var(--b-accent-rose);
}

/* ── Transitions ── */
.module-view {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.module-header {
  padding: 16px;
  border-bottom: var(--b-border-w) solid var(--b-border);
}

.back-link {
  background: none;
  border: none;
  color: var(--b-text-dim);
  font-size: 13px;
  font-weight: 800;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  text-transform: uppercase;
  letter-spacing: 0.1em;
}

.back-link:hover {
  color: var(--b-text);
}
</style>
