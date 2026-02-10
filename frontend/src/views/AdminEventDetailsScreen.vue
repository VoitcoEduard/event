<template>
  <div class="admin-details-screen">
    <div class="header">
      <div class="header-top">
        <button class="back-link" @click="goBack">
          <span>← Back</span>
        </button>
        <p class="header-title">Event Dashboard</p>
      </div>
      <p class="header-subtitle">{{ currentEvent?.name }}</p>
    </div>

    <div class="scroll-content">
      <div class="card event-sidebar">
        <div class="card-header">
          <div>
            <p class="card-title">Event Overview</p>
            <p class="card-description">Manage details & sharing.</p>
          </div>
          <span class="status-badge" :class="currentEvent?.isActive ? 'active' : 'inactive'">
            {{ currentEvent?.isActive ? 'ACTIVE' : 'STOPPED' }}
          </span>
        </div>

        <div class="sidebar-content">
          <div class="info-group">
            <p class="label">Description</p>
            <p class="value">{{ currentEvent?.description }}</p>
          </div>

          <div class="info-group">
            <p class="label">PIN Code</p>
            <div class="pin-display">
              <span class="pin-text">{{ currentEvent?.id }}</span>
            </div>
          </div>

          <div class="info-group">
            <p class="label">Share Access</p>
            <div class="qr-wrapper">
              <QRCodeVue :value="qrCodeValue" :size="120" level="H" />
            </div>
            <button class="copy-btn" @click="copyLink">
              Copy Link
            </button>
          </div>

          <div class="control-group">
            <p class="section-label">Event Controls</p>
            <button 
              class="action-button"
              :class="currentEvent?.isActive ? 'stop-button' : 'start-button'"
              @click="toggleStatus"
            >
              {{ currentEvent?.isActive ? 'STOP EVENT' : 'START EVENT' }}
            </button>
            <p class="control-hint">
              {{ currentEvent?.isActive 
                ? 'Event is live.' 
                : 'Event is stopped.' }}
            </p>
          </div>
        </div>
      </div>

      <div class="card modules-card">
        <div class="card-header">
          <p class="card-title">Event Modules</p>
          <button class="add-button" @click="showModuleModal = true">
            + Add Module
          </button>
        </div>
        <p class="card-description">Manage interactive modules for this event.</p>

        <div v-if="currentEvent?.modules && currentEvent.modules.length > 0" class="modules-list">
          <div v-for="(module, index) in currentEvent.modules" :key="index" class="module-item">
            <div class="module-info">
              <p class="module-name">{{ module.name }}</p>
              <p class="module-type">{{ module.type.toUpperCase() }}</p>
              <span 
                class="module-status-badge"
                :class="{
                  'status-active': module.isActive,
                  'status-inactive': !module.isActive
                }"
              >
                {{ module.isActive ? 'ACTIVE' : 'INACTIVE' }}
              </span>
            </div>
            <div class="module-actions">
              <button class="module-btn config-btn" @click="configureModule(module)">
                Configure
              </button>
              <button 
                class="module-btn"
                :class="module.isActive ? 'stop-button' : 'start-button'"
                @click="toggleModuleStatus(module)"
              >
                {{ module.isActive ? 'Stop' : 'Start' }}
              </button>
            </div>
          </div>
        </div>

        <div v-else class="empty-modules">
          <p class="empty-modules-text">No modules added yet.</p>
        </div>
      </div>
    </div>

    <FormModuleConfig
      v-if="showConfigModal"
      :visible="showConfigModal"
      :module="selectedModule"
      :event-id="currentEvent?.id"
      @close="closeConfigModal"
      @update="fetchEventDetails"
    />

    <div v-if="showModuleModal" class="modal-overlay" @click.self="showModuleModal = false">
      <div class="add-module-container">
        <div class="add-module-header">
          <p class="form-title">Add Module</p>
          <button @click="showModuleModal = false" class="close-btn">✕</button>
        </div>

        <p class="input-label">Module Name</p>
        <input
          v-model="newModuleName"
          type="text"
          placeholder="e.g. Feedback Form"
          class="module-input"
        />

        <p class="input-label">Module Type</p>
        <div class="dropdown">
          <span>📝 Form</span>
        </div>

        <button class="add-module-button" @click="addModule">
          Add Module
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import FormModuleConfig from '../components/FormModuleConfig.vue'
import QRCodeVue from 'qrcode.vue'
import { API_BASE_URL } from '../utils'
import { useToast } from '../composables/useToast'

const router = useRouter()
const route = useRoute()

const currentEvent = ref(null)
const showModuleModal = ref(false)
const newModuleName = ref('')
const selectedModuleType = ref('form')
const showConfigModal = ref(false)
const selectedModule = ref(null)

const { success, error, info } = useToast()

const storedEvent = sessionStorage.getItem('currentEvent')
currentEvent.value = storedEvent ? JSON.parse(storedEvent) : null

const qrCodeValue = computed(() => {
  return currentEvent.value 
    ? `${window.location.origin}/?pin=${currentEvent.value.id}`
    : ''
})

const fetchEventDetails = async () => {
  if (!currentEvent.value?.id) return

  try {
    const response = await fetch(`${API_BASE_URL}/events/${currentEvent.value.id}`)
    if (response.ok) {
      const data = await response.json()
      currentEvent.value = data
      sessionStorage.setItem('currentEvent', JSON.stringify(data))
    }
  } catch (err) {
    console.error('Error fetching event details:', err)
  }
}

onMounted(() => {
  fetchEventDetails()
})

const toggleStatus = async () => {
  try {
    const response = await fetch(`${API_BASE_URL}/events/${currentEvent.value.id}/status`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ isActive: !currentEvent.value.isActive })
    })

    if (response.ok) {
      currentEvent.value.isActive = !currentEvent.value.isActive
      success(`Event ${currentEvent.value.isActive ? 'started' : 'stopped'}`)
    } else {
      error('Failed to update status')
    }
  } catch (err) {
    console.error('Error updating status:', err)
    error('Failed to update status')
  }
}

const getShareLink = () => {
  return `${window.location.origin}/?pin=${currentEvent.value.id}`
}

const copyLink = () => {
  navigator.clipboard.writeText(getShareLink())
  success('Event link copied to clipboard!')
}

const configureModule = (module) => {
  selectedModule.value = module
  showConfigModal.value = true
}

const closeConfigModal = () => {
  showConfigModal.value = false
  selectedModule.value = null
}

const addModule = async () => {
  if (!newModuleName.value.trim()) {
    info('Please enter a module name')
    return
  }

  try {
    const response = await fetch(`${API_BASE_URL}/events/${currentEvent.value.id}/modules`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        type: selectedModuleType.value,
        name: newModuleName.value,
        data: {}
      })
    })

    if (response.ok) {
      fetchEventDetails()
      showModuleModal.value = false
      newModuleName.value = ''
      success('Module added. Now please configure it.')
    } else {
      error('Failed to add module')
    }
  } catch (err) {
    console.error('Error adding module:', err)
    error('Failed to add module')
  }
}

const toggleModuleStatus = async (module) => {
  try {
    const response = await fetch(`${API_BASE_URL}/events/${currentEvent.value.id}/modules/${module.id}/status`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ isActive: !module.isActive })
    })

    if (response.ok) {
      fetchEventDetails()
    } else {
      error('Failed to update module status')
    }
  } catch (err) {
    console.error('Error updating module status:', err)
    error('Failed to update module status')
  }
}

const goBack = () => {
  router.push('/admin')
}
</script>

<style scoped>
.admin-details-screen {
  min-height: 100vh;
  background-color: #121212;
}

.header {
  padding: 20px 24px;
  background-color: #1E1E1E;
  border-bottom: 1px solid #2C2C2C;
}

.header-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.back-link {
  color: #007AFF;
  font-size: 14px;
  background: none;
  border: none;
  cursor: pointer;
}

.header-title {
  font-size: 12px;
  color: #666;
  text-transform: uppercase;
  font-weight: 600;
  letter-spacing: 1px;
}

.header-subtitle {
  font-size: 20px;
  font-weight: 700;
  color: #FFF;
}

.scroll-content {
  padding: 24px;
  display: grid;
  grid-template-columns: minmax(300px, 1fr) minmax(400px, 2.5fr);
  gap: 24px;
  max-width: 1600px;
  margin: 0 auto;
  align-items: start;
}

@media (max-width: 900px) {
  .scroll-content {
    grid-template-columns: 1fr;
  }
}

.card {
  background-color: #1E1E1E;
  border-radius: 16px;
  padding: 24px;
  border: 1px solid #2C2C2C;
}

.event-sidebar {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}

.card-title {
  font-size: 18px;
  font-weight: 700;
  color: #FFF;
  margin-bottom: 4px;
}

.card-description {
  font-size: 13px;
  color: #AAA;
}

.sidebar-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.info-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.label {
  font-size: 11px;
  color: #666;
  text-transform: uppercase;
  font-weight: 600;
  letter-spacing: 0.5px;
}

.value {
  font-size: 15px;
  color: #FFF;
}

.pin-display {
  background-color: #000;
  padding: 12px;
  border-radius: 8px;
  text-align: center;
  border: 1px solid #333;
}

.pin-text {
  font-size: 28px;
  color: #FFF;
  font-weight: bold;
  letter-spacing: 4px;
  font-family: monospace;
}

.qr-wrapper {
  background: white;
  padding: 12px;
  border-radius: 12px;
  align-self: center;
}

.copy-btn {
  margin-top: 8px;
  background-color: #333;
  padding: 10px;
  border-radius: 8px;
  width: 100%;
  color: #FFF;
  font-weight: 600;
  font-size: 13px;
  border: 1px solid #444;
  cursor: pointer;
  transition: background 0.2s;
}

.copy-btn:hover {
  background-color: #444;
}

.control-group {
  margin-top: 10px;
  padding-top: 20px;
  border-top: 1px solid #2C2C2C;
}

.section-label {
  font-size: 12px;
  font-weight: 600;
  color: #888;
  margin-bottom: 12px;
  text-transform: uppercase;
  letter-spacing: 1px;
}

.action-button {
  padding: 14px;
  border-radius: 8px;
  width: 100%;
  font-weight: 700;
  font-size: 14px;
  color: #FFF;
  border: none;
  cursor: pointer;
}

.start-button {
  background-color: #28cd41;
}

.stop-button {
  background-color: #ff3b30;
}

.control-hint {
  font-size: 11px;
  color: #666;
  margin-top: 8px;
  text-align: center;
}

.modules-card {
  min-height: 400px;
}

.add-button {
  background-color: #007AFF;
  padding: 6px 14px;
  border-radius: 20px;
  color: #FFF;
  font-size: 13px;
  font-weight: 600;
}

.modules-list {
  margin-top: 20px;
}

.module-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background-color: #2C2C2E;
  border-radius: 12px;
  margin-bottom: 12px;
}

.module-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.module-name {
  font-size: 16px;
  font-weight: 600;
  color: #FFF;
}

.module-type {
  font-size: 12px;
  color: #888;
}

.module-status-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 700;
}

.module-status-badge.status-active {
  background-color: rgba(48, 209, 88, 0.2);
  color: #30D158;
}

.module-status-badge.status-inactive {
  background-color: rgba(136, 136, 136, 0.2);
  color: #888;
}

.module-actions {
  display: flex;
  gap: 8px;
}

.module-btn {
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
}

.module-btn.config-btn {
  background-color: #333;
  color: #FFF;
}

.module-btn.start-button {
  background-color: #28cd41;
  color: #FFF;
}

.module-btn.stop-button {
  background-color: #ff3b30;
  color: #FFF;
}

.empty-modules {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px;
}

.empty-modules-text {
  color: #666;
  font-size: 14px;
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 100;
}

.add-module-container {
  background-color: #1E1E1E;
  border-radius: 16px;
  padding: 24px;
  width: 85%;
  max-width: 400px;
}

.add-module-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.form-title {
  font-size: 18px;
  font-weight: 700;
  color: #FFF;
}

.close-btn {
  background: none;
  border: none;
  color: #888;
  font-size: 20px;
  cursor: pointer;
}

.input-label {
  font-size: 12px;
  color: #888;
  margin-bottom: 8px;
  font-weight: 600;
}

.module-input {
  background-color: #121212;
  border: 1px solid #333;
  border-radius: 8px;
  padding: 12px 16px;
  color: white;
  font-size: 16px;
  width: 100%;
  margin-bottom: 16px;
}

.dropdown {
  background-color: #2C2C2E;
  border-radius: 8px;
  padding: 12px 16px;
  margin-bottom: 16px;
  color: #FFF;
  font-size: 14px;
}

.add-module-button {
  background-color: #007AFF;
  padding: 14px;
  border-radius: 8px;
  width: 100%;
  color: #FFF;
  font-weight: 600;
  font-size: 16px;
}
</style>
