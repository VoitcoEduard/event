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
            <button class="icon-button" @click="openTemplateModal" title="Load Template">
              📥
            </button>
            <button class="library-btn" @click="openSaveTemplateModal">
               💾 Save as Template
            </button>
            <button class="close-button" @click="handleClose">
              ✕
            </button>
          </div>
        </div>

        <div class="content">
          <FormEditor 
            :timeline="timeline"
            @update:timeline="updateTimeline"
          />

          <div class="action-bar">
            <div class="status-text">
                {{ timeline.length }} Steps
            </div>
            <button class="publish-btn" @click="handlePublish" :disabled="loading">
              {{ loading ? 'Saving...' : 'Save Changes' }}
            </button>
          </div>
        </div>
      </div>

      <!-- Load Template Modal -->
      <div v-if="showTemplateModal" class="modal-overlay" @click.self="showTemplateModal = false">
        <div class="dialog">
          <h3 class="dialog-title">Start from Template</h3>
          <div class="template-list">
            <div v-if="loadingTemplates">Loading...</div>
            <div v-else-if="templates.length === 0">No templates found.</div>
            <div v-else v-for="template in templates" :key="template.id" class="template-item-wrapper">
              <div class="template-item" @click="loadTemplate(template)">
                <p class="template-item-name">{{ template.name }}</p>
                <p class="template-item-desc">{{ template.description }}</p>
              </div>
            </div>
          </div>
          <button class="dialog-close-btn" @click="showTemplateModal = false">
            Cancel
          </button>
        </div>
      </div>

      <!-- Save Template Modal -->
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
import FormEditor from './FormEditor.vue'

const props = defineProps({
  visible: { type: Boolean, default: false },
  module: { type: Object, default: null },
  eventId: { type: String, default: '' }
})

const emit = defineEmits(['close', 'update'])

const timeline = ref([])
const loading = ref(false)
const templates = ref([])
const loadingTemplates = ref(false)
const showTemplateModal = ref(false)
const templateName = ref('')
const showSaveTemplateModal = ref(false)

const { success, error, info } = useToast()

watch(() => [props.visible, props.module], ([visible, mod]) => {
  if (visible && mod?.data?.timeline) {
    timeline.value = JSON.parse(JSON.stringify(mod.data.timeline))
  } else if (visible) {
    timeline.value = []
  }
}, { immediate: true })

const updateTimeline = (newTimeline) => {
    timeline.value = newTimeline
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
  loadingTemplates.value = true
  try {
    const response = await fetch(`${API_BASE_URL}/templates`)
    if (response.ok) {
      const data = await response.json()
      // Filter for form templates
      templates.value = data.filter(t => t.moduleType === 'form' || (!t.moduleType && !t.data?.questions))
    }
  } catch (err) {
    console.error('Error fetching templates:', err)
  } finally {
      loadingTemplates.value = false
  }
}

const openTemplateModal = () => {
  fetchTemplates()
  showTemplateModal.value = true
}

const loadTemplate = (template) => {
  if (!confirm('This will replace your current steps. Continue?')) return
  if (template.data?.timeline) {
    timeline.value = JSON.parse(JSON.stringify(template.data.timeline))
  }
  showTemplateModal.value = false
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
    } else {
        error('Failed to save template')
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

.header-buttons { display: flex; gap: 12px; align-items: center; }

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

.dialog {
  background-color: #1C1C1E;
  border-radius: 16px;
  padding: 20px;
  width: 85%;
  max-width: 400px;
  margin-bottom: 20%;
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
  margin-bottom: 16px;
}

.template-item-wrapper {
  margin-bottom: 8px;
}

.template-item {
  background-color: #2C2C2E;
  padding: 12px;
  border-radius: 8px;
  cursor: pointer;
  border: 1px solid transparent;
}
.template-item:hover { border-color: #0A84FF; }

.template-item-name { color: white; font-weight: 600; }
.template-item-desc { color: #888; font-size: 12px; margin: 0; }

.dialog-close-btn {
  width: 100%;
  padding: 12px;
  background-color: #333;
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-weight: 600;
}

.dialog-input {
    background-color: #121212;
    border: 1px solid #333;
    border-radius: 8px;
    padding: 12px;
    color: white;
    width: 100%;
    margin-bottom: 16px;
}

.dialog-buttons { display: flex; gap: 10px; }
.dialog-btn { flex: 1; padding: 12px; border-radius: 8px; border: none; cursor: pointer; font-weight: 600; }
.cancel { background-color: #333; color: white; }
.save { background-color: #30D158; color: white; }



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
