<template>
  <div class="event-screen" :class="'theme-' + theme">
    <div class="responsive-wrapper">
      <!-- Theme & Language -->
      <div class="top-controls">
        <button @click="toggleTheme" class="theme-toggle" :title="$t('event.toggle_theme')">
          {{ theme === 'bold' ? '🎨' : '🏗️' }}
        </button>
        <div class="language-switcher-container">
          <select v-model="$i18n.locale" @change="changeLanguage" class="lang-select">
            <option value="en">EN</option>
            <option value="ru">RU</option>
            <option value="ro">RO</option>
          </select>
        </div>
      </div>

      <!-- Error State -->
      <div v-if="!currentEvent" class="error-container">
        <p class="error-text">{{ $t('event.no_data') }}</p>
        <button class="back-button" @click="goToMain">{{ $t('event.go_back') }}</button>
      </div>

      <!-- Module View -->
      <div v-else-if="selectedModule" class="module-view">
        <div class="module-header">
          <button class="back-link" @click="selectedModule = null">
            <span class="back-arrow">←</span>
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
            <span class="points-label">PTS</span>
          </div>
        </div>

        <!-- Event Name -->
        <div class="event-header">
          <p class="connected-label">{{ $t('event.connected_to') }}</p>
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
              <div class="card-icon">{{ card.icon }}</div>
              <div class="card-body">
                <h3 class="card-title">{{ card.title }}</h3>
                <p class="card-text">{{ card.content }}</p>
              </div>
            </div>
          </div>

          <!-- Active Modules -->
          <template v-if="activeModules.length > 0">
            <p class="section-title">{{ $t('event.active_activities') }}</p>
            <div
              v-for="module in activeModules"
              :key="module.id"
              class="module-card"
              @click="selectModule(module)"
            >
              <div class="module-icon-box">
                <span class="module-emoji">
                  {{ module.type === 'form' ? '📝' : 
                     (module.type === 'quiz' ? '🎮' : 
                     (module.type === 'wrong_pictures' ? '🖼️' : '📊')) }}
                </span>
              </div>
              <div class="module-main">
                <p class="module-name">{{ module.name }}</p>
                <p class="module-call">{{ $t('event.tap_to_start') }}</p>
              </div>
              <div class="module-arrow">→</div>
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
const theme = ref(localStorage.getItem('userTheme') || 'bold')

const toggleTheme = () => {
  theme.value = theme.value === 'bold' ? 'material' : 'bold'
  localStorage.setItem('userTheme', theme.value)
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

<style>
:root {
  --bg-primary: #20222c; 
  --bg-secondary: #2a2d3d;
  --surface-layer: #373b4d;
  --accent: #8b8eff;
  --text-main: #f8f8fb;
  --text-dim: #b0b3c5;
  --border-solid: #4d526b;
  --gold: #fbbf24;
  --radius-main: 20px;
  --shadow-main: 0 8px 24px rgba(0, 0, 0, 0.2);
}

.theme-material {
  --bg-primary: #121212;
  --bg-secondary: #1e1e1e;
  --surface-layer: #2c2c2c;
  --accent: #bb86fc;
  --text-main: #e1e1e1;
  --text-dim: #9e9e9e;
  --border-solid: #333333;
  --gold: #03dac6; /* Using teal as secondary accent in material */
  --radius-main: 8px; /* Material typically uses smaller radius for cards */
  --shadow-main: 0 2px 8px rgba(0, 0, 0, 0.4);
}
</style>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Outfit:wght@400;500;600;700;800;900&display=swap');

/* foundation variables moved to global style block above */

* { 
  font-family: 'Outfit', sans-serif; 
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
  position: absolute;
  top: 20px;
  left: 20px;
  right: 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
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

/* ── Event Header ── */
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
  color: var(--text-dim);
  font-size: 13px;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 2px;
  margin: 0;
}

/* ── Dynamic Cards ── */
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

/* ── Module Cards ── */
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

/* ── Waiting State ── */
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

/* ── Buttons ── */
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

/* ── Transitions ── */
.module-view {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.module-header {
  padding: 16px;
  border-bottom: 1px solid var(--border-solid);
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
