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
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&display=swap');

* { font-family: 'Inter', sans-serif; box-sizing: border-box; }

.event-screen {
  height: 100vh;
  width: 100vw;
  background: #0d0d0f;
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
  background: #0d0d0f;
  position: relative;
}

.language-switcher-container {
  position: absolute;
  top: 14px;
  right: 14px;
  z-index: 100;
}

.lang-select {
  background: #1a1a1f;
  color: #aaa;
  border: 1px solid #2a2a35;
  padding: 5px 10px;
  border-radius: 8px;
  font-size: 12px;
  cursor: pointer;
  outline: none;
  transition: border-color 0.2s, color 0.2s;
}

.lang-select:hover {
  border-color: #444;
  color: #fff;
}

@media (min-width: 500px) {
  .responsive-wrapper {
    max-width: 480px;
    height: calc(100% - 40px);
    max-height: 850px;
    border-radius: 28px;
    box-shadow: 0 40px 100px rgba(0, 0, 0, 0.7), 0 0 0 1px #202028;
    overflow: hidden;
  }
}

.content {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  flex: 1;
  padding: 24px 20px 20px;
  padding-top: 52px;
  overflow: hidden;
}

/* ── Header ── */
.header {
  margin-bottom: 28px;
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  flex-shrink: 0;
}

.status-text {
  color: #555;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 2px;
  font-weight: 600;
  margin-bottom: 6px;
}

.event-name {
  color: #fff;
  font-size: 30px;
  font-weight: 900;
  text-align: center;
  letter-spacing: -0.5px;
  margin-bottom: 6px;
  line-height: 1.1;
}

.welcome-text {
  font-size: 15px;
  font-weight: 600;
  margin-top: 2px;
  color: #6366f1;
}

/* ── List ── */
.list-container {
  width: 100%;
  flex: 1;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
  padding-bottom: 84px;
  padding-right: 2px;
}

.list-container::-webkit-scrollbar { width: 4px; }
.list-container::-webkit-scrollbar-track { background: transparent; }
.list-container::-webkit-scrollbar-thumb {
  background: #2a2a35;
  border-radius: 4px;
}

.section-title {
  color: #444;
  font-size: 11px;
  font-weight: 700;
  margin-bottom: 14px;
  text-transform: uppercase;
  letter-spacing: 2px;
}

/* ── Module card ── */
.module-card {
  background: #161619;
  border-radius: 18px;
  padding: 0;
  margin-bottom: 12px;
  display: flex;
  flex-direction: row;
  align-items: center;
  border: 1px solid #2a2a35;
  cursor: pointer;
  transition: transform 0.18s ease, box-shadow 0.18s ease, border-color 0.18s ease;
  overflow: hidden;
}

.module-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  border-color: #6366f1;
}

.module-card:active {
  transform: translateY(0);
}

/* Coloured left stripe */
.module-card::before {
  content: '';
  display: block;
  width: 4px;
  align-self: stretch;
  background: linear-gradient(180deg, #6366f1, #4f46e5);
  flex-shrink: 0;
}

.icon-container {
  width: 46px;
  height: 46px;
  border-radius: 12px;
  background: #202028;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 14px 14px 14px 14px;
  flex-shrink: 0;
}

.module-icon {
  font-size: 22px;
}

.module-info {
  flex: 1;
  padding: 14px 0;
}

.module-title {
  color: #fff;
  font-size: 16px;
  font-weight: 700;
  margin-bottom: 3px;
}

.module-subtitle {
  color: #6366f1;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.2px;
}

.chevron {
  color: #3a3a4a;
  font-size: 22px;
  margin-right: 16px;
  transition: color 0.2s, transform 0.2s;
}

.module-card:hover .chevron {
  color: #6366f1;
  transform: translateX(2px);
}

/* ── Waiting card ── */
.waiting-card {
  background: #161619;
  border-radius: 22px;
  padding: 44px 32px;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  border: 1px solid #2a2a35;
}

.loader {
  width: 44px;
  height: 44px;
  border: 3px solid #2a2a35;
  border-top-color: #6366f1;
  border-radius: 50%;
  animation: spin 0.9s linear infinite;
  margin-bottom: 22px;
  box-shadow: 0 0 16px rgba(99, 102, 241, 0.25);
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.message {
  color: #fff;
  font-size: 17px;
  font-weight: 700;
  text-align: center;
  margin-bottom: 8px;
}

.sub-message {
  color: #555;
  font-size: 13px;
  text-align: center;
  line-height: 1.5;
}

/* ── Leave button ── */
.leave-button {
  width: calc(100% - 40px);
  position: absolute;
  bottom: 20px;
  left: 20px;
  padding: 15px 32px;
  border-radius: 14px;
  border: 1px solid #2a2a35;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #f87171;
  font-size: 15px;
  font-weight: 600;
  background: #0d0d0f;
  cursor: pointer;
  z-index: 10;
  transition: background 0.2s, border-color 0.2s;
  box-shadow: 0 -16px 32px rgba(13, 13, 15, 0.9);
}

.leave-button:hover {
  background: rgba(248, 113, 113, 0.07);
  border-color: rgba(248, 113, 113, 0.3);
}

/* ── Error ── */
.error-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 32px;
}

.error-text {
  color: #f87171;
  font-size: 16px;
  margin-bottom: 20px;
  text-align: center;
}

.back-button {
  padding: 12px 24px;
  color: #fff;
  font-size: 15px;
  font-weight: 600;
  background: #161619;
  border: 1px solid #2a2a35;
  border-radius: 12px;
  cursor: pointer;
  transition: border-color 0.2s;
}

.back-button:hover {
  border-color: #6366f1;
}

/* ── Module view ── */
.module-view {
  width: 100%;
  height: 100%;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.module-header {
  display: flex;
  flex-direction: row;
  align-items: center;
  padding: 16px;
  width: 100%;
  flex-shrink: 0;
  z-index: 20;
  border-bottom: 1px solid #1e1e26;
}

.back-link {
  display: flex;
  flex-direction: row;
  align-items: center;
  color: #aaa;
  font-size: 15px;
  font-weight: 500;
  background: none;
  border: none;
  padding: 8px 10px;
  border-radius: 10px;
  cursor: pointer;
  transition: color 0.2s, background 0.2s;
}

.back-link:hover {
  color: #fff;
  background: #1e1e26;
}

.back-arrow {
  margin-right: 6px;
  font-size: 18px;
}
</style>
