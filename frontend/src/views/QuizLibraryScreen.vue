<template>
  <AdminLayout :breadcrumbs="breadcrumbs">
    <template v-if="safeReturnTo" #actions>
      <RouterLink :to="safeReturnTo" class="back-link">
        <AppIcon name="arrowLeft" :size="16" aria-hidden="true" />
        <span>{{ $t('library.back_to_event') }}</span>
      </RouterLink>
    </template>

    <div class="library-screen">
      <header class="header">
        <div class="header-main">
          <div class="header-title-block">
            <p class="header-eyebrow">{{ $t('library.title') }}</p>
            <h1 class="header-heading">
              {{ isMediaTab ? $t('library.heading_media') : $t('library.heading_templates') }}
            </h1>
          </div>
          <div class="header-actions">
            <button
              v-if="!isMediaTab"
              type="button"
              class="create-btn"
              @click="createNewTemplate"
            >
              {{ newTemplateLabel }}
            </button>
            <button
              v-if="activeTab === 'screensavers'"
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
              v-if="!isMediaTab"
              type="button"
              class="import-btn"
              @click="triggerFileInput"
            >
              {{ $t('library.import') }}
            </button>
            <input
              ref="fileInput"
              type="file"
              style="display: none"
              accept=".json"
              @change="handleFileUpload"
            />
            <input
              ref="screensaverInput"
              type="file"
              style="display: none"
              accept="image/jpeg,image/png,image/gif,image/webp"
              @change="handleScreensaverUpload"
            />
            <input
              ref="trackInput"
              type="file"
              style="display: none"
              accept="audio/mpeg,audio/wav,audio/ogg"
              @change="handleTrackUpload"
            />
          </div>
        </div>
      </header>

      <div class="content-wrapper">
        <div class="sidebar">
          <div
            ref="tablistRef"
            class="library-tabs"
            role="tablist"
            :aria-label="$t('library.title')"
            :aria-orientation="isCompactLayout ? 'horizontal' : 'vertical'"
            @keydown="handleTabKeydown"
          >
            <button
              v-for="tab in tabs"
              :key="tab.id"
              :id="'library-tab-' + tab.id"
              :ref="(el) => setTabRef(tab.id, el)"
              type="button"
              role="tab"
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

        <main
          class="main-content"
          role="tabpanel"
          :aria-labelledby="'library-tab-' + activeTab"
        >
          <div v-if="loading" class="loading-state">
            <p>{{ $t('common.loading') }}</p>
          </div>

          <div v-else-if="activeTab === 'screensavers'" class="screensaver-view">
            <div v-if="screensavers.length === 0" class="empty-state">
              <AppIcon class="empty-icon" name="image" :size="48" aria-hidden="true" />
              <p class="empty-text">{{ $t('library.empty.no_screensavers') }}</p>
              <p class="empty-hint">{{ $t('library.empty.screensaver_hint') }}</p>
              <button type="button" class="create-btn empty-cta-btn" @click="triggerScreensaverUpload">
                {{ $t('library.upload_screensaver') }}
              </button>
            </div>

            <div v-else class="screensavers-grid">
              <div v-for="screensaver in screensavers" :key="screensaver.id" class="screensaver-card">
                <div class="screensaver-image-container">
                  <img
                    :src="screensaver.thumbnailUrl"
                    :alt="screensaver.name"
                    class="screensaver-thumbnail"
                    decoding="async"
                    loading="lazy"
                  />
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
                      :aria-label="`${$t('admin.delete_btn')} ${screensaver.name}`"
                      @click="handleDeleteScreensaver(screensaver)"
                    >
                      <AppIcon name="trash" :size="17" aria-hidden="true" />
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-else-if="activeTab === 'music'" class="music-view">
            <div v-if="tracks.length === 0" class="empty-state">
              <AppIcon class="empty-icon" name="music" :size="48" aria-hidden="true" />
              <p class="empty-text">{{ $t('library.empty.no_music') }}</p>
              <p class="empty-hint">{{ $t('library.empty.music_hint') }}</p>
              <button type="button" class="create-btn empty-cta-btn" @click="triggerTrackUpload">
                {{ $t('library.upload_music') }}
              </button>
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
                    :aria-label="`${$t('admin.delete_btn')} ${track.name}`"
                    @click="handleDeleteTrack(track)"
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
            <AppIcon class="empty-icon" :name="activeTabConfig.icon" :size="48" aria-hidden="true" />
            <p class="empty-text">{{ $t('library.empty.no_templates', { type: activeTabLabel }) }}</p>
            <p class="empty-hint">{{ $t(emptyTemplateHintKey) }}</p>
            <button type="button" class="create-btn empty-cta-btn" @click="createNewTemplate">
              {{ newTemplateLabel }}
            </button>
          </div>

          <div v-else class="templates-grid">
            <div v-for="template in filteredTemplates" :key="template.id" class="template-card">
              <h2 class="template-name">{{ template.name }}</h2>
              <p class="template-desc">{{ template.description || $t('library.meta.no_desc') }}</p>
              <div class="template-meta">
                <span class="question-count">{{ formatTemplateItemCount(template) }}</span>
                <span class="created-at">{{ formatDate(template.createdAt) }}</span>
              </div>
              <div class="card-actions">
                <button
                  type="button"
                  class="card-action-btn edit-btn"
                  :aria-label="`${$t('admin.edit_btn')} ${template.name}`"
                  @click="editTemplate(template)"
                >
                  <AppIcon name="edit" :size="16" aria-hidden="true" />
                  <span>{{ $t('admin.edit_btn') }}</span>
                </button>
                <button
                  type="button"
                  class="card-action-btn export-btn"
                  :aria-label="`${$t('library.actions.export_btn')} ${template.name}`"
                  @click="exportTemplate(template)"
                >
                  <AppIcon name="download" :size="16" aria-hidden="true" />
                  <span>{{ $t('library.actions.export_btn') }}</span>
                </button>
                <button
                  type="button"
                  class="card-action-btn delete-btn"
                  :aria-label="`${$t('admin.delete_btn')} ${template.name}`"
                  @click="deleteTemplate(template)"
                >
                  <AppIcon name="trash" :size="16" aria-hidden="true" />
                  <span>{{ $t('admin.delete_btn') }}</span>
                </button>
              </div>
            </div>
          </div>
        </main>
      </div>
    </div>
  </AdminLayout>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AdminLayout from '../components/AdminLayout.vue'
import AppIcon from '../components/AppIcon.vue'
import { API_BASE_URL, getAuthHeaders, normalizeTemplates } from '../utils'
import {
  LIBRARY_TABS,
  getLibraryTabConfig,
  resolveLibraryTab,
  resolveSafeReturnTo,
  sanitizeTemplateQuestions
} from '../utils/templateLibrary'
import { useToast } from '../composables/useToast'
import { useDialog } from '../composables/useDialog'
import {
  getScreensavers,
  uploadScreensaver,
  deleteScreensaver,
  getTracks,
  uploadTrack,
  deleteTrack
} from '../services/media'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const { success, error, info } = useToast()
const { confirm: confirmDialog, prompt: promptDialog } = useDialog()

const tabs = LIBRARY_TABS
const activeTab = ref(resolveLibraryTab(route?.params?.tab))
const activeTabConfig = computed(() => getLibraryTabConfig(activeTab.value))
const activeTabLabel = computed(() => t(activeTabConfig.value.labelKey))
const isMediaTab = computed(() => activeTabConfig.value.kind === 'media')

const safeReturnTo = computed(() => resolveSafeReturnTo(route?.query?.returnTo, router))

const breadcrumbs = computed(() => {
  if (safeReturnTo.value) {
    return [
      { label: t('admin_shell.events'), to: '/admin/events' },
      { label: t('library.back_to_event'), to: safeReturnTo.value },
      { label: t('library.title') },
      { label: activeTabLabel.value }
    ]
  }
  return [
    { label: t('library.title') },
    { label: activeTabLabel.value }
  ]
})

const newTemplateLabel = computed(() => {
  if (activeTab.value === 'forms') return t('library.new_form')
  if (activeTab.value === 'wrong-pictures') return t('library.new_wrong_pictures')
  return t('library.new_quiz')
})

const emptyTemplateHintKey = computed(() => {
  if (activeTab.value === 'forms') return 'library.empty.forms_hint'
  if (activeTab.value === 'wrong-pictures') return 'library.empty.wrong_pictures_hint'
  return 'library.empty.quizzes_hint'
})

const tablistRef = ref(null)
const tabRefs = {}
const isCompactLayout = ref(false)

const setTabRef = (tabId, el) => {
  if (el) {
    tabRefs[tabId] = el
  } else {
    delete tabRefs[tabId]
  }
}

const selectTab = (tabId) => {
  const canonical = resolveLibraryTab(tabId)
  activeTab.value = canonical
  const query = safeReturnTo.value ? { returnTo: safeReturnTo.value } : {}
  if (route?.params?.tab !== canonical) {
    router.push({ path: `/admin/library/${canonical}`, query })
  }
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
  }
}

const templates = ref([])
const templatesLoaded = ref(false)
const screensavers = ref([])
const tracks = ref([])
const loading = ref(true)
const fileInput = ref(null)
const screensaverInput = ref(null)
const trackInput = ref(null)

const filteredTemplates = computed(() => {
  const targetModuleType = activeTabConfig.value.moduleType
  return templates.value.filter(tpl => tpl.moduleType === targetModuleType)
})

const formatTemplateItemCount = (template) => {
  const type = template.moduleType
  if (type === 'form') {
    const count = (template.data?.timeline || []).length
    return t('library.meta.steps', { count }, count)
  }
  const count = (template.data?.questions || []).length
  if (type === 'wrong_pictures') {
    return t('library.meta.rounds', { count }, count)
  }
  return t('library.meta.questions', { count }, count)
}

const fetchTemplates = async () => {
  loading.value = true
  try {
    const response = await fetch(`${API_BASE_URL}/templates`)
    if (response.ok) {
      const data = normalizeTemplates(await response.json())
      templates.value = data.map(tpl => ({
        ...tpl,
        moduleType: tpl.moduleType || (tpl.data?.questions ? 'quiz' : 'form')
      }))
      templatesLoaded.value = true
    } else {
      error(t('library.toast.load_templates_failed'))
    }
  } catch (err) {
    console.error(err)
    error(t('library.toast.load_templates_failed'))
  } finally {
    loading.value = false
  }
}

const fetchScreensavers = async () => {
  loading.value = true
  try {
    screensavers.value = await getScreensavers()
  } catch (err) {
    console.error(err)
    error(t('library.toast.load_screensavers_failed'))
  } finally {
    loading.value = false
  }
}

const fetchTracks = async () => {
  loading.value = true
  try {
    tracks.value = await getTracks()
  } catch (err) {
    console.error(err)
    error(t('library.toast.load_tracks_failed'))
  } finally {
    loading.value = false
  }
}

const loadTabData = (tabId) => {
  if (tabId === 'screensavers') {
    fetchScreensavers()
  } else if (tabId === 'music') {
    fetchTracks()
  } else if (!templatesLoaded.value) {
    fetchTemplates()
  } else {
    loading.value = false
  }
}

watch(
  () => route?.params?.tab,
  (nextParam) => {
    const resolved = resolveLibraryTab(nextParam)
    activeTab.value = resolved
    loadTabData(resolved)
  },
  { immediate: true }
)

const createNewTemplate = () => {
  const query = safeReturnTo.value ? { returnTo: safeReturnTo.value } : {}
  router.push({
    path: `/admin/library/${activeTab.value}/new`,
    query
  })
}

const editTemplate = (template) => {
  const query = safeReturnTo.value ? { returnTo: safeReturnTo.value } : {}
  router.push({
    path: `/admin/library/${activeTab.value}/${encodeURIComponent(template.id)}/edit`,
    query
  })
}

const deleteTemplate = async (template) => {
  const confirmed = await confirmDialog({
    title: t('library.actions.delete_title'),
    message: t('library.actions.delete_confirm', { name: template.name }),
    confirmLabel: t('admin.delete_btn'),
    cancelLabel: t('common.cancel'),
    destructive: true
  })
  if (!confirmed) return

  try {
    const response = await fetch(`${API_BASE_URL}/templates/${template.id}`, {
      method: 'DELETE',
      headers: { ...getAuthHeaders() }
    })

    if (response.ok) {
      success(t('library.toast.template_deleted'))
      fetchTemplates()
    } else {
      error(t('library.toast.template_delete_failed'))
    }
  } catch (err) {
    console.error(err)
    error(t('library.toast.template_delete_failed'))
  }
}

const handleScreensaverUpload = async (event) => {
  const file = event.target.files?.[0]
  if (!file) return

  try {
    const defaultName = file.name.replace(/\.[^.]+$/, '') || file.name
    const name = await promptDialog({
      title: t('library.upload_screensaver'),
      label: t('library.actions.screensaver_name_label'),
      defaultValue: defaultName,
      placeholder: t('library.actions.screensaver_name_placeholder'),
      confirmLabel: t('library.actions.upload_btn'),
      cancelLabel: t('common.cancel')
    })
    if (name === null) return
    const trimmed = name.trim()
    if (!trimmed) {
      info(t('library.actions.media_name_required'))
      return
    }

    await uploadScreensaver(file, trimmed)
    success(t('library.toast.screensaver_uploaded'))
    fetchScreensavers()
  } catch (err) {
    console.error(err)
    error(t('library.toast.screensaver_upload_failed'))
  } finally {
    event.target.value = ''
  }
}

const handleDeleteScreensaver = async (screensaver) => {
  const confirmed = await confirmDialog({
    title: t('library.actions.delete_title'),
    message: t('library.actions.delete_confirm', { name: screensaver.name }),
    confirmLabel: t('admin.delete_btn'),
    cancelLabel: t('common.cancel'),
    destructive: true
  })
  if (!confirmed) return

  try {
    await deleteScreensaver(screensaver.id)
    success(t('admin.details.toast.screensaver_removed'))
    fetchScreensavers()
  } catch (err) {
    console.error(err)
    error(t('library.toast.screensaver_delete_failed'))
  }
}

const handleTrackUpload = async (event) => {
  const file = event.target.files?.[0]
  if (!file) return

  try {
    const defaultName = file.name.replace(/\.[^.]+$/, '') || file.name
    const name = await promptDialog({
      title: t('library.upload_music'),
      label: t('library.actions.track_name_label'),
      defaultValue: defaultName,
      placeholder: t('library.actions.track_name_placeholder'),
      confirmLabel: t('library.actions.upload_btn'),
      cancelLabel: t('common.cancel')
    })
    if (name === null) return
    const trimmed = name.trim()
    if (!trimmed) {
      info(t('library.actions.media_name_required'))
      return
    }

    await uploadTrack(file, trimmed)
    success(t('library.toast.track_uploaded'))
    fetchTracks()
  } catch (err) {
    console.error(err)
    error(t('library.toast.track_upload_failed'))
  } finally {
    event.target.value = ''
  }
}

const handleDeleteTrack = async (track) => {
  const confirmed = await confirmDialog({
    title: t('library.actions.delete_title'),
    message: t('library.actions.delete_confirm', { name: track.name }),
    confirmLabel: t('admin.delete_btn'),
    cancelLabel: t('common.cancel'),
    destructive: true
  })
  if (!confirmed) return

  try {
    await deleteTrack(track.id)
    success(t('library.toast.track_deleted'))
    fetchTracks()
  } catch (err) {
    console.error(err)
    error(t('library.toast.track_delete_failed'))
  }
}

const triggerFileInput = () => {
  fileInput.value?.click()
}

const triggerScreensaverUpload = () => {
  screensaverInput.value?.click()
}

const triggerTrackUpload = () => {
  trackInput.value?.click()
}

const handleFileUpload = (event) => {
  const file = event.target.files?.[0]
  if (!file) return

  const reader = new FileReader()
  reader.onload = async (e) => {
    try {
      const content = JSON.parse(e.target.result)
      const validTypes = new Set(['quiz', 'form', 'wrong_pictures'])
      if (!content?.name || !validTypes.has(content.moduleType)) {
        error(t('library.toast.invalid_template_file'))
        return
      }

      const confirmed = await confirmDialog({
        title: t('library.actions.import_title'),
        message: t('library.actions.import_confirm', { name: content.name }),
        confirmLabel: t('library.actions.import_btn'),
        cancelLabel: t('common.cancel')
      })
      if (!confirmed) return

      const rawData = content.data && typeof content.data === 'object' ? { ...content.data } : {}
      if ((content.moduleType === 'quiz' || content.moduleType === 'wrong_pictures') && Array.isArray(rawData.questions)) {
        rawData.questions = sanitizeTemplateQuestions(content.moduleType, rawData.questions)
      }

      const payload = {
        name: `${content.name} ${t('library.actions.imported_suffix')}`,
        description: content.description || '',
        moduleType: content.moduleType,
        data: rawData
      }

      const res = await fetch(`${API_BASE_URL}/templates`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
        body: JSON.stringify(payload)
      })

      if (res.ok) {
        success(t('library.toast.template_imported'))
        await fetchTemplates()
        const importedTab = resolveLibraryTab(content.moduleType)
        if (importedTab !== activeTab.value) {
          selectTab(importedTab)
        }
      } else {
        error(t('library.toast.template_import_failed'))
      }
    } catch (err) {
      console.error(err)
      error(t('library.toast.invalid_json'))
    } finally {
      event.target.value = ''
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
  a.download = `${String(template.name || 'template').replace(/[^a-z0-9]/gi, '_').toLowerCase()}_template.json`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
  success(t('library.toast.template_exported', { name: template.name }))
}

const formatDate = (dateString) => {
  if (!dateString) return ''
  return new Date(dateString).toLocaleDateString()
}

let mediaQueryList = null
let handleMediaChange = null

onMounted(() => {
  if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
    mediaQueryList = window.matchMedia('(max-width: 768px)')
    isCompactLayout.value = mediaQueryList.matches
    handleMediaChange = (event) => {
      isCompactLayout.value = event.matches
    }
    mediaQueryList.addEventListener('change', handleMediaChange)
  }
})

onUnmounted(() => {
  if (mediaQueryList && handleMediaChange) {
    mediaQueryList.removeEventListener('change', handleMediaChange)
  }
})
</script>

<style scoped>
.library-screen {
  container: library / inline-size;
  min-height: calc(100vh - 60px);
  min-height: calc(100dvh - 60px);
  background-color: #121212;
  color: white;
  display: flex;
  flex-direction: column;
  scroll-padding-bottom: calc(80px + env(safe-area-inset-bottom, 0px));
}

.back-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 38px;
  padding: 6px 12px;
  border-radius: 8px;
  border: 1px solid var(--admin-border-strong, #3a3a3c);
  background: var(--admin-surface, #252525);
  color: var(--admin-primary, #007AFF);
  font-size: 13px;
  font-weight: 600;
  text-decoration: none;
  white-space: nowrap;
}

.back-link:hover {
  background: var(--admin-surface-hover, #303034);
}

.header {
  padding: 20px clamp(16px, 3vw, 32px);
  background-color: #1E1E1E;
  border-bottom: 1px solid #2C2C2C;
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
  min-block-size: 38px;
  min-inline-size: 38px;
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

.tab-icon {
  flex-shrink: 0;
}

.main-content {
  flex: 1;
  min-inline-size: 0;
  padding: 24px clamp(16px, 3vw, 32px) 80px;
  overflow-y: auto;
}

.loading-state,
.empty-state {
  text-align: center;
  padding: 48px 20px;
  color: #888;
  text-wrap: pretty;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.empty-icon {
  margin-bottom: 8px;
}

.empty-text {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.empty-hint {
  font-size: 14px;
  color: #888;
  margin: 0 0 12px;
  max-width: 440px;
}

.empty-cta-btn {
  margin-top: 4px;
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

@container library (inline-size <= 768px) {
  .header {
    padding: 14px max(16px, env(safe-area-inset-right, 0px)) 14px max(16px, env(safe-area-inset-left, 0px));
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
    width: 100%;
    border-right: none;
    border-bottom: 1px solid #2C2C2C;
  }

  .library-tabs {
    flex-direction: row;
    flex-wrap: wrap;
    gap: 8px;
    padding: 10px max(14px, env(safe-area-inset-right, 0px)) 10px max(14px, env(safe-area-inset-left, 0px));
  }

  .tab-btn {
    flex: 1 1 auto;
    justify-content: center;
    white-space: nowrap;
    min-block-size: 44px;
    padding: 8px 12px;
    font-size: 13px;
    gap: 6px;
    border-radius: 999px;
    border: 1px solid var(--admin-border, #333);
  }

  .tab-btn[aria-selected="true"]::after {
    display: none;
  }

  .tab-btn[aria-selected="true"] {
    border-color: var(--admin-primary, #007AFF);
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
    min-block-size: 44px;
  }

  .icon-btn {
    min-block-size: 44px;
    min-inline-size: 44px;
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

:global(html[data-admin-appearance="true"] .library-screen .tab-btn) {
  border-color: var(--admin-border) !important;
}

:global(html[data-admin-appearance="true"] .library-screen .tab-btn[aria-selected="true"]) {
  border-color: var(--admin-primary) !important;
}

@media (pointer: coarse) {
  .back-link,
  .create-btn,
  .import-btn,
  .tab-btn,
  .card-action-btn,
  .icon-btn {
    min-block-size: 44px;
    min-inline-size: 44px;
  }
}
</style>
