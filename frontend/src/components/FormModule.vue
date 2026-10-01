<template>
  <div class="form-module">
    <div v-if="completed" class="completed-view">
      <AppIcon class="success-icon" name="check" :size="34" />
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
                <AppIcon v-if="isCheckboxSelected(currentStep.id, option.value)" class="checkbox-check" name="check" :size="14" />
              </div>
              <span class="selection-text">{{ option.text }}</span>
            </div>
            <div
              v-if="currentStep.data.allowOther"
              class="selection-option checkbox-option"
              @click="toggleCheckboxOption(currentStep.id, '__other__')"
            >
              <div class="selection-indicator" :class="{ selected: isCheckboxSelected(currentStep.id, '__other__') }">
                <AppIcon v-if="isCheckboxSelected(currentStep.id, '__other__')" class="checkbox-check" name="check" :size="14" />
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
                <AppIcon name="trash" :size="16" />
              </button>
            </div>
            <p v-if="templates.length === 0" class="no-templates">
              {{ $t('quiz_config.no_templates') }}
            </p>
          </template>
        </div>
        <button class="dialog-close-btn" @click="showTemplateModal = false">
          {{ $t('common.cancel') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppIcon from './AppIcon.vue'
import { API_BASE_URL, getAuthHeaders } from '../utils'
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
    const response = await fetch(`${API_BASE_URL}/templates/${id}`, { method: 'DELETE', headers: getAuthHeaders() })
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
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&display=swap');

* { font-family: 'Inter', sans-serif; box-sizing: border-box; }

/* ── Shell ── */
.form-module {
  width: 100%;
  height: 100%;
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
  overflow: hidden;
  background: #0d0d0f;
}

/* ── Progress bar ── */
.progress-bar {
  width: 100%;
  height: 4px;
  background: #1e1e26;
  flex-shrink: 0;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--ops-blue), #2d8cff);
  transition: width 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  border-radius: 0 2px 2px 0;
}

/* ── Scrollable area ── */
.scroll-content {
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 24px 20px 120px;
  flex: 1;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
}

.scroll-content::-webkit-scrollbar { width: 4px; }
.scroll-content::-webkit-scrollbar-track { background: transparent; }
.scroll-content::-webkit-scrollbar-thumb { background: #2a2a35; border-radius: 4px; }

/* ── Step counter ── */
.step-counter {
  color: #444;
  margin-bottom: 20px;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 2px;
  align-self: flex-start;
}

/* ── Message card ── */
.message-card {
  background: #161619;
  border-radius: 20px;
  padding: 32px 24px;
  width: 100%;
  border: 1px solid #2a2a35;
  display: flex;
  align-items: center;
  justify-content: center;
}

.message-text {
  font-size: 19px;
  color: #e8e8f0;
  line-height: 1.6;
  text-align: center;
  font-weight: 500;
}

/* ── Field card ── */
.field-card {
  background: #161619;
  border-radius: 20px;
  padding: 24px 22px;
  width: 100%;
  border: 1px solid #2a2a35;
}

.field-label {
  font-size: 17px;
  color: #f0f0f8;
  font-weight: 700;
  margin-bottom: 18px;
  line-height: 1.4;
}

/* ── Text / Number / Date inputs ── */
.field-input {
  background: #0d0d0f;
  border-radius: 14px;
  padding: 15px 16px;
  font-size: 16px;
  color: #f0f0f8;
  border: 2px solid #2a2a35;
  width: 100%;
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s;
  caret-color: var(--ops-blue);
  font-family: 'Inter', sans-serif;
}

.field-input.text-area {
  min-height: 110px;
  resize: vertical;
  line-height: 1.5;
}

.field-input:focus {
  border-color: var(--ops-blue);
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.15);
}

.field-input::placeholder { color: #363640; }

/* ── Selection options (radio + checkbox) ── */
.selection-options {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.selection-option {
  display: flex;
  align-items: center;
  padding: 14px 16px;
  background: #0d0d0f;
  border-radius: 14px;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
  border: 2px solid #2a2a35;
  user-select: none;
}

.selection-option:hover {
  border-color: #4a4a60;
  background: #111115;
}

/* selected state: full border highlight */
.selection-option:has(.selection-indicator.selected) {
  border-color: var(--ops-blue);
  background: rgba(99, 102, 241, 0.05);
}

.selection-indicator {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 2px solid #3a3a4a;
  margin-right: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: border-color 0.2s, background 0.2s;
  flex-shrink: 0;
}

.selection-indicator.selected {
  border-color: var(--ops-blue);
  background: var(--ops-blue);
}

.selection-dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: #fff;
}

.checkbox-option .selection-indicator {
  border-radius: 7px;
}

.checkbox-check {
  color: #fff;
  font-size: 13px;
  font-weight: 800;
}

.selection-text {
  color: #e0e0f0;
  font-size: 15px;
  font-weight: 500;
}

.other-input {
  margin-top: 12px;
}

/* ── Dropdown ── */
.dropdown-select {
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%236366f1' d='M6 8L1 3h10z'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 16px center;
  padding-right: 44px;
  cursor: pointer;
}

.dropdown-select option {
  background: #161619;
  color: #f0f0f8;
}

/* ── Footer ── */
.footer {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  gap: 10px;
  padding: 16px 20px 24px;
  background: linear-gradient(to top, #0d0d0f 60%, transparent 100%);
  z-index: 10;
}

.back-button {
  flex: 1;
  padding: 15px;
  border-radius: 14px;
  background: #1e1e26;
  border: 1px solid #2a2a35;
  color: #aaa;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s, color 0.2s;
  font-family: 'Inter', sans-serif;
}

.back-button:hover { background: #252530; color: #fff; }
.back-button:disabled { opacity: 0.4; cursor: not-allowed; }

.next-button {
  flex: 2;
  padding: 15px;
  border-radius: 14px;
  background: linear-gradient(135deg, #2d8cff, #0a6cf5);
  border: none;
  color: #fff;
  font-size: 15px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: transform 0.15s, box-shadow 0.2s, opacity 0.2s;
  box-shadow: 0 4px 18px rgba(99, 102, 241, 0.35);
  font-family: 'Inter', sans-serif;
}

.next-button:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 6px 24px rgba(99, 102, 241, 0.5);
}

.next-button:active:not(:disabled) { transform: translateY(0); }
.next-button:disabled { opacity: 0.45; cursor: not-allowed; }

/* ── Template button ── */
.template-button {
  position: absolute;
  top: 12px;
  right: 12px;
  display: flex;
  align-items: center;
  padding: 7px 12px;
  background: #1a1a2a;
  border: 1px solid #2a2a45;
  border-radius: 10px;
  color: #2d8cff;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s, border-color 0.2s;
  font-family: 'Inter', sans-serif;
}

.template-button:hover {
  background: #1e1e35;
  border-color: var(--ops-blue);
}

/* ── Completed view ── */
.completed-view {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  padding: 48px 32px;
  text-align: center;
}

.success-icon {
  width: 88px;
  height: 88px;
  border-radius: 50%;
  background: linear-gradient(135deg, #2d8cff, #0a6cf5);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 40px;
  color: #fff;
  margin-bottom: 28px;
  box-shadow: 0 0 40px rgba(99, 102, 241, 0.4);
  animation: pop-in 0.5s cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
}

@keyframes pop-in {
  from { transform: scale(0.5); opacity: 0; }
  to   { transform: scale(1);   opacity: 1; }
}

.completed-title {
  font-size: 26px;
  font-weight: 900;
  color: #fff;
  margin-bottom: 10px;
  letter-spacing: -0.5px;
}

.completed-text {
  font-size: 16px;
  color: #888;
  margin-bottom: 6px;
  line-height: 1.5;
}

.completed-sub {
  font-size: 13px;
  color: #444;
  font-style: italic;
}

/* ── Empty view ── */
.empty-view {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  color: #444;
  font-size: 14px;
}

/* ── Modal overlay ── */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.75);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 100;
  backdrop-filter: blur(4px);
}

.dialog {
  background: #161619;
  border-radius: 22px;
  padding: 26px 24px;
  width: 88%;
  max-width: 400px;
  border: 1px solid #2a2a35;
  box-shadow: 0 32px 80px rgba(0, 0, 0, 0.6);
}

.dialog-title {
  font-size: 18px;
  font-weight: 800;
  color: #fff;
  margin-bottom: 18px;
  text-align: center;
  letter-spacing: -0.3px;
}

.template-list {
  max-height: 280px;
  overflow-y: auto;
}

.template-list::-webkit-scrollbar { width: 4px; }
.template-list::-webkit-scrollbar-thumb { background: #2a2a35; border-radius: 4px; }

.loading-templates {
  display: flex;
  justify-content: center;
  padding: 20px;
}

.template-item-wrapper {
  display: flex;
  align-items: center;
  margin-bottom: 10px;
  gap: 8px;
}

.template-item {
  flex: 1;
  background: #0d0d0f;
  border: 1px solid #2a2a35;
  border-radius: 14px;
  padding: 14px 16px;
  cursor: pointer;
  transition: border-color 0.2s, background 0.2s;
}

.template-item:hover {
  background: #111115;
  border-color: var(--ops-blue);
}

.template-item-name {
  font-size: 15px;
  font-weight: 700;
  color: #f0f0f8;
  margin-bottom: 2px;
}

.template-item-desc {
  font-size: 13px;
  color: #555;
  line-height: 1.4;
}

.delete-template-btn {
  padding: 8px;
  background: none;
  border: none;
  font-size: 18px;
  cursor: pointer;
  opacity: 0.6;
  transition: opacity 0.2s;
  flex-shrink: 0;
}
.delete-template-btn:hover { opacity: 1; }

.no-templates {
  color: #444;
  text-align: center;
  padding: 24px;
  font-size: 14px;
}

.dialog-close-btn {
  margin-top: 16px;
  padding: 14px;
  background: #1e1e26;
  border: 1px solid #2a2a35;
  border-radius: 14px;
  width: 100%;
  color: #aaa;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s, color 0.2s;
  font-family: 'Inter', sans-serif;
}
.dialog-close-btn:hover { background: #252530; color: #fff; }

/* ── Spinner ── */
.loading-spinner-small {
  width: 20px;
  height: 20px;
  border: 2.5px solid rgba(255, 255, 255, 0.25);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
