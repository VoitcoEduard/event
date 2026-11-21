<template>
  <AdminLayout :breadcrumbs="breadcrumbs">
    <template #actions>
      <div class="editor-topbar-actions">
        <span v-if="isDirty" class="unsaved-pill">{{ $t('template_editor.unsaved_badge') }}</span>
        <RouterLink :to="libraryTabLocation" class="cancel-link">
          <AppIcon name="arrowLeft" :size="16" aria-hidden="true" />
          <span>{{ $t('template_editor.back_to_library') }}</span>
        </RouterLink>
        <button
          v-if="!unsupportedTab && !loadError"
          type="button"
          class="save-btn"
          :disabled="saving || loading"
          @click="saveTemplate"
        >
          {{ saving ? $t('library.actions.saving_btn') : $t('library.actions.save_btn') }}
        </button>
      </div>
    </template>

    <div class="library-screen template-editor-screen">
      <main class="template-editor-main">
        <div v-if="unsupportedTab" class="state-card">
          <p class="state-message">{{ $t('template_editor.unsupported_type') }}</p>
          <RouterLink :to="libraryTabLocation" class="state-action-link">
            {{ $t('template_editor.back_to_library') }}
          </RouterLink>
        </div>

        <div v-else-if="loading" class="state-card">
          <p class="state-message">{{ $t('common.loading') }}</p>
        </div>

        <div v-else-if="loadError" class="state-card">
          <p class="state-message state-error" role="alert">{{ loadError }}</p>
          <div class="state-actions">
            <button type="button" class="retry-btn" @click="loadTemplate">
              {{ $t('template_editor.retry') }}
            </button>
            <RouterLink :to="libraryTabLocation" class="state-action-link">
              {{ $t('template_editor.back_to_library') }}
            </RouterLink>
          </div>
        </div>

        <template v-else>
          <header class="template-editor-header">
            <p class="editor-eyebrow">{{ tabLabel }}</p>
            <h1 class="editor-title">{{ pageTitle }}</h1>
          </header>

          <section class="template-meta-card">
            <div class="field-group">
              <label for="template-name-input" class="field-label">
                {{ $t('template_editor.name_label') }}
              </label>
              <input
                id="template-name-input"
                ref="nameInputRef"
                v-model="form.name"
                type="text"
                class="field-input"
                :class="{ 'has-field-error': Boolean(nameError) }"
                :aria-invalid="nameError ? 'true' : undefined"
                :aria-describedby="nameError ? 'template-name-error' : undefined"
                :placeholder="$t('template_editor.name_placeholder')"
                @input="nameError = ''"
              />
              <p v-if="nameError" id="template-name-error" class="field-error" role="alert">
                {{ nameError }}
              </p>
            </div>

            <div class="field-group">
              <label for="template-description-input" class="field-label">
                {{ $t('template_editor.description_label') }}
              </label>
              <textarea
                id="template-description-input"
                v-model="form.description"
                rows="2"
                class="field-input field-textarea"
                :placeholder="$t('template_editor.description_placeholder')"
              ></textarea>
            </div>
          </section>

          <div v-if="validationBannerText" class="validation-banner" role="alert">
            <AppIcon name="alert" :size="18" aria-hidden="true" />
            <span>{{ validationBannerText }}</span>
          </div>

          <section class="template-body-editor">
            <QuizEditor
              v-if="effectiveModuleType === 'quiz'"
              :questions="form.questions"
              v-model:active-index="activeIndex"
              :errors-by-index="errorsByIndex"
              :show-outline="true"
              :page-flow="true"
              event-id=""
              module-id=""
              @update-question="handleUpdateQuestion"
              @remove-question="handleRemoveQuestion"
              @add-question="handleAddQuestion"
              @move-question="handleMoveQuestion"
            />

            <FormEditor
              v-else-if="effectiveModuleType === 'form'"
              :timeline="form.timeline"
              :page-flow="true"
              @update:timeline="handleUpdateTimeline"
            />

            <WrongPicturesEditor
              v-else-if="effectiveModuleType === 'wrong_pictures'"
              v-model:rounds="form.questions"
              v-model:active-index="activeIndex"
              :errors-by-index="errorsByIndex"
              :page-flow="true"
              event-id=""
              module-id=""
            />
          </section>
        </template>
      </main>
    </div>
  </AdminLayout>
</template>

<script setup>
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { onBeforeRouteUpdate, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AdminLayout from '../components/AdminLayout.vue'
import AppIcon from '../components/AppIcon.vue'
import QuizEditor from '../components/QuizEditor.vue'
import FormEditor from '../components/FormEditor.vue'
import WrongPicturesEditor from '../components/WrongPicturesEditor.vue'
import { API_BASE_URL, getAuthHeaders, newUUID, normalizeTemplates } from '../utils'
import {
  getLibraryTabConfig,
  resolveLibraryTab,
  resolveSafeReturnTo,
  sanitizeTemplateQuestions
} from '../utils/templateLibrary'
import {
  groupErrorsByIndex,
  validateQuizQuestions,
  validateWrongPicturesRounds
} from '../utils/quizValidation'
import { useToast } from '../composables/useToast'
import { useDialog } from '../composables/useDialog'
import { useUnsavedChanges } from '../composables/useUnsavedChanges'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const { success, error } = useToast()
const { confirm: confirmDialog } = useDialog()

const templateId = computed(() => {
  const raw = route?.params?.templateId
  return raw ? String(raw) : ''
})
const isEditMode = computed(() => Boolean(templateId.value))

const canonicalTab = computed(() => resolveLibraryTab(route?.params?.tab))
const tabConfig = computed(() => getLibraryTabConfig(canonicalTab.value))
const unsupportedTab = computed(() => tabConfig.value.kind === 'media')
const tabLabel = computed(() => t(tabConfig.value.labelKey))

const safeReturnTo = computed(() => resolveSafeReturnTo(route?.query?.returnTo, router))
const libraryTabLocation = computed(() => ({
  path: `/admin/library/${canonicalTab.value}`,
  query: safeReturnTo.value ? { returnTo: safeReturnTo.value } : {}
}))

const loading = ref(false)
const saving = ref(false)
const loadError = ref('')
const nameError = ref('')
const hasAttemptedValidation = ref(false)
const activeIndex = ref(-1)
const nameInputRef = ref(null)
const initialSnapshot = ref('')

const form = reactive({
  id: '',
  name: '',
  description: '',
  moduleType: tabConfig.value.moduleType,
  questions: [],
  timeline: []
})

const effectiveModuleType = computed(() => form.moduleType || tabConfig.value.moduleType)

const pageTitle = computed(() => {
  if (isEditMode.value) return t('template_editor.edit_title')
  if (effectiveModuleType.value === 'form') return t('template_editor.new_form_title')
  if (effectiveModuleType.value === 'wrong_pictures') return t('template_editor.new_wrong_pictures_title')
  return t('template_editor.new_quiz_title')
})

const breadcrumbs = computed(() => {
  const leaf = isEditMode.value
    ? (form.name.trim() || t('template_editor.untitled'))
    : t('template_editor.new_breadcrumb')
  return [
    { label: t('admin_shell.library'), to: libraryTabLocation.value },
    { label: tabLabel.value, to: libraryTabLocation.value },
    { label: leaf }
  ]
})

const currentSnapshot = computed(() => JSON.stringify({
  name: form.name,
  description: form.description,
  moduleType: effectiveModuleType.value,
  questions: form.questions,
  timeline: form.timeline
}))

const isDirty = computed(() => (
  !loading.value &&
  !loadError.value &&
  !unsupportedTab.value &&
  initialSnapshot.value !== '' &&
  currentSnapshot.value !== initialSnapshot.value
))

const { confirmLeave } = useUnsavedChanges(isDirty)

onBeforeRouteUpdate(async () => confirmLeave())

const activeValidationErrors = computed(() => {
  if (!hasAttemptedValidation.value) return []
  const type = effectiveModuleType.value
  if (type === 'quiz' && form.questions.length > 0) {
    return validateQuizQuestions(form.questions, { mode: 'quiz' })
  }
  if (type === 'wrong_pictures' && form.questions.length > 0) {
    return validateWrongPicturesRounds(form.questions)
  }
  return []
})

const errorsByIndex = computed(() => groupErrorsByIndex(activeValidationErrors.value))

const validationBannerText = computed(() => {
  const count = activeValidationErrors.value.length
  if (!count) return ''
  return t('template_editor.validation_summary', { count }, count)
})

const resetBlankForm = () => {
  form.id = ''
  form.name = ''
  form.description = ''
  form.moduleType = tabConfig.value.moduleType
  form.questions = []
  form.timeline = []
  nameError.value = ''
  loadError.value = ''
  hasAttemptedValidation.value = false
  activeIndex.value = -1
  initialSnapshot.value = currentSnapshot.value
}

const loadTemplate = async () => {
  if (unsupportedTab.value) {
    loading.value = false
    loadError.value = ''
    return
  }

  if (!isEditMode.value) {
    loading.value = false
    resetBlankForm()
    return
  }

  loading.value = true
  loadError.value = ''
  nameError.value = ''
  hasAttemptedValidation.value = false
  activeIndex.value = -1

  try {
    const response = await fetch(`${API_BASE_URL}/templates`)
    if (!response.ok) {
      loadError.value = t('template_editor.load_failed')
      return
    }

    const list = normalizeTemplates(await response.json())
    const match = list.find(item => String(item.id) === templateId.value)
    if (!match) {
      loadError.value = t('template_editor.not_found')
      return
    }

    const inferredType = match.moduleType || (match.data?.questions ? 'quiz' : 'form')
    form.id = String(match.id)
    form.name = match.name || ''
    form.description = match.description || ''
    form.moduleType = inferredType
    form.questions = (inferredType === 'quiz' || inferredType === 'wrong_pictures')
      ? sanitizeTemplateQuestions(inferredType, match.data?.questions || [])
      : []
    form.timeline = inferredType === 'form'
      ? JSON.parse(JSON.stringify(match.data?.timeline || []))
      : []
    initialSnapshot.value = currentSnapshot.value
  } catch (err) {
    console.error(err)
    loadError.value = t('template_editor.load_failed')
  } finally {
    loading.value = false
  }
}

watch(
  [() => route?.params?.tab, () => route?.params?.templateId],
  () => {
    loadTemplate()
  },
  { immediate: true }
)

const handleUpdateQuestion = (index, updatedQ) => {
  form.questions = form.questions.map((item, idx) => (idx === index ? updatedQ : item))
}

const handleRemoveQuestion = async (index) => {
  const confirmed = await confirmDialog({
    title: t('template_editor.remove_question_title'),
    message: t('template_editor.remove_question_confirm'),
    confirmLabel: t('admin.delete_btn'),
    cancelLabel: t('common.cancel'),
    destructive: true
  })
  if (!confirmed) return
  form.questions = form.questions.filter((_, idx) => idx !== index)
}

const handleAddQuestion = (type = 'classic') => {
  const isWrongPictures = effectiveModuleType.value === 'wrong_pictures'
  const finalType = isWrongPictures ? 'wrong_pictures' : type

  const newQuestion = {
    id: newUUID(),
    type: finalType,
    text: '',
    answers: (finalType === 'wrong_pictures' || finalType === 'correct_picture' || finalType === 'simple_text')
      ? []
      : ['', '', '', ''],
    correctIndex: 0,
    timeLimit: finalType === 'wrong_pictures' ? 45 : (finalType === 'simple_text' ? 0 : 20),
    explainTime: 5
  }

  if (finalType === 'range') {
    newQuestion.rangeConfig = { minValue: 0, maxValue: 100, step: 1, unit: '', decimalPlaces: 0 }
    newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 100, tolerance: 10, partialPercent: 0 }
    newQuestion.correctValue = 0
  } else if (finalType === 'number') {
    newQuestion.correctNumber = 0
    newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 50, tolerance: 0, partialPercent: 0 }
  } else if (finalType === 'correct_picture') {
    newQuestion.imageOptions = [
      { id: newUUID(), imageId: '', label: '' },
      { id: newUUID(), imageId: '', label: '' }
    ]
    newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 50, tolerance: 0, partialPercent: 0 }
  } else if (finalType === 'wrong_pictures') {
    newQuestion.wrongPictureConfig = { targets: [], tolerancePercent: 5, maxPoints: 50, showTargetCount: false }
  } else if (finalType === 'true_false') {
    newQuestion.answers = [t('quiz.player.true'), t('quiz.player.false'), '', '']
    newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 50, tolerance: 0, partialPercent: 0 }
  } else if (finalType === 'simple_text') {
    newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 50, tolerance: 0, partialPercent: 0 }
    newQuestion.judgeNotes = ''
    newQuestion.correctAnswerText = ''
  } else {
    newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 50, tolerance: 0, partialPercent: 0 }
  }

  form.questions = [...form.questions, newQuestion]
  activeIndex.value = form.questions.length - 1
}

const handleMoveQuestion = (index, direction) => {
  const newIndex = index + direction
  if (newIndex < 0 || newIndex >= form.questions.length) return
  const next = [...form.questions]
  const temp = next[index]
  next[index] = next[newIndex]
  next[newIndex] = temp
  form.questions = next
  activeIndex.value = newIndex
}

const handleUpdateTimeline = (newTimeline) => {
  form.timeline = Array.isArray(newTimeline) ? newTimeline : []
}

const saveTemplate = async () => {
  const trimmedName = form.name.trim()
  if (!trimmedName) {
    nameError.value = t('template_editor.errors.name_required')
    await nextTick()
    nameInputRef.value?.focus?.()
    return
  }

  const type = effectiveModuleType.value
  const isQuizOrWrongPictures = type === 'quiz' || type === 'wrong_pictures'

  if (isQuizOrWrongPictures && form.questions.length === 0) {
    const confirmed = await confirmDialog({
      title: t('template_editor.save_empty_title'),
      message: t('library.actions.save_empty_confirm'),
      confirmLabel: t('library.actions.save_btn'),
      cancelLabel: t('common.cancel')
    })
    if (!confirmed) return
  } else if (type === 'form' && form.timeline.length === 0) {
    const confirmed = await confirmDialog({
      title: t('template_editor.save_empty_title'),
      message: t('template_editor.save_empty_form_confirm'),
      confirmLabel: t('library.actions.save_btn'),
      cancelLabel: t('common.cancel')
    })
    if (!confirmed) return
  }

  if (isQuizOrWrongPictures && form.questions.length > 0) {
    hasAttemptedValidation.value = true
    const errors = activeValidationErrors.value
    if (errors.length > 0) {
      const firstIndex = errors.find(err => err.index >= 0)?.index ?? 0
      activeIndex.value = firstIndex
      error(t('template_editor.validation_summary', { count: errors.length }, errors.length))
      return
    }
  }

  saving.value = true
  try {
    const payload = {
      name: trimmedName,
      description: form.description,
      moduleType: type,
      data: isQuizOrWrongPictures
        ? { questions: form.questions }
        : { timeline: form.timeline }
    }

    const url = isEditMode.value
      ? `${API_BASE_URL}/templates/${encodeURIComponent(templateId.value)}`
      : `${API_BASE_URL}/templates`
    const method = isEditMode.value ? 'PUT' : 'POST'

    const response = await fetch(url, {
      method,
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify(payload)
    })

    if (!response.ok) {
      error(t('template_editor.save_failed'))
      return
    }

    const result = await response.json()
    const saved = method === 'POST'
      ? response.status === 201 && Boolean(result?.id)
      : result?.message === 'Template updated'
    if (!saved) {
      error(t('template_editor.save_failed'))
      return
    }

    form.name = trimmedName
    initialSnapshot.value = currentSnapshot.value
    success(t('template_editor.saved'))
    await router.push(libraryTabLocation.value)
  } catch (err) {
    console.error(err)
    error(t('template_editor.save_failed'))
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.template-editor-screen {
  min-height: calc(100vh - 60px);
  min-height: calc(100dvh - 60px);
  background-color: var(--admin-bg, #121212);
  color: var(--admin-text, #ffffff);
}

.editor-topbar-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

.unsaved-pill {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  background: var(--admin-warning-soft, rgba(255, 159, 10, 0.16));
  color: var(--admin-warning, #ff9f0a);
}

.cancel-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 38px;
  padding: 6px 12px;
  border-radius: 8px;
  border: 1px solid var(--admin-border-strong, #3a3a3c);
  background: var(--admin-surface, #1e1e1e);
  color: var(--admin-text, #ffffff);
  font-size: 13px;
  font-weight: 600;
  text-decoration: none;
}

.cancel-link:hover {
  background: var(--admin-surface-hover, #2c2c2e);
}

.save-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 38px;
  padding: 8px 18px;
  border-radius: 8px;
  border: 1px solid var(--admin-primary-solid, #0969da);
  background: var(--admin-primary-solid, #0969da);
  color: #ffffff;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.save-btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.template-editor-main {
  max-width: 1120px;
  margin: 0 auto;
  padding: 24px clamp(16px, 3vw, 32px) 96px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  box-sizing: border-box;
}

.template-editor-header {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.editor-eyebrow {
  margin: 0;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--admin-text-dim, #8e8e93);
}

.editor-title {
  margin: 0;
  font-size: clamp(1.25rem, 1.05rem + 1vw, 1.6rem);
  font-weight: 700;
  color: var(--admin-text, #ffffff);
}

.template-meta-card {
  background: var(--admin-surface, #1e1e1e);
  border: 1px solid var(--admin-border, #333333);
  border-radius: 12px;
  padding: 20px;
  display: grid;
  gap: 16px;
  box-shadow: var(--admin-shadow, none);
}

.field-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-inline-size: 0;
}

.field-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--admin-text-muted, #c5c5ca);
}

.field-input {
  width: 100%;
  box-sizing: border-box;
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px solid var(--admin-border-strong, #4b4b50);
  background: var(--admin-surface, #121212);
  color: var(--admin-text, #ffffff);
  font: inherit;
  font-size: 15px;
  line-height: 1.4;
}

.field-input:focus {
  outline: 2px solid var(--admin-focus, #0a84ff);
  outline-offset: 1px;
}

.field-textarea {
  resize: vertical;
  min-height: 68px;
}

.has-field-error {
  border-color: var(--admin-danger, #ff453a) !important;
}

.field-error {
  margin: 2px 0 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--admin-danger, #ff453a);
}

.validation-banner {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  border-radius: 10px;
  border: 1px solid var(--admin-danger, #ff453a);
  background: var(--admin-danger-soft, rgba(255, 69, 58, 0.14));
  color: var(--admin-danger, #ff453a);
  font-size: 14px;
  font-weight: 600;
}

.template-body-editor {
  min-inline-size: 0;
}

.state-card {
  background: var(--admin-surface, #1e1e1e);
  border: 1px solid var(--admin-border, #333333);
  border-radius: 12px;
  padding: 32px 24px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
}

.state-message {
  margin: 0;
  font-size: 15px;
  color: var(--admin-text-muted, #c5c5ca);
}

.state-error {
  color: var(--admin-danger, #ff453a);
  font-weight: 600;
}

.state-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.retry-btn,
.state-action-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 40px;
  padding: 8px 16px;
  border-radius: 8px;
  border: 1px solid var(--admin-border-strong, #4b4b50);
  background: var(--admin-surface-subtle, #252528);
  color: var(--admin-text, #ffffff);
  font-size: 14px;
  font-weight: 600;
  text-decoration: none;
  cursor: pointer;
}

@media (max-width: 768px) {
  .template-editor-main {
    padding: 16px max(14px, env(safe-area-inset-right, 0px)) 96px max(14px, env(safe-area-inset-left, 0px));
  }

  .editor-topbar-actions {
    width: 100%;
    justify-content: flex-end;
  }

  .cancel-link,
  .save-btn {
    min-height: 44px;
  }

  .field-input {
    font-size: 16px;
  }
}

@media (pointer: coarse) {
  .cancel-link,
  .save-btn,
  .retry-btn,
  .state-action-link {
    min-block-size: 44px;
  }
}
</style>
