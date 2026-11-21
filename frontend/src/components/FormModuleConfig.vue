<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay" @click.self="handleClose">
      <div
        ref="modalContainerRef"
        class="modal-container"
        :style="modalStyle"
        role="dialog"
        aria-modal="true"
        :aria-label="$t('form_module.config.title')"
        tabindex="-1"
      >
        <div class="header">
          <div class="header-title-group">
            <p class="title">{{ $t('form_module.config.title') }}</p>
            <p class="subtitle">{{ module?.name || $t('form_module.config.untitled') }}</p>
          </div>
          <div class="header-buttons">
            <button
              type="button"
              class="icon-button"
              @click="openTemplateModal"
              :title="$t('quiz_config.load_library')"
              :aria-label="$t('quiz_config.load_library')"
            >
              <AppIcon name="download" :size="18" aria-hidden="true" />
            </button>
            <button type="button" class="library-btn" @click="openSaveTemplateModal">
               <AppIcon name="copy" :size="18" aria-hidden="true" /> {{ $t('quiz_config.save_template') }}
            </button>
            <button
              type="button"
              class="close-button"
              @click="handleClose"
              :aria-label="$t('common.close')"
            >
              <AppIcon name="close" :size="18" aria-hidden="true" />
            </button>
          </div>
        </div>

        <div class="content">
          <FormEditor 
            :timeline="timeline"
            @update:timeline="updateTimeline"
            class="form-editor-instance"
          />

          <div class="settings-bar">
            <div class="setting-item">
              <label class="checkbox-label">
                <input type="checkbox" v-model="mandatoryOnEntry" />
                {{ $t('form_module.config.mandatory_entry') }}
              </label>
              <p class="setting-desc">{{ $t('form_module.config.mandatory_entry_desc') }}</p>
            </div>
            <div class="setting-item">
              <label class="checkbox-label">
                <input type="checkbox" v-model="hideAfterCompletion" />
                {{ $t('form_module.config.hide_completed') }}
              </label>
              <p class="setting-desc">{{ $t('form_module.config.hide_completed_desc') }}</p>
            </div>
          </div>

          <div class="action-bar">
            <div class="status-text">
                {{ $t('form_module.config.steps_count', { count: timeline.length }) }}
            </div>
            <button type="button" class="publish-btn" @click="handlePublish" :disabled="loading">
              {{ loading ? $t('quiz_config.saving') : $t('form_module.config.save_btn') }}
            </button>
          </div>
        </div>
      </div>

      <!-- Load Template Modal -->
      <div v-if="showTemplateModal" class="modal-overlay dialog-overlay" @click.self="showTemplateModal = false">
        <div
          ref="templateDialogRef"
          class="dialog"
          role="dialog"
          aria-modal="true"
          :aria-label="$t('quiz_config.load_library')"
          tabindex="-1"
        >
          <h3 class="dialog-title">{{ $t('quiz_config.load_library') }}</h3>
          <div class="template-list">
            <div v-if="loadingTemplates">{{ $t('common.loading') }}</div>
            <div v-else-if="templateError" class="template-error">{{ $t('quiz_config.template_load_failed') }}</div>
            <div v-else-if="templates.length === 0">{{ $t('quiz_config.no_templates') }}</div>
            <div v-else v-for="template in templates" :key="template.id" class="template-item-wrapper">
              <div
                class="template-item"
                role="button"
                tabindex="0"
                @click="loadTemplate(template)"
                @keydown.enter.prevent="loadTemplate(template)"
                @keydown.space.prevent="loadTemplate(template)"
              >
                <p class="template-item-name">{{ template.name }}</p>
                <p class="template-item-desc">{{ template.description }}</p>
              </div>
            </div>
          </div>
          <button type="button" class="dialog-close-btn" @click="showTemplateModal = false">
            {{ $t('common.cancel') }}
          </button>
        </div>
      </div>

      <!-- Save Template Modal -->
      <div v-if="showSaveTemplateModal" class="modal-overlay dialog-overlay" @click.self="showSaveTemplateModal = false">
        <div
          ref="saveTemplateDialogRef"
          class="dialog"
          role="dialog"
          aria-modal="true"
          :aria-label="$t('quiz_config.save_template')"
          tabindex="-1"
        >
          <h3 class="dialog-title">{{ $t('quiz_config.save_template') }}</h3>
          <input
            v-model="templateName"
            :placeholder="$t('library.actions.template_name_placeholder')"
            :aria-label="$t('library.actions.template_name_placeholder')"
            class="dialog-input"
          />
          <div class="dialog-buttons">
            <button type="button" class="dialog-btn cancel" @click="showSaveTemplateModal = false">
              {{ $t('common.cancel') }}
            </button>
            <button type="button" class="dialog-btn save" @click="saveAsTemplate">
              {{ $t('admin.details.add_modal_title') }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, watch, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { API_BASE_URL, getAuthHeaders, normalizeTemplates } from '../utils'
import { useToast } from '../composables/useToast'
import FormEditor from './FormEditor.vue'
import AppIcon from './AppIcon.vue'

const { t } = useI18n()
const props = defineProps({
  visible: { type: Boolean, default: false },
  module: { type: Object, default: null },
  eventId: { type: String, default: '' }
})

const emit = defineEmits(['close', 'update'])

const timeline = ref([])
const mandatoryOnEntry = ref(false)
const hideAfterCompletion = ref(false)
const loading = ref(false)
const templates = ref([])
const loadingTemplates = ref(false)
const templateError = ref(false)
const showTemplateModal = ref(false)
const templateName = ref('')
const showSaveTemplateModal = ref(false)
const modalContainerRef = ref(null)
const templateDialogRef = ref(null)
const saveTemplateDialogRef = ref(null)

let previouslyFocusedEl = null
let nestedPreviouslyFocusedEl = null

const { success, error, info } = useToast()

watch(() => [props.visible, props.module], ([visible, mod]) => {
  if (visible && mod?.data) {
    if (mod.data.timeline) {
      timeline.value = JSON.parse(JSON.stringify(mod.data.timeline))
    }
    mandatoryOnEntry.value = mod.data.mandatoryOnEntry ?? false
    hideAfterCompletion.value = mod.data.hideAfterCompletion ?? false
  } else if (visible) {
    timeline.value = []
    mandatoryOnEntry.value = false
    hideAfterCompletion.value = false
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
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify({ 
        data: { 
          ...(props.module?.data && typeof props.module.data === 'object' ? props.module.data : {}),
          timeline: timeline.value,
          mandatoryOnEntry: mandatoryOnEntry.value,
          hideAfterCompletion: hideAfterCompletion.value
        } 
      })
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
  templateError.value = false
  try {
    const response = await fetch(`${API_BASE_URL}/templates`)
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    const data = normalizeTemplates(await response.json())
    // Filter for form templates
    templates.value = data.filter(t => t.moduleType === 'form' || (!t.moduleType && !t.data?.questions))
  } catch (err) {
    console.error('Error fetching templates:', err)
    templateError.value = true
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
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
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
  showTemplateModal.value = false
  showSaveTemplateModal.value = false
  emit('close')
}

const handleKeydown = (event) => {
  if (event.key !== 'Escape' || !props.visible) return
  event.preventDefault()
  event.stopPropagation()
  if (showTemplateModal.value) {
    showTemplateModal.value = false
    return
  }
  if (showSaveTemplateModal.value) {
    showSaveTemplateModal.value = false
    return
  }
  handleClose()
}

watch(() => props.visible, (isOpen) => {
  if (!isOpen) {
    showTemplateModal.value = false
    showSaveTemplateModal.value = false
    if (previouslyFocusedEl && typeof previouslyFocusedEl.focus === 'function') {
      const target = previouslyFocusedEl
      previouslyFocusedEl = null
      nextTick(() => {
        if (target.isConnected) target.focus()
      })
    }
    return
  }
  if (typeof document !== 'undefined') {
    previouslyFocusedEl = document.activeElement
    nextTick(() => {
      modalContainerRef.value?.focus()
    })
  }
})

watch([showTemplateModal, showSaveTemplateModal], ([templateOpen, saveOpen]) => {
  if (typeof document === 'undefined') return
  if (templateOpen || saveOpen) {
    if (!nestedPreviouslyFocusedEl) {
      nestedPreviouslyFocusedEl = document.activeElement
    }
    nextTick(() => {
      if (templateOpen) {
        templateDialogRef.value?.focus()
      } else if (saveOpen) {
        saveTemplateDialogRef.value?.focus()
      }
    })
  } else if (nestedPreviouslyFocusedEl && typeof nestedPreviouslyFocusedEl.focus === 'function') {
    const target = nestedPreviouslyFocusedEl
    nestedPreviouslyFocusedEl = null
    if (props.visible) {
      nextTick(() => {
        if (target.isConnected) target.focus()
      })
    }
  }
})

onMounted(() => {
  if (typeof window !== 'undefined') {
    window.addEventListener('keydown', handleKeydown)
  }
  if (props.visible && typeof document !== 'undefined') {
    previouslyFocusedEl = document.activeElement
    nextTick(() => {
      modalContainerRef.value?.focus()
    })
  }
})

onUnmounted(() => {
  if (typeof window !== 'undefined') {
    window.removeEventListener('keydown', handleKeydown)
  }
})

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

.dialog-overlay {
  align-items: center;
  padding: 12px;
  z-index: 1100;
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
  overflow: hidden;
  overscroll-behavior: contain;
}
.modal-container:focus { outline: none; }

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 20px;
  border-bottom: 1px solid #2C2C2E;
}

.header-title-group {
  min-width: 0;
  flex: 1 1 180px;
}

.title { font-size: 22px; font-weight: bold; color: #FFF; overflow-wrap: anywhere; }
.subtitle { font-size: 14px; color: #8E8E93; margin-top: 2px; overflow-wrap: anywhere; }

.header-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  flex-shrink: 0;
}

.icon-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 8px;
  border-radius: 8px;
  background-color: #2C2C2E;
  font-size: 20px;
  border: 1px solid #333;
  cursor: pointer;
}
.icon-button:hover { background-color: #333; }

.library-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
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
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 8px;
  border-radius: 8px;
  background-color: #3A3A3C;
  color: #FFF;
  border: none;
  cursor: pointer;
}

.form-editor-instance {
  flex: 1;
  min-height: 0;
  height: auto; /* Allow flex to control height */
}

.content {
  flex: 1;
  min-height: 0;
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
  gap: 12px;
}

.settings-bar {
  background-color: #1C1C1E;
  border-top: 1px solid #2C2C2E;
  padding: 16px 20px;
  display: flex;
  gap: 20px;
}

.setting-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
  min-width: 0;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #FFF;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}

.checkbox-label input[type="checkbox"] {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  cursor: pointer;
}

.setting-desc {
  color: #8E8E93;
  font-size: 12px;
  margin: 0;
  padding-left: 24px;
}

.status-text { color: #8E8E93; }

.publish-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
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
  width: calc(100% - 24px);
  max-width: 400px;
  max-height: 85vh;
  max-height: 85dvh;
  overflow-y: auto;
  overscroll-behavior: contain;
  box-sizing: border-box;
}
.dialog:focus { outline: none; }

.dialog-title {
  font-size: 18px;
  font-weight: bold;
  color: #FFF;
  margin-bottom: 16px;
}

.template-list {
  max-height: 300px;
  overflow-y: auto;
  overscroll-behavior: contain;
  margin-bottom: 16px;
}

.template-error {
  color: #FF453A;
  padding: 12px;
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

.template-item-name {
  font-size: 16px;
  font-weight: 600;
  color: #FFF;
}

.template-item-desc {
  font-size: 14px;
  color: #8E8E93;
  margin: 4px 0 0;
}

.dialog-close-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-top: 16px;
  padding: 14px;
  background-color: #2C2C2E;
  border: none;
  border-radius: 12px;
  width: 100%;
  color: #FFF;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
}

.dialog-input {
  background-color: #121212;
  border: 1px solid #333;
  border-radius: 8px;
  padding: 14px 16px;
  color: white;
  font-size: 16px;
  width: 100%;
  min-inline-size: 0;
  box-sizing: border-box;
  margin-bottom: 16px;
}

.dialog-buttons {
  display: flex;
  gap: 12px;
}

.dialog-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  padding: 14px;
  border: none;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
}

.dialog-btn.cancel {
  background-color: #2C2C2E;
  color: #8E8E93;
}

.dialog-btn.save {
  background-color: #007AFF;
  color: #FFF;
}

@media (max-width: 768px) {
  .modal-container {
    height: 100vh;
    height: 100dvh;
    max-height: 100dvh;
    border-radius: 0;
  }

  .header {
    flex-wrap: wrap;
    gap: 10px;
    padding: max(12px, env(safe-area-inset-top, 0px)) max(16px, env(safe-area-inset-right, 0px)) 12px max(16px, env(safe-area-inset-left, 0px));
  }

  .header-buttons {
    gap: 8px;
  }

  .settings-bar {
    flex-direction: column;
    gap: 12px;
    padding: 12px max(16px, env(safe-area-inset-right, 0px)) 12px max(16px, env(safe-area-inset-left, 0px));
  }

  .action-bar {
    flex-wrap: wrap;
    gap: 10px;
    padding: 12px max(16px, env(safe-area-inset-right, 0px)) max(12px, env(safe-area-inset-bottom, 0px)) max(16px, env(safe-area-inset-left, 0px));
  }
}

@media (max-width: 480px) {
  .action-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .publish-btn {
    width: 100%;
  }
}

@media (pointer: coarse) {
  .icon-button,
  .library-btn,
  .close-button,
  .publish-btn,
  .dialog-close-btn,
  .dialog-btn,
  .checkbox-label {
    min-block-size: var(--target-size, 44px);
    min-inline-size: var(--target-size, 44px);
  }
}
</style>
