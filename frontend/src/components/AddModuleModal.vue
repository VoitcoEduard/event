<template>
  <Teleport to="body">
    <div v-if="visible" class="modal-overlay add-module-overlay" @click.self="handleClose">
      <div
        ref="dialogRef"
        class="add-module-container"
        role="dialog"
        aria-modal="true"
        :aria-label="$t('add_module.title')"
        tabindex="-1"
        @keydown.escape.stop.prevent="handleClose"
      >
        <div class="add-module-header">
          <div class="header-titles">
            <span class="step-eyebrow">{{ $t('add_module.step_label', { step }) }}</span>
            <h2 class="form-title">{{ $t('add_module.title') }}</h2>
          </div>
          <button
            type="button"
            class="close-btn"
            :aria-label="$t('common.close')"
            @click="handleClose"
          >
            <AppIcon name="close" :size="18" aria-hidden="true" />
          </button>
        </div>

        <div v-if="step === 1" class="step-body">
          <p class="step-prompt">{{ $t('add_module.choose_type') }}</p>
          <div class="module-types-grid">
            <button
              v-for="item in MODULE_TYPES"
              :key="item.type"
              type="button"
              class="module-type-card"
              :class="{ selected: selectedType === item.type }"
              @click="selectType(item.type)"
            >
              <span class="type-card-icon">
                <AppIcon :name="item.icon" :size="22" aria-hidden="true" />
              </span>
              <span class="type-card-body">
                <span class="type-card-title">{{ $t(`add_module.types.${item.type}.title`) }}</span>
                <span class="type-card-desc">{{ $t(`add_module.types.${item.type}.desc`) }}</span>
              </span>
              <AppIcon name="chevronRight" :size="16" class="type-card-arrow" aria-hidden="true" />
            </button>
          </div>
        </div>

        <form v-else class="step-body" @submit.prevent="handleSubmit" novalidate>
          <div class="selected-type-bar">
            <div class="selected-type-pill">
              <AppIcon :name="selectedTypeMeta.icon" :size="18" aria-hidden="true" />
              <span>{{ $t(`add_module.types.${selectedType}.title`) }}</span>
            </div>
            <button type="button" class="change-type-btn" @click="goToStep1">
              {{ $t('add_module.change_type') }}
            </button>
          </div>

          <div class="field-group">
            <label for="add-module-name" class="input-label">{{ $t('add_module.name_label') }}</label>
            <input
              id="add-module-name"
              ref="nameInputRef"
              v-model="moduleName"
              type="text"
              class="module-input"
              :class="{ 'has-error': Boolean(nameError) }"
              :placeholder="$t(`add_module.placeholders.${selectedType}`)"
              :aria-invalid="nameError ? 'true' : undefined"
              :aria-describedby="nameError ? 'add-module-name-error' : undefined"
              @input="nameError = ''"
            />
            <p v-if="nameError" id="add-module-name-error" class="field-error" role="alert">
              {{ nameError }}
            </p>
          </div>

          <div v-if="supportsTemplates" class="field-group">
            <span class="input-label">{{ $t('add_module.start_label') }}</span>
            <div class="source-toggle" role="radiogroup" :aria-label="$t('add_module.start_label')">
              <button
                type="button"
                role="radio"
                class="source-option"
                :class="{ active: sourceMode === 'blank' }"
                :aria-checked="sourceMode === 'blank' ? 'true' : 'false'"
                @click="setSourceMode('blank')"
              >
                {{ $t('add_module.start_blank') }}
              </button>
              <button
                type="button"
                role="radio"
                class="source-option"
                :class="{ active: sourceMode === 'template' }"
                :aria-checked="sourceMode === 'template' ? 'true' : 'false'"
                @click="setSourceMode('template')"
              >
                {{ $t('add_module.from_template') }}
              </button>
            </div>

            <div v-if="sourceMode === 'template'" class="template-picker">
              <p class="template-note">
                <AppIcon name="info" :size="15" aria-hidden="true" />
                <span>{{ $t('add_module.template_copy_note') }}</span>
              </p>

              <div class="template-search-wrap">
                <AppIcon name="search" :size="16" class="search-icon" aria-hidden="true" />
                <input
                  v-model="searchQuery"
                  type="search"
                  class="template-search-input"
                  :placeholder="$t('add_module.search_templates')"
                  :aria-label="$t('add_module.search_templates')"
                />
              </div>

              <div v-if="loadingTemplates" class="template-state">
                {{ $t('common.loading') }}
              </div>
              <div v-else-if="templatesLoadError" class="template-state template-state--error">
                <span>{{ $t('add_module.errors.templates_load_failed') }}</span>
                <button type="button" class="retry-inline-btn" @click="fetchTemplates">
                  {{ $t('event_details.retry') }}
                </button>
              </div>
              <div v-else-if="filteredTemplates.length === 0" class="template-state">
                {{ $t('add_module.no_matching_templates') }}
              </div>
              <div v-else class="template-list" role="listbox" :aria-label="$t('add_module.from_template')">
                <button
                  v-for="tpl in filteredTemplates"
                  :key="tpl.id"
                  type="button"
                  role="option"
                  class="template-choice"
                  :class="{ selected: selectedTemplateId === tpl.id }"
                  :aria-selected="selectedTemplateId === tpl.id ? 'true' : 'false'"
                  @click="selectTemplate(tpl)"
                >
                  <div class="template-choice-top">
                    <span class="template-choice-name">{{ tpl.name }}</span>
                    <span class="template-choice-count">{{ formatTemplateCount(tpl) }}</span>
                  </div>
                  <p v-if="tpl.description" class="template-choice-desc">{{ tpl.description }}</p>
                </button>
              </div>

              <p v-if="templateSelectionError" class="field-error" role="alert">
                {{ templateSelectionError }}
              </p>
            </div>
          </div>

          <p v-if="submitError" class="submit-error" role="alert">{{ submitError }}</p>

          <div class="modal-footer">
            <button type="button" class="cancel-btn" :disabled="submitting" @click="handleClose">
              {{ $t('common.cancel') }}
            </button>
            <button type="submit" class="add-module-button submit-button" :disabled="submitting">
              {{ submitting ? $t('add_module.creating') : $t('add_module.create_and_edit') }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppIcon from './AppIcon.vue'
import { API_BASE_URL, getAuthHeaders, newUUID, normalizeTemplates } from '../utils'
import { useToast } from '../composables/useToast'

const MODULE_TYPES = [
  { type: 'quiz', icon: 'quiz' },
  { type: 'simple_quiz', icon: 'message' },
  { type: 'wrong_pictures', icon: 'image' },
  { type: 'form', icon: 'form' },
  { type: 'teams', icon: 'teams' }
]

const TEMPLATE_CAPABLE_TYPES = new Set(['quiz', 'form', 'wrong_pictures'])

const props = defineProps({
  visible: { type: Boolean, default: false },
  eventId: { type: [String, Number], default: '' }
})

const emit = defineEmits(['close', 'created'])

const { t } = useI18n()
const router = useRouter()
const { success } = useToast()

const dialogRef = ref(null)
const nameInputRef = ref(null)

const step = ref(1)
const selectedType = ref('quiz')
const moduleName = ref('')
const sourceMode = ref('blank')
const templates = ref([])
const templatesLoaded = ref(false)
const loadingTemplates = ref(false)
const templatesLoadError = ref(false)
const searchQuery = ref('')
const selectedTemplateId = ref(null)
const submitting = ref(false)

const nameError = ref('')
const templateSelectionError = ref('')
const submitError = ref('')

let previousFocusEl = null

const selectedTypeMeta = computed(() => (
  MODULE_TYPES.find((item) => item.type === selectedType.value) || MODULE_TYPES[0]
))

const supportsTemplates = computed(() => TEMPLATE_CAPABLE_TYPES.has(selectedType.value))

const matchingTemplates = computed(() => {
  const type = selectedType.value
  return templates.value.filter((tpl) => {
    if (type === 'quiz') {
      return tpl.moduleType === 'quiz' || (!tpl.moduleType && Array.isArray(tpl.data?.questions))
    }
    if (type === 'form') {
      return tpl.moduleType === 'form' || (!tpl.moduleType && Array.isArray(tpl.data?.timeline))
    }
    if (type === 'wrong_pictures') {
      return tpl.moduleType === 'wrong_pictures'
    }
    return false
  })
})

const filteredTemplates = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return matchingTemplates.value
  return matchingTemplates.value.filter((tpl) => (
    (tpl.name || '').toLowerCase().includes(q)
    || (tpl.description || '').toLowerCase().includes(q)
  ))
})

function resetState() {
  step.value = 1
  selectedType.value = 'quiz'
  moduleName.value = ''
  sourceMode.value = 'blank'
  searchQuery.value = ''
  selectedTemplateId.value = null
  nameError.value = ''
  templateSelectionError.value = ''
  submitError.value = ''
  submitting.value = false
}

watch(
  () => props.visible,
  async (isVisible, wasVisible) => {
    if (typeof document !== 'undefined') {
      if (isVisible && !wasVisible) {
        previousFocusEl = document.activeElement
        resetState()
        await nextTick()
        dialogRef.value?.focus?.()
      } else if (!isVisible && wasVisible) {
        const target = previousFocusEl
        previousFocusEl = null
        if (target && typeof target.focus === 'function' && target.isConnected) {
          target.focus()
        }
      }
    } else if (isVisible && !wasVisible) {
      resetState()
    }
  }
)

async function fetchTemplates() {
  loadingTemplates.value = true
  templatesLoadError.value = false
  try {
    const res = await fetch(`${API_BASE_URL}/templates`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const normalized = normalizeTemplates(await res.json())
    templates.value = normalized
    templatesLoaded.value = true
  } catch {
    templatesLoadError.value = true
  } finally {
    loadingTemplates.value = false
  }
}

async function selectType(type) {
  selectedType.value = type
  sourceMode.value = 'blank'
  selectedTemplateId.value = null
  searchQuery.value = ''
  nameError.value = ''
  templateSelectionError.value = ''
  submitError.value = ''
  step.value = 2
  await nextTick()
  nameInputRef.value?.focus?.()
}

function goToStep1() {
  step.value = 1
  nameError.value = ''
  templateSelectionError.value = ''
  submitError.value = ''
}

function setSourceMode(mode) {
  sourceMode.value = mode
  templateSelectionError.value = ''
  submitError.value = ''
  if (mode === 'template' && !templatesLoaded.value && !loadingTemplates.value) {
    fetchTemplates()
  }
}

function selectTemplate(tpl) {
  selectedTemplateId.value = tpl.id
  templateSelectionError.value = ''
  if (!moduleName.value.trim() && tpl.name) {
    moduleName.value = tpl.name
    nameError.value = ''
  }
}

function formatTemplateCount(tpl) {
  if (selectedType.value === 'form') {
    const count = Array.isArray(tpl.data?.timeline) ? tpl.data.timeline.length : 0
    return t('add_module.items_count_steps', { count })
  }
  const count = Array.isArray(tpl.data?.questions) ? tpl.data.questions.length : 0
  if (selectedType.value === 'wrong_pictures') {
    return t('add_module.items_count_rounds', { count })
  }
  return t('add_module.items_count_questions', { count })
}

function cloneAndNormalizeTemplateData(type, rawData) {
  const cloned = rawData && typeof rawData === 'object'
    ? JSON.parse(JSON.stringify(rawData))
    : {}

  if (type === 'quiz') {
    const rawQuestions = Array.isArray(cloned.questions) ? cloned.questions : []
    cloned.questions = rawQuestions.map((q) => {
      const next = { ...q, id: newUUID() }
      if (next.type === 'range') {
        if (!next.rangeConfig) {
          next.rangeConfig = { minValue: 0, maxValue: 100, step: 1, unit: '', decimalPlaces: 0 }
        }
        if (!next.scoringConfig) {
          next.scoringConfig = { mode: 'symmetric', maxPoints: 100, tolerance: 10, partialPercent: 0 }
        }
      }
      if (next.type === 'true_false') {
        next.answers = ['True', 'False', '', '']
        if (next.correctIndex > 1 || next.correctIndex < 0) next.correctIndex = 0
      }
      if (next.type === 'number') {
        if (next.correctNumber === undefined) next.correctNumber = 0
        if (!next.scoringConfig) {
          next.scoringConfig = { mode: 'symmetric', maxPoints: 50, tolerance: 0, partialPercent: 0 }
        }
      }
      if (next.explainTime === undefined) {
        next.explainTime = 5
      }
      return next
    })
    return cloned
  }

  if (type === 'wrong_pictures') {
    const rawRounds = Array.isArray(cloned.questions) ? cloned.questions : []
    cloned.questions = rawRounds.map((q) => {
      const next = { ...q, id: newUUID(), type: 'wrong_pictures' }
      const wp = next.wrongPictureConfig && typeof next.wrongPictureConfig === 'object'
        ? { ...next.wrongPictureConfig }
        : {}
      const targets = Array.isArray(wp.targets)
        ? wp.targets.map((target) => (target && typeof target === 'object'
          ? { ...target, ...(target.id !== undefined ? { id: newUUID() } : {}) }
          : target))
        : []
      next.wrongPictureConfig = {
        maxPoints: wp.maxPoints ?? 100,
        tolerancePercent: wp.tolerancePercent ?? 10,
        showTargetCount: wp.showTargetCount ?? true,
        ...wp,
        targets
      }
      return next
    })
    return cloned
  }

  if (type === 'form') {
    const rawTimeline = Array.isArray(cloned.timeline) ? cloned.timeline : []
    cloned.timeline = rawTimeline.map((item) => ({
      ...item,
      id: newUUID()
    }))
    return cloned
  }

  return cloned
}

function handleClose() {
  if (submitting.value) return
  emit('close')
}

async function handleSubmit() {
  if (submitting.value) return
  const trimmedName = moduleName.value.trim()
  let hasError = false

  if (!trimmedName) {
    nameError.value = t('add_module.errors.name_required')
    hasError = true
  }

  let selectedTemplate = null
  if (supportsTemplates.value && sourceMode.value === 'template') {
    selectedTemplate = matchingTemplates.value.find((tpl) => tpl.id === selectedTemplateId.value) || null
    if (!selectedTemplate) {
      templateSelectionError.value = t('add_module.errors.select_template')
      hasError = true
    }
  }

  if (hasError) {
    if (nameError.value) nameInputRef.value?.focus?.()
    return
  }

  if (!props.eventId) {
    submitError.value = t('add_module.errors.create_failed')
    return
  }

  const initialData = selectedTemplate
    ? cloneAndNormalizeTemplateData(selectedType.value, selectedTemplate.data)
    : {}

  submitting.value = true
  submitError.value = ''
  try {
    const response = await fetch(`${API_BASE_URL}/events/${encodeURIComponent(props.eventId)}/modules`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify({
        type: selectedType.value,
        name: trimmedName,
        data: initialData
      })
    })
    if (!response.ok) {
      submitError.value = t('add_module.errors.create_failed')
      return
    }
    const createdModule = await response.json()
    emit('created', createdModule)
    emit('close')
    success(t('admin.details.toast.module_added'))
    if (createdModule?.id) {
      await router.push(`/admin/events/${encodeURIComponent(props.eventId)}/modules/${encodeURIComponent(createdModule.id)}/edit`)
    }
  } catch {
    submitError.value = t('add_module.errors.create_failed')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.add-module-overlay {
  position: fixed;
  inset: 0;
  background: var(--admin-overlay, rgba(0, 0, 0, 0.7));
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  z-index: 1000;
  overscroll-behavior: contain;
}

.add-module-container {
  background: var(--admin-surface, #1e1e1e);
  color: var(--admin-text, #f5f5f7);
  border: 1px solid var(--admin-border, #353538);
  border-radius: 16px;
  padding: 24px;
  width: 100%;
  max-width: 560px;
  max-height: 90dvh;
  overflow-y: auto;
  overscroll-behavior: contain;
  box-sizing: border-box;
  box-shadow: var(--admin-shadow, 0 12px 28px rgba(0, 0, 0, 0.35));
}

.add-module-container:focus {
  outline: none;
}

.add-module-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 18px;
}

.header-titles {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.step-eyebrow {
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--admin-primary, #75b7ff);
}

.form-title {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
  color: var(--admin-text, #ffffff);
}

.close-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 44px;
  min-height: 44px;
  padding: 0;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--admin-text-muted, #c5c5ca);
  cursor: pointer;
}

.close-btn:hover {
  background: var(--admin-surface-hover, #303034);
  color: var(--admin-text, #ffffff);
}

.step-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.step-prompt {
  margin: 0;
  font-size: 14px;
  color: var(--admin-text-muted, #c5c5ca);
}

.module-types-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.module-type-card {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  min-height: 60px;
  padding: 12px 14px;
  border: 1px solid var(--admin-border, #353538);
  border-radius: 12px;
  background: var(--admin-surface-subtle, #252528);
  color: var(--admin-text, #f5f5f7);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s, background-color 0.15s;
}

.module-type-card:hover {
  background: var(--admin-surface-hover, #303034);
  border-color: var(--admin-primary, #75b7ff);
}

.module-type-card.selected {
  background: var(--admin-primary-soft, #162e4d);
  border-color: var(--admin-primary, #75b7ff);
}

.type-card-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 42px;
  height: 42px;
  border-radius: 10px;
  background: var(--admin-primary-soft, #162e4d);
  color: var(--admin-primary, #75b7ff);
  flex-shrink: 0;
}

.type-card-body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
}

.type-card-title {
  font-size: 15px;
  font-weight: 700;
  color: var(--admin-text, #ffffff);
}

.type-card-desc {
  font-size: 13px;
  line-height: 1.35;
  color: var(--admin-text-muted, #c5c5ca);
}

.type-card-arrow {
  color: var(--admin-text-dim, #a3a3aa);
  flex-shrink: 0;
}

.selected-type-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 10px;
  background: var(--admin-surface-subtle, #252528);
  border: 1px solid var(--admin-border, #353538);
}

.selected-type-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 700;
  color: var(--admin-primary, #75b7ff);
}

.change-type-btn {
  min-height: 36px;
  padding: 6px 12px;
  border: 1px solid var(--admin-border-strong, #4b4b50);
  border-radius: 8px;
  background: var(--admin-surface, #1d1d1f);
  color: var(--admin-text, #f5f5f7);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.change-type-btn:hover {
  background: var(--admin-surface-hover, #303034);
}

.field-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.input-label {
  font-size: 13px;
  font-weight: 700;
  color: var(--admin-text, #f5f5f7);
}

.module-input {
  width: 100%;
  min-height: 44px;
  padding: 10px 14px;
  border: 1px solid var(--admin-border-strong, #4b4b50);
  border-radius: 8px;
  background: var(--admin-surface, #121212);
  color: var(--admin-text, #ffffff);
  font: inherit;
  font-size: 16px;
  box-sizing: border-box;
}

.module-input.has-error {
  border-color: var(--admin-danger, #ff6961);
}

.field-error,
.submit-error {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--admin-danger, #ff6961);
}

.submit-error {
  padding: 10px 12px;
  border-radius: 8px;
  background: var(--admin-danger-soft, #3b2020);
}

.source-toggle {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.source-option {
  min-height: 44px;
  padding: 8px 12px;
  border: 1px solid var(--admin-border-strong, #4b4b50);
  border-radius: 8px;
  background: var(--admin-surface-subtle, #252528);
  color: var(--admin-text-muted, #c5c5ca);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  text-align: center;
}

.source-option.active {
  background: var(--admin-primary-soft, #162e4d);
  border-color: var(--admin-primary, #75b7ff);
  color: var(--admin-primary, #75b7ff);
}

.template-picker {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 4px;
}

.template-note {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  padding: 8px 10px;
  border-radius: 8px;
  background: var(--admin-surface-subtle, #252528);
  color: var(--admin-text-muted, #c5c5ca);
  font-size: 12px;
}

.template-search-wrap {
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 12px;
  color: var(--admin-text-dim, #a3a3aa);
  pointer-events: none;
}

.template-search-input {
  width: 100%;
  min-height: 40px;
  padding: 8px 12px 8px 36px;
  border: 1px solid var(--admin-border-strong, #4b4b50);
  border-radius: 8px;
  background: var(--admin-surface, #121212);
  color: var(--admin-text, #ffffff);
  font: inherit;
  font-size: 14px;
  box-sizing: border-box;
}

.template-state {
  padding: 16px;
  border-radius: 8px;
  border: 1px dashed var(--admin-border, #353538);
  text-align: center;
  font-size: 13px;
  color: var(--admin-text-muted, #c5c5ca);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.template-state--error {
  color: var(--admin-danger, #ff6961);
}

.retry-inline-btn {
  min-height: 36px;
  padding: 6px 12px;
  border: 1px solid var(--admin-border-strong, #4b4b50);
  border-radius: 8px;
  background: var(--admin-surface, #1d1d1f);
  color: var(--admin-text, #f5f5f7);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.template-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 220px;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding-right: 2px;
}

.template-choice {
  display: flex;
  flex-direction: column;
  gap: 4px;
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--admin-border, #353538);
  border-radius: 10px;
  background: var(--admin-surface-subtle, #252528);
  color: var(--admin-text, #f5f5f7);
  text-align: left;
  cursor: pointer;
}

.template-choice:hover {
  background: var(--admin-surface-hover, #303034);
  border-color: var(--admin-primary, #75b7ff);
}

.template-choice.selected {
  background: var(--admin-primary-soft, #162e4d);
  border-color: var(--admin-primary, #75b7ff);
}

.template-choice-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.template-choice-name {
  font-size: 14px;
  font-weight: 700;
  color: var(--admin-text, #ffffff);
  overflow-wrap: anywhere;
}

.template-choice-count {
  font-size: 12px;
  font-weight: 600;
  color: var(--admin-text-muted, #c5c5ca);
  white-space: nowrap;
}

.template-choice-desc {
  margin: 0;
  font-size: 12px;
  color: var(--admin-text-muted, #c5c5ca);
  overflow-wrap: anywhere;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 6px;
  padding-top: 14px;
  border-top: 1px solid var(--admin-border, #353538);
}

.cancel-btn,
.add-module-button {
  min-height: 44px;
  padding: 10px 18px;
  border-radius: 8px;
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}

.cancel-btn {
  border: 1px solid var(--admin-border-strong, #4b4b50);
  background: var(--admin-surface-subtle, #252528);
  color: var(--admin-text, #f5f5f7);
}

.add-module-button {
  border: 1px solid var(--admin-primary-solid, #0969da);
  background: var(--admin-primary-solid, #0969da);
  color: #ffffff;
}

.add-module-button:disabled,
.cancel-btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

@media (max-width: 640px) {
  .add-module-container {
    padding: 18px;
  }

  .source-toggle {
    grid-template-columns: 1fr;
  }

  .modal-footer {
    flex-direction: column-reverse;
  }

  .modal-footer > button {
    width: 100%;
  }
}
</style>
