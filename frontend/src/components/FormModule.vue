<template>
  <div class="form-module">
    <div v-if="completed" class="completed-view">
      <div class="success-icon">✓</div>
      <h2 class="completed-title">{{ $t('form_module.completed_title') }}</h2>
      <p class="completed-text">{{ $t('form_module.completed_text') }}</p>
      <p class="completed-sub">{{ $t('form_module.completed_sub') }}</p>
    </div>

    <template v-else-if="currentStep">
      <div class="progress-bar">
        <div class="progress-fill" :style="{ width: progressWidth + '%' }"></div>
      </div>

      <div class="scroll-content">
        <p class="step-counter">{{ $t('form_module.step_counter', { current: currentStepIndex + 1, total: timeline.length }) }}</p>

        <div v-if="currentStep.type === 'message'" class="message-card">
          <p class="message-text">{{ currentStep.data.content }}</p>
        </div>

        <div v-else class="field-card">
          <p class="field-label">{{ currentStep.data.label }}</p>
          
          <!-- Text field -->
          <textarea
            v-if="currentStep.data.fieldType === 'text'"
            :value="answers[currentStep.id] || ''"
            @input="updateAnswer"
            :placeholder="$t('form_module.type_answer')"
            class="field-input text-area"
          ></textarea>
          
          <!-- Number field -->
          <input
            v-else-if="currentStep.data.fieldType === 'number'"
            type="number"
            :value="answers[currentStep.id] || ''"
            @input="updateAnswer"
            :placeholder="$t('form_module.type_number')"
            class="field-input"
          />
          
          <!-- Date field -->
          <input
            v-else-if="currentStep.data.fieldType === 'date'"
            type="date"
            :value="answers[currentStep.id] || ''"
            @input="updateAnswer"
            class="field-input"
          />
          
          <!-- Radio buttons -->
          <div v-else-if="currentStep.data.fieldType === 'radio'" class="selection-options">
            <div
              v-for="option in currentStep.data.predefinedAnswers"
              :key="option.id"
              class="selection-option radio-option"
              @click="selectRadioOption(currentStep.id, option.value)"
            >
              <div class="selection-indicator" :class="{ selected: answers[currentStep.id] === option.value }">
                <div class="selection-dot" v-if="answers[currentStep.id] === option.value"></div>
              </div>
              <span class="selection-text">{{ option.text }}</span>
            </div>
            <div
              v-if="currentStep.data.allowOther"
              class="selection-option radio-option"
              @click="selectRadioOption(currentStep.id, '__other__')"
            >
              <div class="selection-indicator" :class="{ selected: answers[currentStep.id] === '__other__' }">
                <div class="selection-dot" v-if="answers[currentStep.id] === '__other__'"></div>
              </div>
              <span class="selection-text">{{ $t('form_module.other') }}</span>
            </div>
            <input
              v-if="currentStep.data.allowOther && answers[currentStep.id] === '__other__'"
              type="text"
              :value="otherAnswers[currentStep.id] || ''"
              @input="updateOtherAnswer"
              :placeholder="currentStep.data.otherPlaceholder || $t('form_module.other_placeholder')"
              class="field-input other-input"
            />
          </div>
          
          <!-- Checkboxes -->
          <div v-else-if="currentStep.data.fieldType === 'checkbox'" class="selection-options">
            <div
              v-for="option in currentStep.data.predefinedAnswers"
              :key="option.id"
              class="selection-option checkbox-option"
              @click="toggleCheckboxOption(currentStep.id, option.value)"
            >
              <div class="selection-indicator" :class="{ selected: isCheckboxSelected(currentStep.id, option.value) }">
                <span v-if="isCheckboxSelected(currentStep.id, option.value)" class="checkbox-check">✓</span>
              </div>
              <span class="selection-text">{{ option.text }}</span>
            </div>
            <div
              v-if="currentStep.data.allowOther"
              class="selection-option checkbox-option"
              @click="toggleCheckboxOption(currentStep.id, '__other__')"
            >
              <div class="selection-indicator" :class="{ selected: isCheckboxSelected(currentStep.id, '__other__') }">
                <span v-if="isCheckboxSelected(currentStep.id, '__other__')" class="checkbox-check">✓</span>
              </div>
              <span class="selection-text">{{ $t('form_module.other') }}</span>
            </div>
            <input
              v-if="currentStep.data.allowOther && isCheckboxSelected(currentStep.id, '__other__')"
              type="text"
              :value="otherAnswers[currentStep.id] || ''"
              @input="updateOtherAnswer"
              :placeholder="currentStep.data.otherPlaceholder || $t('form_module.other_placeholder')"
              class="field-input other-input"
            />
          </div>
          
          <!-- Dropdown -->
          <div v-else-if="currentStep.data.fieldType === 'dropdown'" class="dropdown-wrapper">
            <select
              :value="answers[currentStep.id] || ''"
              @change="updateDropdownAnswer"
              class="field-input dropdown-select"
            >
              <option value="" disabled>{{ $t('form_module.select_option') }}</option>
              <option
                v-for="option in currentStep.data.predefinedAnswers"
                :key="option.id"
                :value="option.value"
              >
                {{ option.text }}
              </option>
              <option v-if="currentStep.data.allowOther" value="__other__">{{ $t('form_module.other') }}</option>
            </select>
            <input
              v-if="currentStep.data.allowOther && answers[currentStep.id] === '__other__'"
              type="text"
              :value="otherAnswers[currentStep.id] || ''"
              @input="updateOtherAnswer"
              :placeholder="currentStep.data.otherPlaceholder || $t('form_module.other_placeholder')"
              class="field-input other-input"
            />
          </div>
          
          <!-- Default text input for unknown types -->
          <input
            v-else
            type="text"
            :value="answers[currentStep.id] || ''"
            @input="updateAnswer"
            :placeholder="$t('form_module.type_answer')"
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
          {{ $t('form_module.back') }}
        </button>
        <button 
          class="next-button" 
          @click="handleNext"
          :disabled="submitting"
        >
          <span v-if="submitting" class="loading-spinner-small"></span>
          <span v-else>{{ isLastStep ? $t('form_module.submit') : $t('form_module.next') }}</span>
        </button>
      </div>

      <button class="template-button" @click="showTemplateModal = true">
        {{ $t('form_module.load_template') }}
      </button>
    </template>

    <div v-if="!currentStep" class="empty-view">
      <p>{{ $t('form_module.empty_config') }}</p>
    </div>

    <div v-if="showTemplateModal" class="modal-overlay" @click.self="showTemplateModal = false">
      <div class="dialog">
        <h3 class="dialog-title">{{ $t('quiz_config.load_library') }}</h3>
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
              {{ $t('quiz_config.no_templates') }}
            </p>
          </template>
        </div>
        <button class="dialog-close-btn" @click="showTemplateModal = false">
          {{ $t('admin.details.cancel_btn') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { API_BASE_URL } from '../utils'
import { useToast } from '../composables/useToast'

const { t } = useI18n()
const props = defineProps({
  module: { type: Object, required: true },
  eventId: { type: String, required: true },
  userId: { type: String, required: true }
})

const emit = defineEmits(['complete'])

const timeline = computed(() => props.module.data?.timeline || [])
const currentStepIndex = ref(0)
const answers = ref({})
const otherAnswers = ref({})
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

const updateOtherAnswer = (e) => {
  const value = e.target.value
  otherAnswers.value = { ...otherAnswers.value, [currentStep.value.id]: value }
}

const selectRadioOption = (stepId, value) => {
  answers.value = { ...answers.value, [stepId]: value }
}

const toggleCheckboxOption = (stepId, value) => {
  const currentValues = answers.value[stepId] || []
  const newValues = Array.isArray(currentValues) ? [...currentValues] : []
  
  if (value === '__other__') {
    const otherIndex = newValues.indexOf('__other__')
    if (otherIndex >= 0) {
      newValues.splice(otherIndex, 1)
    } else {
      newValues.push('__other__')
    }
  } else {
    const optionIndex = newValues.indexOf(value)
    if (optionIndex >= 0) {
      newValues.splice(optionIndex, 1)
    } else {
      newValues.push(value)
    }
  }
  
  answers.value = { ...answers.value, [stepId]: newValues }
}

const isCheckboxSelected = (stepId, value) => {
  const currentValues = answers.value[stepId]
  return Array.isArray(currentValues) && currentValues.includes(value)
}

const updateDropdownAnswer = (e) => {
  const value = e.target.value
  answers.value = { ...answers.value, [currentStep.value.id]: value }
}

const loadTemplate = (template) => {
  if (template.data?.answers) {
    answers.value = { ...answers.value, ...template.data.answers }
  }
  if (template.data?.otherAnswers) {
    otherAnswers.value = { ...otherAnswers.value, ...template.data.otherAnswers }
  }
  showTemplateModal.value = false
  success(t('quiz_config.loaded_template', { name: template.name }))
}

const deleteTemplate = async (id, name) => {
  if (!confirm(t('library.actions.delete_confirm', { name: name }))) return
  
  try {
    const response = await fetch(`${API_BASE_URL}/templates/${id}`, { method: 'DELETE' })
    if (response.ok) {
      fetchTemplates()
    } else {
      error(t('quiz_config.save_failed')) // Reusing save_failed as generic error or add new key
    }
  } catch (err) {
    console.error(err)
    error(t('quiz_config.save_failed'))
  }
}

const handleBack = () => {
  if (currentStepIndex.value > 0) {
    currentStepIndex.value--
  }
}

const isSelectionType = (fieldType) => {
  return ['radio', 'checkbox', 'dropdown'].includes(fieldType)
}

const hasSelection = (stepId) => {
  const answer = answers.value[stepId]
  if (!answer) return false
  if (Array.isArray(answer)) return answer.length > 0
  return !!answer
}

const handleNext = async () => {
  if (currentStep.value.type === 'field') {
    const fieldType = currentStep.value.data?.fieldType
    const stepId = currentStep.value.id
    const answer = answers.value[stepId]
    
    if (isSelectionType(fieldType)) {
      // For selection types, check if at least one option is selected
      if (!hasSelection(stepId)) {
        info(t('form_module.validation.select_option'))
        return
      }
      
      // For checkbox with allowOther, the __other__ value should have text
      if (fieldType === 'checkbox' && Array.isArray(answer) && answer.includes('__other__')) {
        const otherValue = otherAnswers.value[stepId]
        if (!otherValue || !otherValue.trim()) {
          info(t('form_module.validation.specify_other'))
          return
        }
      }
    } else {
      // For text, number, date types
      if (!answer || (typeof answer === 'string' && !answer.trim())) {
        info(t('form_module.validation.provide_answer'))
        return
      }
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
        data: answers.value,
        otherAnswers: otherAnswers.value
      })
    })

    if (response.ok) {
      completed.value = true
      emit('complete', answers.value)
    } else {
      error(t('form_module.error_submit'))
    }
  } catch (err) {
    console.error('Error submitting response:', err)
    error(t('form_module.error_submit'))
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.form-module {
  width: 100%;
  height: 100%; /* Occupy full height of parent */
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
  overflow: hidden; /* Contain children */
}

.progress-bar {
  width: 100%;
  height: 6px;
  background-color: #333;
  border-radius: 3px;
  margin-bottom: 20px;
  overflow: hidden;
  flex-shrink: 0; /* Prevent shrinking */
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
  padding-bottom: 100px; /* Space for footer */
  flex: 1;
  overflow-y: auto; /* Enable internal scrolling */
  -webkit-overflow-scrolling: touch; /* Smooth scrolling on iOS */
  padding-left: 4px; /* Prevent scrollbar overlap visual issues if needed */
  padding-right: 4px;
}

.step-counter {
  color: #666;
  margin-bottom: 16px;
  font-size: 14px;
  font-weight: 600;
  text-transform: uppercase;
  flex-shrink: 0;
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
  flex-shrink: 0;
}

.message-text {
  font-size: 20px;
  color: #FFF;
  line-height: 1.5;
  text-align: center;
}

.field-card {
  background-color: #1E1E1E;
  border-radius: 20px;
  padding: 24px;
  width: 100%;
  border: 1px solid #333;
  flex-shrink: 0;
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

.field-input:focus {
  outline: none;
  border-color: #007AFF;
}

.selection-options {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.selection-option {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background-color: #2C2C2C;
  border-radius: 12px;
  cursor: pointer;
  transition: background-color 0.2s;
  border: 1px solid #444;
}

.selection-option:hover {
  background-color: #3A3A3A;
}

.selection-indicator {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 2px solid #666;
  margin-right: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
  flex-shrink: 0;
}

.selection-indicator.selected {
  border-color: #007AFF;
  background-color: #007AFF;
}

.selection-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background-color: #FFF;
}

.checkbox-option .selection-indicator {
  border-radius: 6px;
}

.checkbox-check {
  color: #FFF;
  font-size: 14px;
  font-weight: bold;
}

.selection-text {
  color: #FFF;
  font-size: 16px;
}

.other-input {
  margin-top: 12px;
}

.dropdown-select {
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%23666' d='M6 8L1 3h10z'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 16px center;
  padding-right: 40px;
}

.footer {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  gap: 12px;
  padding: 20px;
  background-color: inherit; /* Or match card bg if needed, but absolute in relative container is fine */
  /* Add gradient or solid bg behind footer to hide scrolled content? */
  background: linear-gradient(to top, rgba(18,18,18, 1) 0%, rgba(18,18,18, 0.9) 70%, rgba(18,18,18, 0) 100%);
  /* Note: FormModule doesn't control background color, it might depend on theme */
  z-index: 10;
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
  border: 2px solid #666;
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
