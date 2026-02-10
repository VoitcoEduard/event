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
            <button class="close-button" @click="handleClose">
              ✕
            </button>
          </div>
        </div>

        <div class="content">
          <div class="scroll-content">
            <div v-if="questions.length === 0" class="empty-state">
              <p class="empty-icon">🎮</p>
              <p class="empty-text">Start by adding a question to your quiz!</p>
            </div>
            
            <div v-else class="questions-list">
              <div v-for="(question, index) in questions" :key="question.id" class="question-wrapper-outer">
                 <div class="question-sort-controls">
                    <button @click="moveQuestion(index, -1)" :disabled="index === 0" class="move-btn">↑</button>
                    <button @click="moveQuestion(index, 1)" :disabled="index === questions.length - 1" class="move-btn">↓</button>
                 </div>
                 <div class="question-component-wrapper">
                   <QuizQuestionEditor 
                      :question="question" 
                      :index="index"
                      @update="(updatedQ) => updateQuestion(index, updatedQ)"
                      @delete="removeQuestion(index)"
                   />
                 </div>
              </div>
            </div>
            
            <div class="add-section">
              <button class="add-btn" @click="addQuestion">
                + Add Question
              </button>
            </div>

            <div style="height: 100px;"></div>
          </div>

          <div class="action-bar">
            <div class="status-text">
              {{ questions.length }} Questions
            </div>
            <button class="publish-btn" @click="handleSave" :disabled="loading">
              {{ loading ? 'Saving...' : 'Save Quiz' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import { API_BASE_URL } from '../utils'
import { useToast } from '../composables/useToast'
import QuizQuestionEditor from './QuizQuestionEditor.vue'

const props = defineProps({
  visible: { type: Boolean, default: false },
  module: { type: Object, default: null },
  eventId: { type: String, default: '' }
})

const emit = defineEmits(['close', 'update'])
const { success, error, info } = useToast()

const questions = ref([])
const loading = ref(false)

// Load existing questions when module opens
watch(() => [props.visible, props.module], ([visible, mod]) => {
  if (visible && mod?.data?.questions) {
    // Deep copy to avoid mutating prop directly before save
    questions.value = JSON.parse(JSON.stringify(mod.data.questions))
  } else if (visible) {
    questions.value = []
  }
}, { immediate: true })

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

.scroll-content {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: #8E8E93;
}
.empty-icon { font-size: 48px; margin-bottom: 16px; }

.question-wrapper-outer {
  display: flex;
  gap: 12px;
  margin-bottom: 24px;
}

.question-sort-controls {
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  padding-top: 10px;
  gap: 8px;
}

.move-btn {
  background-color: #3A3A3C;
  color: white;
  border: none;
  padding: 8px;
  border-radius: 6px;
  cursor: pointer;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.move-btn:disabled { opacity: 0.3; cursor: not-allowed; }
.move-btn:hover:not(:disabled) { background-color: #48484A; }

.question-component-wrapper {
  flex: 1;
}

.add-section {
  text-align: center;
  margin-top: 20px;
}

.add-btn {
  background-color: rgba(10, 132, 255, 0.1);
  color: #0A84FF;
  padding: 12px 24px;
  border-radius: 8px;
  font-weight: 600;
  border: none;
  cursor: pointer;
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
</style>
