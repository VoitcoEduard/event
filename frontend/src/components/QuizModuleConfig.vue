<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click.self="handleClose">
      <div class="modal-container" :style="modalStyle">
        <div class="header">
          <div>
            <p class="title">Quiz Editor</p>
            <p class="subtitle">{{ module?.name || 'Untitled Quiz' }}</p>
          </div>
          <div class="header-buttons">
            <button class="icon-button" @click="showLibraryModal = true" title="Load from Library">
              📥
            </button>
            <button class="library-btn" @click="triggerFileInput">
              📄 Load JSON
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
            @update-question="updateQuestion"
            @remove-question="removeQuestion"
            @add-question="addQuestion"
            @move-question="moveQuestion"
          />

          <div class="action-bar">
            <div class="status-text">
              {{ questions.length }} Questions
            </div>
            <div class="action-buttons">
                <button class="save-template-btn" @click="exportJSON" :disabled="questions.length === 0">
                    ⬇ Export JSON
                </button>
                <button class="save-template-btn" @click="saveAsTemplate" :disabled="questions.length === 0">
                    💾 Save as Template
                </button>
                <button class="publish-btn" @click="handleSave" :disabled="loading">
                {{ loading ? 'Saving...' : 'Save Quiz' }}
                </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Library Modal -->
      <div v-if="showLibraryModal" class="library-modal-overlay" @click.self="showLibraryModal = false">
        <div class="library-modal">
            <div class="library-header">
                <h3>Select a Template</h3>
                <button class="close-btn" @click="showLibraryModal = false">✕</button>
            </div>
            <div class="library-content">
                <div v-if="loadingTemplates" class="loading">Loading...</div>
                <div v-else-if="templates.length === 0" class="empty">No templates found.</div>
                <div v-else class="template-list">
                    <div v-for="t in templates" :key="t.id" class="template-item" @click="loadTemplate(t)">
                        <div class="template-info">
                            <span class="t-name">{{ t.name }}</span>
                            <span class="t-count">{{ (t.data?.questions || []).length }} Qs</span>
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
import { API_BASE_URL } from '../utils'
import { useToast } from '../composables/useToast'
import QuizEditor from './QuizEditor.vue'

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

// Load existing questions when module opens
const fetchTemplates = async () => {
    loadingTemplates.value = true
    try {
        const res = await fetch(`${API_BASE_URL}/templates`)
        if(res.ok) {
            const data = await res.json()
            templates.value = data.filter(t => t.moduleType === 'quiz' || (!t.moduleType && t.data?.questions))
        }
    } catch(e) {
        console.error(e)
    } finally {
        loadingTemplates.value = false
    }
}

watch(() => [props.visible, props.module], ([visible, mod]) => {
  if (visible && mod?.data?.questions) {
    questions.value = JSON.parse(JSON.stringify(mod.data.questions))
  } else if (visible) {
    questions.value = []
  }
  
  if (visible) {
      fetchTemplates()
  }
}, { immediate: true })



const loadTemplate = (template) => {
    if(questions.value.length > 0) {
        if(!confirm('This will replace your current questions. Continue?')) return
    }
    
    if(template.data?.questions) {
        questions.value = JSON.parse(JSON.stringify(template.data.questions))
        // Regenerate IDs to avoid conflicts if needed, but for now simple copy
        questions.value.forEach(q => q.id = Date.now().toString() + Math.random().toString(36).substr(2, 5))
        success(`Loaded template: ${template.name}`)
        showLibraryModal.value = false
    }
}

const saveAsTemplate = async () => {
    const name = prompt('Enter a name for this template:')
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
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify(payload)
        })
        
        if(res.ok) {
            success('Saved as new template!')
            fetchTemplates()
        } else {
            error('Failed to save template')
        }
    } catch(e) {
        console.error(e)
        error('Error saving template')
    }
}

const addQuestion = () => {
  questions.value.push({
    id: Date.now().toString(),
    text: '',
    answers: ['', '', '', ''],
    correctIndex: 0,
    timeLimit: 20
  })
}

const removeQuestion = (index) => {
  if (!confirm('Delete this question?')) return
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
    const response = await fetch(`${API_BASE_URL}/events/${props.eventId}/modules/${props.module.id}/quiz/questions`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(questions.value)
    })

    if (response.ok) {
      success('Quiz saved successfully!')
      emit('update')
      handleClose()
    } else {
      error('Failed to save quiz.')
    }
  } catch (err) {
    console.error('Error saving quiz:', err)
    error('Failed to save quiz.')
  } finally {
    loading.value = false
  }
}

const validateQuestions = () => {
  if (questions.value.length === 0) return 'Please add at least one question.'
  
  for (const q of questions.value) {
    if (!q.text.trim()) return 'All questions must have text.'
    if (q.answers.some(a => !a.trim())) return 'All answer options must be filled.'
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
                    if(!confirm('This will replace your current questions. Continue?')) return
                }
                questions.value = loadedQuestions
                // Regenerate IDs
                questions.value.forEach(q => q.id = Date.now().toString() + Math.random().toString(36).substr(2, 5))
                success('Questions loaded from file')
            } else {
                info('No questions found in file')
            }
        } catch (err) {
            console.error(err)
            error('Error parsing JSON file')
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
