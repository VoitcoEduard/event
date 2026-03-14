<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click.self="handleClose">
      <div class="modal-container" :style="modalStyle">
        <div class="header">
          <div>
            <p class="title">{{ $t('quiz_config.question_editor.types.wrong_pictures') }} Config</p>
            <p class="subtitle">{{ module?.name || 'Untitled' }}</p>
          </div>
          <div class="header-buttons">
            <button class="close-button" @click="handleClose">
              ✕
            </button>
          </div>
        </div>

        <div class="content scrollable">
          <div class="rounds-container">
            <div 
              v-for="(question, index) in questions" 
              :key="question.id" 
              class="round-card"
            >
              <div class="round-header">
                <h3>Round {{ index + 1 }}</h3>
                <button class="remove-btn" @click="removeRound(index)">✕</button>
              </div>
              
              <div class="config-grid">
                <div class="input-group full-width">
                  <label>Round Title / Instructions</label>
                  <input type="text" v-model="question.text" class="setting-input" placeholder="e.g. Find 3 historical inaccuracies" />
                </div>
                
                <div class="input-group">
                  <label>{{ $t('quiz_config.question_editor.wrong_pictures.max_points') }}</label>
                  <input type="number" v-model.number="question.wrongPictureConfig.maxPoints" class="setting-input" />
                </div>
                
                <div class="input-group">
                  <label>{{ $t('quiz_config.question_editor.wrong_pictures.tolerance') }}</label>
                  <input type="number" v-model.number="question.wrongPictureConfig.tolerancePercent" class="setting-input" />
                </div>
                
                <div class="input-group">
                  <label>{{ $t('quiz_config.question_editor.time_limit') }}</label>
                  <input type="number" v-model.number="question.timeLimit" class="setting-input" />
                </div>
              </div>

              <!-- Image Upload integration -->
              <div class="image-section">
                <ImageUploader 
                  v-model:imageId="question.imageId"
                  v-model:imageAlt="question.imageAlt"
                  :eventId="eventId"
                  :moduleId="module?.id"
                />
              </div>

              <div v-if="question.imageId" class="coordinate-picker-wrapper mt-3">
                <label>{{ $t('quiz_config.question_editor.wrong_pictures.set_targets') }}</label>
                <CoordinatePicker
                  :image-id="question.imageId"
                  :targets="question.wrongPictureConfig.targets"
                  @update:targets="val => question.wrongPictureConfig.targets = val"
                />
              </div>
              <div v-else class="image-required-warning mt-3">
                ⚠️ {{ $t('quiz_config.question_editor.wrong_pictures.image_required') }}
              </div>
            </div>
          </div>
          
          <div class="add-round-container">
            <button class="add-round-btn" @click="addRound">+ Add Round</button>
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
                    <option value="bigscreen">{{ $t('quiz_config.settings.bigscreen_only') }}</option>
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
                <button class="publish-btn" @click="handleSave" :disabled="loading">
                  {{ loading ? $t('quiz_config.saving') : $t('quiz_config.save_quiz') }}
                </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { API_BASE_URL, getAuthHeaders } from '../utils'
import { useToast } from '../composables/useToast'
import { getTracks } from '../services/media'
import ImageUploader from './ImageUploader.vue'
import CoordinatePicker from './CoordinatePicker.vue'

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
const audioOutput = ref('bigscreen')

const fetchTracks = async () => {
    try {
        const data = await getTracks()
        tracks.value = data
    } catch (e) {
        console.error("Failed to load tracks", e)
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
     audioOutput.value = mod.data.audioOutput || 'bigscreen'
  } else if (visible) {
    questions.value = []
    selectedMusicId.value = ''
    audioOutput.value = 'bigscreen'
  }
  
  if (visible) {
      fetchTracks()
      if (questions.value.length === 0) {
        addRound()
      }
  }
}, { immediate: true })

const addRound = () => {
  questions.value.push({
    id: Date.now().toString() + Math.random().toString(36).substr(2, 5),
    type: 'wrong_pictures',
    text: '',
    timeLimit: 45,
    explainTime: 5,
    wrongPictureConfig: {
      targets: [],
      maxPoints: 1000,
      tolerancePercent: 5
    }
  })
}

const removeRound = (index) => {
  if (!confirm('Remove this round?')) return
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
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid #2C2C2E;
}

.title { font-size: 22px; font-weight: bold; color: #FFF; }
.subtitle { font-size: 14px; color: #8E8E93; margin-top: 2px; }

.header-buttons { display: flex; gap: 10px; align-items: center; }

.close-button {
  padding: 8px;
  border-radius: 8px;
  background-color: #3A3A3C;
  color: #FFF;
  border: none;
  cursor: pointer;
}

.content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
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
}

.config-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 16px;
}

.full-width {
  grid-column: 1 / -1;
}

.input-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.input-group label {
  font-size: 12px;
  color: #8E8E93;
}

.setting-input {
  background-color: #1C1C1E;
  border: 1px solid #3A3A3C;
  color: white;
  padding: 10px;
  border-radius: 8px;
  font-size: 14px;
}

.image-section {
  margin-top: 16px;
}

.coordinate-picker-wrapper {
  background-color: #1C1C1E;
  padding: 16px;
  border-radius: 8px;
  border: 1px solid #3A3A3C;
}

.mt-3 {
  margin-top: 16px;
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
</style>
