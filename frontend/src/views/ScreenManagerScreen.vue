<template>
  <div class="screen-manager">
    <div class="header">
      <div class="header-left">
        <button class="back-btn" @click="goBack">← {{ $t('admin.details.back_btn') }}</button>
        <h1>{{ $t('screen_manager.title') }}</h1>
      </div>
      <div class="header-right">
        <button 
          class="launch-btn" 
          :class="{ 'close-btn': isBigScreenOpen }"
          @click="isBigScreenOpen ? closeBigScreen() : openBigScreen()"
        >
          {{ isBigScreenOpen ? $t('admin.details.close_big_screen') : $t('screen_manager.open_big_screen') }}
        </button>
      </div>
    </div>

    <div class="main-content">
      <div class="preview-panel">
        <h2>{{ $t('screen_manager.live_preview') }}</h2>
        <div class="preview-container">
           <!-- Render a mini version of what the Big Screen sees based on screenState -->
           <div v-if="screenState.currentView === 'welcome'" class="mini-view mini-welcome">
              <h1>{{ $t('quiz.host.join_title') }}</h1>
              <div v-if="screenState.showWelcomeQR" class="qr-placeholder-mini">QR</div>
              <div class="pin-mini">PIN: {{ eventId }}</div>
           </div>
           <div v-else-if="screenState.currentView === 'custom_text'" class="mini-view mini-custom-text">
              <h2>{{ screenState.customText || 'Your text here' }}</h2>
           </div>
           <div v-else-if="screenState.currentView === 'screensaver'" class="mini-view mini-screensaver">
              <img v-if="screensaverPreview" :src="screensaverPreview" />
              <div v-else class="empty-mini">Screensaver</div>
           </div>
           <div v-else class="mini-view mini-module">
              <h2>Module Active</h2>
           </div>
        </div>
      </div>

      <div class="controls-panel">
        
        <!-- Welcome Slide -->
        <div class="control-card" :class="{ active: screenState.currentView === 'welcome' }">
          <div class="card-header">
            <h2>{{ $t('screen_manager.welcome_slide') }}</h2>
            <button class="push-btn" @click="pushState('welcome')">
              {{ $t('screen_manager.push_to_screen') }}
            </button>
          </div>
          <div class="card-body">
            <label class="toggle-label">
              <input type="checkbox" v-model="localState.showWelcomeQR" @change="saveState" />
              {{ $t('screen_manager.show_qr') }}
            </label>
            <p class="hint">{{ $t('screen_manager.welcome_hint') }}</p>
          </div>
        </div>

        <!-- Custom Text Slide -->
        <div class="control-card" :class="{ active: screenState.currentView === 'custom_text' }">
          <div class="card-header">
            <h2>{{ $t('screen_manager.custom_text_slide') }}</h2>
            <button class="push-btn" @click="pushState('custom_text')">
              {{ $t('screen_manager.push_to_screen') }}
            </button>
          </div>
          <div class="card-body">
            <textarea 
              v-model="localState.customText" 
              @blur="saveState"
              :placeholder="$t('screen_manager.custom_text_placeholder')"
              rows="3"
            ></textarea>
          </div>
        </div>

        <!-- Screensaver -->
        <div class="control-card" :class="{ active: screenState.currentView === 'screensaver' }">
          <div class="card-header">
            <h2>{{ $t('admin.details.screensaver') }}</h2>
            <button class="push-btn" @click="pushState('screensaver')">
              {{ $t('screen_manager.push_to_screen') }}
            </button>
          </div>
          <div class="card-body">
            <div v-if="screensaverPreview" class="screensaver-preview">
              <img :src="screensaverPreview" alt="Screensaver preview" />
            </div>
            <div class="screensaver-upload">
              <button class="action-button" @click="handleScreensaverUpload">
                {{ $t('admin.details.select_library') }}
              </button>
              <button 
                v-if="screensaverPreview"
                class="action-button delete-btn"
                @click="removeScreensaver"
              >
                {{ $t('admin.details.remove_image') }}
              </button>
            </div>
          </div>
        </div>

      </div>
    </div>

    <!-- Library Selector Modal -->
    <div v-if="showLibrarySelector" class="modal-overlay" @click.self="showLibrarySelector = false">
      <div class="modal-content">
        <div class="modal-header">
          <h3>{{ $t('admin.details.select_title') }}</h3>
          <button class="close-btn" @click="showLibrarySelector = false">✕</button>
        </div>
        <div class="modal-body">
          <div v-if="screensavers.length === 0" class="empty-state">
            <p>{{ $t('admin.details.no_screensavers') }}</p>
          </div>
          <div v-else class="screensavers-grid">
            <div 
              v-for="s in screensavers" 
              :key="s.id" 
              class="screensaver-option"
              @click="selectScreensaver(s)"
            >
              <img :src="s.thumbnailUrl" />
              <p>{{ s.name }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { API_BASE_URL, getAuthHeaders } from '../utils'
import { useToast } from '../composables/useToast'
import { getImageUrl, getScreensavers } from '../services/media'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const { success, error } = useToast()

const eventId = route.params.id

// BigScreen window reference
const bigscreenWindow = ref(null)
const isBigScreenOpen = computed(() => bigscreenWindow.value && !bigscreenWindow.value.closed)

// Current saved state
const screenState = ref({
  currentView: 'screensaver',
  customText: '',
  showWelcomeQR: true,
  activeModuleId: ''
})

// Local editable state
const localState = ref({
  customText: '',
  showWelcomeQR: true
})

const screensaverPreview = ref('')
const showLibrarySelector = ref(false)
const screensavers = ref([])

onMounted(() => {
  fetchScreenState()
  fetchScreensaverConfig()
})

const fetchScreenState = async () => {
  try {
    const res = await fetch(`${API_BASE_URL}/events/${eventId}/screen-state`)
    if (res.ok) {
      const data = await res.json()
      screenState.value = data
      localState.value.customText = data.customText
      localState.value.showWelcomeQR = data.showWelcomeQR
    }
  } catch (err) {
    console.error('Failed to fetch screen state', err)
  }
}

const fetchScreensaverConfig = async () => {
  try {
    const res = await fetch(`${API_BASE_URL}/events/${eventId}/screensaver`)
    if (res.ok) {
      const data = await res.json()
      if (data.imageId) {
        screensaverPreview.value = getImageUrl(data.imageId)
      }
    }
  } catch (err) {
    console.error('Failed to fetch screensaver config', err)
  }
}

const pushState = async (viewType) => {
  const payload = {
    currentView: viewType,
    customText: localState.value.customText,
    showWelcomeQR: localState.value.showWelcomeQR,
    activeModuleId: screenState.value.activeModuleId
  }
  
  try {
    const res = await fetch(`${API_BASE_URL}/events/${eventId}/screen-state`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify(payload)
    })
    if (res.ok) {
      screenState.value = await res.json()
      success(t('screen_manager.toast_pushed'))
    }
  } catch (err) {
    error('Failed to update screen state')
  }
}

const saveState = async () => {
  // Save local edits without pushing the view if it's not the active one
  // Actually, we'll just update the backend so the Big Screen can pull it if it's already active.
  const payload = {
    ...screenState.value,
    customText: localState.value.customText,
    showWelcomeQR: localState.value.showWelcomeQR
  }
  try {
    const res = await fetch(`${API_BASE_URL}/events/${eventId}/screen-state`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify(payload)
    })
    if (res.ok) {
      screenState.value = await res.json()
    }
  } catch (err) {
    console.error(err)
  }
}

const handleScreensaverUpload = async () => {
  try {
    screensavers.value = await getScreensavers()
    showLibrarySelector.value = true
  } catch (err) {
    error('Failed to load screensavers')
  }
}

const selectScreensaver = async (s) => {
  try {
    const configResponse = await fetch(`${API_BASE_URL}/events/${eventId}/screensaver`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify({ imageId: s.mediaId })
    })

    if (configResponse.ok) {
      screensaverPreview.value = s.thumbnailUrl
      showLibrarySelector.value = false
      success(t('admin.details.toast.screensaver_updated'))
    }
  } catch (err) {
    error('Failed to save screensaver')
  }
}

const removeScreensaver = async () => {
  try {
    const res = await fetch(`${API_BASE_URL}/events/${eventId}/screensaver`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify({ imageId: '', imageUrl: '' })
    })
    if (res.ok) {
      screensaverPreview.value = ''
      success(t('admin.details.toast.screensaver_removed'))
    }
  } catch (err) {
    error('Failed to remove screensaver')
  }
}

const goBack = () => {
  router.push(`/admin/event/${eventId}`)
}

const openBigScreen = () => {
  if (bigscreenWindow.value && !bigscreenWindow.value.closed) {
    bigscreenWindow.value.focus()
  } else {
    bigscreenWindow.value = window.open(`/bigscreen/${eventId}`, 'bigscreen', 'width=1200,height=800')
  }
}

const closeBigScreen = () => {
  if (bigscreenWindow.value && !bigscreenWindow.value.closed) {
    bigscreenWindow.value.close()
    bigscreenWindow.value = null
  }
}

</script>

<style scoped>
.screen-manager {
  min-height: 100vh;
  background-color: #121212;
  color: white;
  padding: 40px;
}
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 30px;
  background-color: #1E1E1E;
  padding: 20px;
  border-radius: 12px;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 20px;
}
.back-btn {
  background: #333;
  color: white;
  border: none;
  padding: 10px 20px;
  border-radius: 8px;
  cursor: pointer;
}
.launch-btn {
  background: #FF9500;
  color: white;
  border: none;
  padding: 10px 20px;
  border-radius: 8px;
  cursor: pointer;
  font-weight: bold;
}
.launch-btn.close-btn {
  background-color: #FF3B30;
}
.launch-btn.close-btn:hover {
  background-color: #FF5C54;
}
.main-content {
  display: flex;
  flex-direction: column;
  gap: 30px;
  max-width: 800px;
  margin: 0 auto;
}
.controls-panel {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.control-card {
  background: #1E1E1E;
  border: 2px solid transparent;
  border-radius: 12px;
  padding: 20px;
  transition: border-color 0.3s;
}
.control-card.active {
  border-color: #4CAF50;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}
.card-header h2 {
  margin: 0;
  font-size: 1.2rem;
}
.push-btn {
  background: #4CAF50;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
}
.card-body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.screensaver-preview {
  width: 100%;
  border-radius: 8px;
  overflow: hidden;
  background: #000;
  margin-bottom: 10px;
  aspect-ratio: 16/9;
}
.screensaver-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
textarea {
  width: 100%;
  background: #2A2A2A;
  color: white;
  border: 1px solid #444;
  padding: 10px;
  border-radius: 6px;
  resize: vertical;
}
.preview-panel {
  background: #1E1E1E;
  border-radius: 12px;
  padding: 20px;
  display: flex;
  flex-direction: column;
}
.preview-container {
  flex: 1;
  background: #000;
  border-radius: 8px;
  margin-top: 15px;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  position: relative;
  aspect-ratio: 16/9;
}
.mini-view {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
}
.mini-welcome {
  background: linear-gradient(135deg, #1A1A1D 0%, #000000 100%);
}
.mini-custom-text {
  background: #2c3e50;
}
.mini-custom-text h2 {
  font-size: 2rem;
  max-width: 80%;
}
.mini-screensaver img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.qr-placeholder-mini {
  width: 100px;
  height: 100px;
  background: white;
  color: black;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 10px 0;
}
.action-button {
  background: #2196F3;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
}
.delete-btn {
  background: #f44336;
  margin-left: 10px;
}
.modal-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0,0,0,0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.modal-content {
  background: #1E1E1E;
  padding: 20px;
  border-radius: 12px;
  width: 90%;
  max-width: 800px;
}
.modal-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 20px;
}
.screensavers-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 15px;
}
.screensaver-option {
  cursor: pointer;
  text-align: center;
}
.screensaver-option img {
  width: 100%;
  aspect-ratio: 16/9;
  object-fit: cover;
  border-radius: 8px;
}
</style>
