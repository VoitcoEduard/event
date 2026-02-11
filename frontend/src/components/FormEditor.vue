<template>
  <div class="form-editor">
    <div class="scroll-content">
      <div v-if="localTimeline.length === 0" class="empty-state">
        <p class="empty-icon">✏️</p>
        <p class="empty-text">Start by adding a question or message</p>
      </div>
      <div v-else>
        <div v-for="(step, index) in localTimeline" :key="step.id" class="step-card">
          <div class="step-header" :class="{ 'field-step': step.type === 'field', 'message-step': step.type === 'message' }">
            <div class="step-title-container">
              <span class="step-icon">{{ step.type === 'field' ? '❓' : '💬' }}</span>
              <span class="step-index">Step {{ index + 1 }}</span>
              <span class="step-type-label">{{ step.type === 'field' ? 'Question' : 'Message' }}</span>
            </div>
            <div class="step-actions">
              <button 
                class="action-icon" 
                @click="moveStep(index, 'up')"
                :disabled="index === 0"
              >
                ↑
              </button>
              <button 
                class="action-icon" 
                @click="moveStep(index, 'down')"
                :disabled="index === localTimeline.length - 1"
              >
                ↓
              </button>
              <button class="action-icon delete-btn" @click="removeStep(step.id)">
                🗑
              </button>
            </div>
          </div>

          <div class="step-body">
            <template v-if="step.type === 'field'">
              <p class="label">Question / Label</p>
              <input
                :value="step.data.label"
                @input="updateStep(step.id, 'label', $event.target.value)"
                placeholder="e.g. What is your name?"
                class="input"
              />

              <p class="label">Response Type</p>
              <div class="type-selector">
                <button 
                  v-for="type in ['text', 'number', 'date', 'radio', 'checkbox', 'dropdown']" 
                  :key="type"
                  class="type-option"
                  :class="{ active: step.data.fieldType === type }"
                  @click="updateStep(step.id, 'fieldType', type)"
                >
                  {{ type.charAt(0).toUpperCase() + type.slice(1) }}
                </button>
              </div>

              <!-- Predefined Answers Section for Selection Types -->
              <template v-if="['radio', 'checkbox', 'dropdown'].includes(step.data.fieldType)">
                <div class="predefined-answers-section">
                  <p class="label">Predefined Answers</p>
                  
                  <div class="answers-list">
                    <div 
                      v-for="(answer, answerIndex) in (step.data.predefinedAnswers || [])" 
                      :key="answer.id" 
                      class="answer-item"
                    >
                      <input
                        :value="answer.text"
                        @input="updateAnswer(step.id, answerIndex, 'text', $event.target.value)"
                        placeholder="Display text"
                        class="input answer-input"
                      />
                      <input
                        :value="answer.value"
                        @input="updateAnswer(step.id, answerIndex, 'value', $event.target.value)"
                        placeholder="Value"
                        class="input answer-value-input"
                      />
                      <button 
                        class="remove-answer-btn" 
                        @click="removeAnswer(step.id, answerIndex)"
                        title="Remove answer"
                      >
                        ✕
                      </button>
                    </div>
                  </div>
                  
                  <button class="add-answer-btn" @click="addAnswer(step.id)">
                    + Add Answer
                  </button>

                  <!-- Allow Multiple Selections (Checkbox only) -->
                  <div v-if="step.data.fieldType === 'checkbox'" class="option-toggle">
                    <label class="toggle-label">
                      <input
                        type="checkbox"
                        :checked="step.data.allowMultiple"
                        @change="updateStep(step.id, 'allowMultiple', $event.target.checked)"
                      />
                      <span>Allow multiple selections</span>
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
                      <span>Allow 'Other' option</span>
                    </label>
                  </div>

                  <!-- Other Option Placeholder -->
                  <div v-if="step.data.allowOther" class="other-option-section">
                    <p class="label">Other option placeholder</p>
                    <input
                      :value="step.data.otherPlaceholder || 'Please specify...'"
                      @input="updateStep(step.id, 'otherPlaceholder', $event.target.value)"
                      placeholder="e.g., Please specify..."
                      class="input"
                    />
                  </div>
                </div>
              </template>
            </template>

            <template v-else>
              <p class="label">Message Content</p>
              <textarea
                :value="step.data.content"
                @input="updateStep(step.id, 'content', $event.target.value)"
                placeholder="Enter your message here..."
                class="input text-area"
              ></textarea>
            </template>
          </div>
        </div>
      </div>
      <div style="height: 100px;"></div>
    </div>

    <div class="action-bar">
      <div class="add-buttons">
        <button class="add-btn add-question" @click="addStep('field')">
          + Question
        </button>
        <button class="add-btn add-message" @click="addStep('message')">
          + Message
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, defineProps, defineEmits } from 'vue'

const props = defineProps({
  timeline: { type: Array, default: () => [] }
})

const emit = defineEmits(['update:timeline'])

const localTimeline = ref([])

watch(() => props.timeline, (newVal) => {
  localTimeline.value = JSON.parse(JSON.stringify(newVal))
}, { immediate: true, deep: true })

watch(localTimeline, (newVal) => {
    emit('update:timeline', newVal)
}, { deep: true })

const addStep = (type) => {
  const newStep = {
    id: Date.now().toString(),
    type,
    data: type === 'field'
      ? { label: 'New Question', fieldType: 'text' }
      : { content: 'Welcome to the form!' }
  }
  localTimeline.value = [...localTimeline.value, newStep]
}

const updateStep = (id, key, value) => {
  localTimeline.value = localTimeline.value.map(step =>
    step.id === id ? { ...step, data: { ...step.data, [key]: value } } : step
  )
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
}

const removeStep = (id) => {
  if (!confirm('Are you sure you want to remove this step?')) return
  localTimeline.value = localTimeline.value.filter(step => step.id !== id)
}

const moveStep = (index, direction) => {
  if (direction === 'up' && index > 0) {
    const newTimeline = [...localTimeline.value]
    ;[newTimeline[index], newTimeline[index - 1]] = [newTimeline[index - 1], newTimeline[index]]
    localTimeline.value = newTimeline
  } else if (direction === 'down' && index < localTimeline.value.length - 1) {
    const newTimeline = [...localTimeline.value]
    ;[newTimeline[index], newTimeline[index + 1]] = [newTimeline[index + 1], newTimeline[index]]
    localTimeline.value = newTimeline
  }
}
</script>

<style scoped>
.form-editor {
    display: flex;
    flex-direction: column;
    height: 100%;
}

.scroll-content {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  padding-bottom: 80px;
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
  padding: 16px;
  background-color: #252527;
  border-left: 4px solid;
}

.step-header.field-step { border-left-color: #0A84FF; }
.step-header.message-step { border-left-color: #FF9F0A; }

.step-title-container {
  display: flex;
  align-items: center;
  gap: 12px;
}

.step-icon { font-size: 18px; }
.step-index { font-size: 12px; color: #8E8E93; font-weight: 600; text-transform: uppercase; }
.step-type-label { font-size: 16px; font-weight: 600; color: #FFF; }

.step-actions { display: flex; gap: 8px; }

.action-icon {
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
  margin-bottom: 12px;
}

.input.text-area { min-height: 80px; resize: vertical; }

.type-selector { display: flex; gap: 8px; }

.type-option {
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
.answer-input { flex: 2; margin-bottom: 0; }
.answer-value-input { flex: 1; margin-bottom: 0; }

.remove-answer-btn {
  padding: 8px 12px;
  border-radius: 8px;
  background-color: rgba(255, 69, 58, 0.2);
  color: #FF453A;
  font-size: 14px;
  font-weight: 600;
}

.add-answer-btn {
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
.toggle-label input[type="checkbox"] { width: 18px; height: 18px; accent-color: #007AFF; }
.other-option-section { margin-top: 12px; }

.action-bar {
  display: flex;
  justify-content: flex-start;
  gap: 12px;
  padding: 16px 20px;
  background-color: #1C1C1E;
  border-top: 1px solid #2C2C2E;
}

.add-btn {
  padding: 10px 16px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
}
.add-question { background-color: rgba(0, 122, 255, 0.1); color: #007AFF; }
.add-message { background-color: rgba(255, 159, 10, 0.1); color: #FF9F0A; }
</style>
