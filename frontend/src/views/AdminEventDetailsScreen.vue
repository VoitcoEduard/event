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
      <div class="card unified-card">
        <div class="card-header">
          <div>
            <p class="card-title">Event Overview</p>
            <p class="card-description">Manage details, sharing, and controls.</p>
          </div>
          <span class="status-badge" :class="currentEvent?.isActive ? 'active' : 'inactive'">
            {{ currentEvent?.isActive ? 'ACTIVE - RUNNING' : 'INACTIVE - STOPPED' }}
          </span>
        </div>

        <div class="unified-content">
          <div class="left-column">
            <div class="info-grid">
              <div class="info-item">
                <p class="label">Description</p>
                <p class="value">{{ currentEvent?.description }}</p>
              </div>
              <div class="info-item">
                <p class="label">PIN Code</p>
                <div class="pin-container">
                  <p class="pin-text">{{ currentEvent?.id }}</p>
                </div>
              </div>
            </div>

            <div class="control-section">
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
                  ? 'Event is live. Users can connect.' 
                  : 'Event is stopped. Users cannot connect.' }}
              </p>
            </div>
          </div>

          <div class="vertical-divider"></div>

          <div class="right-column">
            <p class="section-label">Share Access</p>
            <div class="qr-code-container" v-html="qrCodeSVG"></div>
            <button class="copy-button" @click="copyLink">
              Copy Link
            </button>
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
                  'status-active': module.status === 'active',
                  'status-ended': module.status === 'ended'
                }"
              >
                {{ module.status.toUpperCase() }}
              </span>
            </div>
            <div class="module-actions">
              <template v-if="module.status === 'pending'">
                <button class="module-btn config-btn" @click="configureModule(module)">
                  Configure
                </button>
                <button class="module-btn activate-btn" @click="updateModuleStatus(module.id, 'active')">
                  Activate
                </button>
              </template>
              <template v-if="module.status === 'active'">
                <button class="module-btn end-btn" @click="updateModuleStatus(module.id, 'ended')">
                  End
                </button>
              </template>
              <template v-if="module.status === 'ended'">
                <p class="ended-text">Completed</p>
              </template>
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

const router = useRouter()
const route = useRoute()

const currentEvent = ref(null)
const showModuleModal = ref(false)
const newModuleName = ref('')
const selectedModuleType = ref('form')
const showConfigModal = ref(false)
const selectedModule = ref(null)

import { API_BASE_URL } from '../utils'

const storedEvent = sessionStorage.getItem('currentEvent')
currentEvent.value = storedEvent ? JSON.parse(storedEvent) : null

const generateQRCode = (text) => {
  const size = 160
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${size} ${size}" width="${size}" height="${size}">
    <rect width="100%" height="100%" fill="white"/>
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="${size-24}" height="${size-24}" x="12" y="12">
      <rect width="256" height="256" fill="white"/>
      ${generateQRMatrix(text)}
    </svg>
  </svg>`
  return svg
}

const generateQRMatrix = (text) => {
  let matrix = ''
  const cellSize = 16
  for (let y = 0; y < 16; y++) {
    for (let x = 0; x < 16; x++) {
      if (Math.sin(Math.sin(x * y * 123) + text.length) > 0.5) {
        matrix += `<rect x="${x * cellSize}" y="${y * cellSize}" width="${cellSize}" height="${cellSize}" fill="black"/>`
      }
    }
  }
  return matrix
}

const qrCodeSVG = computed(() => {
  const link = currentEvent.value 
    ? `${window.location.origin}/?pin=${currentEvent.value.id}`
    : ''
  return generateQRCode(link)
})

const fetchEventDetails = async () => {
  if (!currentEvent.value?.id) return

  try {
    const response = await fetch(`${API_URL}/events/${currentEvent.value.id}`)
    if (response.ok) {
      const data = await response.json()
      currentEvent.value = data
      sessionStorage.setItem('currentEvent', JSON.stringify(data))
    }
  } catch (error) {
    console.error('Error fetching event details:', error)
  }
}

onMounted(() => {
  fetchEventDetails()
})

const toggleStatus = async () => {
  try {
    const response = await fetch(`${API_URL}/events/${currentEvent.value.id}/status`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ isActive: !currentEvent.value.isActive })
    })

    if (response.ok) {
      currentEvent.value.isActive = !currentEvent.value.isActive
      alert(`Event ${currentEvent.value.isActive ? 'started' : 'stopped'}`)
    } else {
      alert('Failed to update status')
    }
  } catch (error) {
    console.error('Error updating status:', error)
    alert('Failed to update status')
  }
}

const getShareLink = () => {
  return `${window.location.origin}/?pin=${currentEvent.value.id}`
}

const copyLink = () => {
  navigator.clipboard.writeText(getShareLink())
  alert('Event link copied to clipboard!')
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
    alert('Please enter a module name')
    return
  }

  try {
    const response = await fetch(`${API_URL}/events/${currentEvent.value.id}/modules`, {
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
      alert('Module added. Now please configure it.')
    } else {
      alert('Failed to add module')
    }
  } catch (error) {
    console.error('Error adding module:', error)
    alert('Failed to add module')
  }
}

const updateModuleStatus = async (moduleId, newStatus) => {
  try {
    const response = await fetch(`${API_URL}/events/${currentEvent.value.id}/modules/${moduleId}/status`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ status: newStatus })
    })

    if (response.ok) {
      fetchEventDetails()
    } else {
      alert('Failed to update module status')
    }
  } catch (error) {
    console.error('Error updating module status:', error)
    alert('Failed to update module status')
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
  padding-top: 20px;
  padding-bottom: 16px;
  padding-left: 24px;
  padding-right: 24px;
  background-color: #1E1E1E;
  border-bottom: 1px solid #2C2C2C;
}

.header-top {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
}

.back-link {
  color: #007AFF;
  font-size: 14px;
  background: none;
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
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
  max-width: 1600px;
  margin: 0 auto;
}

.card {
  background-color: #1E1E1E;
  border-radius: 16px;
  padding: 24px;
  border: 1px solid #2C2C2C;
  min-width: 350px;
}

.unified-card {
  flex: 3;
  min-width: 600px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 20px;
}

.card-title {
  font-size: 20px;
  font-weight: 700;
  color: #FFF;
  margin-bottom: 12px;
}

.card-description {
  font-size: 14px;
  color: #AAA;
  line-height: 20;
}

.unified-content {
  display: flex;
  gap: 40px;
  flex-wrap: wrap;
}

.left-column {
  flex: 2;
  min-width: 300px;
}

.right-column {
  flex: 1;
  min-width: 250px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background-color: #252525;
  border-radius: 16px;
  padding: 24px;
}

.vertical-divider {
  width: 1px;
  background-color: #333;
}

.info-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  margin-bottom: 30px;
}

.info-item {
  min-width: 150px;
  flex: 1;
}

.label {
  font-size: 12px;
  color: #888;
  margin-bottom: 4px;
  text-transform: uppercase;
}

.value {
  font-size: 16px;
  color: #FFF;
}

.pin-container {
  background-color: #000;
  padding: 10px;
  border-radius: 8px;
  display: inline-block;
}

.pin-text {
  font-size: 24px;
  color: #FFF;
  font-weight: bold;
  letter-spacing: 2px;
  font-family: monospace;
}

.control-section {
  margin-top: auto;
  padding-top: 20px;
  border-top: 1px solid #2C2C2C;
}

.section-label {
  font-size: 14px;
  font-weight: 600;
  color: #FFF;
  margin-bottom: 16px;
  text-transform: uppercase;
  letter-spacing: 1px;
}

.action-button {
  padding: 16px;
  border-radius: 8px;
  width: 100%;
  font-weight: 700;
  font-size: 16px;
  color: #FFF;
}

.start-button {
  background-color: #30D158;
}

.stop-button {
  background-color: #FF453A;
}

.control-hint {
  font-size: 12px;
  color: #666;
  margin-top: 12px;
  text-align: center;
}

.qr-code-container {
  background: white;
  padding: 12px;
  border-radius: 12px;
}

.copy-button {
  margin-top: 20px;
  background-color: #333;
  padding: 14px;
  border-radius: 8px;
  width: 100%;
  color: #FFF;
  font-weight: 600;
  font-size: 14px;
  border: 1px solid #444;
}

.modules-card {
  flex: 2;
  min-width: 600px;
}

.add-button {
  background-color: #007AFF;
  padding: 8px 16px;
  border-radius: 20px;
  color: #FFF;
  font-size: 14px;
  font-weight: 600;
}

.modules-list {
  margin-bottom: 20px;
}

.module-item {
  background-color: #2C2C2C;
  border-radius: 12px;
  padding: 16px;
  margin-bottom: 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.module-info {
  flex: 1;
}

.module-name {
  font-size: 16px;
  font-weight: 600;
  color: #FFF;
  margin-bottom: 4px;
}

.module-type {
  font-size: 12px;
  color: #888;
  margin-bottom: 8px;
}

.module-status-badge {
  padding: 4px 8px;
  border-radius: 6px;
  font-size: 10px;
  font-weight: 700;
  display: inline-block;
}

.module-status-badge.status-active {
  background-color: rgba(48, 209, 88, 0.2);
  color: #30D158;
}

.module-status-badge.status-ended {
  background-color: rgba(255, 69, 58, 0.1);
  color: #FF453A;
}

.module-actions {
  display: flex;
  gap: 10px;
}

.module-btn {
  padding: 8px 16px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
}

.config-btn {
  background-color: #FF9500;
}

.activate-btn {
  background-color: #30D158;
}

.end-btn {
  background-color: #FF453A;
}

.ended-text {
  color: #666;
  font-size: 14px;
  font-style: italic;
}

.empty-modules {
  padding: 40px;
  display: flex;
  justify-content: center;
  align-items: center;
  border: 1px dashed #444;
  border-radius: 12px;
}

.empty-modules-text {
  color: #666;
  font-size: 16px;
}

.status-badge {
  padding: 6px 12px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 700;
}

.status-badge.active {
  background-color: rgba(48, 209, 88, 0.2);
  color: #30D158;
}

.status-badge.inactive {
  background-color: rgba(255, 69, 58, 0.2);
  color: #FF453A;
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.7);
  display: flex;
  justify-content: center;
  align-items: center;
}

.add-module-container {
  background-color: #252525;
  border-radius: 16px;
  padding: 24px;
  width: 85%;
  max-width: 400px;
  border: 1px solid #333;
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
  color: #AAA;
  font-size: 20px;
  background: none;
}

.input-label {
  font-size: 14px;
  font-weight: 600;
  color: #AAA;
  margin-bottom: 8px;
  margin-top: 16px;
}

.module-input {
  background-color: #1E1E1E;
  border-radius: 12px;
  padding: 16px;
  font-size: 16px;
  color: #FFF;
  border: 1px solid #333;
}

.dropdown {
  background-color: #1E1E1E;
  border-radius: 12px;
  padding: 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border: 1px solid #333;
  color: #FFF;
}

.add-module-button {
  background-color: #007AFF;
  padding: 16px;
  border-radius: 12px;
  margin-top: 24px;
  width: 100%;
  color: #FFF;
  font-size: 16px;
  font-weight: 600;
}
</style>
