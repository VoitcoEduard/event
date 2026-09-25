<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click.self="handleClose">
      <div class="modal-container" :style="modalStyle">
        <div class="header">
          <div>
            <p class="title">{{ $t('quiz_config.title') }}</p>
            <p class="subtitle">{{ module?.name || $t('quiz_config.untitled') }}</p>
          </div>
          <div class="header-buttons">
            <button class="icon-button" @click="showLibraryModal = true" :title="$t('quiz_config.load_library')">
              📥
            </button>
            <button class="library-btn" @click="triggerFileInput">
              📄 {{ $t('quiz_config.load_json') }}
            </button>
            <input 
              type="file" 
              ref="fileInput" 
              style="display: none" 
              accept=".json"
              @change="handleFileUpload"
            />
            <button class="close-button" @click="handleClose">
              ✕
            </button>
          </div>
        </div>

        <div class="content">
          <QuizEditor 
            :questions="questions"
            :event-id="eventId"
            :module-id="module?.id || ''"
            @update-question="updateQuestion"
            @remove-question="removeQuestion"
            @add-question="(type) => addQuestion(type)"
            @move-question="moveQuestion"
          />

          <div class="settings-bar">
            <div class="setting-item">
                <label>{{ $t('quiz_config.settings.bg_music') }}</label>
                <select v-model="selectedMusicId" @change="handleMusicChange" class="setting-select">
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
              {{ $t('quiz_config.status', { count: questions.length }) }}
            </div>
            <div class="action-buttons">
                <button class="save-template-btn" @click="exportJSON" :disabled="questions.length === 0">
                    {{ $t('quiz_config.export_json') }}
                </button>
                <button class="save-template-btn" @click="saveAsTemplate" :disabled="questions.length === 0">
                    {{ $t('quiz_config.save_template') }}
                </button>
                <button class="publish-btn" @click="handleSave" :disabled="loading">
                {{ loading ? $t('quiz_config.saving') : $t('quiz_config.save_quiz') }}
                </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Library Modal -->
      <div v-if="showLibraryModal" class="library-modal-overlay" @click.self="showLibraryModal = false">
        <div class="library-modal">
            <div class="library-header">
                <h3>{{ $t('quiz_config.select_template') }}</h3>
                <button class="close-btn" @click="showLibraryModal = false">✕</button>
            </div>
            <div class="library-content">
                <div v-if="loadingTemplates" class="loading">{{ $t('common.loading') }}</div>
                <div v-else-if="templates.length === 0" class="empty">{{ $t('quiz_config.no_templates') }}</div>
                <div v-else class="template-list">
                    <div v-for="t in templates" :key="t.id" class="template-item" @click="loadTemplate(t)">
                        <div class="template-info">
                            <span class="t-name">{{ t.name }}</span>
                            <span class="t-count">{{ $t('quiz_config.questions_count', { count: (t.data?.questions || []).length }) }}</span>
                        </div>
                        <p class="t-desc">{{ t.description }}</p>
                    </div>
                </div>
            </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, watch, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { API_BASE_URL, getAuthHeaders, normalizeTemplates } from '../utils'
import { useToast } from '../composables/useToast'
import QuizEditor from './QuizEditor.vue'
import { getTracks } from '../services/media'

const { t } = useI18n()
const props = defineProps({
  visible: { type: Boolean, default: false },
  module: { type: Object, default: null },
  eventId: { type: String, default: '' }
})

const emit = defineEmits(['close', 'update'])
const { success, error, info } = useToast()

const questions = ref([])
const loading = ref(false)
const showLibraryModal = ref(false)
const templates = ref([])
const loadingTemplates = ref(false)
const fileInput = ref(null)

const tracks = ref([])
const selectedMusicId = ref('')
const audioOutput = ref('projector')

// Load existing data when module opens
const fetchTemplates = async () => {
    loadingTemplates.value = true
    try {
        const res = await fetch(`${API_BASE_URL}/templates`)
        if(res.ok) {
            const data = normalizeTemplates(await res.json())
            templates.value = data.filter(t => t.moduleType === 'quiz' || (!t.moduleType && t.data?.questions))
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

watch(() => [props.visible, props.module], ([visible, mod]) => {
  if (visible && mod?.data) {
     if (mod.data.questions) {
        let loadedQuestions = JSON.parse(JSON.stringify(mod.data.questions))
        questions.value = loadedQuestions.map(q => {
            if (q.type === 'range') {
                if (!q.rangeConfig) q.rangeConfig = { minValue: 0, maxValue: 100, step: 1, unit: '', decimalPlaces: 0 }
                if (!q.scoringConfig) q.scoringConfig = { mode: 'symmetric', maxPoints: 100, tolerance: 10, partialPercent: 0 }
            }
            if (q.type === 'true_false') {
                 q.answers = ['True', 'False', '', '']
                 if (q.correctIndex > 1) q.correctIndex = 0
            }
            if (q.type === 'number') {
                if (q.correctNumber === undefined) q.correctNumber = 0
                if (!q.scoringConfig) q.scoringConfig = { mode: 'symmetric', maxPoints: 1000, tolerance: 0, partialPercent: 0 }
            }
            if (q.explainTime === undefined) q.explainTime = 5
            return q
        })
     } else {
         questions.value = []
     }
     
     // Load music settings
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
  }
}, { immediate: true })

const handleMusicChange = () => {
    // Logic if needed when music changes
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
            // Ensure slider config exists
            if (q.type === 'range' && !q.rangeConfig) {
                q.rangeConfig = { minValue: 0, maxValue: 100, step: 1, unit: '', decimalPlaces: 0 }
            }
            if (q.type === 'range' && !q.scoringConfig) {
                q.scoringConfig = { mode: 'symmetric', maxPoints: 100, tolerance: 10, partialPercent: 0 }
            }
            if (q.type === 'true_false') {
                 q.answers = ['True', 'False', '', '']
                 if (q.correctIndex > 1) q.correctIndex = 0
            }
            if (q.type === 'number') {
                if (q.correctNumber === undefined) q.correctNumber = 0
                if (!q.scoringConfig) q.scoringConfig = { mode: 'symmetric', maxPoints: 1000, tolerance: 0, partialPercent: 0 }
            }
            if (q.explainTime === undefined) q.explainTime = 5
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
            moduleType: 'quiz',
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

const addQuestion = (type = 'classic') => {
  const newQuestion = {
    id: Date.now().toString(),
    type: type,
    text: '',
    answers: ['', '', '', ''],
    correctIndex: 0,
    timeLimit: 20,
    explainTime: 5
  }

  // Add type-specific defaults
  if (type === 'range') {
    newQuestion.rangeConfig = { minValue: 0, maxValue: 100, step: 1, unit: '', decimalPlaces: 0 }
    newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 100, tolerance: 10, partialPercent: 0 }
    newQuestion.correctValue = 0
  } else if (type === 'number') {
    newQuestion.correctNumber = 0
    newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 1000, tolerance: 0, partialPercent: 0 }
  } else if (type === 'simple_text') {
    newQuestion.timeLimit = 0
    newQuestion.explainTime = 0
    newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 100, tolerance: 0, partialPercent: 0 }
    newQuestion.judgeNotes = ''
    newQuestion.correctAnswerText = ''
  } else if (type === 'correct_picture') {
    newQuestion.imageOptions = [
      { id: Date.now().toString(), imageId: '', label: '' },
      { id: (Date.now() + 1).toString(), imageId: '', label: '' }
    ]
    // A7: choice scoring reads scoringConfig.maxPoints.
    newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 1000, tolerance: 0, partialPercent: 0 }
  } else if (type === 'wrong_pictures') {
    newQuestion.wrongPictureConfig = { targets: [], tolerancePercent: 10, maxPoints: 1000 }
  } else if (type === 'true_false') {
    newQuestion.answers = [t('quiz.player.true'), t('quiz.player.false'), '', '']
    // A7: choice scoring reads scoringConfig.maxPoints.
    newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 1000, tolerance: 0, partialPercent: 0 }
  } else if (type === 'multi_select' || type === 'classic') {
    // A7: choice scoring reads scoringConfig.maxPoints.
    newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 1000, tolerance: 0, partialPercent: 0 }
  }

  questions.value.push(newQuestion)
}

const removeQuestion = (index) => {
  if (!confirm(t('quiz_config.delete_question'))) return
  questions.value.splice(index, 1)
}

const updateQuestion = (index, updatedQ) => {
  questions.value[index] = updatedQ
}

const moveQuestion = (index, direction) => {
  const newIndex = index + direction
  if (newIndex < 0 || newIndex >= questions.value.length) return
  
  const temp = questions.value[index]
  questions.value[index] = questions.value[newIndex]
  questions.value[newIndex] = temp
}

const handleSave = async () => {
  const validationError = validateQuestions()
  if (validationError) {
    error(validationError)
    return
  }

  loading.value = true
  try {
    // 1. Save Questions
    const questionsResponse = await fetch(`${API_BASE_URL}/events/${props.eventId}/modules/${props.module.id}/quiz/questions`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify(questions.value)
    })
    
    if (!questionsResponse.ok) throw new Error('Failed to save questions')

    // 2. Save Music Config
    
    // Find URL from selected ID
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

    success(t('quiz_config.saved_success'))
    emit('update')
    handleClose()

  } catch (err) {
    console.error('Error saving quiz:', err)
    error(t('quiz_config.save_error'))
  } finally {
    loading.value = false
  }
}

const validateQuestions = () => {
  if (questions.value.length === 0) return t('quiz_config.validation.min_one')
  
  for (const q of questions.value) {
    if (!q.text.trim()) return t('quiz_config.validation.text_required')
    
    if (q.type === 'range') {
        const config = q.rangeConfig
        if (!config) return t('quiz_config.validation.slider_config')
        if (config.minValue >= config.maxValue) return t('quiz_config.validation.slider_min_max')
        if (config.step <= 0) return t('quiz_config.validation.slider_step')
    } else if (q.type === 'true_false') {
      // Only check first 2 answers for T/F
      if (q.answers.slice(0, 2).some(a => !a.trim())) return t('quiz_config.validation.tf_options')
    } else if (q.type === 'correct_picture') {
        if (!q.imageOptions || q.imageOptions.length < 2) return t('quiz_config.validation.cp_options')
    } else if (q.type === 'number') {
        if (q.correctNumber === undefined || q.correctNumber === null) return t('quiz_config.validation.number_required')
    } else if (q.type === 'simple_text') {
        if (!q.text.trim() && !q.imageId) return t('quiz_config.validation.simple_text_required')
        if ((q.scoringConfig?.maxPoints ?? 0) <= 0) return t('quiz_config.validation.simple_points')
    } else {
      // For standard types (classic, multi_select), ensure all 4 slots are filled
      if (q.answers.some(a => !a.trim())) return t('quiz_config.validation.mc_options')
    }
  }
  return null
}

const handleClose = () => {
  emit('close')
}

const triggerFileInput = () => {
    fileInput.value.click()
}

const handleFileUpload = (event) => {
    const file = event.target.files[0]
    if (!file) return

    const reader = new FileReader()
    reader.onload = (e) => {
        try {
            const content = JSON.parse(e.target.result)
            
            let loadedQuestions = []
            // Handle Template format
            if (content.data && content.data.questions) {
                loadedQuestions = content.data.questions
            } 
            // Handle Raw Questions Array
            else if (Array.isArray(content)) {
                loadedQuestions = content
            } else {
                error('Invalid quiz file format')
                return
            }

            if(loadedQuestions.length > 0) {
                if(questions.value.length > 0) {
                    if(!confirm(t('quiz_config.confirm_replace'))) return
                }
                questions.value = loadedQuestions.map(q => {
                    q.id = Date.now().toString() + Math.random().toString(36).substr(2, 5)
                    // Clear imageId since images from imported files won't exist in this database
                    if (q.imageId) {
                        q.imageId = null
                        q.imageAlt = ''
                        info(t('quiz_config.import.note_images'))
                    }
                     // Ensure slider config exists
                    if (q.type === 'range' && !q.rangeConfig) {
                        q.rangeConfig = { minValue: 0, maxValue: 100, step: 1, unit: '', decimalPlaces: 0 }
                    }
                    if (q.type === 'range' && !q.scoringConfig) {
                        q.scoringConfig = { mode: 'symmetric', maxPoints: 100, tolerance: 10, partialPercent: 0 }
                    }
                    if (q.type === 'true_false') {
                         q.answers = ['True', 'False', '', '']
                         if (q.correctIndex > 1) q.correctIndex = 0
                    }
                    if (q.type === 'number') {
                        if (q.correctNumber === undefined) q.correctNumber = 0
                        if (!q.scoringConfig) q.scoringConfig = { mode: 'symmetric', maxPoints: 1000, tolerance: 0, partialPercent: 0 }
                    }
                    if (q.explainTime === undefined) q.explainTime = 5
                    return q
                })
                success(t('quiz_config.import.success'))
            } else {
                info(t('quiz_config.import.no_questions'))
            }
        } catch (err) {
            console.error(err)
            error(t('quiz_config.import.error_parse'))
        } finally {
            event.target.value = ''
        }
    }
    reader.readAsText(file)
}

const exportJSON = () => {
    // Export as a Template-like structure for better compatibility
    const payload = {
        name: props.module?.name || 'Quiz Export',
        description: `Exported from event ${props.eventId}`,
        moduleType: 'quiz',
        data: { questions: questions.value }
    }
    
    const dataStr = JSON.stringify(payload, null, 2)
    const blob = new Blob([dataStr], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    
    const a = document.createElement('a')
    a.href = url
    a.download = `${(props.module?.name || 'quiz').replace(/[^a-z0-9]/gi, '_').toLowerCase()}.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
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

.icon-button {
  padding: 8px;
  border-radius: 8px;
  background-color: #2C2C2E;
  font-size: 20px;
  border: 1px solid #333;
  cursor: pointer;
}
.icon-button:hover { background-color: #333; }

.library-btn {
    background-color: #333;
    color: white;
    border: 1px solid #444;
    padding: 6px 12px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 13px;
    font-weight: 600;
}
.library-btn:hover { background-color: #444; }

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
  overflow: hidden;
  position: relative;
}

.action-bar {
  background-color: #1C1C1E;
  border-top: 1px solid #2C2C2E;
  padding: 16px 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.settings-bar {
    background-color: #1C1C1E;
    border-top: 1px solid #2C2C2E;
    padding: 16px 20px;
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

.status-text { color: #8E8E93; }

.action-buttons {
    display: flex;
    gap: 12px;
}

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

/* Library Modal */
.library-modal-overlay {
    position: absolute;
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
}

.close-btn { background: none; border: none; color: #888; font-size: 20px; cursor: pointer; }

.template-item {
    background-color: #1C1C1E;
    padding: 12px;
    border-radius: 8px;
    margin-bottom: 10px;
    cursor: pointer;
    border: 1px solid transparent;
}
.template-item:hover { border-color: #0A84FF; }

.template-info { display: flex; justify-content: space-between; margin-bottom: 4px; }
.t-name { color: white; font-weight: 600; }
.t-count { color: #888; font-size: 12px; }
.t-desc { color: #aaa; font-size: 12px; margin: 0; }
</style>
