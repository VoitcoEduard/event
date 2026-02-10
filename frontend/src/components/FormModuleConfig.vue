<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click.self="handleClose">
      <div class="modal-container" :style="modalStyle">
        <div class="header">
          <div>
            <p class="title">Form Editor</p>
            <p class="subtitle">{{ module?.name || 'Untitled Form' }}</p>
          </div>
          <div class="header-buttons">
            <button class="icon-button" @click="openTemplateModal">
              📥
            </button>
            <button class="icon-button" @click="openSaveTemplateModal">
              💾
            </button>
            <button class="close-button" @click="handleClose">
              ✕
            </button>
          </div>
        </div>

        <div class="content">
          <div class="scroll-content">
            <div v-if="timeline.length === 0" class="empty-state">
              <p class="empty-icon">✏️</p>
              <p class="empty-text">Start by adding a question or message</p>
            </div>
            <div v-else>
              <div v-for="(step, index) in timeline" :key="step.id" class="step-card">
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
                      :disabled="index === timeline.length - 1"
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
            <button class="publish-btn" @click="handlePublish" :disabled="loading">
              {{ loading ? 'Saving...' : 'Save Changes' }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="showTemplateModal" class="modal-overlay" @click.self="showTemplateModal = false">
        <div class="dialog">
          <h3 class="dialog-title">Start from Template</h3>
          <div class="template-list">
            <div v-for="template in templates" :key="template.id" class="template-item-wrapper">
              <div class="template-item" @click="loadTemplate(template)">
                <p class="template-item-name">{{ template.name }}</p>
                <p class="template-item-desc">{{ template.description }}</p>
              </div>
              <button class="delete-template-btn" @click.stop="deleteTemplate(template.id, template.name)">
                🗑
              </button>
            </div>
            <p v-if="templates.length === 0" class="no-templates">No templates found.</p>
          </div>
          <button class="dialog-close-btn" @click="showTemplateModal = false">
            Cancel
          </button>
        </div>
      </div>

      <div v-if="showSaveTemplateModal" class="modal-overlay" @click.self="showSaveTemplateModal = false">
        <div class="dialog">
          <h3 class="dialog-title">Save as Template</h3>
          <input
            v-model="templateName"
            placeholder="Template Name"
            class="dialog-input"
          />
          <div class="dialog-buttons">
            <button class="dialog-btn cancel" @click="showSaveTemplateModal = false">
              Cancel
            </button>
            <button class="dialog-btn save" @click="saveAsTemplate">
              Save
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

const props = defineProps({
  visible: { type: Boolean, default: false },
  module: { type: Object, default: null },
  eventId: { type: String, default: '' }
})

const emit = defineEmits(['close', 'update'])

const timeline = ref([])
const loading = ref(false)
const templates = ref([])
const showTemplateModal = ref(false)
const templateName = ref('')
const showSaveTemplateModal = ref(false)

const { success, error, info } = useToast()

watch(() => [props.visible, props.module], ([visible, mod]) => {
  if (visible && mod?.data?.timeline) {
    timeline.value = mod.data.timeline
  } else if (visible) {
    timeline.value = []
  }
}, { immediate: true })

const addStep = (type) => {
  const newStep = {
    id: Date.now().toString(),
    type,
    data: type === 'field'
      ? { label: 'New Question', fieldType: 'text' }
      : { content: 'Welcome to the form!' }
  }
  timeline.value = [...timeline.value, newStep]
}

const updateStep = (id, key, value) => {
  timeline.value = timeline.value.map(step =>
    step.id === id ? { ...step, data: { ...step.data, [key]: value } } : step
  )
}

const addAnswer = (stepId) => {
  timeline.value = timeline.value.map(step => {
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
  timeline.value = timeline.value.map(step => {
    if (step.id === stepId) {
      const answers = [...(step.data.predefinedAnswers || [])]
      answers[answerIndex] = { ...answers[answerIndex], [key]: value }
      return { ...step, data: { ...step.data, predefinedAnswers: answers } }
    }
    return step
  })
}

const removeAnswer = (stepId, answerIndex) => {
  timeline.value = timeline.value.map(step => {
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
  timeline.value = timeline.value.filter(step => step.id !== id)
}

const moveStep = (index, direction) => {
  if (direction === 'up' && index > 0) {
    const newTimeline = [...timeline.value]
    ;[newTimeline[index], newTimeline[index - 1]] = [newTimeline[index - 1], newTimeline[index]]
    timeline.value = newTimeline
  } else if (direction === 'down' && index < timeline.value.length - 1) {
    const newTimeline = [...timeline.value]
    ;[newTimeline[index], newTimeline[index + 1]] = [newTimeline[index + 1], newTimeline[index]]
    timeline.value = newTimeline
  }
}

const handlePublish = async () => {
  if (timeline.value.length === 0) {
    info('Your form has no steps. Please add at least one.')
    return
  }

  loading.value = true
  try {
    const response = await fetch(`${API_BASE_URL}/events/${props.eventId}/modules/${props.module.id}/publish`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ data: { timeline: timeline.value } })
    })

    if (response.ok) {
      success('Form configuration published successfully!')
      emit('update')
      handleClose()
    } else {
      error('Failed to publish configuration.')
    }
  } catch (err) {
    console.error('Error publishing:', err)
    error('Failed to publish configuration.')
  } finally {
    loading.value = false
  }
}

const fetchTemplates = async () => {
  try {
    const response = await fetch(`${API_BASE_URL}/templates`)
    if (response.ok) {
      const data = await response.json()
      templates.value = data || []
    }
  } catch (err) {
    console.error('Error fetching templates:', err)
  }
}

const openTemplateModal = () => {
  fetchTemplates()
  showTemplateModal.value = true
}

const loadTemplate = (template) => {
  if (!confirm('This will replace your current steps. Continue?')) return
  if (template.data?.timeline) {
    timeline.value = template.data.timeline
  }
  showTemplateModal.value = false
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

const openSaveTemplateModal = () => {
  showSaveTemplateModal.value = true
}

const saveAsTemplate = async () => {
  if (!templateName.value.trim()) {
    info('Please enter a template name.')
    return
  }

  try {
    const response = await fetch(`${API_BASE_URL}/templates`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: templateName.value,
        description: 'Created from ' + props.module.name,
        moduleType: 'form',
        data: { timeline: timeline.value }
      })
    })
    if (response.ok) {
      success('Template saved!')
      showSaveTemplateModal.value = false
      templateName.value = ''
    }
  } catch (err) {
    console.error(err)
    error('Failed to save template.')
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
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
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

.title {
  font-size: 22px;
  font-weight: bold;
  color: #FFF;
}

.subtitle {
  font-size: 14px;
  color: #8E8E93;
  margin-top: 2px;
}

.header-buttons {
  display: flex;
  gap: 12px;
}

.icon-button {
  padding: 8px;
  border-radius: 8px;
  background-color: #2C2C2E;
  font-size: 20px;
}

.close-button {
  padding: 8px;
  border-radius: 8px;
  background-color: #3A3A3C;
  margin-left: 8px;
}

.content {
  flex: 1;
  position: relative;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.scroll-content {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  padding-bottom: 120px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 100px 20px;
  opacity: 0.5;
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
}

.empty-text {
  color: #8E8E93;
  font-size: 16px;
}

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

.step-header.field-step {
  border-left-color: #0A84FF;
}

.step-header.message-step {
  border-left-color: #FF9F0A;
}

.step-title-container {
  display: flex;
  align-items: center;
  gap: 12px;
}

.step-icon {
  font-size: 18px;
}

.step-index {
  font-size: 12px;
  color: #8E8E93;
  font-weight: 600;
  text-transform: uppercase;
}

.step-type-label {
  font-size: 16px;
  font-weight: 600;
  color: #FFF;
}

.step-actions {
  display: flex;
  gap: 8px;
}

.action-icon {
  padding: 8px;
  border-radius: 6px;
  background-color: #333;
  font-size: 14px;
}

.action-icon:disabled {
  opacity: 0.3;
}

.delete-btn {
  background-color: rgba(255, 69, 58, 0.1);
}

.step-body {
  padding: 16px;
}

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

.input.text-area {
  min-height: 80px;
  resize: vertical;
}

.type-selector {
  display: flex;
  gap: 8px;
}

.type-option {
  padding: 8px 16px;
  border-radius: 8px;
  background-color: #333;
  color: #888;
  font-size: 14px;
  font-weight: 600;
}

.type-option.active {
  background-color: #007AFF;
  color: #FFF;
}

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

.answer-item {
  display: flex;
  gap: 8px;
  align-items: center;
}

.answer-input {
  flex: 2;
  margin-bottom: 0;
}

.answer-value-input {
  flex: 1;
  margin-bottom: 0;
}

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

.option-toggle {
  margin-top: 12px;
}

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
  accent-color: #007AFF;
}

.other-option-section {
  margin-top: 12px;
}

.action-bar {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background-color: #1C1C1E;
  border-top: 1px solid #2C2C2E;
}

.add-buttons {
  display: flex;
  gap: 8px;
}

.add-btn {
  padding: 10px 16px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
}

.add-question {
  background-color: rgba(0, 122, 255, 0.1);
  color: #007AFF;
}

.add-message {
  background-color: rgba(255, 159, 10, 0.1);
  color: #FF9F0A;
}

.publish-btn {
  background-color: #007AFF;
  color: white;
  padding: 12px 24px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
}

.publish-btn:disabled {
  opacity: 0.5;
}

.dialog {
  background-color: #1C1C1E;
  border-radius: 16px;
  padding: 20px;
  width: 85%;
  max-width: 400px;
}

.dialog-title {
  font-size: 18px;
  font-weight: bold;
  color: #FFF;
  margin-bottom: 16px;
}

.template-list {
  max-height: 300px;
  overflow-y: auto;
}

.template-item-wrapper {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}

.template-item {
  flex: 1;
  background-color: #2C2C2E;
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
  color: #8E8E93;
  margin-top: 4px;
}

.delete-template-btn {
  padding: 8px;
  margin-left: 8px;
  background: none;
  font-size: 20px;
}

.no-templates {
  color: #8E8E93;
  text-align: center;
  padding: 20px;
}

.dialog-close-btn {
  margin-top: 16px;
  padding: 14px;
  background-color: #2C2C2E;
  border-radius: 12px;
  width: 100%;
  color: #FFF;
  font-size: 16px;
  font-weight: 600;
}

.dialog-input {
  background-color: #121212;
  border: 1px solid #333;
  border-radius: 8px;
  padding: 14px 16px;
  color: white;
  font-size: 16px;
  width: 100%;
  margin-bottom: 16px;
}

.dialog-buttons {
  display: flex;
  gap: 12px;
}

.dialog-btn {
  flex: 1;
  padding: 14px;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
}

.dialog-btn.cancel {
  background-color: #2C2C2E;
  color: #8E8E93;
}

.dialog-btn.save {
  background-color: #007AFF;
  color: #FFF;
}
</style>
