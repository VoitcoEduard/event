<template>
  <div class="image-uploader">
    <div v-if="modelValue && !hidePreview" class="image-preview-container">
      <img :src="imageUrl" :alt="altText" class="image-preview" decoding="async" />
      <div class="image-actions">
        <button type="button" class="remove-btn" @click="removeImage" :disabled="disabled">
          {{ $t('image_uploader.remove') }}
        </button>
      </div>
    </div>
    
    <div v-if="modelValue && hidePreview" class="compact-actions">
      <button type="button" class="change-btn" @click="triggerFileSelect" :disabled="disabled">
        <AppIcon name="upload" :size="16" aria-hidden="true" /> {{ $t('image_uploader.change_image') }}
      </button>
      <button type="button" class="remove-btn-compact" @click="removeImage" :disabled="disabled">
        <AppIcon name="close" :size="16" aria-hidden="true" /> {{ $t('image_uploader.remove_short') }}
      </button>
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

    <div 
      v-if="!modelValue" 
      class="upload-zone"
      :class="{ 'drag-over': isDragging, 'disabled': disabled }"
      role="button"
      :tabindex="disabled ? -1 : 0"
      :aria-disabled="disabled ? 'true' : undefined"
      @drop.prevent="handleDrop"
      @dragover.prevent="isDragging = true"
      @dragleave.prevent="isDragging = false"
      @click="!disabled && triggerFileSelect()"
      @keydown.enter.prevent="!disabled && triggerFileSelect()"
      @keydown.space.prevent="!disabled && triggerFileSelect()"
    >
      <div v-if="!isUploading" class="upload-content">
        <AppIcon class="upload-icon" name="camera" :size="32" aria-hidden="true" />
        <span class="upload-text">{{ $t('image_uploader.click_drag') }}</span>
        <span class="upload-hint">{{ $t('image_uploader.hint') }}</span>
      </div>
      <div v-else class="uploading-content">
        <div class="spinner"></div>
        <span>{{ $t('image_uploader.uploading') }}</span>
      </div>
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
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { uploadImage, getImageUrl } from '../services/media'
import AppIcon from './AppIcon.vue'

const { t } = useI18n()
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

const fileInput = ref(null)
const isDragging = ref(false)
const isUploading = ref(false)

const imageUrl = computed(() => {
  return props.modelValue ? getImageUrl(props.modelValue) : null
})

function triggerFileSelect() {
  fileInput.value?.click()
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
  const file = event.dataTransfer.files?.[0]
  if (file && file.type.startsWith('image/')) {
    await uploadFile(file)
  }
}

async function uploadFile(file) {
  if (props.disabled || isUploading.value) return
  
  isUploading.value = true
  try {
    const result = await uploadImage(file, props.eventId, props.moduleId)
    emit('update:modelValue', result.id)
  } catch (error) {
    console.error('Upload failed:', error)
    alert(t('image_uploader.error_upload', { error: error.message }))
  } finally {
    isUploading.value = false
  }
}

async function removeImage() {
  if (props.disabled) return
  
  emit('update:modelValue', null)
  emit('update:altText', '')
}

function updateAltText(value) {
  emit('update:altText', value)
}
</script>

<style scoped>
.image-uploader {
  width: 100%;
  min-inline-size: 0;
  box-sizing: border-box;
}

.upload-zone {
  box-sizing: border-box;
  border: 2px dashed #3A3A3C;
  border-radius: 8px;
  padding: 24px;
  text-align: center;
  cursor: pointer;
  transition: all 0.2s ease;
  background-color: #1C1C1E;
}

.upload-zone:hover:not(.disabled) {
  border-color: #8E8E93;
  background-color: #232325;
}

.upload-zone.drag-over {
  border-color: #0A84FF;
  background-color: rgba(10, 132, 255, 0.1);
}

.upload-zone.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.upload-content {
  display: flex;
  flex-direction: column;
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

.image-actions {
  position: absolute;
  top: 8px;
  right: 8px;
}

.remove-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.7);
  color: white;
  border: 1px solid rgba(255, 255, 255, 0.25);
  border-radius: 4px;
  padding: 6px 12px;
  cursor: pointer;
  font-size: 12px;
  transition: background-color 0.2s;
}

.remove-btn:hover:not(:disabled) {
  background: rgba(255, 59, 48, 0.8);
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

.remove-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
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

@media (pointer: coarse) {
  .change-btn,
  .remove-btn-compact,
  .remove-btn,
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
