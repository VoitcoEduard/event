<template>
  <AdminLayout :breadcrumbs="breadcrumbs">
    <main
      ref="screenRef"
      class="module-editor-screen"
      :style="stickyStyle"
    >
      <div v-if="loading" class="editor-state-wrap">
        <div class="editor-state-card" role="status" aria-live="polite">
          <p class="state-message">{{ $t('module_editor.loading') }}</p>
        </div>
      </div>

      <div v-else-if="loadError" class="editor-state-wrap">
        <div class="editor-state-card editor-state-card--error" role="alert">
          <AppIcon name="alert" :size="28" aria-hidden="true" />
          <h1 class="state-title">{{ loadError }}</h1>
          <div class="state-actions">
            <button type="button" class="editor-btn editor-btn--primary" @click="loadEditorContext">
              {{ $t('module_editor.retry') }}
            </button>
            <RouterLink :to="eventDetailsPath" class="editor-btn editor-btn--secondary">
              <AppIcon name="arrowLeft" :size="16" aria-hidden="true" />
              <span>{{ $t('module_editor.back_to_event') }}</span>
            </RouterLink>
          </div>
        </div>
      </div>

      <template v-else-if="moduleObj">
        <header class="module-editor-header">
          <div class="module-editor-header-inner">
            <div class="editor-identity">
              <div class="name-field-wrap">
                <label for="module-editor-name" class="sr-only">
                  {{ $t('add_module.name_label') }}
                </label>
                <input
                  id="module-editor-name"
                  ref="nameInputRef"
                  v-model="moduleName"
                  type="text"
                  class="module-name-input"
                  :class="{ 'has-field-error': Boolean(nameError) }"
                  :placeholder="$t('module_editor.name_placeholder')"
                  :aria-invalid="Boolean(nameError) || undefined"
                  :aria-describedby="nameError ? 'module-editor-name-error' : undefined"
                  @input="nameError = ''"
                />
                <p
                  v-if="nameError"
                  id="module-editor-name-error"
                  class="name-field-error"
                  role="alert"
                >
                  {{ nameError }}
                </p>
              </div>

              <span class="module-type-badge">
                <AppIcon :name="moduleTypeIcon(moduleType)" :size="14" aria-hidden="true" />
                <span>{{ moduleTypeLabel(moduleType) }}</span>
              </span>
            </div>

            <div class="editor-actions">
              <button
                v-if="supportsTemplates"
                type="button"
                class="editor-btn editor-btn--secondary load-library-btn"
                @click="openLibraryModal"
              >
                <AppIcon name="book" :size="16" aria-hidden="true" />
                <span>{{ $t('quiz_config.load_library') }}</span>
              </button>

              <div v-if="hasSecondaryActions" ref="overflowWrapRef" class="overflow-wrap">
                <button
                  type="button"
                  class="editor-btn editor-btn--secondary overflow-menu-toggle"
                  aria-haspopup="menu"
                  :aria-expanded="overflowOpen ? 'true' : 'false'"
                  :aria-label="$t('module_editor.more_actions')"
                  @click="overflowOpen = !overflowOpen"
                >
                  <AppIcon name="moreVertical" :size="16" aria-hidden="true" />
                  <span class="overflow-toggle-label">{{ $t('module_editor.more_actions') }}</span>
                </button>

                <div
                  class="secondary-actions-menu"
                  :class="{ 'is-open': overflowOpen }"
                  role="menu"
                >
                  <button
                    v-if="supportsJson"
                    type="button"
                    role="menuitem"
                    class="editor-btn editor-btn--secondary secondary-action-btn load-json-btn"
                    @click="triggerFileInput"
                  >
                    <AppIcon name="upload" :size="15" aria-hidden="true" />
                    <span>{{ $t('module_editor.load_json') }}</span>
                  </button>

                  <button
                    v-if="supportsJson"
                    type="button"
                    role="menuitem"
                    class="editor-btn editor-btn--secondary secondary-action-btn export-json-btn"
                    :disabled="questions.length === 0"
                    @click="exportJSON"
                  >
                    <AppIcon name="download" :size="15" aria-hidden="true" />
                    <span>{{ $t('module_editor.export_json') }}</span>
                  </button>

                  <button
                    v-if="supportsTemplates"
                    type="button"
                    role="menuitem"
                    class="editor-btn editor-btn--secondary secondary-action-btn save-template-btn"
                    :disabled="!hasTemplateableItems"
                    @click="saveAsTemplate"
                  >
                    <AppIcon name="copy" :size="15" aria-hidden="true" />
                    <span>{{ $t('module_editor.save_template') }}</span>
                  </button>
                </div>

                <input
                  v-if="supportsJson"
                  ref="fileInputRef"
                  type="file"
                  accept=".json"
                  class="sr-only-input"
                  @change="handleFileUpload"
                />
              </div>

              <button
                v-if="isHostableModule(moduleType)"
                type="button"
                class="editor-btn editor-btn--host host-shortcut-btn"
                @click="goToHost"
              >
                <AppIcon name="play" :size="15" aria-hidden="true" />
                <span>{{ $t('module_editor.host') }}</span>
              </button>

              <button
                type="button"
                class="editor-btn editor-btn--primary save-primary-btn"
                :disabled="saving"
                @click="handleSave"
              >
                <span
                  v-if="isDirty"
                  class="dirty-dot"
                  :title="$t('module_editor.unsaved_indicator')"
                  :aria-label="$t('module_editor.unsaved_indicator')"
                ></span>
                <span>{{ saving ? $t('quiz_config.saving') : $t('module_editor.save') }}</span>
              </button>
            </div>
          </div>
        </header>

        <div class="editor-body">
          <div
            v-if="validationErrors.length > 0"
            class="validation-summary-banner"
            role="alert"
          >
            <AppIcon name="alert" :size="18" aria-hidden="true" />
            <div class="validation-summary-text">
              <strong>{{ $t('module_editor.validation_banner', validationErrors.length) }}</strong>
              <span v-if="topLevelValidationMessage"> — {{ topLevelValidationMessage }}</span>
            </div>
          </div>

          <!-- Quiz Editor -->
          <template v-if="moduleType === 'quiz'">
            <QuizEditor
              ref="quizEditorRef"
              v-model:active-index="activeIndex"
              :questions="questions"
              :event-id="eventId"
              :module-id="moduleId"
              :errors-by-index="errorsByIndex"
              :show-outline="true"
              :page-flow="true"
              @update-question="updateQuestion"
              @remove-question="removeQuestion"
              @add-question="addQuestion"
              @move-question="moveQuestion"
            />

            <section class="editor-settings-card">
              <button
                type="button"
                class="settings-collapse-trigger"
                :aria-expanded="audioSettingsOpen ? 'true' : 'false'"
                @click="audioSettingsOpen = !audioSettingsOpen"
              >
                <div class="settings-trigger-text">
                  <span class="settings-title">{{ $t('module_editor.quiz_settings') }}</span>
                  <span class="settings-summary">{{ audioSummaryText }}</span>
                </div>
                <AppIcon
                  :name="audioSettingsOpen ? 'chevronUp' : 'chevronDown'"
                  :size="18"
                  aria-hidden="true"
                />
              </button>

              <div v-show="audioSettingsOpen" class="settings-collapse-body">
                <div class="settings-grid">
                  <div class="setting-field">
                    <label for="quiz-bg-music">{{ $t('quiz_config.settings.bg_music') }}</label>
                    <select id="quiz-bg-music" v-model="selectedMusicId" class="setting-select">
                      <option value="">{{ $t('quiz_config.settings.none') }}</option>
                      <option v-for="track in tracks" :key="track.id" :value="track.id">
                        {{ track.name }}
                      </option>
                    </select>
                  </div>

                  <div class="setting-field">
                    <label for="quiz-audio-output">{{ $t('quiz_config.settings.audio_output') }}</label>
                    <select id="quiz-audio-output" v-model="audioOutput" class="setting-select">
                      <option value="projector">{{ $t('quiz_config.settings.projector_only') }}</option>
                      <option value="admin">{{ $t('quiz_config.settings.admin_only') }}</option>
                      <option value="both">{{ $t('quiz_config.settings.both') }}</option>
                    </select>
                  </div>
                </div>
              </div>
            </section>
          </template>

          <!-- Simple Quiz Editor -->
          <template v-else-if="moduleType === 'simple_quiz'">
            <QuizEditor
              ref="quizEditorRef"
              v-model:active-index="activeIndex"
              :questions="questions"
              :event-id="eventId"
              :module-id="moduleId"
              locked-type="simple_text"
              :errors-by-index="errorsByIndex"
              :show-outline="true"
              :page-flow="true"
              @update-question="updateQuestion"
              @remove-question="removeQuestion"
              @add-question="addQuestion"
              @move-question="moveQuestion"
            />

            <section class="editor-settings-card simple-quiz-settings">
              <label class="checkbox-setting-label">
                <input v-model="allowReveal" type="checkbox" />
                <span>{{ $t('simple_quiz.config.allow_reveal') }}</span>
              </label>
              <p class="setting-help">{{ $t('simple_quiz.config.allow_reveal_hint') }}</p>

              <div v-if="judgeToken" class="judge-link-bar">
                <span class="judge-label">{{ $t('simple_quiz.config.judge_link') }}</span>
                <code class="judge-link">{{ judgeUrl }}</code>
                <button type="button" class="editor-btn editor-btn--secondary copy-judge-btn" @click="copyJudgeLink">
                  <AppIcon name="copy" :size="14" aria-hidden="true" />
                  <span>{{ $t('simple_quiz.config.copy') }}</span>
                </button>
              </div>
            </section>
          </template>

          <!-- Wrong Pictures Editor -->
          <template v-else-if="moduleType === 'wrong_pictures'">
            <WrongPicturesEditor
              ref="wrongPicturesEditorRef"
              v-model:active-index="activeIndex"
              :rounds="questions"
              :event-id="eventId"
              :module-id="moduleId"
              :errors-by-index="errorsByIndex"
              :page-flow="true"
              @update:rounds="handleRoundsUpdate"
            />

            <section class="editor-settings-card">
              <button
                type="button"
                class="settings-collapse-trigger"
                :aria-expanded="audioSettingsOpen ? 'true' : 'false'"
                @click="audioSettingsOpen = !audioSettingsOpen"
              >
                <div class="settings-trigger-text">
                  <span class="settings-title">{{ $t('module_editor.audio_settings') }}</span>
                  <span class="settings-summary">{{ audioSummaryText }}</span>
                </div>
                <AppIcon
                  :name="audioSettingsOpen ? 'chevronUp' : 'chevronDown'"
                  :size="18"
                  aria-hidden="true"
                />
              </button>

              <div v-show="audioSettingsOpen" class="settings-collapse-body">
                <div class="settings-grid">
                  <div class="setting-field">
                    <label for="wp-bg-music">{{ $t('quiz_config.settings.bg_music') }}</label>
                    <select id="wp-bg-music" v-model="selectedMusicId" class="setting-select">
                      <option value="">{{ $t('quiz_config.settings.none') }}</option>
                      <option v-for="track in tracks" :key="track.id" :value="track.id">
                        {{ track.name }}
                      </option>
                    </select>
                  </div>

                  <div class="setting-field">
                    <label for="wp-audio-output">{{ $t('quiz_config.settings.audio_output') }}</label>
                    <select id="wp-audio-output" v-model="audioOutput" class="setting-select">
                      <option value="projector">{{ $t('quiz_config.settings.projector_only') }}</option>
                      <option value="admin">{{ $t('quiz_config.settings.admin_only') }}</option>
                      <option value="both">{{ $t('quiz_config.settings.both') }}</option>
                    </select>
                  </div>
                </div>
              </div>
            </section>
          </template>

          <!-- Form Editor -->
          <template v-else-if="moduleType === 'form'">
            <FormEditor
              :timeline="timeline"
              :page-flow="true"
              @update:timeline="updateTimeline"
            />

            <section class="editor-settings-card">
              <h2 class="settings-card-heading">{{ $t('form_module.config.settings_title') }}</h2>
              <div class="settings-grid">
                <div class="setting-field">
                  <label class="checkbox-setting-label">
                    <input v-model="mandatoryOnEntry" type="checkbox" />
                    <span>{{ $t('form_module.config.mandatory_entry') }}</span>
                  </label>
                  <p class="setting-help">{{ $t('form_module.config.mandatory_entry_desc') }}</p>
                </div>

                <div class="setting-field">
                  <label class="checkbox-setting-label">
                    <input v-model="hideAfterCompletion" type="checkbox" />
                    <span>{{ $t('form_module.config.hide_completed') }}</span>
                  </label>
                  <p class="setting-help">{{ $t('form_module.config.hide_completed_desc') }}</p>
                </div>
              </div>
            </section>
          </template>

          <!-- Teams Editor -->
          <template v-else-if="moduleType === 'teams'">
            <section class="editor-settings-card teams-settings-card">
              <h2 class="settings-card-heading">{{ $t('teams_module.configure') }}</h2>
              <TeamsConfigFields
                v-model="teamsConfig"
                :sources="teamsSources"
                :source-locked="teamsInitialized"
              />
              <div class="teams-editor-footer">
                <button
                  type="button"
                  class="editor-btn editor-btn--secondary manage-teams-btn"
                  @click="goToHost"
                >
                  <AppIcon name="teams" :size="16" aria-hidden="true" />
                  <span>{{ $t('teams_module.manage_teams') }}</span>
                </button>
              </div>
            </section>
          </template>
        </div>
      </template>

      <!-- Load from Library Modal -->
      <Teleport to="body">
        <div
          v-if="showLibraryModal"
          class="library-modal-overlay"
          @click.self="closeLibraryModal"
        >
          <div
            ref="libraryModalRef"
            class="library-modal"
            role="dialog"
            aria-modal="true"
            :aria-label="$t('quiz_config.select_template')"
            tabindex="-1"
            @keydown.escape.stop.prevent="closeLibraryModal"
          >
            <div class="library-header">
              <h2>{{ $t('quiz_config.select_template') }}</h2>
              <button
                type="button"
                class="close-btn"
                :aria-label="$t('common.close')"
                @click="closeLibraryModal"
              >
                <AppIcon name="close" :size="18" aria-hidden="true" />
              </button>
            </div>

            <div class="library-search-bar">
              <AppIcon name="search" :size="16" class="library-search-icon" aria-hidden="true" />
              <input
                v-model="librarySearchQuery"
                type="search"
                class="library-search-input"
                :placeholder="$t('module_editor.search_templates')"
                :aria-label="$t('module_editor.search_templates')"
              />
            </div>

            <div class="library-content">
              <div v-if="loadingTemplates" class="library-state">{{ $t('common.loading') }}</div>
              <div v-else-if="templateError" class="library-state library-state--error">
                <span>{{ $t('quiz_config.template_load_failed') }}</span>
                <button type="button" class="editor-btn editor-btn--secondary" @click="fetchTemplates">
                  {{ $t('module_editor.retry') }}
                </button>
              </div>
              <div v-else-if="filteredTemplates.length === 0" class="library-state">
                {{ $t('quiz_config.no_templates') }}
              </div>
              <div v-else class="template-list">
                <button
                  v-for="tpl in filteredTemplates"
                  :key="tpl.id"
                  type="button"
                  class="template-item"
                  @click="loadTemplate(tpl)"
                >
                  <div class="template-info">
                    <span class="t-name">{{ tpl.name }}</span>
                    <span class="t-count">{{ formatTemplateItemCount(tpl) }}</span>
                  </div>
                  <p v-if="tpl.description" class="t-desc">{{ tpl.description }}</p>
                </button>
              </div>
            </div>
          </div>
        </div>
      </Teleport>
    </main>
  </AdminLayout>
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AdminLayout from '../components/AdminLayout.vue'
import AppIcon from '../components/AppIcon.vue'
import QuizEditor from '../components/QuizEditor.vue'
import WrongPicturesEditor from '../components/WrongPicturesEditor.vue'
import FormEditor from '../components/FormEditor.vue'
import TeamsConfigFields from '../components/TeamsConfigFields.vue'
import { API_BASE_URL, getAuthHeaders, normalizeTemplates } from '../utils'
import { getTracks } from '../services/media'
import { teamsActivity } from '../services/teamsActivity'
import { defaultTeamsConfig } from '../utils/teamsActivity'
import { useToast } from '../composables/useToast'
import { useDialog } from '../composables/useDialog'
import { useUnsavedChanges } from '../composables/useUnsavedChanges'
import {
  groupErrorsByIndex,
  validateQuizQuestions,
  validateWrongPicturesRounds
} from '../utils/quizValidation'

const MODULE_ICONS = {
  quiz: 'quiz',
  simple_quiz: 'message',
  wrong_pictures: 'image',
  form: 'form',
  teams: 'teams'
}

const TEMPLATE_CAPABLE_TYPES = new Set(['quiz', 'form', 'wrong_pictures'])
const JSON_CAPABLE_TYPES = new Set(['quiz', 'wrong_pictures'])

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const { success, error, info } = useToast()
const { confirm: confirmDialog, prompt: promptDialog } = useDialog()

const eventId = computed(() => String(route.params.eventId || ''))
const moduleId = computed(() => String(route.params.moduleId || ''))
const eventDetailsPath = computed(() => (
  eventId.value ? `/admin/events/${encodeURIComponent(eventId.value)}` : '/admin/events'
))

const screenRef = ref(null)
const nameInputRef = ref(null)
const quizEditorRef = ref(null)
const wrongPicturesEditorRef = ref(null)
const fileInputRef = ref(null)
const overflowWrapRef = ref(null)
const libraryModalRef = ref(null)

const loading = ref(true)
const loadError = ref('')
const saving = ref(false)
const eventName = ref('')
const moduleObj = ref(null)
const moduleName = ref('')
const moduleType = ref('')
const moduleData = ref({})
const nameError = ref('')

// Quiz / Simple Quiz / Wrong Pictures state
const questions = ref([])
const activeIndex = ref(-1)
const tracks = ref([])
const selectedMusicId = ref('')
const audioOutput = ref('projector')
const audioSettingsOpen = ref(false)
const allowReveal = ref(true)
const judgeToken = ref('')

// Form state
const timeline = ref([])
const mandatoryOnEntry = ref(false)
const hideAfterCompletion = ref(false)

// Teams state
const teamsConfig = ref(defaultTeamsConfig())
const teamsSources = ref([])
const teamsInitialized = ref(false)

// Validation state
const hasValidated = ref(false)
const validationErrors = ref([])
const errorsByIndex = ref({})

// Library / Overflow state
const overflowOpen = ref(false)
const showLibraryModal = ref(false)
const templates = ref([])
const loadingTemplates = ref(false)
const templateError = ref(false)
const librarySearchQuery = ref('')

// Dirty tracking
const savedSnapshot = ref('')
const stickyTopPx = ref(107)

let loadSeq = 0
let previousLibraryFocus = null
let topbarObserver = null

const breadcrumbs = computed(() => [
  { label: t('admin_shell.events'), to: '/admin/events' },
  { label: eventName.value || eventId.value || t('event_details.event_fallback'), to: eventDetailsPath.value },
  { label: moduleName.value.trim() || moduleObj.value?.name || t('module_editor.untitled_module') }
])

const supportsTemplates = computed(() => TEMPLATE_CAPABLE_TYPES.has(moduleType.value))
const supportsJson = computed(() => JSON_CAPABLE_TYPES.has(moduleType.value))
const hasSecondaryActions = computed(() => supportsTemplates.value || supportsJson.value)

const hasTemplateableItems = computed(() => (
  moduleType.value === 'form' ? timeline.value.length > 0 : questions.value.length > 0
))

const judgeUrl = computed(() => {
  if (!judgeToken.value || typeof window === 'undefined') return ''
  return `${window.location.origin}/simple-quiz/${eventId.value}/${moduleId.value}/judge?token=${encodeURIComponent(judgeToken.value)}`
})

const audioSummaryText = computed(() => {
  const trackName = tracks.value.find(tr => tr.id === selectedMusicId.value)?.name || t('quiz_config.settings.none')
  const outputMap = {
    projector: t('quiz_config.settings.projector_only'),
    admin: t('quiz_config.settings.admin_only'),
    both: t('quiz_config.settings.both')
  }
  return `${trackName} · ${outputMap[audioOutput.value] || audioOutput.value}`
})

const topLevelValidationMessage = computed(() => {
  const topErr = validationErrors.value.find(err => err.index < 0) || validationErrors.value[0]
  return topErr ? t(topErr.messageKey, topErr.params || {}) : ''
})

const matchingTemplates = computed(() => {
  const type = moduleType.value
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
  const q = librarySearchQuery.value.trim().toLowerCase()
  if (!q) return matchingTemplates.value
  return matchingTemplates.value.filter(tpl => (
    String(tpl.name || '').toLowerCase().includes(q) ||
    String(tpl.description || '').toLowerCase().includes(q)
  ))
})

const stickyStyle = computed(() => ({
  '--module-editor-sticky-top': `${stickyTopPx.value}px`
}))

function moduleTypeIcon(type) {
  return MODULE_ICONS[type] || 'sliders'
}

function moduleTypeLabel(type) {
  if (type && Object.prototype.hasOwnProperty.call(MODULE_ICONS, type)) {
    return t(`event_details.module_types.${type}`)
  }
  return type || ''
}

function isHostableModule(type) {
  return type === 'quiz' || type === 'wrong_pictures' || type === 'simple_quiz' || type === 'teams'
}

function makeItemId() {
  return Date.now().toString() + Math.random().toString(36).slice(2, 7)
}

function normalizeQuizQuestion(raw, { regenerateId = false, clearImages = false } = {}) {
  const q = JSON.parse(JSON.stringify(raw || {}))
  if (regenerateId || !q.id) {
    q.id = makeItemId()
  }
  if (clearImages) {
    if (q.imageId) {
      q.imageId = null
      q.imageAlt = ''
    }
    if (Array.isArray(q.imageOptions)) {
      q.imageOptions = q.imageOptions.map(opt => ({ ...opt, imageId: '' }))
    }
  }
  if (q.type === 'range') {
    if (!q.rangeConfig) q.rangeConfig = { minValue: 0, maxValue: 100, step: 1, unit: '', decimalPlaces: 0 }
    if (!q.scoringConfig) q.scoringConfig = { mode: 'symmetric', maxPoints: 100, tolerance: 10, partialPercent: 0 }
  }
  if (q.type === 'true_false') {
    q.answers = ['True', 'False', '', '']
    if (q.correctIndex > 1) q.correctIndex = 0
  }
  if (q.type === 'number') {
    if (q.correctNumber === undefined) q.correctNumber = 0
    if (!q.scoringConfig) q.scoringConfig = { mode: 'symmetric', maxPoints: 50, tolerance: 0, partialPercent: 0 }
  }
  if (q.explainTime === undefined) q.explainTime = 5
  return q
}

function normalizeSimpleQuizQuestion(raw, { regenerateId = false } = {}) {
  const q = JSON.parse(JSON.stringify(raw || {}))
  return {
    id: regenerateId || !q.id ? makeItemId() : q.id,
    type: 'simple_text',
    text: q.text || '',
    timeLimit: q.timeLimit ?? 0,
    explainTime: 0,
    scoringConfig: q.scoringConfig || { mode: 'symmetric', maxPoints: 100, tolerance: 0, partialPercent: 0 },
    imageId: q.imageId || null,
    imageAlt: q.imageAlt || '',
    judgeNotes: q.judgeNotes || '',
    correctAnswerText: q.correctAnswerText || ''
  }
}

function normalizeWrongPicturesRound(raw, { regenerateId = false, clearImages = false } = {}) {
  const r = JSON.parse(JSON.stringify(raw || {}))
  return {
    id: regenerateId || !r.id ? makeItemId() : r.id,
    type: 'wrong_pictures',
    text: r.text || '',
    timeLimit: r.timeLimit !== undefined ? r.timeLimit : 60,
    explainTime: r.explainTime !== undefined ? r.explainTime : 5,
    imageId: clearImages ? null : (r.imageId || null),
    imageAlt: clearImages ? '' : (r.imageAlt || ''),
    wrongPictureConfig: {
      targets: Array.isArray(r.wrongPictureConfig?.targets)
        ? r.wrongPictureConfig.targets.map(target => ({ ...target }))
        : [],
      maxPoints: r.wrongPictureConfig?.maxPoints !== undefined ? r.wrongPictureConfig.maxPoints : 50,
      tolerancePercent: r.wrongPictureConfig?.tolerancePercent !== undefined ? r.wrongPictureConfig.tolerancePercent : 5,
      showTargetCount: Boolean(r.wrongPictureConfig?.showTargetCount)
    }
  }
}

function createDefaultQuestion(type = 'classic') {
  const newQuestion = {
    id: makeItemId(),
    type,
    text: '',
    answers: ['', '', '', ''],
    correctIndex: 0,
    timeLimit: 20,
    explainTime: 5
  }

  if (type === 'range') {
    newQuestion.rangeConfig = { minValue: 0, maxValue: 100, step: 1, unit: '', decimalPlaces: 0 }
    newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 100, tolerance: 10, partialPercent: 0 }
    newQuestion.correctValue = 0
  } else if (type === 'number') {
    newQuestion.correctNumber = 0
    newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 50, tolerance: 0, partialPercent: 0 }
  } else if (type === 'simple_text') {
    newQuestion.timeLimit = 0
    newQuestion.explainTime = 0
    newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 100, tolerance: 0, partialPercent: 0 }
    newQuestion.judgeNotes = ''
    newQuestion.correctAnswerText = ''
    newQuestion.imageId = null
    newQuestion.imageAlt = ''
  } else if (type === 'correct_picture') {
    newQuestion.imageOptions = [
      { id: makeItemId(), imageId: '', label: '' },
      { id: makeItemId(), imageId: '', label: '' }
    ]
    newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 50, tolerance: 0, partialPercent: 0 }
  } else if (type === 'wrong_pictures') {
    newQuestion.wrongPictureConfig = { targets: [], tolerancePercent: 10, maxPoints: 50 }
  } else if (type === 'true_false') {
    newQuestion.answers = [t('quiz.player.true'), t('quiz.player.false'), '', '']
    newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 50, tolerance: 0, partialPercent: 0 }
  } else {
    newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 50, tolerance: 0, partialPercent: 0 }
  }

  return newQuestion
}

function serializeState() {
  const base = { name: moduleName.value }
  if (moduleType.value === 'quiz' || moduleType.value === 'wrong_pictures') {
    return JSON.stringify({
      ...base,
      questions: questions.value,
      backgroundMusicId: selectedMusicId.value,
      audioOutput: audioOutput.value
    })
  }
  if (moduleType.value === 'simple_quiz') {
    return JSON.stringify({
      ...base,
      questions: questions.value,
      allowReveal: allowReveal.value
    })
  }
  if (moduleType.value === 'form') {
    return JSON.stringify({
      ...base,
      timeline: timeline.value,
      mandatoryOnEntry: mandatoryOnEntry.value,
      hideAfterCompletion: hideAfterCompletion.value
    })
  }
  if (moduleType.value === 'teams') {
    return JSON.stringify({
      ...base,
      teamsConfig: teamsConfig.value
    })
  }
  return JSON.stringify(base)
}

const isDirty = computed(() => {
  if (loading.value || loadError.value || !moduleObj.value) return false
  return serializeState() !== savedSnapshot.value
})

useUnsavedChanges(isDirty)

async function fetchTracks() {
  try {
    tracks.value = await getTracks()
  } catch {
    tracks.value = []
  }
}

async function loadEditorContext() {
  const seq = ++loadSeq
  const currentEventId = eventId.value
  const currentModuleId = moduleId.value

  loading.value = true
  loadError.value = ''
  nameError.value = ''
  hasValidated.value = false
  validationErrors.value = []
  errorsByIndex.value = {}
  overflowOpen.value = false

  if (!currentEventId || !currentModuleId) {
    loading.value = false
    loadError.value = t('module_editor.load_error')
    return
  }

  try {
    const headers = getAuthHeaders()
    const [eventRes, moduleRes] = await Promise.allSettled([
      fetch(`${API_BASE_URL}/events/${encodeURIComponent(currentEventId)}`, { headers }),
      fetch(
        `${API_BASE_URL}/events/${encodeURIComponent(currentEventId)}/modules/${encodeURIComponent(currentModuleId)}`,
        { headers }
      )
    ])

    if (seq !== loadSeq) return

    let eventData = null
    if (eventRes.status === 'fulfilled' && eventRes.value.ok) {
      eventData = await eventRes.value.json()
      if (seq !== loadSeq) return
      eventName.value = eventData?.name || ''
    } else {
      eventName.value = ''
    }

    if (moduleRes.status !== 'fulfilled' || !moduleRes.value.ok) {
      moduleObj.value = null
      loadError.value = t('module_editor.load_error')
      return
    }

    const mod = await moduleRes.value.json()
    if (seq !== loadSeq) return

    moduleObj.value = mod
    moduleName.value = mod?.name || ''
    moduleType.value = mod?.type || ''
    moduleData.value = mod?.data && typeof mod.data === 'object' ? JSON.parse(JSON.stringify(mod.data)) : {}

    const rawData = moduleData.value
    if (moduleType.value === 'quiz') {
      const rawQuestions = Array.isArray(rawData.questions) ? rawData.questions : []
      questions.value = rawQuestions.map(q => normalizeQuizQuestion(q))
      selectedMusicId.value = rawData.backgroundMusicId || ''
      audioOutput.value = rawData.audioOutput || 'projector'
      activeIndex.value = questions.value.length > 0 ? 0 : -1
      await fetchTracks()
    } else if (moduleType.value === 'simple_quiz') {
      const rawQuestions = Array.isArray(rawData.questions) ? rawData.questions : []
      questions.value = rawQuestions.map(q => normalizeSimpleQuizQuestion(q))
      allowReveal.value = rawData.allowAnswerReveal !== false
      judgeToken.value = rawData.judgeToken || ''
      activeIndex.value = questions.value.length > 0 ? 0 : -1
    } else if (moduleType.value === 'wrong_pictures') {
      const rawRounds = Array.isArray(rawData.questions) ? rawData.questions : []
      questions.value = rawRounds.map(r => normalizeWrongPicturesRound(r))
      selectedMusicId.value = rawData.backgroundMusicId || ''
      audioOutput.value = rawData.audioOutput || 'projector'
      activeIndex.value = questions.value.length > 0 ? 0 : -1
      await fetchTracks()
    } else if (moduleType.value === 'form') {
      timeline.value = Array.isArray(rawData.timeline) ? JSON.parse(JSON.stringify(rawData.timeline)) : []
      mandatoryOnEntry.value = Boolean(rawData.mandatoryOnEntry)
      hideAfterCompletion.value = Boolean(rawData.hideAfterCompletion)
    } else if (moduleType.value === 'teams') {
      const modulesList = Array.isArray(eventData?.modules) ? eventData.modules : []
      const idx = modulesList.findIndex(m => String(m.id) === String(currentModuleId))
      const priorModules = idx >= 0 ? modulesList.slice(0, idx) : modulesList
      teamsSources.value = priorModules.filter(m => m.type === 'teams' && String(m.id) !== String(currentModuleId))
      teamsConfig.value = { ...defaultTeamsConfig(), ...(rawData.teamsConfig || {}) }
      teamsInitialized.value = false
      try {
        const teamsAdmin = await teamsActivity.admin(currentEventId, currentModuleId)
        if (seq !== loadSeq) return
        if (teamsAdmin?.activity) {
          teamsInitialized.value = true
          teamsConfig.value = { ...defaultTeamsConfig(), ...(teamsAdmin.activity.config || {}) }
        }
      } catch {
        // Activity not initialized yet or unavailable; keep module config defaults.
      }
    }

    if (seq !== loadSeq) return
    savedSnapshot.value = serializeState()
  } catch {
    if (seq === loadSeq) {
      moduleObj.value = null
      loadError.value = t('module_editor.load_error')
    }
  } finally {
    if (seq === loadSeq) {
      loading.value = false
    }
  }
}

watch([eventId, moduleId], loadEditorContext, { immediate: true })

function runValidation() {
  let errs = []
  if (moduleType.value === 'quiz') {
    errs = validateQuizQuestions(questions.value, { mode: 'quiz' })
  } else if (moduleType.value === 'simple_quiz') {
    errs = validateQuizQuestions(questions.value, { mode: 'simple_quiz' })
  } else if (moduleType.value === 'wrong_pictures') {
    errs = validateWrongPicturesRounds(questions.value)
  }
  validationErrors.value = errs
  errorsByIndex.value = groupErrorsByIndex(errs)
  return errs
}

watch(
  questions,
  () => {
    if (!hasValidated.value) return
    runValidation()
  },
  { deep: true }
)

async function scrollToFirstValidationError(firstErr) {
  if (!firstErr || firstErr.index < 0) return
  activeIndex.value = firstErr.index
  await nextTick()

  if (moduleType.value === 'wrong_pictures') {
    wrongPicturesEditorRef.value?.scrollToRound(firstErr.index)
  } else {
    quizEditorRef.value?.scrollToQuestion(firstErr.index)
  }

  const rootEl = screenRef.value
  if (!rootEl) return
  const attr = moduleType.value === 'wrong_pictures' ? 'data-round-index' : 'data-question-index'
  const itemEl = rootEl.querySelector(`[${attr}="${firstErr.index}"]`)
  const fieldEl = itemEl?.querySelector('.has-field-error')
  if (fieldEl && typeof fieldEl.scrollIntoView === 'function') {
    fieldEl.scrollIntoView({ behavior: 'smooth', block: 'center' })
    if (['INPUT', 'SELECT', 'TEXTAREA'].includes(fieldEl.tagName)) {
      fieldEl.focus?.({ preventScroll: true })
    }
  }
}

const addQuestion = (type = 'classic') => {
  const effectiveType = moduleType.value === 'simple_quiz' ? 'simple_text' : type
  const nextQuestion = createDefaultQuestion(effectiveType)
  questions.value = [...questions.value, nextQuestion]
  activeIndex.value = questions.value.length - 1
  nextTick(() => {
    quizEditorRef.value?.scrollToQuestion(activeIndex.value)
  })
}

const updateQuestion = (index, updatedQ) => {
  const next = [...questions.value]
  next[index] = moduleType.value === 'simple_quiz'
    ? { ...updatedQ, type: 'simple_text' }
    : updatedQ
  questions.value = next
}

const removeQuestion = async (index) => {
  const confirmed = await confirmDialog({
    title: t('module_editor.delete_question_title'),
    message: t('quiz_config.delete_question'),
    confirmLabel: t('admin.details.delete'),
    cancelLabel: t('common.cancel'),
    destructive: true
  })
  if (!confirmed) return
  questions.value = questions.value.filter((_, i) => i !== index)
  if (activeIndex.value >= questions.value.length) {
    activeIndex.value = questions.value.length - 1
  }
}

const moveQuestion = (index, direction) => {
  const newIndex = index + direction
  if (newIndex < 0 || newIndex >= questions.value.length) return
  const next = [...questions.value]
  ;[next[index], next[newIndex]] = [next[newIndex], next[index]]
  questions.value = next
  activeIndex.value = newIndex
}

const handleRoundsUpdate = (nextRounds) => {
  questions.value = Array.isArray(nextRounds) ? nextRounds : []
}

const updateTimeline = (nextTimeline) => {
  timeline.value = Array.isArray(nextTimeline) ? nextTimeline : []
}

function getSelectedMusicUrl() {
  if (!selectedMusicId.value) return ''
  return tracks.value.find(tr => tr.id === selectedMusicId.value)?.url || ''
}

async function handleSave() {
  if (saving.value) return

  const trimmedName = moduleName.value.trim()
  if (!trimmedName) {
    nameError.value = t('add_module.errors.name_required')
    error(nameError.value)
    nameInputRef.value?.focus()
    return
  }

  if (['quiz', 'simple_quiz', 'wrong_pictures'].includes(moduleType.value)) {
    hasValidated.value = true
    const errs = runValidation()
    if (errs.length > 0) {
      const firstErr = errs[0]
      error(t(firstErr.messageKey, firstErr.params || {}))
      await scrollToFirstValidationError(firstErr)
      return
    }
  } else if (moduleType.value === 'form' && timeline.value.length === 0) {
    info(t('form_module.config.no_steps'))
    return
  }

  saving.value = true
  try {
    const eId = encodeURIComponent(eventId.value)
    const mId = encodeURIComponent(moduleId.value)
    const headers = { 'Content-Type': 'application/json', ...getAuthHeaders() }

    if (moduleType.value === 'quiz' || moduleType.value === 'wrong_pictures') {
      const questionsRes = await fetch(`${API_BASE_URL}/events/${eId}/modules/${mId}/quiz/questions`, {
        method: 'PUT',
        headers,
        body: JSON.stringify(questions.value)
      })
      if (!questionsRes.ok) throw new Error('Failed to save questions')

      const musicUrl = getSelectedMusicUrl()
      const modRes = await fetch(`${API_BASE_URL}/events/${eId}/modules/${mId}`, {
        method: 'PUT',
        headers,
        body: JSON.stringify({
          name: trimmedName,
          data: {
            questions: questions.value,
            backgroundMusicId: selectedMusicId.value,
            backgroundMusicUrl: musicUrl,
            audioOutput: audioOutput.value
          }
        })
      })
      if (!modRes.ok) throw new Error('Failed to save module config')
    } else if (moduleType.value === 'simple_quiz') {
      const modRes = await fetch(`${API_BASE_URL}/events/${eId}/modules/${mId}`, {
        method: 'PUT',
        headers,
        body: JSON.stringify({
          name: trimmedName,
          data: {
            questions: questions.value,
            allowAnswerReveal: allowReveal.value
          }
        })
      })
      if (!modRes.ok) throw new Error('Failed to save module config')
    } else if (moduleType.value === 'form') {
      const nextData = {
        ...(moduleData.value && typeof moduleData.value === 'object' ? moduleData.value : {}),
        timeline: timeline.value,
        mandatoryOnEntry: mandatoryOnEntry.value,
        hideAfterCompletion: hideAfterCompletion.value
      }
      const modRes = await fetch(`${API_BASE_URL}/events/${eId}/modules/${mId}/publish`, {
        method: 'PUT',
        headers,
        body: JSON.stringify({
          name: trimmedName,
          data: nextData
        })
      })
      if (!modRes.ok) throw new Error('Failed to publish form configuration')
      moduleData.value = JSON.parse(JSON.stringify(nextData))
    } else if (moduleType.value === 'teams') {
      if (teamsInitialized.value) {
        await teamsActivity.action(eventId.value, moduleId.value, {
          action: 'configure',
          config: teamsConfig.value
        })
      }
      const modRes = await fetch(`${API_BASE_URL}/events/${eId}/modules/${mId}`, {
        method: 'PUT',
        headers,
        body: JSON.stringify({
          name: trimmedName,
          data: {
            teamsConfig: teamsConfig.value
          }
        })
      })
      if (!modRes.ok) throw new Error('Failed to save teams configuration')
    }

    moduleName.value = trimmedName
    if (moduleObj.value) {
      moduleObj.value = { ...moduleObj.value, name: trimmedName }
    }
    savedSnapshot.value = serializeState()
    success(t('module_editor.saved_success'))
  } catch {
    error(t('module_editor.save_error'))
  } finally {
    saving.value = false
  }
}

function goToHost() {
  overflowOpen.value = false
  if (moduleType.value === 'teams') {
    router.push({
      path: `/admin/events/${encodeURIComponent(eventId.value)}/teams`,
      query: { moduleId: moduleId.value }
    })
    return
  }
  router.push(`/admin/events/${encodeURIComponent(eventId.value)}/modules/${encodeURIComponent(moduleId.value)}/host`)
}

async function copyJudgeLink() {
  if (!judgeUrl.value) return
  try {
    await navigator.clipboard?.writeText(judgeUrl.value)
    success(t('simple_quiz.config.copied'))
  } catch {
    // Clipboard blocked or unavailable
  }
}

async function fetchTemplates() {
  loadingTemplates.value = true
  templateError.value = false
  try {
    const res = await fetch(`${API_BASE_URL}/templates`, { headers: getAuthHeaders() })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    templates.value = normalizeTemplates(await res.json())
  } catch {
    templateError.value = true
  } finally {
    loadingTemplates.value = false
  }
}

function openLibraryModal() {
  overflowOpen.value = false
  librarySearchQuery.value = ''
  previousLibraryFocus = typeof document !== 'undefined' ? document.activeElement : null
  showLibraryModal.value = true
  fetchTemplates()
  nextTick(() => {
    libraryModalRef.value?.focus?.()
  })
}

function closeLibraryModal() {
  showLibraryModal.value = false
  const target = previousLibraryFocus
  previousLibraryFocus = null
  if (target && typeof target.focus === 'function') {
    nextTick(() => {
      if (target.isConnected) target.focus()
    })
  }
}

function formatTemplateItemCount(tpl) {
  if (moduleType.value === 'form') {
    const count = Array.isArray(tpl.data?.timeline) ? tpl.data.timeline.length : 0
    return t('add_module.items_count_steps', { count })
  }
  if (moduleType.value === 'wrong_pictures') {
    const count = Array.isArray(tpl.data?.questions) ? tpl.data.questions.length : 0
    return t('add_module.items_count_rounds', { count })
  }
  const count = Array.isArray(tpl.data?.questions) ? tpl.data.questions.length : 0
  return t('add_module.items_count_questions', { count })
}

async function loadTemplate(tpl) {
  const hasExisting = moduleType.value === 'form' ? timeline.value.length > 0 : questions.value.length > 0
  if (hasExisting) {
    const confirmed = await confirmDialog({
      title: t('module_editor.replace_title'),
      message: moduleType.value === 'form'
        ? t('module_editor.confirm_replace_steps')
        : t('quiz_config.confirm_replace'),
      confirmLabel: t('module_editor.replace_confirm'),
      cancelLabel: t('common.cancel'),
      destructive: true
    })
    if (!confirmed) return
  }

  if (moduleType.value === 'form') {
    const loadedSteps = Array.isArray(tpl.data?.timeline) ? JSON.parse(JSON.stringify(tpl.data.timeline)) : []
    timeline.value = loadedSteps.map(step => ({ ...step, id: makeItemId() }))
  } else if (moduleType.value === 'wrong_pictures') {
    const loadedRounds = Array.isArray(tpl.data?.questions) ? tpl.data.questions : []
    questions.value = loadedRounds.map(r => normalizeWrongPicturesRound(r, { regenerateId: true }))
    activeIndex.value = questions.value.length > 0 ? 0 : -1
  } else {
    const loadedQuestions = Array.isArray(tpl.data?.questions) ? tpl.data.questions : []
    questions.value = loadedQuestions.map(q => normalizeQuizQuestion(q, { regenerateId: true }))
    activeIndex.value = questions.value.length > 0 ? 0 : -1
  }

  hasValidated.value = false
  validationErrors.value = []
  errorsByIndex.value = {}
  success(t('quiz_config.loaded_template', { name: tpl.name }))
  closeLibraryModal()
}

function triggerFileInput() {
  overflowOpen.value = false
  fileInputRef.value?.click()
}

function handleFileUpload(event) {
  const file = event.target?.files?.[0]
  if (!file) return

  const reader = new FileReader()
  reader.onload = async (e) => {
    try {
      const content = JSON.parse(e.target.result)
      let loadedQuestions = null
      if (content?.data && Array.isArray(content.data.questions)) {
        loadedQuestions = content.data.questions
      } else if (Array.isArray(content)) {
        loadedQuestions = content
      }

      if (!loadedQuestions) {
        error(t('module_editor.import_invalid_format'))
        return
      }
      if (loadedQuestions.length === 0) {
        info(t('quiz_config.import.no_questions'))
        return
      }

      if (questions.value.length > 0) {
        const confirmed = await confirmDialog({
          title: t('module_editor.replace_title'),
          message: t('quiz_config.confirm_replace'),
          confirmLabel: t('module_editor.replace_confirm'),
          cancelLabel: t('common.cancel'),
          destructive: true
        })
        if (!confirmed) return
      }

      let hadImages = false
      for (const item of loadedQuestions) {
        if (item?.imageId || (Array.isArray(item?.imageOptions) && item.imageOptions.some(opt => opt?.imageId))) {
          hadImages = true
          break
        }
      }

      if (moduleType.value === 'wrong_pictures') {
        questions.value = loadedQuestions.map(r => (
          normalizeWrongPicturesRound(r, { regenerateId: true, clearImages: true })
        ))
      } else {
        questions.value = loadedQuestions.map(q => (
          normalizeQuizQuestion(q, { regenerateId: true, clearImages: true })
        ))
      }

      if (hadImages) {
        info(t('quiz_config.import.note_images'))
      }
      activeIndex.value = questions.value.length > 0 ? 0 : -1
      hasValidated.value = false
      validationErrors.value = []
      errorsByIndex.value = {}
      success(t('quiz_config.import.success'))
    } catch {
      error(t('quiz_config.import.error_parse'))
    } finally {
      if (event.target) {
        event.target.value = ''
      }
    }
  }
  reader.readAsText(file)
}

function exportJSON() {
  overflowOpen.value = false
  const payload = {
    name: moduleName.value.trim() || t('module_editor.export_default_name'),
    description: t('module_editor.export_description', { eventId: eventId.value }),
    moduleType: moduleType.value,
    data: { questions: questions.value }
  }
  const dataStr = JSON.stringify(payload, null, 2)
  const blob = new Blob([dataStr], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  const slug = (moduleName.value.trim() || moduleType.value || 'module')
    .replace(/[^a-z0-9]/gi, '_')
    .toLowerCase()
  a.download = `${slug}.json`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
  success(t('module_editor.exported_json'))
}

async function saveAsTemplate() {
  overflowOpen.value = false
  const rawName = await promptDialog({
    title: t('quiz_config.save_template'),
    label: t('module_editor.template_name_label'),
    placeholder: t('library.actions.template_name_placeholder'),
    defaultValue: moduleName.value.trim(),
    confirmLabel: t('library.actions.save_btn'),
    cancelLabel: t('common.cancel')
  })
  if (rawName === null) return

  const name = rawName.trim()
  if (!name) {
    info(t('module_editor.template_name_required'))
    return
  }

  try {
    const dataPayload = moduleType.value === 'form'
      ? { timeline: timeline.value }
      : { questions: questions.value }

    const res = await fetch(`${API_BASE_URL}/templates`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify({
        name,
        description: t('module_editor.template_description', { eventId: eventId.value }),
        moduleType: moduleType.value,
        data: dataPayload
      })
    })
    if (res.ok) {
      success(t('quiz_config.saved_template'))
    } else {
      error(t('quiz_config.save_failed'))
    }
  } catch {
    error(t('quiz_config.save_failed'))
  }
}

function handleDocumentClick(event) {
  if (!overflowOpen.value) return
  if (overflowWrapRef.value && !overflowWrapRef.value.contains(event.target)) {
    overflowOpen.value = false
  }
}

function handleWindowKeydown(event) {
  if (event.key === 'Escape' && overflowOpen.value) {
    overflowOpen.value = false
  }
}

function updateStickyTop() {
  if (typeof document === 'undefined') return
  const topbar = document.querySelector('.admin-topbar')
  if (topbar && typeof topbar.getBoundingClientRect === 'function') {
    const height = Math.round(topbar.getBoundingClientRect().height)
    if (height > 0 && height < 220) {
      stickyTopPx.value = height
    }
  }
}

onMounted(() => {
  if (typeof document !== 'undefined') {
    document.addEventListener('click', handleDocumentClick)
    const topbar = document.querySelector('.admin-topbar')
    if (topbar && typeof ResizeObserver !== 'undefined') {
      topbarObserver = new ResizeObserver(updateStickyTop)
      topbarObserver.observe(topbar)
    }
    updateStickyTop()
  }
  if (typeof window !== 'undefined') {
    window.addEventListener('keydown', handleWindowKeydown)
  }
})

onUnmounted(() => {
  if (typeof document !== 'undefined') {
    document.removeEventListener('click', handleDocumentClick)
  }
  if (typeof window !== 'undefined') {
    window.removeEventListener('keydown', handleWindowKeydown)
  }
  if (topbarObserver) {
    topbarObserver.disconnect()
    topbarObserver = null
  }
})
</script>

<style scoped>
.module-editor-screen {
  min-height: 100%;
  background: var(--admin-bg, #f5f6f8);
  color: var(--admin-text, #101828);
  padding-bottom: max(48px, env(safe-area-inset-bottom, 0px));
}

.module-editor-screen,
.module-editor-screen * {
  box-sizing: border-box;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

.sr-only-input {
  display: none;
}

.editor-state-wrap {
  max-width: 640px;
  margin: 40px auto;
  padding: 0 20px;
}

.editor-state-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  text-align: center;
  padding: 28px 24px;
  background: var(--admin-surface, #ffffff);
  border: 1px solid var(--admin-border, #dde1e7);
  border-radius: 14px;
  box-shadow: var(--admin-shadow, 0 1px 2px rgba(16, 24, 40, 0.06));
}

.editor-state-card--error {
  border-color: color-mix(in srgb, var(--admin-danger, #b42318) 38%, var(--admin-border, #dde1e7));
}

.state-title {
  margin: 0;
  font-size: 1.15rem;
  font-weight: 700;
  color: var(--admin-text, #101828);
}

.state-message {
  margin: 0;
  color: var(--admin-text-muted, #475467);
  font-weight: 600;
}

.state-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 10px;
}

.module-editor-header {
  position: sticky;
  top: var(--module-editor-sticky-top, 107px);
  z-index: 300;
  background: var(--admin-surface, #ffffff);
  border-bottom: 1px solid var(--admin-border, #dde1e7);
  box-shadow: var(--admin-shadow, 0 1px 2px rgba(16, 24, 40, 0.06));
}

.module-editor-header-inner {
  max-width: 1440px;
  margin: 0 auto;
  padding: 12px clamp(16px, 3vw, 32px);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px 16px;
}

.editor-identity {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  flex: 1 1 280px;
  min-inline-size: 0;
}

.name-field-wrap {
  flex: 1 1 220px;
  max-width: 420px;
  min-inline-size: 0;
}

.module-name-input {
  width: 100%;
  min-height: 40px;
  padding: 8px 12px;
  border: 1px solid var(--admin-border-strong, #c8ced8);
  border-radius: 8px;
  background: var(--admin-surface, #ffffff);
  color: var(--admin-text, #101828);
  font: inherit;
  font-size: 1rem;
  font-weight: 700;
}

.module-name-input:focus {
  border-color: var(--admin-primary, #0969da);
  outline: none;
}

.module-name-input.has-field-error {
  border-color: var(--admin-danger, #b42318);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--admin-danger, #b42318) 20%, transparent);
}

.name-field-error {
  margin: 4px 0 0;
  font-size: 12px;
  font-weight: 600;
  color: var(--admin-danger, #b42318);
}

.module-type-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 11px;
  border-radius: 999px;
  background: var(--admin-primary-soft, #e8f2ff);
  color: var(--admin-primary, #0969da);
  border: 1px solid color-mix(in srgb, var(--admin-primary, #0969da) 24%, var(--admin-border, #dde1e7));
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
}

.editor-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  flex-shrink: 0;
}

.editor-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  min-height: 40px;
  padding: 8px 14px;
  border-radius: 8px;
  font: inherit;
  font-size: 13px;
  font-weight: 650;
  line-height: 1.2;
  text-decoration: none;
  white-space: nowrap;
  cursor: pointer;
  border: 1px solid var(--admin-border-strong, #c8ced8);
  background: var(--admin-surface, #ffffff);
  color: var(--admin-text, #101828);
  transition: background-color 0.15s, border-color 0.15s;
}

.editor-btn:hover:not(:disabled) {
  background: var(--admin-surface-hover, #eef2f7);
}

.editor-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.editor-btn--primary {
  background: var(--admin-primary-solid, #0969da);
  border-color: var(--admin-primary-solid, #0969da);
  color: #ffffff;
}

.editor-btn--primary:hover:not(:disabled) {
  background: var(--admin-primary-hover, #0758b8);
}

.editor-btn--host {
  background: var(--admin-primary-soft, #e8f2ff);
  border-color: color-mix(in srgb, var(--admin-primary, #0969da) 36%, var(--admin-border, #dde1e7));
  color: var(--admin-primary, #0969da);
}

.editor-btn--host:hover:not(:disabled) {
  background: color-mix(in srgb, var(--admin-primary-soft, #e8f2ff) 70%, var(--admin-surface-hover, #eef2f7));
}

.dirty-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #ffcf4d;
  box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.35);
  flex-shrink: 0;
}

.overflow-wrap {
  position: relative;
  display: inline-flex;
  align-items: center;
}

.overflow-menu-toggle {
  display: none;
}

.secondary-actions-menu {
  display: flex;
  align-items: center;
  gap: 8px;
}

.editor-body {
  max-width: 1440px;
  margin: 0 auto;
  padding: 24px clamp(16px, 3vw, 32px) 0;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.validation-summary-banner {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  border-radius: 10px;
  background: var(--admin-danger-soft, #feebe9);
  border: 1px solid color-mix(in srgb, var(--admin-danger, #b42318) 38%, var(--admin-border, #dde1e7));
  color: var(--admin-danger, #b42318);
  font-size: 14px;
}

.validation-summary-text {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  line-height: 1.4;
}

.editor-settings-card {
  background: var(--admin-surface, #ffffff);
  border: 1px solid var(--admin-border, #dde1e7);
  border-radius: 12px;
  padding: 18px 20px;
  box-shadow: var(--admin-shadow, 0 1px 2px rgba(16, 24, 40, 0.06));
}

.settings-card-heading {
  margin: 0 0 14px;
  font-size: 16px;
  font-weight: 700;
  color: var(--admin-text, #101828);
}

.settings-collapse-trigger {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 4px 0;
  border: none;
  background: transparent;
  color: var(--admin-text, #101828);
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.settings-trigger-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-inline-size: 0;
}

.settings-title {
  font-size: 15px;
  font-weight: 700;
  color: var(--admin-text, #101828);
}

.settings-summary {
  font-size: 13px;
  color: var(--admin-text-muted, #475467);
}

.settings-collapse-body {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--admin-border, #dde1e7);
}

.settings-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 16px;
}

.setting-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-inline-size: 0;
}

.setting-field label {
  font-size: 13px;
  font-weight: 600;
  color: var(--admin-text-muted, #475467);
}

.setting-select {
  min-height: 40px;
  padding: 8px 12px;
  border: 1px solid var(--admin-border-strong, #c8ced8);
  border-radius: 8px;
  background: var(--admin-surface, #ffffff);
  color: var(--admin-text, #101828);
  font: inherit;
  font-size: 14px;
}

.checkbox-setting-label {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  font-size: 14px;
  font-weight: 650;
  color: var(--admin-text, #101828);
  cursor: pointer;
}

.checkbox-setting-label input[type="checkbox"] {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  accent-color: var(--admin-primary-solid, #0969da);
  cursor: pointer;
}

.setting-help {
  margin: 4px 0 0 28px;
  font-size: 13px;
  line-height: 1.4;
  color: var(--admin-text-muted, #475467);
}

.judge-link-bar {
  margin-top: 16px;
  padding: 12px 14px;
  border-radius: 10px;
  background: var(--admin-surface-subtle, #f8f9fb);
  border: 1px solid var(--admin-border, #dde1e7);
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

.judge-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--admin-text-muted, #475467);
}

.judge-link {
  flex: 1 1 220px;
  min-inline-size: 0;
  font-size: 12px;
  color: var(--admin-primary, #0969da);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.teams-settings-card {
  max-width: 720px;
}

.teams-editor-footer {
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--admin-border, #dde1e7);
  display: flex;
  justify-content: flex-start;
}

/* Library Modal */
.library-modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 1200;
  background: var(--admin-overlay, rgba(16, 24, 40, 0.5));
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
}

.library-modal {
  width: min(100%, 560px);
  max-height: min(85vh, 85dvh);
  background: var(--admin-surface, #ffffff);
  color: var(--admin-text, #101828);
  border: 1px solid var(--admin-border, #dde1e7);
  border-radius: 14px;
  box-shadow: 0 20px 44px rgba(16, 24, 40, 0.22);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  outline: none;
}

.library-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--admin-border, #dde1e7);
}

.library-header h2 {
  margin: 0;
  font-size: 17px;
  font-weight: 700;
}

.close-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  border: 1px solid transparent;
  background: transparent;
  color: var(--admin-text-muted, #475467);
  cursor: pointer;
}

.close-btn:hover {
  background: var(--admin-surface-hover, #eef2f7);
  color: var(--admin-text, #101828);
}

.library-search-bar {
  position: relative;
  padding: 12px 20px 0;
}

.library-search-icon {
  position: absolute;
  left: 32px;
  top: 23px;
  color: var(--admin-text-dim, #667085);
  pointer-events: none;
}

.library-search-input {
  width: 100%;
  min-height: 40px;
  padding: 8px 12px 8px 36px;
  border: 1px solid var(--admin-border-strong, #c8ced8);
  border-radius: 8px;
  background: var(--admin-surface, #ffffff);
  color: var(--admin-text, #101828);
  font: inherit;
  font-size: 14px;
}

.library-content {
  flex: 1;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 16px 20px 20px;
}

.library-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 28px 16px;
  text-align: center;
  color: var(--admin-text-muted, #475467);
  font-size: 14px;
}

.library-state--error {
  color: var(--admin-danger, #b42318);
}

.template-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.template-item {
  width: 100%;
  text-align: left;
  padding: 12px 14px;
  border-radius: 10px;
  border: 1px solid var(--admin-border, #dde1e7);
  background: var(--admin-surface-subtle, #f8f9fb);
  color: var(--admin-text, #101828);
  font: inherit;
  cursor: pointer;
  transition: border-color 0.15s, background-color 0.15s;
}

.template-item:hover {
  background: var(--admin-surface-hover, #eef2f7);
  border-color: var(--admin-primary, #0969da);
}

.template-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.t-name {
  font-weight: 700;
  font-size: 14px;
}

.t-count {
  font-size: 12px;
  font-weight: 600;
  color: var(--admin-text-muted, #475467);
  white-space: nowrap;
}

.t-desc {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--admin-text-muted, #475467);
  line-height: 1.4;
}

/* Theme embedded modular editors cleanly inside .module-editor-screen */
:global(html[data-admin-appearance="true"] .module-editor-screen :is(.step-card, .question-editor, .round-card)) {
  background: var(--admin-surface) !important;
  border-color: var(--admin-border) !important;
  color: var(--admin-text) !important;
  box-shadow: var(--admin-shadow) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen .step-header) {
  background: var(--admin-surface-subtle) !important;
  border-bottom: 1px solid var(--admin-border) !important;
  color: var(--admin-text) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen .step-header.field-step) {
  border-left-color: var(--admin-primary) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen .step-header.message-step) {
  border-left-color: var(--admin-warning) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen :is(input, textarea, select):not(.answer-input):not(.correct-value-input)) {
  background: var(--admin-surface) !important;
  border-color: var(--admin-border-strong) !important;
  color: var(--admin-text) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen .step-card .answer-input) {
  background: var(--admin-surface) !important;
  border-color: var(--admin-border-strong) !important;
  color: var(--admin-text) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen .answer-card .answer-input) {
  background: transparent !important;
  border: none !important;
  color: var(--admin-text) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen .correct-value-input) {
  background: var(--admin-success-soft) !important;
  border-color: var(--admin-success) !important;
  color: var(--admin-text) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen :is(.range-config-panel, .number-config-panel, .wrong-pictures-config, .simple-text-config, .targets-list-container, .upload-zone, .image-option-card)) {
  background: var(--admin-surface-subtle) !important;
  border-color: var(--admin-border) !important;
  color: var(--admin-text) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen :is(.preview-box, .number-input-preview, .target-row, .image-preview-container)) {
  background: var(--admin-surface) !important;
  border-color: var(--admin-border) !important;
  color: var(--admin-text) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen :is(.round-header h3, .step-type-label, .toggle-label, .upload-text, .targets-header h4, .coordinate-picker-wrapper > label, .teams-config-fields label)) {
  color: var(--admin-text) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen :is(.label, .input-label, .input-group label, .image-section label, .alt-text-input label, .options-header label, .default-hint, .preview-note, .config-note, .upload-hint, .question-number, .step-index, .target-params, .empty-text, .empty-targets, .teams-config-fields .hint)) {
  color: var(--admin-text-muted) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen :is(.move-btn, .action-icon, .change-btn, .outline-move-btn)) {
  background: var(--admin-surface-subtle) !important;
  border: 1px solid var(--admin-border-strong) !important;
  color: var(--admin-text) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen :is(.action-icon.delete-btn, .remove-answer-btn, .remove-btn-compact)) {
  background: var(--admin-danger-soft) !important;
  border: 1px solid color-mix(in srgb, var(--admin-danger) 35%, var(--admin-border)) !important;
  color: var(--admin-danger) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen .round-header .remove-btn) {
  background: var(--admin-danger-soft) !important;
  border: 1px solid color-mix(in srgb, var(--admin-danger) 35%, var(--admin-border)) !important;
  color: var(--admin-danger) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen .question-type-tag) {
  background: var(--admin-primary-soft) !important;
  border: 1px solid color-mix(in srgb, var(--admin-primary) 30%, var(--admin-border)) !important;
  color: var(--admin-primary) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen .type-option) {
  background: var(--admin-surface-subtle) !important;
  border: 1px solid var(--admin-border-strong) !important;
  color: var(--admin-text-muted) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen .type-option.active) {
  background: var(--admin-primary-solid) !important;
  border-color: var(--admin-primary-solid) !important;
  color: #ffffff !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen .quiz-outline-sidebar) {
  background: var(--admin-surface) !important;
  border-color: var(--admin-border) !important;
  color: var(--admin-text) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen .outline-item) {
  background: var(--admin-surface-subtle) !important;
  border-color: var(--admin-border) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen .outline-item.is-active) {
  background: var(--admin-primary-soft) !important;
  border-color: var(--admin-primary) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen .outline-item.has-error) {
  border-color: var(--admin-danger) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen :is(.image-required-warning, .multi-select-warning)) {
  background: var(--admin-warning-soft) !important;
  border: 1px solid color-mix(in srgb, var(--admin-warning) 40%, var(--admin-border)) !important;
  color: var(--admin-warning) !important;
}

:global(html[data-admin-appearance="true"] .module-editor-screen .has-field-error) {
  border-color: var(--admin-danger) !important;
}

@media (max-width: 768px) {
  .module-editor-header {
    position: static;
  }

  .module-editor-header-inner {
    padding: 12px max(12px, env(safe-area-inset-right, 0px)) 12px max(12px, env(safe-area-inset-left, 0px));
  }

  .editor-identity {
    flex: 1 0 100%;
  }

  .name-field-wrap {
    flex: 1 1 180px;
    max-width: none;
  }

  .editor-actions {
    width: 100%;
    justify-content: flex-start;
  }

  .overflow-menu-toggle {
    display: inline-flex;
  }

  .secondary-actions-menu {
    display: none;
    position: absolute;
    top: calc(100% + 6px);
    left: 0;
    z-index: 360;
    flex-direction: column;
    align-items: stretch;
    min-width: 210px;
    padding: 6px;
    border-radius: 10px;
    background: var(--admin-surface, #ffffff);
    border: 1px solid var(--admin-border, #dde1e7);
    box-shadow: 0 12px 28px rgba(16, 24, 40, 0.2);
  }

  .secondary-actions-menu.is-open {
    display: flex;
  }

  .secondary-action-btn {
    width: 100%;
    justify-content: flex-start;
    border-color: transparent;
  }

  .save-primary-btn {
    margin-left: auto;
  }

  .editor-body {
    padding: 16px max(12px, env(safe-area-inset-right, 0px)) 0 max(12px, env(safe-area-inset-left, 0px));
  }

  .editor-btn,
  .module-name-input,
  .setting-select,
  .checkbox-setting-label,
  .settings-collapse-trigger,
  .template-item,
  .close-btn {
    min-height: 44px;
  }

  .close-btn,
  .overflow-menu-toggle {
    min-width: 44px;
  }
}

@media (pointer: coarse) {
  .editor-btn,
  .module-name-input,
  .setting-select,
  .checkbox-setting-label,
  .settings-collapse-trigger,
  .template-item,
  .close-btn {
    min-height: 44px;
  }

  .close-btn {
    min-width: 44px;
  }
}
</style>
