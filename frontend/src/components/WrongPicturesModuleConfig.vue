<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click.self="handleClose">
      <div
        ref="modalContainerRef"
        class="modal-container"
        :style="modalStyle"
        role="dialog"
        aria-modal="true"
        :aria-label="$t('quiz_config.question_editor.types.wrong_pictures')"
        tabindex="-1"
        @keydown.escape="handleModalEscape"
      >
        <div class="header">
          <div class="header-title-block">
            <p class="title">{{ $t('quiz_config.question_editor.types.wrong_pictures') }} Config</p>
            <p class="subtitle">{{ module?.name || $t('quiz_config.untitled') }}</p>
          </div>
          <div class="header-buttons">
            <button
              type="button"
              class="icon-button"
              @click="showLibraryModal = true"
              :title="$t('quiz_config.load_library')"
              :aria-label="$t('quiz_config.load_library')"
            >
              <AppIcon name="download" :size="18" aria-hidden="true" />
            </button>
            <button type="button" class="close-button" @click="handleClose" :aria-label="$t('common.close')">
              <AppIcon name="close" :size="18" aria-hidden="true" />
            </button>
          </div>
        </div>

        <div class="content scrollable">
          <div class="rounds-container">
            <div 
              v-for="(question, index) in questions" 
              :key="question.id" 
              class="round-card"
              @click="activeRoundIndex = index"
              @focusin="activeRoundIndex = index"
              @paste="(event) => handleRoundPaste(index, event)"
            >
              <div class="round-header">
                <h3>Round {{ index + 1 }}</h3>
                <button type="button" class="remove-btn" @click="removeRound(index)" :aria-label="$t('quiz_config.question_editor.wrong_pictures.remove_round')"><AppIcon name="close" :size="16" aria-hidden="true" /></button>
              </div>
              
              <div class="config-compact-row">
                <div class="input-group flex-grow">
                  <label>{{ $t('quiz_config.question_editor.wrong_pictures.instructions_label') }}</label>
                  <input 
                    type="text" 
                    v-model="question.text" 
                    class="setting-input" 
                    :placeholder="$t('quiz_config.question_editor.text_placeholder')"
                  />
                </div>
                
                <div class="input-group mini">
                  <label>{{ $t('quiz_config.question_editor.wrong_pictures.points_short') }}</label>
                  <input type="number" v-model.number="question.wrongPictureConfig.maxPoints" class="setting-input" />
                </div>
                
                <div class="input-group mini">
                  <label>{{ $t('quiz_config.question_editor.wrong_pictures.tolerance_short') }}</label>
                  <input type="number" v-model.number="question.wrongPictureConfig.tolerancePercent" class="setting-input" />
                </div>
                
                <div class="input-group mini">
                  <label>{{ $t('quiz_config.question_editor.wrong_pictures.time_short') }}</label>
                  <input type="number" v-model.number="question.timeLimit" class="setting-input" />
                </div>
                
                <div class="input-group mini center">
                  <label>{{ $t('quiz_config.question_editor.wrong_pictures.show_count') }}</label>
                  <label class="switch small">
                    <input type="checkbox" v-model="question.wrongPictureConfig.showTargetCount">
                    <span class="slider round"></span>
                  </label>
                </div>
              </div>

              <div class="image-management-section">
                <ImageUploader 
                  :ref="(el) => setRoundUploaderRef(index, el)"
                  v-model="question.imageId"
                  v-model:altText="question.imageAlt"
                  :eventId="eventId"
                  :moduleId="module?.id"
                  :hidePreview="true"
                />

                <div v-if="question.imageId" class="coordinate-picker-container">
                  <CoordinatePicker
                    :image-id="question.imageId"
                    :targets="question.wrongPictureConfig.targets"
                    @update:targets="val => question.wrongPictureConfig.targets = val"
                  />
                </div>
                <div v-else class="image-required-warning">
                  <AppIcon name="alert" :size="16" aria-hidden="true" /> {{ $t('quiz_config.question_editor.wrong_pictures.image_required') }}
                </div>
              </div>
            </div>
          </div>
          
          <div class="add-round-container">
            <button type="button" class="add-round-btn" @click="addRound">{{ $t('quiz_config.question_editor.wrong_pictures.add_round') }}</button>
          </div>

          <div class="settings-bar mt-auto">
            <div class="setting-item">
                <label>{{ $t('quiz_config.settings.bg_music') }}</label>
                <select v-model="selectedMusicId" class="setting-select">
                    <option value="">{{ $t('quiz_config.settings.none') }}</option>
                    <option v-for="track in tracks" :key="track.id" :value="track.id">
                        {{ track.name }}
                    </option>
                </select>
            </div>
            <div class="setting-item">
                <label>{{ $t('quiz_config.settings.audio_output') }}</label>
                <select v-model="audioOutput" class="setting-select">
                    <option value="projector">{{ $t('quiz_config.settings.projector_only') }}</option>
                    <option value="admin">{{ $t('quiz_config.settings.admin_only') }}</option>
                    <option value="both">{{ $t('quiz_config.settings.both') }}</option>
                </select>
            </div>
          </div>

          <div class="action-bar">
            <div class="status-text">
              {{ questions.length }} Rounds
            </div>
            <div class="action-buttons">
                <button type="button" class="save-template-btn" @click="saveAsTemplate" :disabled="questions.length === 0">
                    {{ $t('quiz_config.save_template') }}
                </button>
                <button type="button" class="publish-btn" @click="handleSave" :disabled="loading">
                  {{ loading ? $t('quiz_config.saving') : $t('quiz_config.save_quiz') }}
                </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Teleport>

  <!-- Library Modal -->
  <Teleport to="body">
    <div v-if="showLibraryModal" class="library-modal-overlay" @click.self="showLibraryModal = false">
      <div
        ref="libraryModalRef"
        class="library-modal"
        role="dialog"
        aria-modal="true"
        :aria-label="$t('quiz_config.select_template')"
        tabindex="-1"
        @keydown.escape.stop.prevent="showLibraryModal = false"
      >
          <div class="library-header">
              <h3>{{ $t('quiz_config.select_template') }}</h3>
              <button type="button" class="close-btn" @click="showLibraryModal = false" :aria-label="$t('common.close')"><AppIcon name="close" :size="18" aria-hidden="true" /></button>
          </div>
          <div class="library-content">
              <div v-if="loadingTemplates" class="loading">{{ $t('common.loading') }}</div>
              <div v-else-if="templates.length === 0" class="empty">{{ $t('quiz_config.no_templates') }}</div>
              <div v-else class="template-list">
                  <div v-for="t in templates" :key="t.id" class="template-item" @click="loadTemplate(t)">
                      <div class="template-info">
                          <span class="t-name">{{ t.name }}</span>
                          <span class="t-count">{{ t.data?.questions?.length || 0 }} Rounds</span>
                      </div>
                      <p class="t-desc">{{ t.description }}</p>
                  </div>
              </div>
          </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, watch, computed, nextTick, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { API_BASE_URL, getAuthHeaders, normalizeTemplates } from '../utils'
import { useToast } from '../composables/useToast'
import { getTracks } from '../services/media'
import { hasActiveBlockingModal, isPasteTargetInContainer } from '../utils/clipboard'
import ImageUploader from './ImageUploader.vue'
import CoordinatePicker from './CoordinatePicker.vue'
import AppIcon from './AppIcon.vue'

const { t } = useI18n()
const props = defineProps({
  visible: { type: Boolean, default: false },
  module: { type: Object, default: null },
  eventId: { type: String, default: '' }
})

const emit = defineEmits(['close', 'update'])
const { success, error } = useToast()

const questions = ref([])
const loading = ref(false)
const tracks = ref([])
const selectedMusicId = ref('')
const audioOutput = ref('projector')

const showLibraryModal = ref(false)
const templates = ref([])
const loadingTemplates = ref(false)
const modalContainerRef = ref(null)
const libraryModalRef = ref(null)
const activeRoundIndex = ref(0)
const roundUploaderRefs = new Map()

const setRoundUploaderRef = (index, el) => {
  if (el) {
    roundUploaderRefs.set(index, el)
  } else {
    roundUploaderRefs.delete(index)
  }
}

const handleRoundPaste = (index, event) => {
  if (event?.defaultPrevented) return false
  return roundUploaderRefs.get(index)?.handleClipboardPasteEvent?.(event) ?? false
}

const handleWindowPaste = (event) => {
  if (!props.visible || showLibraryModal.value || event?.defaultPrevented) return
  if (hasActiveBlockingModal(modalContainerRef.value)) return
  if (!isPasteTargetInContainer(event?.target, modalContainerRef.value)) return

  const idx = activeRoundIndex.value >= 0 && activeRoundIndex.value < questions.value.length
    ? activeRoundIndex.value
    : (questions.value.length === 1 ? 0 : -1)
  if (idx >= 0) {
    handleRoundPaste(idx, event)
  }
}

let previouslyFocusedEl = null
let previousLibraryTriggerEl = null

const handleModalEscape = (event) => {
  if (!props.visible || event.defaultPrevented) return
  if (showLibraryModal.value) {
    event.preventDefault()
    event.stopPropagation()
    showLibraryModal.value = false
    return
  }
  event.preventDefault()
  event.stopPropagation()
  handleClose()
}

const handleGlobalKeydown = (event) => {
  if (event.key !== 'Escape' || !props.visible || event.defaultPrevented) return
  if (showLibraryModal.value) {
    event.preventDefault()
    showLibraryModal.value = false
    return
  }
  event.preventDefault()
  handleClose()
}

watch(() => props.visible, (isVisible, wasVisible) => {
  if (typeof window !== 'undefined') {
    if (isVisible) {
      window.addEventListener('keydown', handleGlobalKeydown)
      window.addEventListener('paste', handleWindowPaste)
    } else {
      window.removeEventListener('keydown', handleGlobalKeydown)
      window.removeEventListener('paste', handleWindowPaste)
    }
  }
  if (typeof document === 'undefined') return
  if (isVisible && !wasVisible) {
    previouslyFocusedEl = document.activeElement
    nextTick(() => {
      if (modalContainerRef.value && !modalContainerRef.value.contains(document.activeElement)) {
        modalContainerRef.value.focus?.()
      }
    })
  } else if (!isVisible && wasVisible) {
    showLibraryModal.value = false
    const target = previouslyFocusedEl
    previouslyFocusedEl = null
    if (target && typeof target.focus === 'function') {
      nextTick(() => {
        if (target.isConnected) target.focus()
      })
    }
  }
}, { immediate: true })

watch(showLibraryModal, (isOpen, wasOpen) => {
  if (typeof document === 'undefined') return
  if (isOpen && !wasOpen) {
    previousLibraryTriggerEl = document.activeElement
    nextTick(() => {
      libraryModalRef.value?.focus?.()
    })
  } else if (!isOpen && wasOpen) {
    const target = previousLibraryTriggerEl
    previousLibraryTriggerEl = null
    if (props.visible && target && typeof target.focus === 'function') {
      nextTick(() => {
        if (target.isConnected) target.focus()
      })
    }
  }
})

onUnmounted(() => {
  if (typeof window !== 'undefined') {
    window.removeEventListener('keydown', handleGlobalKeydown)
    window.removeEventListener('paste', handleWindowPaste)
  }
})

const fetchTemplates = async () => {
    loadingTemplates.value = true
    try {
        const res = await fetch(`${API_BASE_URL}/templates`)
        if(res.ok) {
            const data = normalizeTemplates(await res.json())
            templates.value = data.filter(t => t.moduleType === 'wrong_pictures')
        }
    } catch(e) {
        console.error(e)
    } finally {
        loadingTemplates.value = false
    }
}

const fetchTracks = async () => {
    try {
        const data = await getTracks()
        tracks.value = data
    } catch (e) {
        console.error("Failed to load tracks", e)
    }
}

const addRound = () => {
  questions.value.push({
    id: Date.now().toString() + Math.random().toString(36).substr(2, 5),
    type: 'wrong_pictures',
    text: '',
    timeLimit: 60,
    explainTime: 5,
    wrongPictureConfig: {
      targets: [],
      maxPoints: 50,
      tolerancePercent: 5,
      showTargetCount: false
    }
  })
}

const loadTemplate = (template) => {
    if(questions.value.length > 0) {
        if(!confirm(t('quiz_config.confirm_replace'))) return
    }
    
    if(template.data?.questions) {
        let loadedQuestions = JSON.parse(JSON.stringify(template.data.questions))
        // Hydrate and regenerate IDs
        questions.value = loadedQuestions.map(q => {
            q.id = Date.now().toString() + Math.random().toString(36).substr(2, 5)
            // Keep imageId - library templates share the same database so images still exist
            return q
        })
        success(t('quiz_config.loaded_template', { name: template.name }))
        showLibraryModal.value = false
    }
}

const saveAsTemplate = async () => {
    const name = prompt(t('quiz_config.enter_template_name'))
    if(!name) return

    try {
        const payload = {
            name,
            description: `Created from event ${props.eventId}`,
            moduleType: 'wrong_pictures',
            data: { questions: questions.value }
        }
        
        const res = await fetch(`${API_BASE_URL}/templates`, {
            method: 'POST',
            headers: {'Content-Type': 'application/json', ...getAuthHeaders()},
            body: JSON.stringify(payload)
        })
        
        if(res.ok) {
            success(t('quiz_config.saved_template'))
            fetchTemplates()
        } else {
            error(t('quiz_config.save_failed'))
        }
    } catch(e) {
        console.error(e)
        error(t('quiz_config.save_failed'))
    }
}

watch(() => [props.visible, props.module], ([visible, mod]) => {
  if (visible && mod?.data) {
     if (mod.data.questions) {
        // Hydrate questions array
        questions.value = JSON.parse(JSON.stringify(mod.data.questions))
     } else {
         questions.value = []
     }
     selectedMusicId.value = mod.data.backgroundMusicId || ''
     audioOutput.value = mod.data.audioOutput || 'projector'
  } else if (visible) {
    questions.value = []
    selectedMusicId.value = ''
    audioOutput.value = 'projector'
  }
  
  if (visible) {
      fetchTemplates()
      fetchTracks()
      if (questions.value.length === 0) {
        addRound()
      }
  }
}, { immediate: true })


const removeRound = (index) => {
  if (!confirm(t('quiz_config.question_editor.wrong_pictures.remove_round_confirm'))) return
  questions.value.splice(index, 1)
}

const handleSave = async () => {
  if (questions.value.length === 0) {
    error('Please add at least one round.')
    return
  }

  for (const q of questions.value) {
    if (!q.text.trim()) {
      error('Round title/instructions are required.')
      return
    }
    if (!q.imageId) {
      error('Each round must have an image.')
      return
    }
    if (q.wrongPictureConfig.targets.length === 0) {
      error('Each round must have at least one target location defined.')
      return
    }
  }

  loading.value = true
  try {
    const questionsResponse = await fetch(`${API_BASE_URL}/events/${props.eventId}/modules/${props.module.id}/quiz/questions`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify(questions.value)
    })
    
    if (!questionsResponse.ok) throw new Error('Failed to save rounds')

    let musicUrl = ''
    if (selectedMusicId.value) {
        const track = tracks.value.find(t => t.id === selectedMusicId.value)
        if (track) musicUrl = track.url
    }

    const configUpdate = {
        data: {
            questions: questions.value,
            backgroundMusicId: selectedMusicId.value,
            backgroundMusicUrl: musicUrl,
            audioOutput: audioOutput.value
        }
    }

    const moduleResponse = await fetch(`${API_BASE_URL}/events/${props.eventId}/modules/${props.module.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
        body: JSON.stringify(configUpdate)
    })

    if (!moduleResponse.ok) throw new Error('Failed to save module config')

    success('Wrong Pictures module saved successfully!')
    emit('update')
    handleClose()

  } catch (err) {
    console.error('Error saving module:', err)
    error(t('quiz_config.save_error'))
  } finally {
    loading.value = false
  }
}

const handleClose = () => {
  emit('close')
}

const modalStyle = computed(() => ({
  transform: `translateY(${props.visible ? '0' : '100%'})`,
  opacity: props.visible ? 1 : 0,
  transition: 'all 0.3s ease'
}))
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: flex-end;
  z-index: 1000;
}

.modal-container {
  background-color: #1C1C1E;
  height: 92%;
  width: 100%;
  max-width: 800px;
  border-top-left-radius: 20px;
  border-top-right-radius: 20px;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
  overscroll-behavior: contain;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid #2C2C2E;
  gap: 12px;
}

.header-title-block {
  min-inline-size: 0;
  flex: 1 1 auto;
}

.title { font-size: 22px; font-weight: bold; color: #FFF; overflow-wrap: anywhere; }
.subtitle { font-size: 14px; color: #8E8E93; margin-top: 2px; overflow-wrap: anywhere; }

.header-buttons { display: flex; gap: 10px; align-items: center; }

.close-button {
  padding: 8px;
  border-radius: 8px;
  background-color: #3A3A3C;
  color: #FFF;
  border: none;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.content {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  overscroll-behavior: contain;
}

.rounds-container {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.round-card {
  background: #2C2C2E;
  border-radius: 12px;
  padding: 20px;
  border: 1px solid #444;
  min-inline-size: 0;
  box-sizing: border-box;
}

.round-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.round-header h3 {
  margin: 0;
  color: #FFF;
  font-size: 18px;
}

.remove-btn {
  background: none;
  border: none;
  color: #FF453A;
  cursor: pointer;
  font-size: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.config-compact-row {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
  align-items: flex-end;
}

.input-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.flex-grow { flex: 1; }
.mini { width: 80px; }
.center { align-items: center; }

.input-group label {
  font-size: 11px;
  color: #8E8E93;
  font-weight: 500;
  white-space: nowrap;
}

.setting-input {
  background-color: #1C1C1E;
  border: 1px solid #3A3A3C;
  color: white;
  padding: 8px 10px;
  border-radius: 6px;
  font-size: 13px;
  width: 100%;
  box-sizing: border-box;
}

.image-management-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.coordinate-picker-container {
  border-top: 1px solid #3A3A3C;
  padding-top: 12px;
}

.mt-auto {
  margin-top: auto;
}

.add-round-container {
  padding: 0 20px 20px;
  display: flex;
  justify-content: center;
}

.add-round-btn {
  background: #3A3A3C;
  color: white;
  border: 1px dashed #666;
  padding: 12px 24px;
  border-radius: 8px;
  cursor: pointer;
  width: 100%;
  font-weight: 600;
}

.add-round-btn:hover {
  background: #4A4A4C;
}

.settings-bar, .action-bar {
  background-color: #1C1C1E;
  border-top: 1px solid #2C2C2E;
  padding: 16px 20px;
}

.settings-bar {
  display: flex;
  gap: 20px;
}

.setting-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
}

.setting-item label {
  font-size: 12px;
  color: #8E8E93;
}

.setting-select {
  background-color: #2C2C2E;
  border: 1px solid #3A3A3C;
  color: white;
  padding: 8px;
  border-radius: 8px;
  font-size: 14px;
}

.action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.status-text { color: #8E8E93; }

.action-buttons {
  display: flex;
  gap: 12px;
}

.publish-btn {
  background-color: #0A84FF;
  color: white;
  padding: 10px 20px;
  border-radius: 8px;
  font-weight: 600;
  border: none;
  cursor: pointer;
}
.publish-btn:disabled { opacity: 0.5; }

.image-required-warning {
  color: #FF9F0A;
  background: rgba(255, 159, 10, 0.1);
  padding: 12px;
  border-radius: 8px;
  font-size: 14px;
  border: 1px solid rgba(255, 159, 10, 0.3);
}

/* Switch Styles */
.switch.small {
  width: 36px;
  height: 20px;
  margin-top: 0;
}

.switch.small .slider:before {
  height: 14px;
  width: 14px;
  left: 3px;
  bottom: 3px;
}

.switch.small input:checked + .slider:before {
  transform: translateX(16px);
}

.switch {
  position: relative;
  display: inline-block;
  width: 44px;
  height: 24px;
  margin-top: 4px;
}

.switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: #3A3A3C;
  transition: .4s;
}

.slider:before {
  position: absolute;
  content: "";
  height: 18px;
  width: 18px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  transition: .4s;
}

input:checked + .slider {
  background-color: #0A84FF;
}

input:focus + .slider {
  box-shadow: 0 0 1px #0A84FF;
}

input:checked + .slider:before {
  transform: translateX(20px);
}

.slider.round {
  border-radius: 24px;
}

.slider.round:before {
  border-radius: 50%;
}

.icon-button {
  padding: 8px;
  border-radius: 8px;
  background-color: #2C2C2E;
  font-size: 20px;
  border: 1px solid #333;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}
.icon-button:hover { background-color: #333; }

.save-template-btn {
    background: none;
    border: 1px solid #333;
    color: #ccc;
    padding: 10px 16px;
    border-radius: 8px;
    font-weight: 600;
    cursor: pointer;
}
.save-template-btn:hover { background-color: #333; color: white; }
.save-template-btn:disabled { opacity: 0.5; cursor: not-allowed; }

/* Library Modal */
.library-modal-overlay {
    position: fixed;
    top: 0; left: 0; right: 0; bottom: 0;
    background-color: rgba(0,0,0,0.8);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 2000;
}

.library-modal {
    background-color: #2C2C2E;
    width: 90%;
    max-width: 500px;
    max-height: 80%;
    border-radius: 12px;
    display: flex;
    flex-direction: column;
    box-shadow: 0 4px 20px rgba(0,0,0,0.5);
    box-sizing: border-box;
    overscroll-behavior: contain;
}

.library-header {
    padding: 16px;
    border-bottom: 1px solid #3A3A3C;
    display: flex;
    justify-content: space-between;
    align-items: center;
}
.library-header h3 { color: white; margin: 0; font-size: 18px; }

.library-content {
    flex: 1;
    overflow-y: auto;
    padding: 16px;
    overscroll-behavior: contain;
}

.close-btn {
  background: none;
  border: none;
  color: #888;
  font-size: 20px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.template-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
}

.template-item {
    background-color: #1C1C1E;
    padding: 12px;
    border-radius: 8px;
    cursor: pointer;
    border: 1px solid transparent;
}
.template-item:hover { border-color: #0A84FF; }

.template-info { display: flex; justify-content: space-between; margin-bottom: 4px; gap: 8px; }
.t-name { color: white; font-weight: 600; }
.t-count { color: #888; font-size: 12px; flex-shrink: 0; }
.t-desc { color: #aaa; font-size: 12px; margin: 0; }

@media (max-width: 768px) {
  .modal-container {
    height: 100vh;
    height: 100dvh;
    max-height: 100dvh;
    border-radius: 0;
  }

  .header {
    flex-wrap: wrap;
    gap: 10px;
    padding: max(12px, env(safe-area-inset-top, 0px)) max(16px, env(safe-area-inset-right, 0px)) 12px max(16px, env(safe-area-inset-left, 0px));
  }

  .title {
    font-size: 18px;
  }

  .header-buttons {
    flex-wrap: wrap;
    gap: 8px;
    margin-inline-start: auto;
  }

  .settings-bar {
    flex-direction: column;
    gap: 12px;
    padding: 12px max(16px, env(safe-area-inset-right, 0px)) 12px max(16px, env(safe-area-inset-left, 0px));
  }

  .setting-item {
    min-inline-size: 0;
  }

  .setting-select {
    min-inline-size: 0;
    width: 100%;
    box-sizing: border-box;
    font-size: 16px;
  }

  .action-bar {
    flex-direction: column;
    align-items: stretch;
    gap: 10px;
    padding: 12px max(16px, env(safe-area-inset-right, 0px)) max(12px, env(safe-area-inset-bottom, 0px)) max(16px, env(safe-area-inset-left, 0px));
  }

  .action-buttons {
    flex-wrap: wrap;
    gap: 8px;
  }

  .save-template-btn {
    flex: 1 1 140px;
    min-inline-size: 0;
    text-align: center;
  }

  .publish-btn {
    flex: 1 0 100%;
    width: 100%;
  }

  .library-modal {
    width: calc(100% - 24px);
    max-height: 85vh;
    max-height: 85dvh;
  }
}

@media (max-width: 640px) {
  .rounds-container {
    padding: 12px;
    gap: 14px;
  }

  .round-card {
    padding: 14px;
  }

  .add-round-container {
    padding: 0 12px 16px;
  }

  .config-compact-row {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
    align-items: end;
  }

  .config-compact-row .flex-grow {
    grid-column: 1 / -1;
    min-inline-size: 0;
  }

  .config-compact-row .mini {
    width: auto;
    min-inline-size: 0;
  }

  .config-compact-row .mini.center {
    align-items: flex-start;
  }

  .input-group label {
    white-space: normal;
    overflow-wrap: break-word;
  }

  .setting-input {
    font-size: 16px;
  }
}

@media (pointer: coarse) {
  .icon-button,
  .close-button,
  .remove-btn,
  .close-btn {
    min-block-size: var(--target-size, 44px);
    min-inline-size: var(--target-size, 44px);
  }

  .add-round-btn,
  .save-template-btn,
  .publish-btn,
  .setting-select,
  .setting-input,
  .template-item {
    min-block-size: var(--target-size, 44px);
  }

  .switch.small::after {
    content: '';
    position: absolute;
    inset: -12px -4px;
  }
}
</style>
