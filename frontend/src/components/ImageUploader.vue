<template>
  <div ref="rootRef" class="image-uploader" tabindex="-1" @paste="handlePaste">
    <div v-if="modelValue && !hidePreview" class="image-preview-container">
      <img :src="imageUrl" :alt="altText" class="image-preview" decoding="async" />
      <div v-if="isUploading" class="preview-uploading-overlay" role="status">
        <div class="spinner"></div>
        <span>{{ $t('image_uploader.uploading') }}</span>
      </div>
      <div class="image-actions">
        <button
          type="button"
          class="paste-overlay-btn"
          @click="handlePasteButtonClick"
          :disabled="disabled || isUploading"
          :title="$t('image_uploader.paste_replace_title')"
        >
          <AppIcon name="clipboard" :size="14" aria-hidden="true" />
          <span>{{ $t('image_uploader.paste_short') }}</span>
        </button>
        <button type="button" class="remove-btn" @click="removeImage" :disabled="disabled || isUploading">
          {{ $t('image_uploader.remove') }}
        </button>
      </div>
    </div>

    <div v-if="modelValue && hidePreview" class="compact-actions">
      <button type="button" class="change-btn" @click="triggerFileSelect" :disabled="disabled || isUploading">
        <AppIcon name="upload" :size="16" aria-hidden="true" /> {{ $t('image_uploader.change_image') }}
      </button>
      <button type="button" class="change-btn paste-compact-btn" @click="handlePasteButtonClick" :disabled="disabled || isUploading">
        <AppIcon name="clipboard" :size="16" aria-hidden="true" />
        {{ isUploading ? $t('image_uploader.uploading') : $t('image_uploader.paste_clipboard') }}
      </button>
      <button type="button" class="remove-btn-compact" @click="removeImage" :disabled="disabled || isUploading">
        <AppIcon name="close" :size="16" aria-hidden="true" /> {{ $t('image_uploader.remove_short') }}
      </button>
    </div>

    <div v-if="modelValue && awaitingPaste" class="awaiting-paste-banner" role="status">
      <AppIcon name="clipboard" :size="14" aria-hidden="true" />
      <span>{{ $t('image_uploader.awaiting_paste_hint') }}</span>
    </div>

    <div v-if="modelValue" class="alt-text-input">
      <label>{{ $t('image_uploader.alt_label') }}</label>
      <input
        type="text"
        :value="altText"
        @input="updateAltText($event.target.value)"
        :placeholder="$t('image_uploader.alt_placeholder')"
        class="alt-input"
        :disabled="disabled"
      />
    </div>

    <div v-if="!modelValue" class="upload-panel">
      <button
        ref="uploadZoneRef"
        type="button"
        class="upload-zone"
        :class="{ 'drag-over': isDragging, 'awaiting-paste': awaitingPaste, 'disabled': disabled }"
        :disabled="disabled || isUploading"
        @drop.prevent="handleDrop"
        @dragover.prevent="handleDragOver"
        @dragleave.prevent="handleDragLeave"
        @click="triggerFileSelect"
        @blur="clearAwaitingPaste"
      >
        <div v-if="!isUploading" class="upload-content">
          <AppIcon class="upload-icon" name="camera" :size="32" aria-hidden="true" />
          <span class="upload-text">{{ $t('image_uploader.click_drag') }}</span>
          <span class="upload-hint">{{ awaitingPaste ? $t('image_uploader.awaiting_paste_hint') : $t('image_uploader.hint') }}</span>
        </div>
        <div v-else class="uploading-content">
          <div class="spinner"></div>
          <span>{{ $t('image_uploader.uploading') }}</span>
        </div>
      </button>

      <button
        type="button"
        class="paste-btn"
        :disabled="disabled || isUploading"
        @click="handlePasteButtonClick"
      >
        <AppIcon name="clipboard" :size="15" aria-hidden="true" />
        <span>{{ $t('image_uploader.paste_clipboard') }}</span>
      </button>
    </div>

    <input
      ref="fileInput"
      type="file"
      accept="image/jpeg,image/png,image/gif,image/webp"
      @change="handleFileSelect"
      style="display: none"
      :disabled="disabled"
    />
  </div>
</template>

<script setup>
import { ref, computed, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { uploadImage, getImageUrl } from '../services/media'
import { useToast } from '../composables/useToast'
import {
  extractImageFromClipboardData,
  isEditableElement,
  readImageFromClipboard
} from '../utils/clipboard'
import AppIcon from './AppIcon.vue'

const { t } = useI18n()
const toast = useToast()

const props = defineProps({
  modelValue: {
    type: String,
    default: null
  },
  altText: {
    type: String,
    default: ''
  },
  eventId: {
    type: String,
    required: true
  },
  moduleId: {
    type: String,
    required: true
  },
  disabled: {
    type: Boolean,
    default: false
  },
  hidePreview: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue', 'update:altText'])

const rootRef = ref(null)
const uploadZoneRef = ref(null)
const fileInput = ref(null)
const isDragging = ref(false)
const isUploading = ref(false)
const awaitingPaste = ref(false)

let awaitingPasteTimer = null

const imageUrl = computed(() => {
  return props.modelValue ? getImageUrl(props.modelValue) : null
})

function clearAwaitingPaste() {
  awaitingPaste.value = false
  if (awaitingPasteTimer) {
    clearTimeout(awaitingPasteTimer)
    awaitingPasteTimer = null
  }
}

function focusForPaste() {
  if (uploadZoneRef.value && typeof uploadZoneRef.value.focus === 'function') {
    uploadZoneRef.value.focus()
  } else if (rootRef.value && typeof rootRef.value.focus === 'function') {
    rootRef.value.focus()
  }
}

function enterAwaitingPasteMode(toastMessage) {
  clearAwaitingPaste()
  awaitingPaste.value = true
  awaitingPasteTimer = setTimeout(() => {
    awaitingPaste.value = false
    awaitingPasteTimer = null
  }, 6000)
  focusForPaste()
  toast.info(toastMessage)
}

onUnmounted(() => {
  clearAwaitingPaste()
})

function triggerFileSelect() {
  if (props.disabled || isUploading.value) return
  fileInput.value?.click()
}

function handleDragOver() {
  if (props.disabled || isUploading.value) return
  isDragging.value = true
}

function handleDragLeave(event) {
  if (!event?.currentTarget?.contains?.(event.relatedTarget)) {
    isDragging.value = false
  }
}

async function handleFileSelect(event) {
  const file = event.target.files?.[0]
  if (file) {
    await uploadFile(file)
  }
  event.target.value = ''
}

async function handleDrop(event) {
  isDragging.value = false
  if (props.disabled || isUploading.value) return
  const file = event.dataTransfer?.files?.[0]
  if (file && file.type.startsWith('image/')) {
    await uploadFile(file)
  }
}

async function uploadFile(file, { fromClipboard = false } = {}) {
  if (props.disabled || isUploading.value) return false

  isUploading.value = true
  try {
    const result = await uploadImage(file, props.eventId, props.moduleId)
    emit('update:modelValue', result.id)
    if (fromClipboard) {
      toast.success(t('image_uploader.pasted_success'))
    }
    return true
  } catch (error) {
    console.error('Upload failed:', error)
    toast.error(t('image_uploader.error_upload', { error: error?.message || 'Unknown error' }))
    return false
  } finally {
    isUploading.value = false
  }
}

async function handlePasteButtonClick() {
  if (props.disabled || isUploading.value) return

  const result = await readImageFromClipboard()
  switch (result.status) {
    case 'ok':
      clearAwaitingPaste()
      await uploadFile(result.file, { fromClipboard: true })
      break
    case 'empty':
      toast.warning(t('image_uploader.no_image_in_clipboard'))
      break
    case 'unsupported':
      enterAwaitingPasteMode(t('image_uploader.paste_shortcut_fallback'))
      break
    case 'denied':
      enterAwaitingPasteMode(t('image_uploader.paste_permission_denied'))
      break
    case 'error':
    default:
      enterAwaitingPasteMode(t('image_uploader.clipboard_read_failed'))
      break
  }
}

async function handlePaste(event) {
  if (props.disabled || isUploading.value) return false

  const insideEditable = isEditableElement(event?.target)
  const file = extractImageFromClipboardData(event?.clipboardData, {
    ignoreWhenTextPresent: insideEditable
  })

  if (!file) {
    if (!insideEditable && awaitingPaste.value) {
      clearAwaitingPaste()
      toast.warning(t('image_uploader.no_image_in_clipboard'))
    }
    return false
  }

  event?.preventDefault?.()
  event?.stopPropagation?.()
  clearAwaitingPaste()
  await uploadFile(file, { fromClipboard: true })
  return true
}

async function removeImage() {
  if (props.disabled || isUploading.value) return

  clearAwaitingPaste()
  emit('update:modelValue', null)
  emit('update:altText', '')
}

function updateAltText(value) {
  emit('update:altText', value)
}

defineExpose({
  handleClipboardPasteEvent: handlePaste,
  uploadFile
})
</script>

<style scoped>
.image-uploader {
  width: 100%;
  min-inline-size: 0;
  box-sizing: border-box;
  outline: none;
}

.upload-panel {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.upload-zone {
  width: 100%;
  display: block;
  box-sizing: border-box;
  border: 2px dashed #3A3A3C;
  border-radius: 8px;
  padding: 24px;
  text-align: center;
  font: inherit;
  color: inherit;
  cursor: pointer;
  transition: all 0.2s ease;
  background-color: #1C1C1E;
}

.upload-zone:hover:not(:disabled) {
  border-color: #8E8E93;
  background-color: #232325;
}

.upload-zone:focus-visible {
  outline: 3px solid #0A84FF;
  outline-offset: 2px;
}

.upload-zone.drag-over,
.upload-zone.awaiting-paste {
  border-color: #0A84FF;
  background-color: rgba(10, 132, 255, 0.1);
}

.upload-zone.disabled,
.upload-zone:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.upload-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.upload-icon {
  font-size: 32px;
}

.upload-text {
  color: white;
  font-size: 14px;
}

.upload-hint {
  color: #8E8E93;
  font-size: 12px;
}

.paste-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: 100%;
  min-block-size: 36px;
  padding: 8px 12px;
  border-radius: 8px;
  border: 1px solid #3A3A3C;
  background-color: #232325;
  color: #E5E5EA;
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.2s ease, border-color 0.2s ease;
}

.paste-btn:hover:not(:disabled) {
  background-color: #2C2C2E;
  border-color: #8E8E93;
}

.paste-btn:focus-visible {
  outline: 3px solid #0A84FF;
  outline-offset: 2px;
}

.paste-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.uploading-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: #8E8E93;
}

.spinner {
  width: 24px;
  height: 24px;
  border: 2px solid #3A3A3C;
  border-top-color: #0A84FF;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.image-preview-container {
  position: relative;
  border-radius: 8px;
  overflow: hidden;
  background-color: #1C1C1E;
  border: 1px solid #3A3A3C;
}

.image-preview {
  width: 100%;
  max-height: 300px;
  object-fit: contain;
  display: block;
}

.preview-uploading-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.65);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: #FFFFFF;
  font-size: 13px;
  z-index: 2;
}

.image-actions {
  position: absolute;
  top: 8px;
  right: 8px;
  display: flex;
  align-items: center;
  gap: 6px;
  z-index: 3;
}

.paste-overlay-btn,
.remove-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  background: rgba(0, 0, 0, 0.72);
  color: white;
  border: 1px solid rgba(255, 255, 255, 0.25);
  border-radius: 4px;
  padding: 6px 10px;
  cursor: pointer;
  font-size: 12px;
  transition: background-color 0.2s;
}

.paste-overlay-btn:hover:not(:disabled) {
  background: rgba(10, 132, 255, 0.85);
}

.remove-btn:hover:not(:disabled) {
  background: rgba(255, 59, 48, 0.8);
}

.paste-overlay-btn:disabled,
.remove-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.compact-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}

.change-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  background: #3A3A3C;
  color: white;
  border: 1px solid #444;
  border-radius: 6px;
  padding: 8px 16px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  transition: all 0.2s;
}

.change-btn:hover:not(:disabled) {
  background: #4A4A4C;
}

.change-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.remove-btn-compact {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  background: rgba(255, 69, 58, 0.1);
  color: #FF453A;
  border: 1px solid rgba(255, 69, 58, 0.3);
  border-radius: 6px;
  padding: 8px 16px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  transition: all 0.2s;
}

.remove-btn-compact:hover:not(:disabled) {
  background: rgba(255, 69, 58, 0.2);
}

.remove-btn-compact:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.awaiting-paste-banner {
  margin-top: 8px;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  border-radius: 6px;
  background: rgba(10, 132, 255, 0.14);
  border: 1px solid rgba(10, 132, 255, 0.35);
  color: #75B7FF;
  font-size: 12px;
}

.alt-text-input {
  margin-top: 12px;
}

.alt-text-input label {
  display: block;
  font-size: 12px;
  color: #8E8E93;
  margin-bottom: 6px;
}

.alt-input {
  width: 100%;
  min-inline-size: 0;
  background-color: #1C1C1E;
  border: 1px solid #3A3A3C;
  border-radius: 8px;
  padding: 10px;
  color: white;
  font-size: 14px;
  box-sizing: border-box;
}

.alt-input:focus {
  outline: none;
  border-color: #0A84FF;
}

.alt-input::placeholder {
  color: #636366;
}

:global(html[data-admin-appearance="true"] .image-uploader .upload-zone) {
  background: var(--admin-surface-subtle) !important;
  border: 2px dashed var(--admin-border-strong) !important;
  color: var(--admin-text) !important;
}

:global(html[data-admin-appearance="true"] .image-uploader .upload-zone:hover:not(:disabled)) {
  background: var(--admin-surface-hover) !important;
  border-color: var(--admin-primary) !important;
}

:global(html[data-admin-appearance="true"] .image-uploader .upload-zone.drag-over),
:global(html[data-admin-appearance="true"] .image-uploader .upload-zone.awaiting-paste) {
  background: var(--admin-primary-soft) !important;
  border-color: var(--admin-primary) !important;
}

:global(html[data-admin-appearance="true"] .image-uploader .upload-text) {
  color: var(--admin-text) !important;
}

:global(html[data-admin-appearance="true"] .image-uploader .upload-hint) {
  color: var(--admin-text-muted) !important;
}

:global(html[data-admin-appearance="true"] .image-uploader .paste-btn) {
  background: var(--admin-surface) !important;
  border: 1px solid var(--admin-border-strong) !important;
  color: var(--admin-text) !important;
}

:global(html[data-admin-appearance="true"] .image-uploader .paste-btn:hover:not(:disabled)) {
  background: var(--admin-surface-hover) !important;
  border-color: var(--admin-primary) !important;
}

:global(html[data-admin-appearance="true"] .image-uploader .awaiting-paste-banner) {
  background: var(--admin-primary-soft) !important;
  border-color: var(--admin-primary) !important;
  color: var(--admin-primary) !important;
}

@media (pointer: coarse) {
  .change-btn,
  .remove-btn-compact,
  .remove-btn,
  .paste-overlay-btn,
  .paste-btn,
  .upload-zone {
    min-block-size: 44px;
    min-inline-size: 44px;
  }

  .alt-input {
    min-block-size: 44px;
    font-size: 16px;
  }
}
</style>
