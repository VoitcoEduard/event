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
                        v-for="type in ['text', 'number', 'date']" 
                        :key="type"
                        class="type-option"
                        :class="{ active: step.data.fieldType === type }"
                        @click="updateStep(step.id, 'fieldType', type)"
                      >
                        {{ type.charAt(0).toUpperCase() + type.slice(1) }}
                      </button>
                    </div>
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
              <button class="delete-template-btn" @click="deleteTemplate(template.id, template.name)">
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

import { API_BASE_URL } from '../utils'

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
    alert('Your form has no steps. Please add at least one.')
    return
  }

  loading.value = true
  try {
    const response = await fetch(`${API_URL}/events/${props.eventId}/modules/${props.module.id}/publish`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ data: { timeline: timeline.value } })
    })

    if (response.ok) {
      alert('Form configuration published successfully!')
      emit('update')
      handleClose()
    } else {
      alert('Failed to publish configuration.')
    }
  } catch (error) {
    console.error('Error publishing:', error)
    alert('Failed to publish configuration.')
  } finally {
    loading.value = false
  }
}

const fetchTemplates = async () => {
  try {
    const response = await fetch(`${API_URL}/templates`)
    if (response.ok) {
      const data = await response.json()
      templates.value = data || []
    }
  } catch (error) {
    console.error('Error fetching templates:', error)
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
    const response = await fetch(`${API_URL}/templates/${id}`, { method: 'DELETE' })
    if (response.ok) {
      fetchTemplates()
    } else {
      alert('Failed to delete template.')
    }
  } catch (error) {
    console.error(error)
    alert('Failed to delete template.')
  }
}

const openSaveTemplateModal = () => {
  showSaveTemplateModal.value = true
}

const saveAsTemplate = async () => {
  if (!templateName.value.trim()) {
    alert('Please enter a template name.')
    return
  }

  try {
    const response = await fetch(`${API_URL}/templates`, {
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
      alert('Template saved!')
      showSaveTemplateModal.value = false
      templateName.value = ''
    }
  } catch (error) {
    console.error(error)
    alert('Failed to save template.')
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
  text-transform: uppercase;
  font-weight: 600;
}

.input {
  background-color: #1C1C1E;
  border-radius: 8px;
  padding: 12px;
  font-size: 16px;
  color: #FFF;
  border: 1px solid #3A3A3C;
  margin-bottom: 16px;
  width: 100%;
}

.input.text-area {
  min-height: 80px;
  resize: vertical;
}

.type-selector {
  display: flex;
  background-color: #1C1C1E;
  border-radius: 8px;
  padding: 4px;
  border: 1px solid #3A3A3C;
}

.type-option {
  flex: 1;
  padding: 8px;
  border-radius: 6px;
  font-size: 14px;
  color: #8E8E93;
  font-weight: 500;
  background: none;
}

.type-option.active {
  background-color: #3A3A3C;
  color: #FFF;
  font-weight: 600;
}

.action-bar {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  background-color: #1C1C1E;
  border-top: 1px solid #2C2C2E;
  padding: 16px;
  padding-bottom: 32px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.add-buttons {
  display: flex;
  gap: 12px;
}

.add-btn {
  padding: 10px 16px;
  border-radius: 10px;
  font-weight: 600;
  font-size: 14px;
  color: #FFF;
}

.add-question {
  background-color: #0A84FF;
}

.add-message {
  background-color: #FF9F0A;
}

.publish-btn {
  background-color: #30D158;
  padding: 12px 24px;
  border-radius: 10px;
  color: #FFF;
  font-weight: bold;
  font-size: 15px;
}

.dialog {
  background-color: #1C1C1E;
  border-radius: 16px;
  padding: 24px;
  width: 85%;
  max-width: 400px;
  border: 1px solid #3A3A3C;
}

.dialog-title {
  font-size: 20px;
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
  margin-bottom: 8px;
}

.template-item {
  flex: 1;
  background-color: #2C2C2E;
  border-radius: 8px;
  padding: 12px;
  cursor: pointer;
}

.template-item-name {
  font-size: 16px;
  color: #FFF;
  font-weight: 600;
}

.template-item-desc {
  font-size: 14px;
  color: #8E8E93;
  margin-top: 4px;
}

.delete-template-btn {
  padding: 12px;
  background-color: rgba(255, 69, 58, 0.1);
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.no-templates {
  color: #666;
  text-align: center;
  padding: 20px;
}

.dialog-close-btn {
  margin-top: 16px;
  padding: 12px;
  background-color: #3A3A3C;
  border-radius: 8px;
  width: 100%;
  color: #FFF;
  font-weight: 600;
}

.dialog-input {
  background-color: #2C2C2E;
  color: #FFF;
  padding: 12px;
  border-radius: 8px;
  border: 1px solid #3A3A3C;
  width: 100%;
  margin-bottom: 16px;
}

.dialog-buttons {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
}

.dialog-btn {
  padding: 10px 16px;
  border-radius: 8px;
  font-weight: 600;
}

.dialog-btn.cancel {
  background-color: #333;
  color: #FFF;
}

.dialog-btn.save {
  background-color: #0A84FF;
  color: #FFF;
}
</style>
