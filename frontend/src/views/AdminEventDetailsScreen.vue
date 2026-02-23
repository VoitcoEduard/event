<template>
  <div class="admin-details-screen">
    <div class="header">
      <div class="header-content">
        <div class="header-left">
          <div class="event-info">
            <div class="language-switcher">
              <select v-model="$i18n.locale" @change="changeLanguage">
                <option value="en">English</option>
                <option value="ru">Русский</option>
                <option value="ro">Română</option>
              </select>
            </div>
            <h1 class="event-title">{{ currentEvent?.name }}</h1>
            <span class="event-status" :class="currentEvent?.isActive ? 'active' : 'inactive'">
              {{ currentEvent?.isActive ? $t('admin.status_active') : $t('admin.status_inactive') }}
            </span>
          </div>
        </div>
        <div class="header-right">
          <button class="nav-button screen-manager-btn" @click="goToScreenManager">
            {{ $t('admin.details.screen_manager') }}
          </button>
          <button class="nav-button copy-link-btn" @click="copyBigScreenLink" :title="$t('quiz.host.copy_link')">
             🔗
          </button>
          <button class="nav-button library-btn" @click="goToLibrary">
            {{ $t('admin.details.library_btn') }}
          </button>
          <button class="nav-button data-btn" @click="goToDataView">
            {{ $t('common.data_btn') }}
          </button>
          <button class="nav-button back-btn" @click="goBack">
            {{ $t('admin.details.back_btn') }}
          </button>
        </div>
      </div>
    </div>

    <div class="scroll-content">
      <div class="card event-sidebar">
        <div class="card-header">
          <div>
            <p class="card-title">{{ $t('admin.details.overview') }}</p>
            <p class="card-description">{{ $t('admin.details.overview_desc') }}</p>
          </div>
          <span class="status-badge" :class="currentEvent?.isActive ? 'active' : 'inactive'">
            {{ currentEvent?.isActive ? $t('admin.status_active') : $t('admin.status_inactive') }}
          </span>
        </div>

        <div class="sidebar-content">
          <div class="info-group">
            <p class="label">{{ $t('admin.details.description') }}</p>
            <p class="value">{{ currentEvent?.description }}</p>
          </div>

          <div class="info-group">
            <p class="label">{{ $t('admin.details.pin_code') }}</p>
            <div class="pin-display">
              <span class="pin-text">{{ currentEvent?.id }}</span>
            </div>
          </div>

          <div class="info-group">
            <p class="label">{{ $t('admin.details.share_access') }}</p>
            <div class="qr-wrapper">
              <QRCodeVue :value="qrCodeValue" :size="120" level="H" />
            </div>
            <button class="copy-btn" @click="copyLink">
              {{ $t('admin.details.copy_link') }}
            </button>
          </div>

          <div class="control-group">
            <p class="section-label">{{ $t('admin.details.controls') }}</p>
            <button 
              class="action-button"
              :class="currentEvent?.isActive ? 'stop-button' : 'start-button'"
              @click="toggleStatus"
            >
              {{ currentEvent?.isActive ? $t('admin.details.stop_event') : $t('admin.details.start_event') }}
            </button>
            <p class="control-hint">
              {{ currentEvent?.isActive 
                ? $t('admin.details.live_hint') 
                : $t('admin.details.stopped_hint') }}
            </p>
          </div>

          </div>
        </div>

      <div class="card modules-card">
        <div class="card-header">
          <p class="card-title">{{ $t('admin.details.modules') }}</p>
          <button class="add-button" @click="showModuleModal = true">
            {{ $t('admin.details.add_module') }}
          </button>
        </div>
        <p class="card-description">{{ $t('admin.details.modules_desc') }}</p>

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
                {{ module.isActive ? $t('admin.status_active') : $t('admin.status_inactive') }}
              </span>
            </div>
                        <div class="module-actions">
                          <button class="module-btn config-btn" @click="configureModule(module)">
                            {{ $t('admin.details.configure') }}
                          </button>
                          <button
                            v-if="module.type === 'quiz'"
                            class="module-btn host-btn"
                            @click="hostQuiz(module)"
                          >
                            {{ $t('admin.details.host') }}
                          </button>
                          
                          <button
                            v-if="module.type === 'quiz'"
                            class="module-btn push-btn"
                            @click="pushQuizToBigScreen(module)"
                          >
                            🖥️ {{ $t('admin.details.push_to_big_screen') }}
                          </button>
                          
                          <button
                            class="module-btn"
                            :class="module.isActive ? 'stop-button' : 'start-button'"
                            @click="toggleModuleStatus(module)"
                          >
                            {{ module.isActive ? $t('admin.details.stop') : $t('admin.details.start') }}
                          </button>
                          <button
                            class="module-btn delete-btn"
                            @click="deleteModule(module)"
                          >
                            {{ $t('admin.details.delete') }}
                          </button>
                        </div>
                      </div>
        </div>

        <div v-else class="empty-modules">
          <p class="empty-modules-text">{{ $t('admin.details.no_modules') }}</p>
        </div>
      </div>
    </div>

    <FormModuleConfig
      v-if="showConfigModal && selectedModule?.type === 'form'"
      :visible="showConfigModal && selectedModule?.type === 'form'"
      :module="selectedModule"
      :event-id="currentEvent?.id"
      @close="closeConfigModal"
      @update="fetchEventDetails"
    />

    <QuizModuleConfig
      v-if="showConfigModal && selectedModule?.type === 'quiz'"
      :visible="showConfigModal && selectedModule?.type === 'quiz'"
      :module="selectedModule"
      :event-id="currentEvent?.id"
      @close="closeConfigModal"
      @update="fetchEventDetails"
    />

    <div v-if="showModuleModal" class="modal-overlay" @click.self="showModuleModal = false">
      <div class="add-module-container">
        <div class="add-module-header">
          <p class="form-title">{{ $t('admin.details.add_modal_title') }}</p>
          <button @click="showModuleModal = false" class="close-btn">✕</button>
        </div>

        <p class="input-label">{{ $t('admin.details.module_name_label') }}</p>
        <input
          v-model="newModuleName"
          type="text"
          :placeholder="$t('admin.details.module_name_placeholder')"
          class="module-input"
        />

        <p class="input-label">{{ $t('admin.details.module_type_label') }}</p>
        <select v-model="selectedModuleType" class="dropdown-select">
          <option value="form">{{ $t('admin.details.form_option') }}</option>
          <option value="quiz">{{ $t('admin.details.quiz_option') }}</option>
        </select>

        <button class="add-module-button" @click="addModule">
          {{ $t('admin.details.add_modal_title') }}
        </button>
      </div>
    </div>


  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import FormModuleConfig from '../components/FormModuleConfig.vue'
import QuizModuleConfig from '../components/QuizModuleConfig.vue'

import QRCodeVue from 'qrcode.vue'
import { API_BASE_URL, getAuthHeaders } from '../utils'
import { useToast } from '../composables/useToast'
import { getImageUrl } from '../services/media'
import { getScreensavers } from '../services/media'

const { t, locale } = useI18n()
const router = useRouter()
const route = useRoute()

const currentEvent = ref(null)
const showModuleModal = ref(false)
const newModuleName = ref('')
const selectedModuleType = ref('form')
const showConfigModal = ref(false)
const selectedModule = ref(null)

// Screensaver state
const screensaverPreview = ref('')
const screensaverLoading = ref(false)
const screensavers = ref([])
const showLibrarySelector = ref(false)

// BigScreen window reference
const bigscreenWindow = ref(null)
const isBigScreenOpen = computed(() => bigscreenWindow.value && !bigscreenWindow.value.closed)

const { success, error, info } = useToast()

const changeLanguage = () => {
  localStorage.setItem('userLanguage', locale.value)
}

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
  fetchScreensaverConfig()
})

// Screensaver functions
const fetchScreensaverConfig = async () => {
  if (!currentEvent.value?.id) return
  
  try {
    const response = await fetch(`${API_BASE_URL}/events/${currentEvent.value.id}/screensaver`)
    if (response.ok) {
      const data = await response.json()
      if (data.imageId) {
        currentEvent.value.screensaverConfig = data
        screensaverPreview.value = getImageUrl(data.imageId)
      }
    }
  } catch (err) {
    console.error('Error fetching screensaver config:', err)
  }
}

const handleScreensaverUpload = async () => {
  screensaverLoading.value = true
  
  try {
    // Load screensavers from library
    screensavers.value = await getScreensavers()
    showLibrarySelector.value = true
  } catch (err) {
    console.error('Error loading screensavers:', err)
    error('Failed to load screensavers from library')
  } finally {
    screensaverLoading.value = false
  }
}

const selectScreensaver = async (screensaver) => {
  try {
    // Save screensaver config using the library screensaver's mediaId
    const configResponse = await fetch(`${API_BASE_URL}/events/${currentEvent.value.id}/screensaver`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify({ imageId: screensaver.mediaId })
    })

    if (configResponse.ok) {
      currentEvent.value.screensaverConfig = await configResponse.json()
      screensaverPreview.value = screensaver.thumbnailUrl
      showLibrarySelector.value = false
      success(t('admin.details.toast.screensaver_updated'))
    } else {
      error('Failed to save screensaver config')
    }
  } catch (err) {
    console.error('Error selecting screensaver:', err)
    error('Failed to select screensaver')
  }
}

const removeScreensaver = async () => {
  try {
    const response = await fetch(`${API_BASE_URL}/events/${currentEvent.value.id}/screensaver`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify({ imageId: '', imageUrl: '' })
    })
    
    if (response.ok) {
      currentEvent.value.screensaverConfig = null
      screensaverPreview.value = ''
      success(t('admin.details.toast.screensaver_removed'))
    } else {
      error('Failed to remove screensaver')
    }
  } catch (err) {
    console.error('Error removing screensaver:', err)
    error('Failed to remove screensaver image')
  }
}

const toggleStatus = async () => {
  try {
    const response = await fetch(`${API_BASE_URL}/events/${currentEvent.value.id}/status`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify({ isActive: !currentEvent.value.isActive })
    })

    if (response.ok) {
      currentEvent.value.isActive = !currentEvent.value.isActive
      success(t(`admin.status_${currentEvent.value.isActive ? 'active' : 'inactive'}`))
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
  success(t('admin.details.toast.event_link_copied'))
}

const configureModule = (module) => {
  selectedModule.value = module
  showConfigModal.value = true
}

const closeConfigModal = () => {
  showConfigModal.value = false
  selectedModule.value = null
}

const hostQuiz = (module) => {
  router.push(`/quiz/${currentEvent.value.id}/${module.id}/host`)
}

const launchBigScreen = (module) => {
  // Open big screen in a new window
  const url = `/bigscreen/${currentEvent.value.id}/${module.id}`
  bigscreenWindow.value = window.open(url, 'bigscreen', 'width=1200,height=800')
}

const pushQuizToBigScreen = async (module) => {
  // Push the quiz to big screen without opening a new window
  // This will send a screen state update to the big screen
  try {
    const response = await fetch(`${API_BASE_URL}/events/${currentEvent.value.id}/screen-state`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify({
        currentView: 'quiz',
        moduleId: module.id
      })
    })
    
    if (response.ok) {
      success(t('admin.details.pushed_to_big_screen'))
    } else {
      error(t('admin.details.push_error'))
    }
  } catch (err) {
    console.error('Error pushing to big screen:', err)
    error(t('admin.details.push_error'))
  }
}

const addModule = async () => {
  if (!newModuleName.value.trim()) {
    info(t('admin.details.messages.fill_fields'))
    return
  }

  try {
    const response = await fetch(`${API_BASE_URL}/events/${currentEvent.value.id}/modules`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
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
      success(t('admin.details.toast.module_added'))
    } else {
      error('Failed to add module')
    }
  } catch (err) {
    console.error('Error adding module:', err)
    error('Failed to add module')
  }
}

const toggleModuleStatus = async (module) => {
  console.log('[DEBUG] toggleModuleStatus: Starting for module:', module.id, 'current isActive:', module.isActive)
  
  // If stopping a quiz module, reset the quiz session first
  if (module.isActive && module.type === 'quiz') {
    try {
      console.log('[DEBUG] toggleModuleStatus: Resetting quiz session for module:', module.id)
      await fetch(`${API_BASE_URL}/events/${currentEvent.value.id}/modules/${module.id}/quiz/reset`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...getAuthHeaders() }
      })
      console.log('[DEBUG] toggleModuleStatus: Quiz session reset complete')
    } catch (err) {
      console.error('Error resetting quiz session:', err)
    }
  }

  try {
    console.log('[DEBUG] toggleModuleStatus: Updating status to:', !module.isActive)
    const response = await fetch(`${API_BASE_URL}/events/${currentEvent.value.id}/modules/${module.id}/status`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify({ isActive: !module.isActive })
    })

    if (response.ok) {
      console.log('[DEBUG] toggleModuleStatus: Status update successful, fetching event details')
      await fetchEventDetails()
      console.log('[DEBUG] toggleModuleStatus: Event details fetched, new module isActive:', currentEvent.value.modules.find(m => m.id === module.id)?.isActive)
      success(`Module ${!module.isActive ? 'started' : 'stopped'}`)
    } else {
      error('Failed to update module status')
    }
  } catch (err) {
    console.error('Error updating module status:', err)
    error('Failed to update module status')
  }
}

const deleteModule = async (module) => {
  if (!confirm(t('admin.confirm_delete'))) return

  try {
    const response = await fetch(`${API_BASE_URL}/events/${currentEvent.value.id}/modules/${module.id}`, {
      method: 'DELETE',
      headers: { ...getAuthHeaders() }
    })

    if (response.ok) {
      success('Module deleted')
      fetchEventDetails()
    } else {
      error('Failed to delete module')
    }
  } catch (err) {
    console.error('Error deleting module:', err)
    error('Failed to delete module')
  }
}

const goBack = () => {
  router.push('/admin')
}

const goToLibrary = () => {
  router.push('/quiz/library')
}

const goToDataView = () => {
  router.push(`/data/library?eventId=${currentEvent.value.id}`)
}

const goToScreenManager = () => {
  if (currentEvent.value?.id) {
    router.push(`/admin/event/${currentEvent.value.id}/screen-manager`)
  }
}

const copyBigScreenLink = () => {
  let urlPart = ''
  // Find the first quiz module (optional - bigscreen can work without quiz)
  const quizModule = currentEvent.value?.modules?.find(m => m.type === 'quiz')
  if (quizModule) {
    urlPart = `/bigscreen/${currentEvent.value.id}/${quizModule.id}`
  } else {
    // Open bigscreen without quiz module - will show screensaver only
    urlPart = `/bigscreen/${currentEvent.value.id}`
  }
  
  const fullUrl = `${window.location.origin}${urlPart}`
  navigator.clipboard.writeText(fullUrl)
  success(t('quiz.host.toast.link_copied'))
}
</script>

<style scoped>
.admin-details-screen {
  min-height: 100vh;
  background-color: #121212;
}

.language-switcher {
  margin-right: 20px;
}

.language-switcher select {
  background-color: #252525;
  color: white;
  border: 1px solid #333333;
  padding: 8px 12px;
  border-radius: 8px;
  cursor: pointer;
  outline: none;
}

.header {
  padding: 20px 40px;
  background-color: #1E1E1E;
  border-bottom: 1px solid #333;
  box-shadow: 0 2px 4px rgba(0,0,0,0.2);
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  max-width: 1400px;
  margin: 0 auto;
}

.header-left {
  flex: 1;
}

.event-info {
  display: flex;
  align-items: center;
  gap: 20px;
}

.event-title {
  font-size: 32px;
  font-weight: 700;
  color: #ffffff;
  margin: 0;
  line-height: 1.2;
}

.event-status {
  padding: 8px 20px;
  border-radius: 20px;
  font-size: 14px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.event-status.active {
  background-color: #d4edda;
  color: #155724;
  border: 1px solid #c3e6cb;
}

.event-status.inactive {
  background-color: #f8d7da;
  color: #721c24;
  border: 1px solid #f5c6cb;
}

.header-right {
  display: flex;
  gap: 16px;
  align-items: center;
}

.nav-button {
  padding: 12px 24px;
  border: none;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 8px;
  text-decoration: none;
}

.library-btn {
  background-color: #4CAF50;
  color: white;
}

.library-btn:hover {
  background-color: #45a049;
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(76, 175, 80, 0.3);
}

.screen-manager-btn {
  background-color: #FF9500;
  color: white;
}

.screen-manager-btn:hover {
  background-color: #FFA830;
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(255, 149, 0, 0.3);
}

.copy-link-btn {
  background-color: #2196F3;
  color: white;
  min-width: 50px;
  justify-content: center;
}

.copy-link-btn:hover {
  background-color: #1e88e5;
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(33, 150, 243, 0.3);
}

.data-btn {
  background-color: #9C27B0;
  color: white;
}

.data-btn:hover {
  background-color: #8e24aa;
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(156, 39, 176, 0.3);
}

.back-btn {
  background-color: #607D8B;
  color: white;
}

.back-btn:hover {
  background-color: #546e7a;
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(96, 125, 139, 0.3);
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
  border: 1px solid #333;
  box-shadow: 0 2px 8px rgba(0,0,0,0.2);
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
  color: #ffffff;
  margin-bottom: 4px;
}

.card-description {
  font-size: 13px;
  color: #666;
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

.screensaver-preview {
  width: 100%;
  height: 120px;
  background: #1e1e1e;
  border-radius: 8px;
  overflow: hidden;
  margin-bottom: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.screensaver-preview img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}

.screensaver-upload {
  margin-bottom: 12px;
}

.file-input {
  display: none;
}

.file-label {
  display: block;
  padding: 12px;
  background: #333;
  color: white;
  text-align: center;
  border-radius: 8px;
  cursor: pointer;
  font-weight: 600;
  transition: background 0.2s;
}

.file-label:hover {
  background: #444;
}

.delete-screensaver-btn {
  background-color: #ff3b30 !important;
  margin-top: 8px;
}

/* Library Selector Modal */
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
  z-index: 1000;
}

.modal-content {
  background-color: #1C1C1E;
  width: 90%;
  max-width: 600px;
  max-height: 80vh;
  border-radius: 12px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #333;
}

.modal-header h3 {
  margin: 0;
  color: white;
  font-size: 18px;
}

.close-btn {
  background: none;
  border: none;
  color: #888;
  font-size: 20px;
  cursor: pointer;
}

.close-btn:hover {
  color: white;
}

.modal-body {
  padding: 20px;
  overflow-y: auto;
}

.screensavers-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 16px;
}

.screensaver-option {
  cursor: pointer;
  border-radius: 8px;
  overflow: hidden;
  border: 2px solid transparent;
  transition: border-color 0.2s;
}

.screensaver-option:hover {
  border-color: #0A84FF;
}

.screensaver-option img {
  width: 100%;
  height: 120px;
  object-fit: cover;
}

.screensaver-name {
  padding: 8px;
  margin: 0;
  font-size: 12px;
  color: white;
  background: #2C2C2E;
  text-align: center;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.empty-state {
  text-align: center;
  padding: 40px;
  color: #888;
}

.empty-state .hint {
  font-size: 14px;
  margin-top: 8px;
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

.module-btn.delete-btn {
  background-color: #1c1c1e;
  color: #ff453a;
  border: 1px solid #ff453a;
  margin-left: 8px;
}

.module-btn.delete-btn:hover {
  background-color: rgba(255, 69, 58, 0.1);
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

.module-btn.host-btn {
  background-color: #5E5CE6;
  color: #FFF;
}

.module-btn.bigscreen-btn {
  background-color: #FF9500;
  color: #FFF;
}

.module-btn.bigscreen-btn:hover {
  background-color: #FFA830;
}

.module-btn.push-btn {
  background-color: #30D158;
  color: #FFF;
}

.module-btn.push-btn:hover {
  background-color: #28cd41;
}

.dropdown-select {
  background-color: #2C2C2E;
  border-radius: 8px;
  padding: 12px 16px;
  margin-bottom: 16px;
  color: #FFF;
  font-size: 14px;
  border: 1px solid #3A3A3C;
  width: 100%;
  appearance: none;
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

.data-view-modal-content {
  background-color: #1E1E1E;
  border-radius: 16px;
  padding: 24px;
  width: 90%;
  max-width: 1200px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  gap: 20px;
  overflow: hidden; /* Ensure content within modal respects its bounds */
}

.data-view-modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
