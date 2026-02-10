<template>
  <div class="event-screen">
    <div v-if="!currentEvent" class="error-container">
      <p class="error-text">No event data found.</p>
      <button class="back-button" @click="goToMain">Go Back</button>
    </div>

    <div v-else-if="selectedModule" class="module-view">
      <div class="header">
        <button class="back-link" @click="selectedModule = null">
          <span class="back-arrow">←</span>
          <span>Back to Event</span>
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
        <p class="status-text">Connected to</p>
        <p class="event-name">{{ currentEvent.name }}</p>
      </div>

      <div class="list-container">
        <template v-if="activeModules.length > 0">
          <p class="section-title">Active Activities</p>
          <div
            v-for="module in activeModules"
            :key="module.id"
            class="module-card"
            @click="selectModule(module)"
          >
            <div class="icon-container">
              <span class="module-icon">{{ module.type === 'form' ? '📝' : '📊' }}</span>
            </div>
            <div class="module-info">
              <p class="module-title">{{ module.name }}</p>
              <p class="module-subtitle">Tap to start</p>
            </div>
            <span class="chevron">›</span>
          </div>
        </template>
        <template v-else>
          <div class="waiting-card">
            <div class="loader"></div>
            <p class="message">Waiting for host to start the next activity...</p>
            <p class="sub-message">Sit tight and get ready!</p>
          </div>
        </template>
      </div>

      <button class="leave-button" @click="goToMain">
        Leave Event
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import FormModule from '../components/FormModule.vue'
import { API_BASE_URL, getUserId } from '../utils'

const router = useRouter()
const userId = ref(getUserId())

const storedEvent = sessionStorage.getItem('currentEvent')
const currentEvent = ref(storedEvent ? JSON.parse(storedEvent) : null)
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
        activeModules.value = data.modules.filter(m => m.status === 'active')
      }
    }
  } catch (error) {
    console.error('Polling error:', error)
  }
}

onMounted(() => {
  if (currentEvent.value?.id) {
    pollingInterval = setInterval(fetchEvent, 3000)
  }
})

onUnmounted(() => {
  if (pollingInterval) {
    clearInterval(pollingInterval)
  }
})

const selectModule = (module) => {
  selectedModule.value = module
}

const handleModuleComplete = () => {
  setTimeout(() => {
    selectedModule.value = null
  }, 2000)
}

const goToMain = () => {
  router.push('/')
}
</script>

<style scoped>
.event-screen {
  min-height: 100vh;
  background-color: #121212;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.content {
  width: 90%;
  max-width: 400px;
  display: flex;
  flex-direction: column;
  align-items: center;
  flex: 1;
  padding-top: 60px;
}

.header {
  margin-bottom: 40px;
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
}

.status-text {
  color: #888;
  font-size: 14px;
  text-transform: uppercase;
  letter-spacing: 1px;
  margin-bottom: 8px;
}

.event-name {
  color: #FFF;
  font-size: 32px;
  font-weight: 800;
  text-align: center;
}

.list-container {
  width: 100%;
  flex: 1;
}

.section-title {
  color: #888;
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 16px;
  text-transform: uppercase;
  letter-spacing: 1px;
}

.module-card {
  background-color: #1E1E1E;
  border-radius: 16px;
  padding: 16px;
  margin-bottom: 16px;
  display: flex;
  flex-direction: row;
  align-items: center;
  border: 1px solid #333;
  cursor: pointer;
}

.icon-container {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background-color: #333;
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
  color: #FFF;
  font-size: 18px;
  font-weight: 700;
  margin-bottom: 4px;
}

.module-subtitle {
  color: #30D158;
  font-size: 14px;
  font-weight: 500;
}

.chevron {
  color: #666;
  font-size: 24px;
}

.waiting-card {
  background-color: #1E1E1E;
  border-radius: 20px;
  padding: 40px;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 40px;
  border: 1px solid #333;
}

.loader {
  width: 40px;
  height: 40px;
  border: 3px solid #333;
  border-top-color: #007AFF;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 20px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.message {
  color: #FFF;
  font-size: 18px;
  font-weight: 600;
  text-align: center;
  margin-bottom: 8px;
}

.sub-message {
  color: #888;
  font-size: 14px;
  text-align: center;
}

.leave-button {
  padding: 16px 32px;
  border-radius: 12px;
  border: 1px solid #333;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 40px;
  color: #FF453A;
  font-size: 16px;
  font-weight: 600;
  background: none;
}

.error-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
}

.error-text {
  color: #FF453A;
  font-size: 18px;
  margin-bottom: 20px;
}

.back-button {
  padding: 10px;
  color: #FFF;
  font-size: 16px;
  background: none;
}

.module-view {
  width: 100%;
  flex: 1;
}

.header {
  display: flex;
  flex-direction: row;
  align-items: center;
  padding: 16px;
  margin-bottom: 10px;
}

.back-link {
  display: flex;
  flex-direction: row;
  align-items: center;
  color: #FFF;
  font-size: 16px;
  background: none;
  padding: 8px;
}

.back-arrow {
  margin-right: 8px;
}
</style>
