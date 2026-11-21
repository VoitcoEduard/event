<template>
  <div class="form-editor" :class="{ 'form-editor--page-flow': pageFlow }">
    <div class="scroll-content">
      <div v-if="localTimeline.length === 0" class="empty-state">
        <AppIcon class="empty-icon" name="edit" :size="42" aria-hidden="true" />
        <p class="empty-text">{{ $t('form_editor.empty_text') }}</p>
      </div>
      <div v-else>
        <div v-for="(step, index) in localTimeline" :key="step.id" class="step-card">
          <div class="step-header" :class="{ 'field-step': step.type === 'field', 'message-step': step.type === 'message' }">
            <div class="step-title-container">
              <AppIcon class="step-icon" :name="step.type === 'field' ? 'question' : 'message'" :size="18" aria-hidden="true" />
              <span class="step-index">{{ $t('form_module.step_counter', { current: index + 1, total: localTimeline.length }) }}</span>
              <span class="step-type-label">{{ step.type === 'field' ? $t('form_editor.question') : $t('form_editor.message') }}</span>
            </div>
            <div class="step-actions">
              <button
                type="button"
                class="action-icon"
                @click="moveStep(index, 'up')"
                :disabled="index === 0"
                :title="$t('form_editor.move_up')"
                :aria-label="$t('form_editor.move_up')"
              >
                <AppIcon name="chevronUp" :size="16" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="action-icon"
                @click="moveStep(index, 'down')"
                :disabled="index === localTimeline.length - 1"
                :title="$t('form_editor.move_down')"
                :aria-label="$t('form_editor.move_down')"
              >
                <AppIcon name="chevronDown" :size="16" aria-hidden="true" />
              </button>
              <button
                type="button"
                class="action-icon delete-btn"
                @click="removeStep(step.id)"
                :title="$t('form_editor.delete_step')"
                :aria-label="$t('form_editor.delete_step')"
              >
                <AppIcon name="trash" :size="16" aria-hidden="true" />
              </button>
            </div>
          </div>

          <div class="step-body">
            <template v-if="step.type === 'field'">
              <p class="label">{{ $t('form_editor.question_label') }}</p>
              <input
                :value="step.data.label"
                @input="updateStep(step.id, 'label', $event.target.value)"
                :placeholder="$t('form_editor.question_placeholder')"
                :aria-label="$t('form_editor.question_label')"
                class="input"
              />

              <p class="label">{{ $t('form_editor.response_type') }}</p>
              <div class="type-selector">
                <button
                  v-for="type in ['text', 'number', 'date', 'radio', 'checkbox', 'dropdown']"
                  :key="type"
                  type="button"
                  class="type-option"
                  :class="{ active: step.data.fieldType === type }"
                  :aria-pressed="step.data.fieldType === type"
                  @click="updateStep(step.id, 'fieldType', type)"
                >
                  {{ $t(`form_editor.types.${type}`) }}
                </button>
              </div>

              <!-- Predefined Answers Section for Selection Types -->
              <template v-if="['radio', 'checkbox', 'dropdown'].includes(step.data.fieldType)">
                <div class="predefined-answers-section">
                  <p class="label">{{ $t('form_editor.predefined_answers') }}</p>
                  
                  <div class="answers-list">
                    <div 
                      v-for="(answer, answerIndex) in (step.data.predefinedAnswers || [])" 
                      :key="answer.id" 
                      class="answer-item"
                    >
                      <input
                        :value="answer.text"
                        @input="updateAnswer(step.id, answerIndex, 'text', $event.target.value)"
                        :placeholder="$t('form_editor.display_text')"
                        :aria-label="$t('form_editor.display_text')"
                        class="input answer-input"
                      />
                      <input
                        :value="answer.value"
                        @input="updateAnswer(step.id, answerIndex, 'value', $event.target.value)"
                        :placeholder="$t('form_editor.value')"
                        :aria-label="$t('form_editor.value')"
                        class="input answer-value-input"
                      />
                      <button
                        type="button"
                        class="remove-answer-btn"
                        @click="removeAnswer(step.id, answerIndex)"
                        aria-label="Remove answer"
                      >
                        <AppIcon name="close" :size="14" aria-hidden="true" />
                      </button>
                    </div>
                  </div>
                  
                  <button type="button" class="add-answer-btn" @click="addAnswer(step.id)">
                    {{ $t('form_editor.add_answer') }}
                  </button>

                  <!-- Allow Multiple Selections (Checkbox only) -->
                  <div v-if="step.data.fieldType === 'checkbox'" class="option-toggle">
                    <label class="toggle-label">
                      <input
                        type="checkbox"
                        :checked="step.data.allowMultiple"
                        @change="updateStep(step.id, 'allowMultiple', $event.target.checked)"
                      />
                      <span>{{ $t('form_editor.allow_multiple') }}</span>
                    </label>
                  </div>

                  <!-- Allow 'Other' Option -->
                  <div class="option-toggle">
                    <label class="toggle-label">
                      <input
                        type="checkbox"
                        :checked="step.data.allowOther"
                        @change="updateStep(step.id, 'allowOther', $event.target.checked)"
                      />
                      <span>{{ $t('form_editor.allow_other') }}</span>
                    </label>
                  </div>

                  <!-- Other Option Placeholder -->
                  <div v-if="step.data.allowOther" class="other-option-section">
                    <p class="label">{{ $t('form_editor.other_placeholder_label') }}</p>
                    <input
                      :value="step.data.otherPlaceholder || $t('form_module.other_placeholder')"
                      @input="updateStep(step.id, 'otherPlaceholder', $event.target.value)"
                      :placeholder="$t('form_module.other_placeholder')"
                      :aria-label="$t('form_editor.other_placeholder_label')"
                      class="input"
                    />
                  </div>
                </div>
              </template>
            </template>

            <template v-else>
              <p class="label">{{ $t('form_editor.message_content') }}</p>
              <textarea
                :value="step.data.content"
                @input="updateStep(step.id, 'content', $event.target.value)"
                :placeholder="$t('form_editor.message_placeholder')"
                :aria-label="$t('form_editor.message_content')"
                class="input text-area"
              ></textarea>
            </template>
          </div>
        </div>
      </div>
      <div class="modal-bottom-spacer"></div>
    </div>

    <div class="action-bar">
      <div class="add-buttons">
        <button type="button" class="add-btn add-question" @click="addStep('field')">
          {{ $t('form_editor.add_question_btn') }}
        </button>
        <button type="button" class="add-btn add-message" @click="addStep('message')">
          {{ $t('form_editor.add_message_btn') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { extractLabel } from '../utils/labelExtractor'
import { useDialog } from '../composables/useDialog'
import AppIcon from './AppIcon.vue'

const { t } = useI18n()
const { confirm: confirmDialog } = useDialog()
const props = defineProps({
  timeline: { type: Array, default: () => [] },
  pageFlow: { type: Boolean, default: false }
})

const emit = defineEmits(['update:timeline'])

const localTimeline = ref([])

// Only sync from props when the array reference itself changes (e.g. loading a
// template or opening the modal). Deep-watching props and also emitting deep
// changes back would create an infinite reactive loop.
watch(() => props.timeline, (newVal) => {
  if (newVal === localTimeline.value) return
  localTimeline.value = JSON.parse(JSON.stringify(newVal || []))
}, { immediate: true })

// Helper: emit the current localTimeline to the parent after each mutation.
const emitTimeline = () => {
  emit('update:timeline', localTimeline.value)
}

const addStep = (type) => {
  // For fields, generate a default label based on type
  const defaultLabels = {
    text: t('form_module.type_answer'),
    number: t('form_module.type_number'),
    date: t('form_editor.date_label'),
    radio: t('form_editor.radio_label'),
    checkbox: t('form_editor.checkbox_label'),
    dropdown: t('form_module.select_option')
  }
  const defaultQuestion = defaultLabels.text
  
  const newStep = {
    id: Date.now().toString(),
    type,
    data: type === 'field'
      ? { label: defaultQuestion, fieldType: 'text' }
      : { content: t('form_editor.welcome_msg') }
  }
  
  // Auto-generate compact label for the field
  if (type === 'field') {
    newStep.data.compactLabel = extractLabel(defaultQuestion)
  }
  
  localTimeline.value = [...localTimeline.value, newStep]
  emitTimeline()
}

const updateStep = (id, key, value) => {
  localTimeline.value = localTimeline.value.map(step => {
    if (step.id === id) {
      const updates = { [key]: value }
      
      // Auto-generate compact label when question/label changes
      if (key === 'label' && value) {
        updates.compactLabel = extractLabel(value)
      }
      
      return { ...step, data: { ...step.data, ...updates } }
    }
    return step
  })
  emitTimeline()
}

const addAnswer = (stepId) => {
  localTimeline.value = localTimeline.value.map(step => {
    if (step.id === stepId) {
      const answers = step.data.predefinedAnswers || []
      return {
        ...step,
        data: {
          ...step.data,
          predefinedAnswers: [
            ...answers,
            { id: Date.now().toString() + Math.random().toString(36).substr(2, 9), text: '', value: '' }
          ]
        }
      }
    }
    return step
  })
  emitTimeline()
}

const updateAnswer = (stepId, answerIndex, key, value) => {
  localTimeline.value = localTimeline.value.map(step => {
    if (step.id === stepId) {
      const answers = [...(step.data.predefinedAnswers || [])]
      answers[answerIndex] = { ...answers[answerIndex], [key]: value }
      return { ...step, data: { ...step.data, predefinedAnswers: answers } }
    }
    return step
  })
  emitTimeline()
}

const removeAnswer = (stepId, answerIndex) => {
  localTimeline.value = localTimeline.value.map(step => {
    if (step.id === stepId) {
      const answers = [...(step.data.predefinedAnswers || [])]
      answers.splice(answerIndex, 1)
      return { ...step, data: { ...step.data, predefinedAnswers: answers } }
    }
    return step
  })
  emitTimeline()
}

const removeStep = async (id) => {
  const confirmed = await confirmDialog({
    title: t('form_editor.delete_step_title'),
    message: t('form_editor.delete_step_confirm'),
    confirmLabel: t('form_editor.delete_step_btn'),
    cancelLabel: t('common.cancel'),
    destructive: true
  })
  if (!confirmed) return
  localTimeline.value = localTimeline.value.filter(step => step.id !== id)
  emitTimeline()
}

const moveStep = (index, direction) => {
  if (direction === 'up' && index > 0) {
    const newTimeline = [...localTimeline.value]
    ;[newTimeline[index], newTimeline[index - 1]] = [newTimeline[index - 1], newTimeline[index]]
    localTimeline.value = newTimeline
    emitTimeline()
  } else if (direction === 'down' && index < localTimeline.value.length - 1) {
    const newTimeline = [...localTimeline.value]
    ;[newTimeline[index], newTimeline[index + 1]] = [newTimeline[index + 1], newTimeline[index]]
    localTimeline.value = newTimeline
    emitTimeline()
  }
}
</script>

<style scoped>
.form-editor {
  display: flex;
  flex-direction: column;
  min-height: 0;
  min-width: 0;
  width: 100%;
  box-sizing: border-box;
  container: form-editor / inline-size;
}

.form-editor *,
.form-editor *::before,
.form-editor *::after {
  box-sizing: border-box;
}

.scroll-content {
  flex: 1;
  min-height: 0;
  overflow-y: visible;
  padding: 0 0 24px;
}

.modal-bottom-spacer {
  display: none;
}

:global(.modal-container) .form-editor:not(.form-editor--page-flow) .scroll-content {
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 20px;
  padding-bottom: 80px;
}

:global(.modal-container) .form-editor:not(.form-editor--page-flow) .modal-bottom-spacer {
  display: block;
  height: 100px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 100px 20px;
  opacity: 0.5;
}

.empty-icon { font-size: 48px; margin-bottom: 16px; }
.empty-text { color: #8E8E93; font-size: 16px; }

.step-card {
  background-color: #2C2C2E;
  border-radius: 16px;
  margin-bottom: 16px;
  overflow: hidden;
  border: 1px solid #3A3A3C;
}

.step-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background-color: #252527;
  border-left: 4px solid;
}

.step-header.field-step { border-left-color: #0A84FF; }
.step-header.message-step { border-left-color: #FF9F0A; }

.step-title-container {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  min-width: 0;
}

.step-icon { font-size: 18px; }
.step-index { font-size: 12px; color: #8E8E93; font-weight: 600; text-transform: uppercase; }
.step-type-label { font-size: 16px; font-weight: 600; color: #FFF; }

.step-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.action-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 8px;
  border-radius: 6px;
  background-color: #333;
  font-size: 14px;
}
.action-icon:disabled { opacity: 0.3; }
.delete-btn { background-color: rgba(255, 69, 58, 0.1); }

.step-body { padding: 16px; }

.label {
  font-size: 12px;
  color: #8E8E93;
  margin-bottom: 8px;
  font-weight: 600;
}

.input {
  background-color: #121212;
  border: 1px solid #333;
  border-radius: 8px;
  padding: 12px 16px;
  color: white;
  font-size: 16px;
  width: 100%;
  min-inline-size: 0;
  box-sizing: border-box;
  margin-bottom: 12px;
}

.input.text-area { min-height: 80px; resize: vertical; }

.type-selector {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.type-option {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 8px 16px;
  border-radius: 8px;
  background-color: #333;
  color: #888;
  font-size: 14px;
  font-weight: 600;
}
.type-option.active { background-color: #007AFF; color: #FFF; }

.predefined-answers-section {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #3A3A3C;
}

.answers-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
}

.answer-item { display: flex; gap: 8px; align-items: center; }
.answer-input { flex: 2; min-inline-size: 0; box-sizing: border-box; margin-bottom: 0; }
.answer-value-input { flex: 1; min-inline-size: 0; box-sizing: border-box; margin-bottom: 0; }

.remove-answer-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  padding: 8px 12px;
  border-radius: 8px;
  background-color: rgba(255, 69, 58, 0.2);
  color: #FF453A;
  font-size: 14px;
  font-weight: 600;
}

.add-answer-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 10px 16px;
  border-radius: 8px;
  background-color: rgba(52, 199, 89, 0.1);
  color: #34C759;
  font-size: 14px;
  font-weight: 600;
  width: 100%;
  margin-bottom: 16px;
}

.option-toggle { margin-top: 12px; }
.toggle-label {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  color: #FFF;
  font-size: 14px;
}
.toggle-label input[type="checkbox"] {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  accent-color: #007AFF;
}
.other-option-section { margin-top: 12px; }

.action-bar {
  display: flex;
  justify-content: flex-start;
  gap: 12px;
  padding: 16px 20px;
  background-color: #1C1C1E;
  border-top: 1px solid #2C2C2E;
}

.add-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.add-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 10px 16px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
}
.add-question { background-color: rgba(0, 122, 255, 0.1); color: #007AFF; }
.add-message { background-color: rgba(255, 159, 10, 0.1); color: #FF9F0A; }

:global(html[data-admin-appearance="true"] .form-editor .action-bar) {
  background: var(--admin-surface) !important;
  border-color: var(--admin-border) !important;
}

:global(html[data-admin-appearance="true"] .form-editor .add-btn.add-question) {
  background: var(--admin-primary-solid) !important;
  border-color: var(--admin-primary-solid) !important;
  color: #fff !important;
}

:global(html[data-admin-appearance="true"] .form-editor .add-btn.add-message) {
  background: var(--admin-warning-soft) !important;
  border: 1px solid color-mix(in srgb, var(--admin-warning) 35%, var(--admin-border)) !important;
  color: var(--admin-warning) !important;
}

@container form-editor (inline-size <= 540px) {
  :global(.modal-container) .form-editor:not(.form-editor--page-flow) .scroll-content {
    padding: 12px;
    padding-bottom: 84px;
  }

  .step-header {
    flex-wrap: wrap;
    gap: 8px;
    padding: 12px;
  }

  .step-actions {
    margin-inline-start: auto;
  }

  .step-body {
    padding: 12px;
  }

  .answer-item {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }

  .answer-input,
  .answer-value-input {
    flex: none;
    width: 100%;
    min-inline-size: 0;
    box-sizing: border-box;
  }

  .remove-answer-btn {
    width: 100%;
  }

  .action-bar {
    padding: 12px;
  }

  .add-buttons {
    width: 100%;
    flex-wrap: wrap;
    gap: 8px;
  }

  .add-btn {
    flex: 1 1 130px;
    min-inline-size: 0;
  }
}

@media (pointer: coarse) {
  .action-icon,
  .type-option,
  .remove-answer-btn,
  .add-answer-btn,
  .add-btn,
  .toggle-label {
    min-block-size: var(--target-size, 44px);
    min-inline-size: var(--target-size, 44px);
  }
}
</style>
