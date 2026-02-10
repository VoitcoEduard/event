<template>
  <div class="form-module">
    <div v-if="completed" class="completed-view">
      <div class="success-icon">✓</div>
      <h2 class="completed-title">All Done!</h2>
      <p class="completed-text">Your responses have been saved.</p>
      <p class="completed-sub">Sit tight for the next activity.</p>
    </div>

    <template v-else-if="currentStep">
      <div class="progress-bar">
        <div class="progress-fill" :style="{ width: progressWidth + '%' }"></div>
      </div>

      <div class="scroll-content">
        <p class="step-counter">Step {{ currentStepIndex + 1 }} of {{ timeline.length }}</p>

        <div v-if="currentStep.type === 'message'" class="message-card">
          <p class="message-text">{{ currentStep.data.content }}</p>
        </div>

        <div v-else class="field-card">
          <p class="field-label">{{ currentStep.data.label }}</p>
          <textarea
            v-if="currentStep.data.fieldType === 'text'"
            :value="answers[currentStep.id] || ''"
            @input="updateAnswer"
            placeholder="Type your answer here..."
            class="field-input text-area"
          ></textarea>
          <input
            v-else
            :type="currentStep.data.fieldType === 'number' ? 'number' : 'text'"
            :value="answers[currentStep.id] || ''"
            @input="updateAnswer"
            :placeholder="currentStep.data.fieldType === 'number' ? 'Type a number...' : ''"
            class="field-input"
          />
        </div>
      </div>

      <div class="footer">
        <button 
          v-if="currentStepIndex > 0" 
          class="back-button" 
          @click="handleBack"
          :disabled="submitting"
        >
          Back
        </button>
        <button 
          class="next-button" 
          @click="handleNext"
          :disabled="submitting"
        >
          <span v-if="submitting" class="loading-spinner-small"></span>
          <span v-else>{{ isLastStep ? 'Submit' : 'Next' }}</span>
        </button>
      </div>

      <button class="template-button" @click="showTemplateModal = true">
        📂 Load Template
      </button>
    </template>

    <div v-if="!currentStep" class="empty-view">
      <p>Empty Form Configuration</p>
    </div>

    <div v-if="showTemplateModal" class="modal-overlay" @click.self="showTemplateModal = false">
      <div class="dialog">
        <h3 class="dialog-title">Load from Template</h3>
        <div class="template-list">
          <div v-if="loadingTemplates" class="loading-templates">
            <div class="loading-spinner-small"></div>
          </div>
          <template v-else>
            <div 
              v-for="template in templates" 
              :key="template.id" 
              class="template-item-wrapper"
            >
              <div class="template-item" @click="loadTemplate(template)">
                <p class="template-item-name">{{ template.name }}</p>
                <p class="template-item-desc">{{ template.description }}</p>
              </div>
              <button class="delete-template-btn" @click.stop="deleteTemplate(template.id, template.name)">
                🗑
              </button>
            </div>
            <p v-if="templates.length === 0" class="no-templates">
              No templates found.
            </p>
          </template>
        </div>
        <button class="dialog-close-btn" @click="showTemplateModal = false">
          Cancel
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { API_BASE_URL } from '../utils'
import { useToast } from '../composables/useToast'

const props = defineProps({
  module: { type: Object, required: true },
  eventId: { type: String, required: true },
  userId: { type: String, required: true }
})

const emit = defineEmits(['complete'])

const timeline = computed(() => props.module.data?.timeline || [])
const currentStepIndex = ref(0)
const answers = ref({})
const submitting = ref(false)
const completed = ref(false)
const templates = ref([])
const showTemplateModal = ref(false)
const loadingTemplates = ref(false)

const currentStep = computed(() => timeline.value[currentStepIndex.value])
const isLastStep = computed(() => currentStepIndex.value === timeline.value.length - 1)
const progressWidth = computed(() => ((currentStepIndex.value + 1) / timeline.value.length) * 100)

const { success, error, info } = useToast()

const fetchTemplates = async () => {
  loadingTemplates.value = true
  try {
    const response = await fetch(`${API_BASE_URL}/templates`)
    if (response.ok) {
      const data = await response.json()
      templates.value = data || []
    }
  } catch (error) {
    console.error('Error fetching templates:', error)
  } finally {
    loadingTemplates.value = false
  }
}

onMounted(() => {
  fetchTemplates()
})

const updateAnswer = (e) => {
  const value = e.target.value
  answers.value = { ...answers.value, [currentStep.value.id]: value }
}

const loadTemplate = (template) => {
  if (template.data?.answers) {
    answers.value = { ...answers.value, ...template.data.answers }
  }
  showTemplateModal.value = false
  success('Template loaded successfully!')
}

const deleteTemplate = async (id, name) => {
  if (!confirm(`Are you sure you want to delete "${name}"?`)) return
  
  try {
    const response = await fetch(`${API_BASE_URL}/templates/${id}`, { method: 'DELETE' })
    if (response.ok) {
      fetchTemplates()
    } else {
      error('Failed to delete template.')
    }
  } catch (err) {
    console.error(err)
    error('Failed to delete template.')
  }
}

const handleBack = () => {
  if (currentStepIndex.value > 0) {
    currentStepIndex.value--
  }
}

const handleNext = async () => {
  if (currentStep.value.type === 'field') {
    const answer = answers.value[currentStep.value.id]
    if (!answer || (typeof answer === 'string' && !answer.trim())) {
      info('Please provide an answer.')
      return
    }
  }

  if (isLastStep.value) {
    await handleSubmit()
  } else {
    currentStepIndex.value++
  }
}

const handleSubmit = async () => {
  submitting.value = true
  try {
    const response = await fetch(`${API_BASE_URL}/events/${props.eventId}/modules/${props.module.id}/response`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        userId: props.userId,
        data: answers.value
      })
    })

    if (response.ok) {
      completed.value = true
      emit('complete')
    } else {
      error('Failed to submit response.')
    }
  } catch (err) {
    console.error('Error submitting response:', err)
    error('Failed to submit response.')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.form-module {
  width: 100%;
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
}

.progress-bar {
  width: 100%;
  height: 6px;
  background-color: #333;
  border-radius: 3px;
  margin-bottom: 20px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background-color: #30D158;
  transition: width 0.3s ease;
}

.scroll-content {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-bottom: 100px;
}

.step-counter {
  color: #666;
  margin-bottom: 16px;
  font-size: 14px;
  font-weight: 600;
  text-transform: uppercase;
}

.message-card {
  background-color: #1E1E1E;
  border-radius: 20px;
  padding: 30px;
  width: 100%;
  border: 1px solid #333;
  display: flex;
  align-items: center;
  justify-content: center;
}

.message-text {
  font-size: 20px;
  color: #FFF;
  line-height: 30;
  text-align: center;
}

.field-card {
  background-color: #1E1E1E;
  border-radius: 20px;
  padding: 24px;
  width: 100%;
  border: 1px solid #333;
}

.field-label {
  font-size: 18px;
  color: #FFF;
  font-weight: 700;
  margin-bottom: 16px;
}

.field-input {
  background-color: #2C2C2C;
  border-radius: 12px;
  padding: 16px;
  font-size: 16px;
  color: #FFF;
  border: 1px solid #444;
  width: 100%;
}

.field-input.text-area {
  min-height: 100px;
  resize: vertical;
}

.footer {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  gap: 12px;
  padding: 20px;
}

.back-button {
  flex: 1;
  padding: 16px;
  border-radius: 12px;
  background-color: #333;
  color: #FFF;
  font-size: 16px;
  font-weight: 600;
}

.next-button {
  flex: 2;
  padding: 16px;
  border-radius: 12px;
  background-color: #007AFF;
  color: #FFF;
  font-size: 16px;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
}

.template-button {
  position: absolute;
  top: 10px;
  right: 10px;
  display: flex;
  flex-direction: row;
  align-items: center;
  padding: 8px 12px;
  background-color: rgba(0, 122, 255, 0.1);
  border-radius: 8px;
  color: #007AFF;
  font-size: 14px;
  font-weight: 600;
}

.completed-view {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  padding: 40px;
}

.success-icon {
  font-size: 80px;
  color: #30D158;
  margin-bottom: 20px;
}

.completed-title {
  font-size: 28px;
  font-weight: bold;
  color: #FFF;
  margin-bottom: 10px;
}

.completed-text {
  font-size: 18px;
  color: #CCC;
  margin-bottom: 8px;
  text-align: center;
}

.completed-sub {
  font-size: 14px;
  color: #888;
  font-style: italic;
}

.empty-view {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  color: #666;
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
  z-index: 100;
}

.dialog {
  background-color: #1E1E1E;
  border-radius: 20px;
  padding: 24px;
  width: 85%;
  max-width: 400px;
}

.dialog-title {
  font-size: 20px;
  font-weight: bold;
  color: #FFF;
  margin-bottom: 16px;
  text-align: center;
}

.template-list {
  max-height: 300px;
  overflow-y: auto;
}

.loading-templates {
  display: flex;
  justify-content: center;
  padding: 20px;
}

.template-item-wrapper {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}

.template-item {
  flex: 1;
  background-color: #2C2C2C;
  border-radius: 12px;
  padding: 16px;
  cursor: pointer;
}

.template-item-name {
  font-size: 16px;
  font-weight: 600;
  color: #FFF;
}

.template-item-desc {
  font-size: 14px;
  color: #888;
  margin-top: 4px;
}

.delete-template-btn {
  padding: 8px;
  margin-left: 8px;
  background: none;
  font-size: 20px;
}

.no-templates {
  color: #666;
  text-align: center;
  padding: 20px;
}

.dialog-close-btn {
  margin-top: 16px;
  padding: 16px;
  background-color: #333;
  border-radius: 12px;
  width: 100%;
  color: #FFF;
  font-size: 16px;
  font-weight: 600;
}

.loading-spinner-small {
  width: 20px;
  height: 20px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: #007AFF;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
