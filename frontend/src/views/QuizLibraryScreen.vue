<template>
  <div class="library-screen">
    <div class="header">
      <div class="header-top">
        <button class="back-link" @click="goBack">
          <span>← Back</span>
        </button>
        <p class="header-title">Library</p>
      </div>
      <div class="header-actions">
        <p class="header-subtitle">Manage Templates</p>
        <button class="create-btn" @click="createNewTemplate">
          + New {{ activeTab === 'form' ? 'Form' : 'Quiz' }} Template
        </button>
        <button class="import-btn" @click="triggerFileInput">
            📂 Import
        </button>
        <input 
            type="file" 
            ref="fileInput" 
            style="display: none" 
            accept=".json"
            @change="handleFileUpload"
        />
      </div>
    </div>

    <div class="content-wrapper">
        <!-- Vertical Tabs Sidebar -->
        <div class="sidebar">
            <button 
                class="tab-btn" 
                :class="{ active: activeTab === 'form' }"
                @click="activeTab = 'form'"
            >
                <span class="tab-icon">📝</span>
                Forms
            </button>
            <button 
                class="tab-btn" 
                :class="{ active: activeTab === 'quiz' }"
                @click="activeTab = 'quiz'"
            >
                <span class="tab-icon">🎮</span>
                Quizzes
            </button>
        </div>

        <!-- Main Content Area -->
        <div class="main-content">
            <div v-if="loading" class="loading-state">
                <p>Loading templates...</p>
            </div>

            <div v-else-if="filteredTemplates.length === 0" class="empty-state">
                <p class="empty-icon">{{ activeTab === 'form' ? '📝' : '🎮' }}</p>
                <p class="empty-text">No {{ activeTab }} templates found.</p>
            </div>

            <div v-else class="templates-grid">
                <div v-for="template in filteredTemplates" :key="template.id" class="template-card">
                <div class="card-header">
                    <h3 class="template-name">{{ template.name }}</h3>
                    <div class="card-actions">
                        <button class="icon-btn" @click="exportTemplate(template)" title="Export JSON">⬇</button>
                        <button class="icon-btn edit-btn" @click="editTemplate(template)" title="Edit">✎</button>
                        <button class="icon-btn delete-btn" @click="deleteTemplate(template)" title="Delete">🗑</button>
                    </div>
                </div>
                <p class="template-desc">{{ template.description || 'No description' }}</p>
                <div class="template-meta">
                    <span v-if="activeTab === 'quiz'" class="question-count">{{ (template.data?.questions || []).length }} Questions</span>
                    <span v-else class="question-count">{{ (template.data?.timeline || []).length }} Steps</span>
                    <span class="created-at">{{ formatDate(template.createdAt) }}</span>
                </div>
                </div>
            </div>
        </div>
    </div>

    <!-- Edit Template Modal -->
    <div v-if="showEditor" class="modal-overlay" @click.self="closeEditor">
      <div class="modal-container">
        <div class="modal-header">
          <input 
            v-model="currentTemplate.name" 
            class="template-name-input" 
            placeholder="Template Name"
          />
          <div class="header-buttons">
            <button class="save-btn" @click="saveTemplate" :disabled="saving">
              {{ saving ? 'Saving...' : 'Save Template' }}
            </button>
            <button class="close-btn" @click="closeEditor">✕</button>
          </div>
        </div>
        
        <div class="modal-body">
            <textarea 
                v-model="currentTemplate.description" 
                class="template-desc-input" 
                placeholder="Description (optional)"
                rows="2"
            ></textarea>
            
            <!-- Dynamic Editor based on Type -->
            <QuizEditor 
                v-if="currentTemplate.moduleType === 'quiz'"
                :questions="currentTemplate.questions"
                event-id=""
                module-id=""
                @update-question="handleUpdateQuestion"
                @remove-question="handleRemoveQuestion"
                @add-question="handleAddQuestion"
                @move-question="handleMoveQuestion"
                class="editor-component"
            />

            <FormEditor
                v-if="currentTemplate.moduleType === 'form'"
                :timeline="currentTemplate.timeline"
                @update:timeline="handleUpdateTimeline"
                class="editor-component"
            />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { API_BASE_URL } from '../utils'
import { useToast } from '../composables/useToast'
import QuizEditor from '../components/QuizEditor.vue'
import FormEditor from '../components/FormEditor.vue'

const router = useRouter()
const { success, error, info } = useToast()

const activeTab = ref('form')
const templates = ref([])
const loading = ref(true)
const showEditor = ref(false)
const saving = ref(false)
const fileInput = ref(null)
const currentTemplate = ref({
  id: null,
  name: '',
  description: '',
  moduleType: 'form',
  questions: [],
  timeline: []
})

const filteredTemplates = computed(() => {
    return templates.value.filter(t => t.moduleType === activeTab.value)
})

const fetchTemplates = async () => {
  loading.value = true
  try {
    const response = await fetch(`${API_BASE_URL}/templates`)
    if (response.ok) {
        const data = await response.json()
        // Ensure legacy templates have a type (default to quiz if missing, based on data structure)
        templates.value = data.map(t => ({
            ...t,
            moduleType: t.moduleType || (t.data?.questions ? 'quiz' : 'form')
        }))
    } else {
      error('Failed to load templates')
    }
  } catch (err) {
    console.error(err)
    error('Error loading templates')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchTemplates()
})

const goBack = () => {
  router.push('/admin')
}

const createNewTemplate = () => {
  currentTemplate.value = {
    id: null,
    name: activeTab.value === 'form' ? 'New Form Template' : 'New Quiz Template',
    description: '',
    moduleType: activeTab.value,
    questions: [],
    timeline: []
  }
  showEditor.value = true
}

const editTemplate = (template) => {
  currentTemplate.value = {
    id: template.id,
    name: template.name,
    description: template.description,
    moduleType: template.moduleType,
    questions: JSON.parse(JSON.stringify(template.data?.questions || [])),
    timeline: JSON.parse(JSON.stringify(template.data?.timeline || []))
  }
  showEditor.value = true
}

const deleteTemplate = async (template) => {
  if (!confirm(`Delete template "${template.name}"?`)) return

  try {
    const response = await fetch(`${API_BASE_URL}/templates/${template.id}`, {
      method: 'DELETE'
    })

    if (response.ok) {
      success('Template deleted')
      fetchTemplates()
    } else {
      error('Failed to delete template')
    }
  } catch (err) {
    console.error(err)
    error('Error deleting template')
  }
}

const saveTemplate = async () => {
  if (!currentTemplate.value.name.trim()) {
    info('Please enter a template name')
    return
  }
  
  const isQuiz = currentTemplate.value.moduleType === 'quiz'
  
  if (isQuiz && currentTemplate.value.questions.length === 0) {
       if(!confirm("Save quiz template with no questions?")) return
   }

   saving.value = true
  try {
    const payload = {
      name: currentTemplate.value.name,
      description: currentTemplate.value.description,
      moduleType: currentTemplate.value.moduleType,
      data: isQuiz 
        ? { questions: currentTemplate.value.questions } 
        : { timeline: currentTemplate.value.timeline }
    }

    let url = `${API_BASE_URL}/templates`
    let method = 'POST'

    if (currentTemplate.value.id) {
      url = `${API_BASE_URL}/templates/${currentTemplate.value.id}`
      method = 'PUT'
    }

    const response = await fetch(url, {
      method: method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    })

    if (response.ok) {
      success('Template saved successfully')
      showEditor.value = false
      fetchTemplates()
    } else {
      error('Failed to save template')
    }
  } catch (err) {
    console.error(err)
    error('Error saving template')
  } finally {
    saving.value = false
  }
}

const closeEditor = () => {
    if(confirm('Discard changes?')) {
        showEditor.value = false
    }
}

// Quiz Editor Handlers
const handleUpdateQuestion = (index, updatedQ) => {
    currentTemplate.value.questions[index] = updatedQ
}

const handleRemoveQuestion = (index) => {
    if (!confirm('Delete this question?')) return
    currentTemplate.value.questions.splice(index, 1)
}

const handleAddQuestion = () => {
    currentTemplate.value.questions.push({
        id: Date.now().toString(),
        text: '',
        answers: ['', '', '', ''],
        correctIndex: 0,
        timeLimit: 20
    })
}

const handleMoveQuestion = (index, direction) => {
    const newIndex = index + direction
    if (newIndex < 0 || newIndex >= currentTemplate.value.questions.length) return
    const temp = currentTemplate.value.questions[index]
    currentTemplate.value.questions[index] = currentTemplate.value.questions[newIndex]
    currentTemplate.value.questions[newIndex] = temp
}

// Form Editor Handlers
const handleUpdateTimeline = (newTimeline) => {
    currentTemplate.value.timeline = newTimeline
}


const formatDate = (dateString) => {
  if (!dateString) return ''
  return new Date(dateString).toLocaleDateString()
}
const triggerFileInput = () => {
    fileInput.value.click()
}

const handleFileUpload = (event) => {
    const file = event.target.files[0]
    if (!file) return

    const reader = new FileReader()
    reader.onload = async (e) => {
        try {
            const content = JSON.parse(e.target.result)
            
            // Validate content structure roughly
            if(!content.name || !content.moduleType) {
                error('Invalid template file format')
                return
            }

            // Ask User for confirmation or just save
            if(!confirm(`Import template "${content.name}"?`)) return

            // Prepare payload (clean up ID if present to create new)
            const payload = {
                name: content.name + ' (Imported)',
                description: content.description || '',
                moduleType: content.moduleType,
                data: content.data || {}
            }

            const res = await fetch(`${API_BASE_URL}/templates`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            })

            if(res.ok) {
                success('Template imported successfully')
                fetchTemplates()
            } else {
                error('Failed to import template')
            }
        } catch (err) {
            console.error(err)
            error('Error parsing JSON file')
        } finally {
            event.target.value = '' // Reset input
        }
    }
    reader.readAsText(file)
}

const exportTemplate = (template) => {
    const dataStr = JSON.stringify(template, null, 2)
    const blob = new Blob([dataStr], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    
    const a = document.createElement('a')
    a.href = url
    a.download = `${template.name.replace(/[^a-z0-9]/gi, '_').toLowerCase()}_template.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
}
</script>

<style scoped>
.library-screen {
  min-height: 100vh;
  background-color: #121212;
  color: white;
  display: flex;
  flex-direction: column;
}

.header {
  padding: 20px 24px;
  background-color: #1E1E1E;
  border-bottom: 1px solid #2C2C2C;
}

.header-top {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
  gap: 16px;
}

.back-link {
  color: #007AFF;
  font-size: 14px;
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
}

.header-title {
  font-size: 12px;
  color: #666;
  text-transform: uppercase;
  font-weight: 600;
  letter-spacing: 1px;
}

.header-actions {
    display: flex;
    gap: 12px;
    align-items: center;
}

.header-subtitle {
  font-size: 24px;
  font-weight: 700;
  color: #FFF;
}

.create-btn {
  background-color: #007AFF;
  color: white;
  border: none;
  padding: 10px 20px;
  border-radius: 8px;
  font-weight: 600;
  cursor: pointer;
}
.create-btn:hover { background-color: #005bb5; }

.import-btn {
  background-color: #333;
  color: white;
  border: 1px solid #555;
  padding: 10px 16px;
  border-radius: 8px;
  font-weight: 600;
  cursor: pointer;
}
.import-btn:hover { background-color: #444; }

.content-wrapper {
    display: flex;
    flex: 1;
    overflow: hidden;
}

.sidebar {
    width: 200px;
    background-color: #1E1E1E; /* Slightly different shade */
    border-right: 1px solid #2C2C2C;
    display: flex;
    flex-direction: column;
    padding: 20px 0;
}

.tab-btn {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 16px 24px;
    background: none;
    border: none;
    color: #888;
    cursor: pointer;
    font-size: 16px;
    font-weight: 600;
    text-align: left;
    transition: all 0.2s;
    border-left: 3px solid transparent;
}

.tab-btn:hover {
    background-color: #252525;
    color: white;
}

.tab-btn.active {
    background-color: #252525;
    color: #007AFF;
    border-left-color: #007AFF;
}

.tab-icon { font-size: 20px; }

.main-content {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
}

.loading-state, .empty-state {
  text-align: center;
  padding: 60px;
  color: #888;
}

.empty-icon { font-size: 48px; margin-bottom: 16px; }

.templates-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
}

.template-card {
  background-color: #1E1E1E;
  border: 1px solid #333;
  border-radius: 12px;
  padding: 20px;
  display: flex;
  flex-direction: column;
  transition: transform 0.2s;
}
.template-card:hover { border-color: #555; }

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 10px;
}

.template-name {
  font-size: 18px;
  font-weight: 600;
  margin: 0;
  color: #fff;
}

.card-actions { display: flex; gap: 8px; }

.icon-btn {
  background: none;
  border: none;
  color: #888;
  cursor: pointer;
  padding: 4px;
  font-size: 16px;
  border-radius: 4px;
}
.icon-btn:hover { background-color: #333; color: white; }
.delete-btn:hover { color: #ff3b30; }

.template-desc {
  font-size: 14px;
  color: #aaa;
  margin-bottom: 16px;
  flex: 1;
}

.template-meta {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: #666;
  border-top: 1px solid #333;
  padding-top: 12px;
}

/* Modal Styles */
.modal-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background-color: rgba(0, 0, 0, 0.7);
  display: flex;
  justify-content: center;
  align-items: flex-end; /* Slide up from bottom vibe */
  z-index: 1000;
}

.modal-container {
  background-color: #1C1C1E;
  width: 100%;
  height: 95vh;
  max-width: 1000px;
  border-top-left-radius: 20px;
  border-top-right-radius: 20px;
  display: flex;
  flex-direction: column;
  box-shadow: 0 -4px 20px rgba(0,0,0,0.5);
}

.modal-header {
  padding: 20px;
  border-bottom: 1px solid #333;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.template-name-input {
  background: transparent;
  border: none;
  color: white;
  font-size: 24px;
  font-weight: bold;
  width: 100%;
  outline: none;
}
.template-name-input::placeholder { color: #555; }

.header-buttons { display: flex; gap: 12px; align-items: center; }

.save-btn {
  background-color: #30D158;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  font-weight: 600;
  cursor: pointer;
}
.save-btn:disabled { opacity: 0.5; }

.close-btn {
  background: none;
  border: none;
  color: #888;
  font-size: 24px;
  cursor: pointer;
}

.modal-body {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    padding: 0; 
}

.template-desc-input {
    background-color: #2C2C2E;
    border: none;
    color: white;
    padding: 16px 20px;
    font-size: 14px;
    resize: none;
    outline: none;
    border-bottom: 1px solid #333;
}

.editor-component {
    flex: 1;
}

@media (min-width: 800px) {
    .modal-overlay { align-items: center; }
    .modal-container { height: 90vh; border-radius: 20px; }
}
</style>
