<template>
  <div class="event-screen">
    <div class="responsive-wrapper">
      <div class="language-switcher-container">
        <select v-model="$i18n.locale" @change="changeLanguage" class="lang-select">
          <option value="en">English</option>
          <option value="ru">Русский</option>
          <option value="ro">Română</option>
        </select>
      </div>
      <div v-if="!currentEvent" class="error-container">
        <p class="error-text">{{ $t('event.no_data') }}</p>
        <button class="back-button" @click="goToMain">{{ $t('event.go_back') }}</button>
      </div>

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

      <div v-else class="content">
        <div class="header">
          <p class="status-text">{{ $t('event.connected_to') }}</p>
          <p class="event-name">{{ currentEvent.name }}</p>
          <p v-if="userName" class="welcome-text">{{ $t('event.welcome', { name: userName }) }}</p>
        </div>

        <div class="list-container">
          <template v-if="activeModules.length > 0">
            <p class="section-title">{{ $t('event.active_activities') }}</p>
            <div
              v-for="module in activeModules"
              :key="module.id"
              class="module-card"
              @click="selectModule(module)"
            >
              <div class="icon-container">
                <span class="module-icon">{{ module.type === 'form' ? '📝' : (module.type === 'quiz' ? '🎮' : '📊') }}</span>
              </div>
              <div class="module-info">
                <p class="module-title">{{ module.name }}</p>
                <p class="module-subtitle">{{ $t('event.tap_to_start') }}</p>
              </div>
              <span class="chevron">›</span>
            </div>
          </template>
          <template v-else>
            <div class="waiting-card">
              <div class="loader"></div>
              <p class="message">{{ $t('event.waiting_msg') }}</p>
              <p class="sub-message">{{ $t('event.waiting_submsg') }}</p>
            </div>
          </template>
        </div>

        <button class="leave-button" @click="goToMain">
          {{ $t('event.leave_btn') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import FormModule from '../components/FormModule.vue'
import { API_BASE_URL, getUserId, getUserName, setUserName } from '../utils'

const { locale } = useI18n()
const router = useRouter()
const userId = ref(getUserId())
const userName = ref(getUserName())

const changeLanguage = () => {
  localStorage.setItem('userLanguage', locale.value)
}

const storedEvent = sessionStorage.getItem('currentEvent')
const currentEvent = ref(storedEvent ? JSON.parse(storedEvent) : null)
// Check logic: if no currentEvent but we have ID in query, set ID to fetch
if (!currentEvent.value) {
    const params = new URLSearchParams(window.location.search)
    const idParam = params.get('id')
    if (idParam) {
        currentEvent.value = { id: idParam }
    }
}

const activeModules = ref([])
const selectedModule = ref(null)
let pollingInterval = null

const fetchEvent = async () => {
  if (!currentEvent.value?.id) return
  
  try {
    const response = await fetch(`${API_BASE_URL}/events/${currentEvent.value.id}`)
    if (response.ok) {
      const data = await response.json()
      currentEvent.value = data
      if (data.modules) {
        const active = data.modules.filter(m => m.isActive)
        
        // Filter out completed modules that should be hidden
        activeModules.value = active.filter(m => {
          if (m.data?.hideAfterCompletion) {
            return localStorage.getItem(`completed_module_${m.id}`) !== 'true'
          }
          return true
        })

        // Auto-redirect to mandatory uncompleted module if not already in a module
        if (!selectedModule.value) {
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
.event-screen {
  height: 100vh; /* Fixed height to enable centering */
  width: 100vw;
  background-color: var(--background-color, #121212);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center; /* Center vertically on desktop */
  overflow: hidden; /* Prevent body scroll */
}

.responsive-wrapper {
  width: 100%;
  height: 100%; /* Full height by default (mobile) */
  display: flex;
  flex-direction: column;
  background-color: var(--background-color, #121212);
  position: relative;
}

.language-switcher-container {
  position: absolute;
  top: 10px;
  right: 10px;
  z-index: 100;
}

.lang-select {
  background: rgba(255, 255, 255, 0.06);
  color: white;
  border: 1px solid rgba(255, 255, 255, 0.1);
  padding: 4px 8px;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  outline: none;
}

@media (min-width: 500px) {
  .responsive-wrapper {
    max-width: 480px; /* Constrains width */
    height: calc(100% - 40px); /* Margin top/bottom essentially */
    max-height: 850px; /* Reasonable max height for a "phone" view */
    border-radius: 24px;
    box-shadow: 0 40px 80px rgba(0,0,0,0.5); /* Strong shadow for depth */
    border: 1px solid var(--border-color, #333);
    overflow: hidden; /* Clip content to corners */
  }
}

.content {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  flex: 1;
  padding: 20px;
  padding-top: 40px;
  overflow: hidden; /* Prevent container expansion */
}

.header {
  margin-bottom: 24px;
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  flex-shrink: 0; /* Header stays at top */
}

.status-text {
  color: var(--text-secondary, #888);
  font-size: 14px;
  text-transform: uppercase;
  letter-spacing: 1px;
  margin-bottom: 8px;
}

.event-name {
  color: var(--text-primary, #FFF);
  font-size: 32px;
  font-weight: 800;
  text-align: center;
  margin-bottom: 4px;
}

.welcome-text {
  color: var(--primary-color, #30D158);
  font-size: 18px;
  font-weight: 600;
  margin-top: 4px;
}

.list-container {
  width: 100%;
  flex: 1;
  overflow-y: auto; /* Internal scrolling */
  -webkit-overflow-scrolling: touch;
  padding-bottom: 80px; /* Space for bottom button */
  padding-right: 4px; /* Prevent scrollbar visual clash */
}

/* Scrollbar styling for webkit */
.list-container::-webkit-scrollbar,
.scroll-content::-webkit-scrollbar {
  width: 6px;
}

.list-container::-webkit-scrollbar-track,
.scroll-content::-webkit-scrollbar-track {
  background: transparent;
}

.list-container::-webkit-scrollbar-thumb,
.scroll-content::-webkit-scrollbar-thumb {
  background-color: rgba(255, 255, 255, 0.1);
  border-radius: 3px;
}

.section-title {
  color: var(--text-secondary, #888);
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 16px;
  text-transform: uppercase;
  letter-spacing: 1px;
}

.module-card {
  background-color: var(--surface-color, #1E1E1E);
  border-radius: 16px;
  padding: 16px;
  margin-bottom: 16px;
  display: flex;
  flex-direction: row;
  align-items: center;
  border: 1px solid var(--border-color, #333);
  cursor: pointer;
  transition: transform 0.2s, background-color 0.2s;
}

.module-card:hover {
  transform: translateY(-2px);
  background-color: var(--surface-hover, #252525);
}

.icon-container {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background-color: var(--background-color, #333);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 16px;
}

.module-icon {
  font-size: 24px;
}

.module-info {
  flex: 1;
}

.module-title {
  color: var(--text-primary, #FFF);
  font-size: 18px;
  font-weight: 700;
  margin-bottom: 4px;
}

.module-subtitle {
  color: var(--primary-color, #30D158);
  font-size: 14px;
  font-weight: 500;
}

.chevron {
  color: var(--text-secondary, #666);
  font-size: 24px;
}

.waiting-card {
  background-color: var(--surface-color, #1E1E1E);
  border-radius: 20px;
  padding: 40px;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 40px;
  border: 1px solid var(--border-color, #333);
}

.loader {
  width: 40px;
  height: 40px;
  border: 3px solid #333;
  border-top-color: var(--primary-color, #007AFF);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 20px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.message {
  color: var(--text-primary, #FFF);
  font-size: 18px;
  font-weight: 600;
  text-align: center;
  margin-bottom: 8px;
}

.sub-message {
  color: var(--text-secondary, #888);
  font-size: 14px;
  text-align: center;
}

.leave-button {
  width: calc(100% - 40px); /* Match padding of container */
  position: absolute;
  bottom: 20px;
  left: 20px;
  padding: 16px 32px;
  border-radius: 12px;
  border: 1px solid var(--border-color, #333);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--danger-color, #FF453A);
  font-size: 16px;
  font-weight: 600;
  background-color: var(--background-color, #121212); /* Opaque bg mostly */
  cursor: pointer;
  z-index: 10;
  box-shadow: 0 -10px 20px rgba(0,0,0,0.2); /* Shadow to separate from list */
}

.leave-button:hover {
  background-color: rgba(255, 69, 58, 0.1);
}

.error-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
}

.error-text {
  color: var(--danger-color, #FF453A);
  font-size: 18px;
  margin-bottom: 20px;
}

.back-button {
  padding: 10px;
  color: var(--text-primary, #FFF);
  font-size: 16px;
  background: none;
  border: none;
  cursor: pointer;
}

.module-view {
  width: 100%;
  height: 100%;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden; /* Ensure module scrolling is contained */
}

.module-header {
  display: flex;
  flex-direction: row;
  align-items: center;
  padding: 16px;
  width: 100%;
  flex-shrink: 0;
  z-index: 20;
}

.back-link {
  display: flex;
  flex-direction: row;
  align-items: center;
  color: var(--text-primary, #FFF);
  font-size: 16px;
  background: none;
  border: none;
  padding: 8px;
  cursor: pointer;
}

.back-link:hover {
  opacity: 0.8;
}

.back-arrow {
  margin-right: 8px;
}
</style>
