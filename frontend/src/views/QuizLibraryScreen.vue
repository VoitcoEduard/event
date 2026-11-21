<template>
  <div class="library-screen">
    <header class="header">
      <div class="header-top">
        <button type="button" class="back-link" @click="goBack">
          <AppIcon name="arrowLeft" :size="18" aria-hidden="true" />
          <span>{{ $t('library.back_btn') }}</span>
        </button>
        <div class="language-switcher">
          <select
            v-model="$i18n.locale"
            :aria-label="$t('library.language_label')"
            @change="changeLanguage"
          >
            <option value="en">English</option>
            <option value="ru">Русский</option>
            <option value="ro">Română</option>
          </select>
        </div>
      </div>
      <div class="header-main">
        <div class="header-title-block">
          <p class="header-eyebrow">{{ $t('library.title') }}</p>
          <h1 class="header-heading">{{ $t('library.manage_templates') }}</h1>
        </div>
        <div class="header-actions">
          <button
            v-if="activeTab !== 'screensaver' && activeTab !== 'music'"
            type="button"
            class="create-btn"
            @click="createNewTemplate"
          >
            {{ activeTab === 'form' ? $t('library.new_form') : (activeTab === 'quiz' ? $t('library.new_quiz') : $t('library.new_wrong_pictures')) }}
          </button>
          <button
            v-if="activeTab === 'screensaver'"
            type="button"
            class="create-btn"
            @click="triggerScreensaverUpload"
          >
            {{ $t('library.upload_screensaver') }}
          </button>
          <button
            v-if="activeTab === 'music'"
            type="button"
            class="create-btn"
            @click="triggerTrackUpload"
          >
            {{ $t('library.upload_music') }}
          </button>
          <button
            v-if="activeTab !== 'screensaver' && activeTab !== 'music'"
            type="button"
            class="import-btn"
            @click="triggerFileInput"
          >
            {{ $t('library.import') }}
          </button>
          <input
            type="file"
            ref="fileInput"
            style="display: none"
            accept=".json"
            @change="handleFileUpload"
          />
          <input
            type="file"
            ref="screensaverInput"
            style="display: none"
            accept="image/jpeg,image/png,image/gif,image/webp"
            @change="handleScreensaverUpload"
          />
          <input
            type="file"
            ref="trackInput"
            style="display: none"
            accept="audio/mpeg,audio/wav,audio/ogg"
            @change="handleTrackUpload"
          />
        </div>
      </div>
    </header>

    <div class="content-wrapper">
      <!-- Navigation Tabs (Vertical sidebar on desktop, horizontal scrollable bar on mobile) -->
      <div class="sidebar" :class="{ 'more-start': hasMoreStart, 'more-end': hasMoreEnd }">
        <div
          ref="tablistRef"
          class="library-tabs"
          :class="{ 'more-start': hasMoreStart, 'more-end': hasMoreEnd }"
          role="tablist"
          :aria-label="$t('library.title')"
          :aria-orientation="isCompactLayout ? 'horizontal' : 'vertical'"
          @keydown="handleTabKeydown"
          @scroll.passive="updateTabScrollCues"
        >
          <button
            v-for="tab in tabs"
            :key="tab.id"
            :ref="(el) => setTabRef(tab.id, el)"
            type="button"
            role="tab"
            :id="'library-tab-' + tab.id"
            class="tab-btn"
            :class="{ active: activeTab === tab.id }"
            :aria-selected="activeTab === tab.id ? 'true' : 'false'"
            :tabindex="activeTab === tab.id ? 0 : -1"
            @click="selectTab(tab.id)"
          >
            <AppIcon class="tab-icon" :name="tab.icon" :size="20" aria-hidden="true" />
            <span class="tab-label">{{ $t(tab.labelKey) }}</span>
          </button>
        </div>
      </div>

      <!-- Main Content Area -->
      <main
        class="main-content"
        role="tabpanel"
        :aria-labelledby="'library-tab-' + activeTab"
      >
        <div v-if="loading" class="loading-state">
          <p>{{ $t('common.loading') }}</p>
        </div>

        <!-- Screensavers Tab Content -->
        <div v-else-if="activeTab === 'screensaver'" class="screensaver-view">
          <div v-if="screensavers.length === 0" class="empty-state">
            <AppIcon class="empty-icon" name="image" :size="48" aria-hidden="true" />
            <p class="empty-text">{{ $t('library.empty.no_screensavers') }}</p>
            <p class="empty-hint">{{ $t('library.empty.screensaver_hint') }}</p>
          </div>

          <div v-else class="screensavers-grid">
            <div v-for="screensaver in screensavers" :key="screensaver.id" class="screensaver-card">
              <div class="screensaver-image-container">
                <img :src="screensaver.thumbnailUrl" :alt="screensaver.name" class="screensaver-thumbnail" decoding="async" loading="lazy" />
              </div>
              <div class="screensaver-footer">
                <div class="screensaver-info">
                  <h2 class="screensaver-name">{{ screensaver.name }}</h2>
                  <p class="screensaver-date">{{ $t('library.meta.added') }}: {{ formatDate(screensaver.createdAt) }}</p>
                </div>
                <div class="screensaver-actions">
                  <button
                    type="button"
                    class="icon-btn delete-btn"
                    @click="handleDeleteScreensaver(screensaver)"
                    :aria-label="`${$t('admin.delete_btn')} ${screensaver.name}`"
                  >
                    <AppIcon name="trash" :size="17" aria-hidden="true" />
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Music Tab Content -->
        <div v-else-if="activeTab === 'music'" class="music-view">
          <div v-if="tracks.length === 0" class="empty-state">
            <AppIcon class="empty-icon" name="music" :size="48" aria-hidden="true" />
            <p class="empty-text">{{ $t('library.empty.no_music') }}</p>
            <p class="empty-hint">{{ $t('library.empty.music_hint') }}</p>
          </div>

          <div v-else class="tracks-list">
            <div v-for="track in tracks" :key="track.id" class="track-card">
              <div class="track-icon">
                <AppIcon name="music" :size="22" aria-hidden="true" />
              </div>
              <div class="track-info">
                <h2 class="track-name">{{ track.name }}</h2>
                <p class="track-date">{{ $t('library.meta.added') }}: {{ formatDate(track.createdAt) }}</p>
              </div>
              <div class="track-actions">
                <button
                  type="button"
                  class="icon-btn delete-btn"
                  @click="handleDeleteTrack(track)"
                  :aria-label="`${$t('admin.delete_btn')} ${track.name}`"
                >
                  <AppIcon name="trash" :size="17" aria-hidden="true" />
                </button>
              </div>
              <div class="track-preview">
                <audio controls :src="`${API_BASE_URL}${track.url}`" class="audio-player"></audio>
              </div>
            </div>
          </div>
        </div>

        <div v-else-if="filteredTemplates.length === 0" class="empty-state">
          <AppIcon class="empty-icon" :name="activeTab === 'form' ? 'form' : (activeTab === 'wrong_pictures' ? 'image' : 'quiz')" :size="48" aria-hidden="true" />
          <p class="empty-text">{{ $t('library.empty.no_templates', { type: activeTabLabel }) }}</p>
        </div>

        <div v-else class="templates-grid">
          <div v-for="template in filteredTemplates" :key="template.id" class="template-card">
            <h2 class="template-name">{{ template.name }}</h2>
            <p class="template-desc">{{ template.description || $t('library.meta.no_desc') }}</p>
            <div class="template-meta">
              <span v-if="activeTab === 'quiz' || activeTab === 'wrong_pictures'" class="question-count">{{ $t('library.meta.questions', { count: (template.data?.questions || []).length }) }}</span>
              <span v-else class="question-count">{{ $t('library.meta.steps', { count: (template.data?.timeline || []).length }) }}</span>
              <span class="created-at">{{ formatDate(template.createdAt) }}</span>
            </div>
            <div class="card-actions">
              <button
                type="button"
                class="card-action-btn edit-btn"
                @click="editTemplate(template)"
                :aria-label="`${$t('admin.edit_btn')} ${template.name}`"
              >
                <AppIcon name="edit" :size="16" aria-hidden="true" />
                <span>{{ $t('admin.edit_btn') }}</span>
              </button>
              <button
                type="button"
                class="card-action-btn export-btn"
                @click="exportTemplate(template)"
                :aria-label="`${$t('library.actions.export_btn')} ${template.name}`"
              >
                <AppIcon name="download" :size="16" aria-hidden="true" />
                <span>{{ $t('library.actions.export_btn') }}</span>
              </button>
              <button
                type="button"
                class="card-action-btn delete-btn"
                @click="deleteTemplate(template)"
                :aria-label="`${$t('admin.delete_btn')} ${template.name}`"
              >
                <AppIcon name="trash" :size="16" aria-hidden="true" />
                <span>{{ $t('admin.delete_btn') }}</span>
              </button>
            </div>
          </div>
        </div>
      </main>
    </div>

    <!-- Edit Template Modal -->
    <div v-if="showEditor" class="modal-overlay" @click.self="closeEditor">
      <div
        ref="modalContainerRef"
        class="modal-container"
        role="dialog"
        aria-modal="true"
        :aria-label="currentTemplate.name || $t('library.manage_templates')"
        tabindex="-1"
        @keydown.escape.stop.prevent="closeEditor"
      >
        <div class="modal-header">
          <input
            v-model="currentTemplate.name"
            class="template-name-input"
            :aria-label="$t('library.actions.template_name_placeholder')"
            :placeholder="$t('library.actions.template_name_placeholder')"
          />
          <div class="header-buttons">
            <button type="button" class="save-btn" @click="saveTemplate" :disabled="saving">
              {{ saving ? $t('library.actions.saving_btn') : $t('library.actions.save_btn') }}
            </button>
            <button type="button" class="close-btn" @click="closeEditor" :aria-label="$t('common.close')">
              <AppIcon name="close" :size="18" aria-hidden="true" />
            </button>
          </div>
        </div>

        <div class="modal-body">
          <textarea
            v-model="currentTemplate.description"
            class="template-desc-input"
            :placeholder="$t('library.actions.desc_placeholder')"
            rows="2"
          ></textarea>

          <!-- Dynamic Editor based on Type -->
          <QuizEditor
            v-if="currentTemplate.moduleType === 'quiz'"
            :questions="currentTemplate.questions"
            event-id=""
            module-id=""
            @update-question="handleUpdateQuestion"
            @remove-question="handleRemoveQuestion"
            @add-question="(type) => handleAddQuestion(type)"
            @move-question="handleMoveQuestion"
            class="editor-component"
          />

          <FormEditor
            v-if="currentTemplate.moduleType === 'form'"
            :timeline="currentTemplate.timeline"
            @update:timeline="handleUpdateTimeline"
            class="editor-component"
          />

          <QuizEditor
            v-if="currentTemplate.moduleType === 'wrong_pictures'"
            :questions="currentTemplate.questions"
            event-id=""
            module-id=""
            @update-question="handleUpdateQuestion"
            @remove-question="handleRemoveQuestion"
            @add-question="(type) => handleAddQuestion(type)"
            @move-question="handleMoveQuestion"
            class="editor-component"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { API_BASE_URL, getAuthHeaders, normalizeTemplates } from '../utils'
import { useToast } from '../composables/useToast'
import QuizEditor from '../components/QuizEditor.vue'
import FormEditor from '../components/FormEditor.vue'
import { getScreensavers, uploadScreensaver, deleteScreensaver, getTracks, uploadTrack, deleteTrack, getImageUrl } from '../services/media'
import AppIcon from '../components/AppIcon.vue'

const { t, locale } = useI18n()
const router = useRouter()
const { success, error, info } = useToast()

const changeLanguage = () => {
  localStorage.setItem('userLanguage', locale.value)
}

const tabs = [
  { id: 'form', labelKey: 'library.tabs.forms', icon: 'form' },
  { id: 'quiz', labelKey: 'library.tabs.quizzes', icon: 'quiz' },
  { id: 'screensaver', labelKey: 'library.tabs.screensavers', icon: 'image' },
  { id: 'music', labelKey: 'library.tabs.music', icon: 'music' },
  { id: 'wrong_pictures', labelKey: 'library.tabs.wrong_pictures', icon: 'image' }
]

const activeTab = ref('form')
const activeTabLabel = computed(() => {
  const currentTab = tabs.find(tab => tab.id === activeTab.value)
  return t(currentTab?.labelKey || 'library.tabs.forms')
})
const tablistRef = ref(null)
const modalContainerRef = ref(null)
const tabRefs = {}
const isCompactLayout = ref(false)
const hasMoreStart = ref(false)
const hasMoreEnd = ref(false)

const updateTabScrollCues = () => {
  const scroller = tablistRef.value
  if (!scroller) return
  const maxScrollLeft = scroller.scrollWidth - scroller.clientWidth
  if (maxScrollLeft <= 1) {
    hasMoreStart.value = false
    hasMoreEnd.value = false
    return
  }
  const scrollLeft = Math.max(0, scroller.scrollLeft)
  hasMoreStart.value = scrollLeft > 2
  hasMoreEnd.value = scrollLeft < maxScrollLeft - 2
}

const setTabRef = (tabId, el) => {
  if (el) {
    tabRefs[tabId] = el
  } else {
    delete tabRefs[tabId]
  }
}

const scrollTabIntoView = (el) => {
  if (!el) return
  const scroller = tablistRef.value
  const shouldCenter = isCompactLayout.value || (scroller && scroller.scrollWidth > scroller.clientWidth)
  el.scrollIntoView({
    block: 'nearest',
    inline: shouldCenter ? 'center' : 'nearest'
  })
  updateTabScrollCues()
}

const selectTab = (tabId) => {
  activeTab.value = tabId
  nextTick(() => {
    scrollTabIntoView(tabRefs[tabId])
  })
}

const handleTabKeydown = (event) => {
  const prevKeys = ['ArrowLeft', 'ArrowUp']
  const nextKeys = ['ArrowRight', 'ArrowDown']
  if (![...prevKeys, ...nextKeys, 'Home', 'End'].includes(event.key)) return

  let currentIndex = tabs.findIndex(tab => tabRefs[tab.id] === document.activeElement)
  if (currentIndex === -1) {
    currentIndex = tabs.findIndex(tab => tab.id === activeTab.value)
  }

  let nextIndex = currentIndex
  if (nextKeys.includes(event.key)) {
    nextIndex = (currentIndex + 1) % tabs.length
  } else if (prevKeys.includes(event.key)) {
    nextIndex = (currentIndex - 1 + tabs.length) % tabs.length
  } else if (event.key === 'Home') {
    nextIndex = 0
  } else if (event.key === 'End') {
    nextIndex = tabs.length - 1
  }

  event.preventDefault()
  const targetEl = tabRefs[tabs[nextIndex].id]
  if (targetEl) {
    targetEl.focus()
    scrollTabIntoView(targetEl)
  }
}

const templates = ref([])
const screensavers = ref([])
const tracks = ref([])
const loading = ref(true)
const showEditor = ref(false)
const saving = ref(false)
const fileInput = ref(null)
const screensaverInput = ref(null)
const trackInput = ref(null)
const currentTemplate = ref({
  id: null,
  name: '',
  description: '',
  moduleType: 'form',
  questions: [],
  timeline: []
})

const filteredTemplates = computed(() => {
    return templates.value.filter(t => t.moduleType === activeTab.value)
})

const fetchTemplates = async () => {
  loading.value = true
  try {
    const response = await fetch(`${API_BASE_URL}/templates`)
    if (response.ok) {
        const data = normalizeTemplates(await response.json())
        // Ensure legacy templates have a type (default to quiz if missing, based on data structure)
        templates.value = data.map(t => ({
            ...t,
            moduleType: t.moduleType || (t.data?.questions ? 'quiz' : 'form')
        }))
    } else {
      error('Failed to load templates')
    }
  } catch (err) {
    console.error(err)
    error('Error loading templates')
  } finally {
    loading.value = false
  }
}

const fetchScreensavers = async () => {
  loading.value = true
  try {
    const data = await getScreensavers()
    screensavers.value = data
  } catch (err) {
    console.error(err)
    error('Error loading screensavers')
  } finally {
    loading.value = false
  }
}

const fetchTracks = async () => {
  loading.value = true
  try {
    const data = await getTracks()
    tracks.value = data
  } catch (err) {
    console.error(err)
    error('Error loading tracks')
  } finally {
    loading.value = false
  }
}

const handleScreensaverUpload = async (event) => {
    const file = event.target.files[0]
    if (!file) return

    try {
        const name = prompt(t('library.actions.template_name_placeholder'), file.name)
        if (!name) return
        
        await uploadScreensaver(file, name)
        success(t('admin.details.toast.screensaver_updated'))
        fetchScreensavers()
    } catch (err) {
        console.error(err)
        error('Failed to upload screensaver')
    } finally {
        event.target.value = ''
    }
}

const handleDeleteScreensaver = async (screensaver) => {
    if (!confirm(t('library.actions.delete_confirm', { name: screensaver.name }))) return
    
    try {
        await deleteScreensaver(screensaver.id)
        success(t('admin.details.toast.screensaver_removed'))
        fetchScreensavers()
    } catch (err) {
        console.error(err)
        error('Failed to delete screensaver')
    }
}

const handleTrackUpload = async (event) => {
    const file = event.target.files[0]
    if (!file) return

    try {
        const name = prompt(t('library.actions.template_name_placeholder'), file.name)
        if (!name) return
        
        await uploadTrack(file, name)
        success('Track uploaded successfully')
        fetchTracks()
    } catch (err) {
        console.error(err)
        error('Failed to upload track')
    } finally {
        event.target.value = ''
    }
}

const handleDeleteTrack = async (track) => {
    if (!confirm(t('library.actions.delete_confirm', { name: track.name }))) return
    
    try {
        await deleteTrack(track.id)
        success('Track deleted')
        fetchTracks()
    } catch (err) {
        console.error(err)
        error('Failed to delete track')
    }
}

let mediaQueryList = null
let handleMediaChange = null
let tabResizeObserver = null
let previouslyFocusedEl = null

const handleGlobalKeydown = (event) => {
  if (event.key === 'Escape' && showEditor.value && !event.defaultPrevented) {
    if (typeof document !== 'undefined' && document.querySelector('body > .modal-overlay')) {
      return
    }
    event.preventDefault()
    closeEditor()
  }
}

watch(showEditor, (isOpen) => {
  if (typeof document === 'undefined') return
  if (isOpen) {
    previouslyFocusedEl = document.activeElement
    nextTick(() => {
      modalContainerRef.value?.focus()
    })
  } else if (previouslyFocusedEl && typeof previouslyFocusedEl.focus === 'function') {
    const target = previouslyFocusedEl
    previouslyFocusedEl = null
    nextTick(() => {
      if (target.isConnected) target.focus()
    })
  }
})

watch(locale, () => {
  nextTick(updateTabScrollCues)
})

onMounted(() => {
  fetchTemplates()

  if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
    mediaQueryList = window.matchMedia('(max-width: 768px)')
    isCompactLayout.value = mediaQueryList.matches
    handleMediaChange = (event) => {
      isCompactLayout.value = event.matches
      nextTick(updateTabScrollCues)
    }
    mediaQueryList.addEventListener('change', handleMediaChange)
  }

  const scroller = tablistRef.value
  if (scroller) {
    nextTick(updateTabScrollCues)
    if (typeof ResizeObserver !== 'undefined') {
      tabResizeObserver = new ResizeObserver(() => {
        updateTabScrollCues()
      })
      tabResizeObserver.observe(scroller)
      Array.from(scroller.children).forEach((child) => tabResizeObserver.observe(child))
    }
  }

  if (typeof window !== 'undefined') {
    window.addEventListener('resize', updateTabScrollCues, { passive: true })
    window.addEventListener('keydown', handleGlobalKeydown)
  }

  if (typeof document !== 'undefined' && document.fonts?.ready) {
    document.fonts.ready.then(() => updateTabScrollCues()).catch(() => {})
  }
})

onUnmounted(() => {
  if (mediaQueryList && handleMediaChange) {
    mediaQueryList.removeEventListener('change', handleMediaChange)
  }
  if (tabResizeObserver) {
    tabResizeObserver.disconnect()
  }
  if (typeof window !== 'undefined') {
    window.removeEventListener('resize', updateTabScrollCues)
    window.removeEventListener('keydown', handleGlobalKeydown)
  }
})

// Watch for tab changes
watch(activeTab, (newTab) => {
    if (newTab === 'screensaver') {
        fetchScreensavers()
    } else if (newTab === 'music') {
        fetchTracks()
    }
})

const goBack = () => {
  router.push('/admin')
}

const createNewTemplate = () => {
  let defaultName = 'New Template'
  if (activeTab.value === 'form') defaultName = 'New Form Template'
  else if (activeTab.value === 'quiz') defaultName = 'New Quiz Template'
  else if (activeTab.value === 'wrong_pictures') defaultName = 'New Wrong Pictures Template'

  currentTemplate.value = {
    id: null,
    name: defaultName,
    description: '',
    moduleType: activeTab.value,
    questions: [],
    timeline: []
  }
  showEditor.value = true
}

const editTemplate = (template) => {
  currentTemplate.value = {
    id: template.id,
    name: template.name,
    description: template.description,
    moduleType: template.moduleType,
    questions: JSON.parse(JSON.stringify(template.data?.questions || [])).map(q => {
        if (q.type === 'range') {
            if (!q.rangeConfig) q.rangeConfig = { minValue: 0, maxValue: 100, step: 1, unit: '', decimalPlaces: 0 }
            if (!q.scoringConfig) q.scoringConfig = { mode: 'symmetric', maxPoints: 100, tolerance: 10, partialPercent: 0 }
        }
        if (q.type === 'true_false') {
             q.answers = ['True', 'False', '', '']
            if (q.correctIndex > 1) q.correctIndex = 0
        }
        return q
    }),
    timeline: JSON.parse(JSON.stringify(template.data?.timeline || []))
  }
  showEditor.value = true
}

const deleteTemplate = async (template) => {
  if (!confirm(t('library.actions.delete_confirm', { name: template.name }))) return

  try {
    const response = await fetch(`${API_BASE_URL}/templates/${template.id}`, {
      method: 'DELETE',
      headers: { ...getAuthHeaders() }
    })

    if (response.ok) {
      success('Template deleted')
      fetchTemplates()
    } else {
      error('Failed to delete template')
    }
  } catch (err) {
    console.error(err)
    error('Error deleting template')
  }
}

const saveTemplate = async () => {
  if (!currentTemplate.value.name.trim()) {
    info(t('admin.messages.fill_fields'))
    return
  }
  
  const isQuiz = currentTemplate.value.moduleType === 'quiz'
  const isWrongPictures = currentTemplate.value.moduleType === 'wrong_pictures'
  
  if ((isQuiz || isWrongPictures) && currentTemplate.value.questions.length === 0) {
    if (!confirm(t('library.actions.save_empty_confirm'))) return
  }

   saving.value = true
  try {
    const payload = {
      name: currentTemplate.value.name,
      description: currentTemplate.value.description,
      moduleType: currentTemplate.value.moduleType,
      data: (isQuiz || isWrongPictures) 
        ? { questions: currentTemplate.value.questions } 
        : { timeline: currentTemplate.value.timeline }
    }

    let url = `${API_BASE_URL}/templates`
    let method = 'POST'

    if (currentTemplate.value.id) {
      url = `${API_BASE_URL}/templates/${currentTemplate.value.id}`
      method = 'PUT'
    }

    const response = await fetch(url, {
      method: method,
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify(payload)
    })

    if (response.ok) {
      const result = await response.json()
      const saved = method === 'POST' ? response.status === 201 && !!result?.id : result?.message === 'Template updated'
      if (!saved) {
        error('Failed to save template')
        return
      }
      success('Template saved successfully')
      showEditor.value = false
      fetchTemplates()
    } else {
      error('Failed to save template')
    }
  } catch (err) {
    console.error(err)
    error('Error saving template')
  } finally {
    saving.value = false
  }
}

const closeEditor = () => {
    if(confirm(t('library.actions.discard_confirm'))) {
        showEditor.value = false
    }
}

// Quiz Editor Handlers
const handleUpdateQuestion = (index, updatedQ) => {
    currentTemplate.value.questions[index] = updatedQ
}

const handleRemoveQuestion = (index) => {
    if (!confirm(t('library.actions.delete_confirm', { name: 'this question' }))) return
    currentTemplate.value.questions.splice(index, 1)
}

const handleAddQuestion = (type = 'classic') => {
    const isWrongPictures = currentTemplate.value.moduleType === 'wrong_pictures'
    const finalType = isWrongPictures ? 'wrong_pictures' : type
    
    const newQuestion = {
        id: Date.now().toString() + Math.random().toString(36).substr(2, 5),
        type: finalType,
        text: '',
        answers: (finalType === 'wrong_pictures' || finalType === 'correct_picture') ? [] : ['', '', '', ''],
        correctIndex: 0,
        timeLimit: finalType === 'wrong_pictures' ? 45 : 20,
        explainTime: 5
    }

    // Add type-specific defaults
    if (finalType === 'range') {
        newQuestion.rangeConfig = { minValue: 0, maxValue: 100, step: 1, unit: '', decimalPlaces: 0 }
        newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 100, tolerance: 10, partialPercent: 0 }
        newQuestion.correctValue = 0
    } else if (finalType === 'number') {
        newQuestion.correctNumber = 0
        newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 50, tolerance: 0, partialPercent: 0 }
    } else if (finalType === 'correct_picture') {
        newQuestion.imageOptions = [
            { id: Date.now().toString(), imageId: '', label: '' },
            { id: (Date.now() + 1).toString(), imageId: '', label: '' }
        ]
        // A7: choice scoring reads scoringConfig.maxPoints.
        newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 50, tolerance: 0, partialPercent: 0 }
    } else if (finalType === 'wrong_pictures') {
        newQuestion.wrongPictureConfig = { targets: [], tolerancePercent: 5, maxPoints: 50 }
    } else if (finalType === 'true_false') {
        newQuestion.answers = [t('quiz.player.true'), t('quiz.player.false'), '', '']
        // A7: choice scoring reads scoringConfig.maxPoints.
        newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 50, tolerance: 0, partialPercent: 0 }
    } else if (finalType === 'multi_select' || finalType === 'classic') {
        // A7: choice scoring reads scoringConfig.maxPoints.
        newQuestion.scoringConfig = { mode: 'symmetric', maxPoints: 50, tolerance: 0, partialPercent: 0 }
    }

    currentTemplate.value.questions.push(newQuestion)
}

const handleMoveQuestion = (index, direction) => {
    const newIndex = index + direction
    if (newIndex < 0 || newIndex >= currentTemplate.value.questions.length) return
    const temp = currentTemplate.value.questions[index]
    currentTemplate.value.questions[index] = currentTemplate.value.questions[newIndex]
    currentTemplate.value.questions[newIndex] = temp
}

// Form Editor Handlers
const handleUpdateTimeline = (newTimeline) => {
    currentTemplate.value.timeline = newTimeline
}


const formatDate = (dateString) => {
  if (!dateString) return ''
  return new Date(dateString).toLocaleDateString()
}
const triggerFileInput = () => {
    fileInput.value.click()
}

const triggerScreensaverUpload = () => {
    screensaverInput.value.click()
}

const triggerTrackUpload = () => {
    trackInput.value.click()
}

const handleFileUpload = (event) => {
    const file = event.target.files[0]
    if (!file) return

    const reader = new FileReader()
    reader.onload = async (e) => {
        try {
            const content = JSON.parse(e.target.result)
            
            // Validate content structure roughly
            if(!content.name || !content.moduleType) {
                error('Invalid template file format')
                return
            }

            // Ask User for confirmation or just save
            if(!confirm(`Import template "${content.name}"?`)) return

            // Prepare payload (clean up ID if present to create new)
            const payload = {
                name: content.name + ' (Imported)',
                description: content.description || '',
                moduleType: content.moduleType,
                data: content.data || {}
            }
            // Sanitize questions if quiz
            if (payload.moduleType === 'quiz' && payload.data.questions) {
                 payload.data.questions = payload.data.questions.map(q => {
                    if (q.type === 'range') {
                        if (!q.rangeConfig) q.rangeConfig = { minValue: 0, maxValue: 100, step: 1, unit: '', decimalPlaces: 0 }
                        if (!q.scoringConfig) q.scoringConfig = { mode: 'symmetric', maxPoints: 100, tolerance: 10, partialPercent: 0 }
                    }
                    if (q.type === 'true_false') {
                        q.answers = ['True', 'False', '', '']
                        if (q.correctIndex > 1) q.correctIndex = 0
                    }
                    return q
                 })
            }

            const res = await fetch(`${API_BASE_URL}/templates`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
                body: JSON.stringify(payload)
            })

            if(res.ok) {
                success('Template imported successfully')
                fetchTemplates()
            } else {
                error('Failed to import template')
            }
        } catch (err) {
            console.error(err)
            error('Error parsing JSON file')
        } finally {
            event.target.value = '' // Reset input
        }
    }
    reader.readAsText(file)
}

const exportTemplate = (template) => {
    const dataStr = JSON.stringify(template, null, 2)
    const blob = new Blob([dataStr], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    
    const a = document.createElement('a')
    a.href = url
    a.download = `${template.name.replace(/[^a-z0-9]/gi, '_').toLowerCase()}_template.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
}
</script>

<style scoped>
.library-screen {
  container: library / inline-size;
  min-height: 100vh;
  min-height: 100dvh;
  background-color: #121212;
  color: white;
  display: flex;
  flex-direction: column;
  scroll-padding-bottom: calc(80px + env(safe-area-inset-bottom, 0px));
}

.header {
  padding: 16px 24px 20px;
  background-color: #1E1E1E;
  border-bottom: 1px solid #2C2C2C;
}

.header-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.back-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: #007AFF;
  font-size: 14px;
  font-weight: 600;
  background: none;
  border: 1px solid transparent;
  border-radius: 8px;
  cursor: pointer;
  padding: 6px 10px;
  margin-left: -10px;
  min-block-size: 36px;
  min-inline-size: 36px;
}

.language-switcher {
  margin-inline-start: auto;
}

.language-switcher select {
  background-color: #252525;
  color: white;
  border: 1px solid #333333;
  padding: 6px 10px;
  border-radius: 8px;
  cursor: pointer;
  outline: none;
  min-block-size: 36px;
}

.header-main {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  gap: 16px;
  flex-wrap: wrap;
}

.header-title-block {
  min-inline-size: 0;
}

.header-eyebrow {
  font-size: 12px;
  color: #888;
  text-transform: uppercase;
  font-weight: 600;
  letter-spacing: 1px;
  margin: 0 0 4px;
}

.header-heading {
  font-size: clamp(1.25rem, 1.05rem + 1vw, 1.5rem);
  font-weight: 700;
  color: #FFF;
  margin: 0;
  line-height: 1.2;
  text-wrap: balance;
}

.header-actions {
  display: flex;
  gap: 10px;
  align-items: center;
  flex-wrap: wrap;
}

.create-btn,
.import-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 18px;
  border-radius: 8px;
  font-weight: 600;
  font-size: 14px;
  cursor: pointer;
  min-block-size: 36px;
  min-inline-size: 36px;
  text-align: center;
}

.create-btn {
  background-color: #007AFF;
  color: white;
  border: 1px solid #007AFF;
}
.create-btn:hover { background-color: #005bb5; }

.import-btn {
  background-color: #333;
  color: white;
  border: 1px solid #555;
}
.import-btn:hover { background-color: #444; }

.content-wrapper {
  display: flex;
  flex: 1;
  min-height: 0;
}

.sidebar {
  width: 220px;
  flex-shrink: 0;
  background-color: #1E1E1E;
  border-right: 1px solid #2C2C2C;
}

.library-tabs {
  display: flex;
  flex-direction: column;
  padding: 12px 0;
}

.tab-btn {
  position: relative;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 20px;
  background: none;
  border: none;
  color: #888;
  cursor: pointer;
  font-size: 15px;
  font-weight: 600;
  text-align: left;
  transition: background-color 0.2s, color 0.2s;
  min-block-size: 40px;
}

.tab-btn:hover {
  background-color: #252525;
  color: white;
}

.tab-btn[aria-selected="true"] {
  background-color: #252525;
  color: #007AFF;
}

.tab-btn[aria-selected="true"]::after {
  content: "";
  position: absolute;
  inset: 4px auto 4px 0;
  border-inline-start: 3px solid var(--admin-primary, #007AFF);
}

.tab-icon { font-size: 20px; }

.main-content {
  flex: 1;
  min-inline-size: 0;
  padding: 24px 24px 80px;
  overflow-y: auto;
}

.loading-state,
.empty-state {
  text-align: center;
  padding: 40px 20px;
  color: #888;
  text-wrap: pretty;
}

.empty-icon { font-size: 48px; margin-bottom: 16px; }

.empty-hint {
  font-size: 14px;
  color: #666;
  margin-top: 8px;
}

.templates-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 300px), 1fr));
  gap: 16px;
}

.template-card {
  background-color: #1E1E1E;
  border: 1px solid #333;
  border-radius: 12px;
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-inline-size: 0;
  transition: border-color 0.2s;
}
.template-card:hover { border-color: #555; }

.template-name {
  font-size: 17px;
  font-weight: 650;
  line-height: 1.3;
  margin: 0;
  color: #fff;
  overflow-wrap: break-word;
  text-wrap: balance;
}

.template-desc {
  font-size: 14px;
  line-height: 1.45;
  color: #aaa;
  margin: 0;
  flex: 1;
  overflow-wrap: break-word;
  text-wrap: pretty;
}

.template-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px 12px;
  font-size: 12px;
  color: #888;
  border-top: 1px solid #333;
  padding-top: 10px;
}

.card-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding-top: 10px;
  border-top: 1px solid #333;
}

.card-action-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 12px;
  border-radius: 8px;
  border: 1px solid #444;
  background-color: #252525;
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  min-block-size: 36px;
  min-inline-size: 36px;
}

.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: 1px solid transparent;
  color: #888;
  cursor: pointer;
  padding: 6px;
  border-radius: 8px;
  min-block-size: 36px;
  min-inline-size: 36px;
}
.icon-btn:hover { background-color: #333; color: white; }
.delete-btn:hover { color: #ff3b30; }

/* Screensavers Grid */
.screensaver-view {
  padding: 0;
}

.screensavers-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 240px), 1fr));
  gap: 16px;
}

.screensaver-card {
  background-color: #1E1E1E;
  border: 1px solid #333;
  border-radius: 12px;
  overflow: hidden;
  min-inline-size: 0;
  transition: border-color 0.2s;
}
.screensaver-card:hover { border-color: #555; }

.screensaver-image-container {
  width: 100%;
  aspect-ratio: 16 / 9;
  overflow: hidden;
  background-color: #000;
}

.screensaver-thumbnail {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.screensaver-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
}

.screensaver-info {
  min-inline-size: 0;
  flex: 1;
}

.screensaver-name {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 4px 0;
  color: #fff;
  overflow-wrap: break-word;
  text-wrap: balance;
}

.screensaver-date {
  font-size: 12px;
  color: #888;
  margin: 0;
}

.screensaver-actions {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

/* Music View Styles */
.music-view {
  padding: 0;
  max-width: 860px;
  margin: 0 auto;
}

.tracks-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.track-card {
  background-color: #1E1E1E;
  border: 1px solid #333;
  border-radius: 12px;
  padding: 14px;
  display: grid;
  grid-template-columns: 44px minmax(0, 1fr) minmax(220px, 280px) auto;
  align-items: center;
  gap: 12px 14px;
  min-inline-size: 0;
}

.track-icon {
  background-color: #2C2C2E;
  color: #007AFF;
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
}

.track-info {
  min-inline-size: 0;
}

.track-name {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 4px 0;
  color: #fff;
  overflow-wrap: break-word;
}

.track-date {
  font-size: 12px;
  color: #888;
  margin: 0;
}

.track-preview {
  display: flex;
  align-items: center;
  min-inline-size: 0;
}

.audio-player {
  width: 100%;
  height: 36px;
}

.track-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

/* Modal Styles */
.modal-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background-color: rgba(0, 0, 0, 0.7);
  display: flex;
  justify-content: center;
  align-items: flex-end;
  z-index: 1000;
  overscroll-behavior: contain;
}

.modal-container {
  background-color: #1C1C1E;
  width: 100%;
  height: 95vh;
  height: 95dvh;
  max-width: 1000px;
  border-top-left-radius: 20px;
  border-top-right-radius: 20px;
  display: flex;
  flex-direction: column;
  box-shadow: 0 -4px 20px rgba(0,0,0,0.5);
  overscroll-behavior: contain;
}
.modal-container:focus { outline: none; }

.modal-header {
  padding: 20px;
  border-bottom: 1px solid #333;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.template-name-input {
  background: transparent;
  border: none;
  color: white;
  font-size: 24px;
  font-weight: bold;
  width: 100%;
  min-inline-size: 0;
  outline: none;
}
.template-name-input:focus-visible {
  outline: 2px solid var(--admin-primary, #007AFF);
  outline-offset: 2px;
  border-radius: 6px;
}
.template-name-input::placeholder { color: #555; }

.header-buttons {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-shrink: 0;
}

.save-btn {
  background-color: #30D158;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  font-weight: 600;
  cursor: pointer;
  min-block-size: 36px;
}
.save-btn:disabled { opacity: 0.5; }

.close-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  color: #888;
  cursor: pointer;
  border-radius: 8px;
  min-block-size: 36px;
  min-inline-size: 36px;
}

.modal-body {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 0;
}

.template-desc-input {
  background-color: #2C2C2E;
  border: none;
  color: white;
  padding: 16px 20px;
  font-size: 14px;
  resize: none;
  outline: none;
  border-bottom: 1px solid #333;
}

.editor-component {
  flex: 1;
  min-height: 0;
}

@media (min-width: 800px) {
  .modal-overlay { align-items: center; }
  .modal-container { height: 90vh; height: 90dvh; border-radius: 20px; }
}

/* Mobile & vertical container adaptation */
@container library (inline-size <= 768px) {
  .header {
    padding: max(12px, env(safe-area-inset-top, 0px)) max(16px, env(safe-area-inset-right, 0px)) 14px max(16px, env(safe-area-inset-left, 0px));
  }

  .header-main {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }

  .header-actions {
    width: 100%;
    display: flex;
    align-items: stretch;
    flex-wrap: wrap;
    gap: 8px;
  }

  .create-btn,
  .import-btn {
    min-block-size: 44px;
    text-wrap: balance;
    line-height: 1.25;
  }

  .create-btn {
    flex: 2 1 200px;
  }

  .import-btn {
    flex: 1 1 110px;
  }

  .content-wrapper {
    flex-direction: column;
    overflow: visible;
  }

  .sidebar {
    position: sticky;
    top: 0;
    z-index: 20;
    width: 100%;
    border-right: none;
    border-bottom: 1px solid #2C2C2C;
  }

  .sidebar::before,
  .sidebar::after {
    content: "";
    position: absolute;
    inset-block: 0;
    inline-size: 28px;
    pointer-events: none;
    z-index: 2;
    opacity: 0;
    transition: opacity 0.18s ease;
    background: linear-gradient(
      var(--edge-dir, to right),
      var(--admin-surface, #1E1E1E) 0%,
      color-mix(in srgb, var(--admin-surface, #1E1E1E) 75%, transparent) 45%,
      transparent 100%
    );
    box-shadow: inset var(--edge-shadow-x, 8px) 0 10px -10px rgba(0, 0, 0, 0.35);
  }

  .sidebar::before {
    --edge-dir: to right;
    --edge-shadow-x: 8px;
    inset-inline-start: 0;
  }

  .sidebar::after {
    --edge-dir: to left;
    --edge-shadow-x: -8px;
    inset-inline-end: 0;
  }

  .sidebar.more-start::before,
  .sidebar.more-end::after {
    opacity: 1;
  }

  .library-tabs {
    flex-direction: row;
    flex-wrap: nowrap;
    overflow-x: auto;
    overscroll-behavior-x: contain;
    padding: 0 max(16px, env(safe-area-inset-right, 0px)) 0 max(16px, env(safe-area-inset-left, 0px));
    gap: 4px;
    scroll-snap-type: inline proximity;
    scrollbar-width: none;
    --fade-size: 2.25rem;
    scroll-padding-inline: var(--fade-size);
    -webkit-mask-image: linear-gradient(
      to right,
      transparent 0,
      black var(--fade-start, 0px),
      black calc(100% - var(--fade-end, 0px)),
      transparent 100%
    );
    mask-image: linear-gradient(
      to right,
      transparent 0,
      black var(--fade-start, 0px),
      black calc(100% - var(--fade-end, 0px)),
      transparent 100%
    );
    transition: --fade-start 0.18s ease, --fade-end 0.18s ease;
  }

  .library-tabs::after {
    content: "";
    flex: 0 0 max(16px, env(safe-area-inset-right, 0px));
  }

  .library-tabs.more-start {
    --fade-start: var(--fade-size);
  }

  .library-tabs.more-end {
    --fade-end: var(--fade-size);
  }

  .library-tabs::-webkit-scrollbar {
    display: none;
  }

  .tab-btn {
    flex: none;
    white-space: nowrap;
    scroll-snap-align: start;
    min-block-size: 44px;
    padding: 10px 14px;
    font-size: 14px;
    gap: 8px;
    border-radius: 0;
  }

  .tab-btn[aria-selected="true"]::after {
    inset: auto 8px 0 8px;
    border-inline-start: none;
    border-block-end: 3px solid var(--admin-primary, #007AFF);
  }

  .main-content {
    padding: 16px max(16px, env(safe-area-inset-right, 0px)) calc(84px + env(safe-area-inset-bottom, 0px)) max(16px, env(safe-area-inset-left, 0px));
    overflow-y: visible;
  }

  .templates-grid,
  .screensavers-grid {
    gap: 12px;
  }

  .template-card {
    padding: 16px;
  }

  .card-action-btn {
    flex: 1 1 auto;
    min-block-size: 42px;
  }

  .track-card {
    grid-template-columns: 40px minmax(0, 1fr) auto;
    padding: 14px;
    gap: 10px 12px;
  }

  .track-icon {
    width: 40px;
    height: 40px;
  }

  .track-preview {
    grid-column: 1 / -1;
    width: 100%;
  }

  .audio-player {
    height: 40px;
  }
}

@media (max-width: 768px) {
  .modal-container {
    height: 100vh;
    height: 100dvh;
    border-radius: 0;
  }

  .modal-header {
    padding: max(12px, env(safe-area-inset-top, 0px)) max(14px, env(safe-area-inset-right, 0px)) 12px max(14px, env(safe-area-inset-left, 0px));
    flex-wrap: wrap;
    align-items: center;
    gap: 10px 8px;
  }

  .header-buttons {
    order: 1;
    width: 100%;
    justify-content: space-between;
    gap: 8px;
  }

  .template-name-input {
    order: 2;
    flex: 1 0 100%;
    width: 100%;
    font-size: clamp(16px, 4.5vw, 18px);
    line-height: 1.3;
    padding: 4px 0;
  }

  .template-desc-input {
    padding: 12px max(14px, env(safe-area-inset-right, 0px)) 12px max(14px, env(safe-area-inset-left, 0px));
    font-size: 16px;
  }

  .save-btn {
    min-block-size: 44px;
  }

  .close-btn {
    min-block-size: 44px;
    min-inline-size: 44px;
  }
}

@media (pointer: coarse) {
  .back-link,
  .language-switcher select,
  .create-btn,
  .import-btn,
  .tab-btn,
  .card-action-btn,
  .icon-btn,
  .save-btn,
  .close-btn {
    min-block-size: 44px;
    min-inline-size: 44px;
  }
}
</style>
