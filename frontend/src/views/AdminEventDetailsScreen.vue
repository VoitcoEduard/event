<template>
  <AdminLayout :breadcrumbs="breadcrumbs">
    <div ref="screenRef" class="admin-details-screen">
      <header class="event-header">
        <div class="header-content">
          <div class="header-main">
            <div class="event-info">
              <h1 class="event-title">{{ currentEvent?.name }}</h1>
              <span
                v-if="currentEvent"
                class="event-status"
                :class="currentEvent.isActive ? 'active' : 'inactive'"
              >
                {{ currentEvent.isActive ? $t('admin.status_active') : $t('admin.status_inactive') }}
              </span>
            </div>

            <nav class="event-subnav" :aria-label="$t('event_details.nav_label')">
              <span class="nav-button subnav-tab is-current" aria-current="page">
                {{ $t('event_details.tab_modules') }}
              </span>
              <button
                type="button"
                class="nav-button screen-manager-btn"
                :disabled="!currentEvent"
                @click="goToScreenManager"
              >
                {{ $t('event_details.tab_slides') }}
              </button>
              <button
                type="button"
                class="nav-button data-btn"
                :disabled="!currentEvent"
                @click="goToDataView"
              >
                {{ $t('event_details.tab_data') }}
              </button>
              <button
                type="button"
                class="nav-button copy-link-btn"
                :disabled="!currentEvent"
                :title="$t('event_details.copy_projector_link')"
                :aria-label="$t('event_details.copy_projector_link')"
                @click="copyProjectorLink"
              >
                <AppIcon name="link" :size="16" aria-hidden="true" />
                <span class="copy-link-label">{{ $t('event_details.copy_projector_link') }}</span>
              </button>
              <button
                type="button"
                class="nav-button library-btn"
                @click="goToLibrary"
              >
                {{ $t('event_details.tab_library') }}
              </button>
            </nav>
          </div>
        </div>
      </header>

      <main class="scroll-content">
        <div v-if="!currentEvent && isLoadingEvent" class="card event-state-card">
          <p class="card-title">{{ $t('admin.loading_events') }}</p>
        </div>
        <div v-else-if="!currentEvent && eventLoadError" class="card event-state-card">
          <p class="card-title">{{ eventLoadError }}</p>
          <div class="state-actions">
            <button type="button" class="nav-button library-btn" @click="fetchEventDetails()">
              {{ $t('event_details.retry') }}
            </button>
            <button type="button" class="nav-button back-btn" @click="goBack">
              {{ $t('admin_shell.events') }}
            </button>
          </div>
        </div>

        <template v-else>
          <aside class="card event-sidebar">
            <div class="card-header overview-desktop-header">
              <div class="card-header-text">
                <p class="card-title">{{ $t('admin.details.overview') }}</p>
                <p class="card-description">{{ $t('admin.details.overview_desc') }}</p>
              </div>
              <span class="status-badge" :class="currentEvent?.isActive ? 'active' : 'inactive'">
                {{ currentEvent?.isActive ? $t('admin.status_active') : $t('admin.status_inactive') }}
              </span>
            </div>

            <div class="sidebar-content">
              <div class="overview-primary-strip">
                <div class="info-group pin-group">
                  <p class="label">{{ $t('admin.details.pin_code') }}</p>
                  <div class="pin-display">
                    <span class="pin-text">{{ currentEvent?.pin || currentEvent?.id }}</span>
                  </div>
                </div>

                <div class="control-group">
                  <p class="section-label">{{ $t('admin.details.controls') }}</p>
                  <button
                    type="button"
                    class="action-button"
                    :class="currentEvent?.isActive ? 'stop-button' : 'start-button'"
                    @click="toggleStatus"
                  >
                    {{ currentEvent?.isActive ? $t('admin.details.stop_event') : $t('admin.details.start_event') }}
                  </button>
                  <p class="control-hint">
                    {{ currentEvent?.isActive
                      ? $t('admin.details.live_hint')
                      : $t('admin.details.stopped_hint') }}
                  </p>
                </div>

                <button
                  type="button"
                  class="share-drawer-toggle"
                  :aria-expanded="shareDrawerOpen ? 'true' : 'false'"
                  @click="shareDrawerOpen = !shareDrawerOpen"
                >
                  <span>{{ $t('event_details.share_qr_toggle') }}</span>
                  <AppIcon :name="shareDrawerOpen ? 'chevronUp' : 'chevronDown'" :size="16" aria-hidden="true" />
                </button>
              </div>

              <div class="overview-collapsible" :class="{ 'is-open': shareDrawerOpen }">
                <div v-if="currentEvent?.description" class="info-group">
                  <p class="label">{{ $t('admin.details.description') }}</p>
                  <p class="value">{{ currentEvent.description }}</p>
                </div>

                <div class="info-group">
                  <p class="label">{{ $t('admin.details.share_access') }}</p>
                  <div class="qr-wrapper">
                    <QRCodeVue :value="qrCodeValue" :size="120" level="H" />
                  </div>
                  <button type="button" class="copy-btn" @click="copyLink">
                    {{ $t('admin.details.copy_link') }}
                  </button>
                </div>
              </div>
            </div>
          </aside>

          <section class="card modules-card">
            <div class="card-header">
              <div>
                <p class="card-title">{{ $t('admin.details.modules') }}</p>
                <p class="card-description">{{ $t('admin.details.modules_desc') }}</p>
              </div>
              <button type="button" class="add-button" @click="openModuleModal">
                {{ $t('admin.details.add_module') }}
              </button>
            </div>

            <div v-if="currentEvent?.modules && currentEvent.modules.length > 0" class="modules-list">
              <div v-for="(module, index) in currentEvent.modules" :key="module.id ?? index" class="module-item">
                <div class="module-info">
                  <p class="module-name">{{ module.name }}</p>
                  <div class="module-badges">
                    <span class="module-type-badge">
                      <AppIcon :name="moduleTypeIcon(module.type)" :size="14" aria-hidden="true" />
                      <span>{{ moduleTypeLabel(module.type) }}</span>
                    </span>
                    <span
                      class="module-status-badge"
                      :class="{
                        'status-active': module.isActive,
                        'status-inactive': !module.isActive
                      }"
                    >
                      {{ module.isActive ? $t('admin.status_active') : $t('admin.status_inactive') }}
                    </span>
                  </div>
                </div>

                <div class="module-actions">
                  <button
                    type="button"
                    class="module-btn config-btn"
                    @click="editModule(module)"
                  >
                    {{ $t('event_details.edit_module') }}
                  </button>
                  <button
                    v-if="isHostableModule(module.type)"
                    type="button"
                    class="module-btn host-btn"
                    @click="hostModule(module)"
                  >
                    {{ $t('event_details.host_module') }}
                  </button>
                  <button
                    type="button"
                    class="module-btn"
                    :class="module.isActive ? 'stop-button' : 'start-button'"
                    @click="toggleModuleStatus(module)"
                  >
                    {{ module.isActive ? $t('admin.details.stop') : $t('admin.details.start') }}
                  </button>
                  <button
                    type="button"
                    class="module-btn delete-btn"
                    @click="deleteModule(module)"
                  >
                    {{ $t('admin.details.delete') }}
                  </button>
                </div>
              </div>
            </div>

            <div v-else class="empty-modules">
              <p class="empty-modules-text">{{ $t('admin.details.no_modules') }}</p>
            </div>
          </section>
        </template>
      </main>

      <AddModuleModal
        :visible="showModuleModal"
        :event-id="currentEvent?.id || routeEventId"
        @close="closeModuleModal"
        @created="handleModuleCreated"
      />
    </div>
  </AdminLayout>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import QRCodeVue from 'qrcode.vue'
import AdminLayout from '../components/AdminLayout.vue'
import AddModuleModal from '../components/AddModuleModal.vue'
import AppIcon from '../components/AppIcon.vue'
import { API_BASE_URL, getAuthHeaders } from '../utils'
import { useToast } from '../composables/useToast'
import { useDialog } from '../composables/useDialog'

const MODULE_ICONS = {
  quiz: 'quiz',
  simple_quiz: 'message',
  wrong_pictures: 'image',
  form: 'form',
  teams: 'teams'
}

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const { success, error } = useToast()
const { confirm } = useDialog()

const screenRef = ref(null)
const currentEvent = ref(null)
const isLoadingEvent = ref(false)
const eventLoadError = ref('')
const showModuleModal = ref(false)
const shareDrawerOpen = ref(false)

let eventFetchSeq = 0

const routeEventId = computed(() => {
  const raw = route.params?.eventId ?? route.params?.id ?? route.query?.eventId
  if (Array.isArray(raw)) return raw[0] ? String(raw[0]) : ''
  return raw ? String(raw) : ''
})

const breadcrumbs = computed(() => [
  { label: t('admin_shell.events'), to: '/admin/events' },
  { label: currentEvent.value?.name || t('event_details.event_fallback') }
])

const readStoredEventForId = (targetId) => {
  if (!targetId) return null
  try {
    const raw = sessionStorage.getItem('currentEvent')
    if (!raw) return null
    const parsed = JSON.parse(raw)
    if (parsed && parsed.id != null && String(parsed.id) === String(targetId)) {
      return parsed
    }
  } catch {
    // ignore corrupted or unavailable sessionStorage
  }
  return null
}

const writeStoredEvent = (eventData) => {
  try {
    if (eventData) {
      sessionStorage.setItem('currentEvent', JSON.stringify(eventData))
    } else {
      sessionStorage.removeItem('currentEvent')
    }
  } catch {
    // ignore storage errors
  }
}

currentEvent.value = readStoredEventForId(routeEventId.value)

const qrCodeValue = computed(() => (
  currentEvent.value
    ? `${window.location.origin}/?pin=${currentEvent.value.pin || currentEvent.value.id}`
    : ''
))

const moduleTypeIcon = (type) => MODULE_ICONS[type] || 'sliders'

const moduleTypeLabel = (type) => {
  if (type && Object.prototype.hasOwnProperty.call(MODULE_ICONS, type)) {
    return t(`event_details.module_types.${type}`)
  }
  return type || ''
}

const isHostableModule = (type) => (
  type === 'quiz' || type === 'wrong_pictures' || type === 'simple_quiz' || type === 'teams'
)

const fetchEventDetails = async (targetId = routeEventId.value) => {
  const seq = ++eventFetchSeq
  const eventId = targetId ? String(targetId) : ''
  if (!eventId) {
    currentEvent.value = null
    eventLoadError.value = t('event_details.not_found')
    isLoadingEvent.value = false
    return
  }

  isLoadingEvent.value = true
  eventLoadError.value = ''

  try {
    const response = await fetch(`${API_BASE_URL}/events/${encodeURIComponent(eventId)}`)
    if (seq !== eventFetchSeq) return
    if (response.ok) {
      const data = await response.json()
      if (seq !== eventFetchSeq) return
      currentEvent.value = data
      writeStoredEvent(data)
      eventLoadError.value = ''
    } else if (response.status === 404) {
      currentEvent.value = null
      writeStoredEvent(null)
      eventLoadError.value = t('event_details.not_found')
    } else {
      eventLoadError.value = t('event_details.load_failed')
      error(eventLoadError.value)
    }
  } catch (err) {
    if (seq !== eventFetchSeq) return
    console.error('Error fetching event details:', err)
    eventLoadError.value = t('event_details.load_failed')
    error(eventLoadError.value)
  } finally {
    if (seq === eventFetchSeq) {
      isLoadingEvent.value = false
    }
  }
}

watch(routeEventId, (newId, oldId) => {
  if (newId === oldId) return
  showModuleModal.value = false
  shareDrawerOpen.value = false
  eventLoadError.value = ''
  currentEvent.value = readStoredEventForId(newId)
  fetchEventDetails(newId)
})

onMounted(() => {
  fetchEventDetails(routeEventId.value)
})

const openModuleModal = () => {
  showModuleModal.value = true
}

const closeModuleModal = () => {
  showModuleModal.value = false
}

const handleModuleCreated = () => {
  fetchEventDetails()
}

const toggleStatus = async () => {
  if (!currentEvent.value?.id) return
  try {
    const nextActive = !currentEvent.value.isActive
    const response = await fetch(`${API_BASE_URL}/events/${encodeURIComponent(currentEvent.value.id)}/status`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify({ isActive: nextActive })
    })

    if (response.ok) {
      currentEvent.value.isActive = nextActive
      writeStoredEvent(currentEvent.value)
      success(t(nextActive ? 'event_details.toast_event_started' : 'event_details.toast_event_stopped'))
    } else {
      error(t('event_details.toast_status_failed'))
    }
  } catch (err) {
    console.error('Error updating status:', err)
    error(t('event_details.toast_status_failed'))
  }
}

const copyLink = () => {
  if (!currentEvent.value) return
  const link = `${window.location.origin}/?pin=${currentEvent.value.pin || currentEvent.value.id}`
  navigator.clipboard?.writeText(link)
  success(t('admin.details.toast.event_link_copied'))
}

const editModule = (module) => {
  if (!currentEvent.value?.id || !module?.id) return
  router.push(`/admin/events/${encodeURIComponent(currentEvent.value.id)}/modules/${encodeURIComponent(module.id)}/edit`)
}

const hostModule = (module) => {
  if (!currentEvent.value?.id || !module?.id) return
  const eventId = encodeURIComponent(currentEvent.value.id)
  const moduleId = encodeURIComponent(module.id)
  if (module.type === 'teams') {
    router.push(`/admin/events/${eventId}/teams?moduleId=${moduleId}`)
    return
  }
  router.push(`/admin/events/${eventId}/modules/${moduleId}/host`)
}

const toggleModuleStatus = async (module) => {
  if (!currentEvent.value?.id) return

  if (module.isActive && (module.type === 'quiz' || module.type === 'wrong_pictures' || module.type === 'simple_quiz')) {
    try {
      const resetPath = module.type === 'simple_quiz'
        ? `${API_BASE_URL}/events/${encodeURIComponent(currentEvent.value.id)}/modules/${encodeURIComponent(module.id)}/simple-quiz/reset`
        : `${API_BASE_URL}/events/${encodeURIComponent(currentEvent.value.id)}/modules/${encodeURIComponent(module.id)}/quiz/reset`
      await fetch(resetPath, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...getAuthHeaders() }
      })
    } catch (err) {
      console.error('Error resetting quiz session:', err)
    }
  }

  try {
    const nextActive = !module.isActive
    const response = await fetch(`${API_BASE_URL}/events/${encodeURIComponent(currentEvent.value.id)}/modules/${encodeURIComponent(module.id)}/status`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify({ isActive: nextActive })
    })

    if (response.ok) {
      await fetchEventDetails()
      success(t(nextActive ? 'event_details.toast_module_started' : 'event_details.toast_module_stopped'))
    } else {
      error(t('event_details.toast_module_status_failed'))
    }
  } catch (err) {
    console.error('Error updating module status:', err)
    error(t('event_details.toast_module_status_failed'))
  }
}

const deleteModule = async (module) => {
  if (!currentEvent.value?.id) return
  const confirmed = await confirm({
    title: t('event_details.delete_module_title'),
    message: t('event_details.delete_module_confirm', { name: module.name }),
    confirmLabel: t('admin.details.delete'),
    cancelLabel: t('common.cancel'),
    destructive: true
  })
  if (!confirmed) return

  try {
    const response = await fetch(`${API_BASE_URL}/events/${encodeURIComponent(currentEvent.value.id)}/modules/${encodeURIComponent(module.id)}`, {
      method: 'DELETE',
      headers: { ...getAuthHeaders() }
    })

    if (response.ok) {
      success(t('event_details.toast_module_deleted'))
      await fetchEventDetails()
    } else {
      error((await response.text()).trim() || t('event_details.toast_module_delete_failed'))
    }
  } catch (err) {
    console.error('Error deleting module:', err)
    error(t('event_details.toast_module_delete_failed'))
  }
}

const goBack = () => {
  router.push('/admin/events')
}

const goToLibrary = () => {
  const id = currentEvent.value?.id || routeEventId.value
  if (id) {
    router.push(`/admin/library/quizzes?returnTo=${encodeURIComponent(`/admin/events/${id}`)}`)
    return
  }
  router.push('/admin/library/quizzes')
}

const goToDataView = () => {
  if (!currentEvent.value?.id) return
  router.push(`/admin/events/${encodeURIComponent(currentEvent.value.id)}/data`)
}

const goToScreenManager = () => {
  if (!currentEvent.value?.id) return
  router.push(`/admin/events/${encodeURIComponent(currentEvent.value.id)}/projector`)
}

const copyProjectorLink = () => {
  if (!currentEvent.value?.id) return
  const fullUrl = `${window.location.origin}/projector/${currentEvent.value.id}`
  navigator.clipboard?.writeText(fullUrl)
  success(t('quiz.host.toast.link_copied'))
}
</script>

<style scoped>
.admin-details-screen {
  container: event-details / inline-size;
  min-height: 100%;
  background-color: var(--admin-bg, #121212);
  scroll-padding-bottom: calc(84px + env(safe-area-inset-bottom, 0px));
}

.admin-details-screen,
.admin-details-screen * {
  box-sizing: border-box;
}

.event-header {
  padding: 18px clamp(16px, 3vw, 40px);
  background-color: var(--admin-surface, #1E1E1E);
  border-bottom: 1px solid var(--admin-border, #333);
}

.header-content {
  max-width: 1560px;
  margin: 0 auto;
}

.header-main {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 14px 20px;
}

.event-info {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  flex: 1 1 260px;
  min-inline-size: 0;
}

.event-title {
  font-size: clamp(1.35rem, 1rem + 1.8vw, 1.85rem);
  font-weight: 700;
  color: var(--admin-text, #ffffff);
  margin: 0;
  line-height: 1.2;
  text-wrap: balance;
  overflow-wrap: break-word;
  min-inline-size: 0;
}

.event-status {
  padding: 5px 12px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.4px;
  flex-shrink: 0;
}

.event-status.active {
  background-color: #d4edda;
  color: #155724;
  border: 1px solid #c3e6cb;
}

.event-status.inactive {
  background-color: #f8d7da;
  color: #721c24;
  border: 1px solid #f5c6cb;
}

.event-subnav {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.nav-button {
  padding: 8px 14px;
  min-block-size: 40px;
  border: 1px solid var(--admin-border-strong, #444);
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  text-decoration: none;
  white-space: nowrap;
}

.subnav-tab.is-current {
  background: var(--admin-primary-soft, #162e4d);
  border-color: var(--admin-primary, #75b7ff);
  color: var(--admin-primary, #75b7ff);
  cursor: default;
}

.copy-link-label {
  display: inline;
}

.screen-manager-btn {
  background-color: #FF9500;
  color: white;
}

.copy-link-btn {
  background-color: #2196F3;
  color: white;
}

.data-btn {
  background-color: #9C27B0;
  color: white;
}

.library-btn {
  background-color: #4CAF50;
  color: white;
}

.back-btn {
  background-color: #607D8B;
  color: white;
}

.scroll-content {
  padding: 24px clamp(16px, 3vw, 24px) 64px;
  display: grid;
  grid-template-columns: minmax(290px, 1fr) minmax(380px, 2.5fr);
  gap: 24px;
  max-width: 1600px;
  margin: 0 auto;
  align-items: start;
}

.event-state-card {
  grid-column: 1 / -1;
  display: flex;
  flex-direction: column;
  gap: 14px;
  align-items: flex-start;
}

.state-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.card {
  background-color: var(--admin-surface, #1E1E1E);
  border-radius: 16px;
  padding: 24px;
  border: 1px solid var(--admin-border, #333);
  box-shadow: var(--admin-shadow, 0 2px 8px rgba(0, 0, 0, 0.2));
  min-inline-size: 0;
}

.event-sidebar {
  display: flex;
  flex-direction: column;
  gap: 18px;
  min-inline-size: 0;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 12px;
}

.card-header-text {
  min-inline-size: 0;
  flex: 1;
}

.status-badge {
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.4px;
  border: 1px solid transparent;
  flex-shrink: 0;
}

.status-badge.active {
  background-color: #d4edda;
  color: #155724;
  border-color: #c3e6cb;
}

.status-badge.inactive {
  background-color: #f8d7da;
  color: #721c24;
  border-color: #f5c6cb;
}

.card-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--admin-text, #ffffff);
  margin: 0 0 4px;
}

.card-description {
  font-size: 13px;
  color: var(--admin-text-muted, #888);
  margin: 0;
}

.sidebar-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
  min-inline-size: 0;
}

.overview-primary-strip {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.overview-collapsible {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.share-drawer-toggle {
  display: none;
}

.info-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-inline-size: 0;
}

.label {
  font-size: 11px;
  color: var(--admin-text-muted, #888);
  text-transform: uppercase;
  font-weight: 600;
  letter-spacing: 0.5px;
  margin: 0;
}

.value {
  font-size: 15px;
  color: var(--admin-text, #FFF);
  overflow-wrap: break-word;
  margin: 0;
  min-inline-size: 0;
}

.pin-display {
  background-color: var(--admin-surface-subtle, #000);
  padding: 12px;
  border-radius: 8px;
  text-align: center;
  border: 1px solid var(--admin-border, #333);
}

.pin-text {
  font-size: clamp(1.25rem, 0.9rem + 2vw, 1.75rem);
  color: var(--admin-text, #FFF);
  font-weight: bold;
  letter-spacing: clamp(2px, 0.6vw, 4px);
  font-family: monospace;
  overflow-wrap: anywhere;
}

.qr-wrapper {
  background: white;
  padding: 12px;
  border-radius: 12px;
  align-self: center;
  max-width: 100%;
  display: inline-flex;
  justify-content: center;
}

.copy-btn {
  margin-top: 6px;
  background-color: #333;
  padding: 10px 16px;
  border-radius: 8px;
  width: 100%;
  min-block-size: 44px;
  color: #FFF;
  font-weight: 600;
  font-size: 13px;
  border: 1px solid #444;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.control-group {
  padding-top: 16px;
  border-top: 1px solid var(--admin-border, #2C2C2C);
}

.section-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--admin-text-muted, #888);
  margin: 0 0 10px;
  text-transform: uppercase;
  letter-spacing: 0.8px;
}

.action-button {
  padding: 12px 16px;
  border-radius: 8px;
  width: 100%;
  min-block-size: 44px;
  font-weight: 700;
  font-size: 14px;
  color: #FFF;
  border: none;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.start-button {
  background-color: #28cd41;
}

.stop-button {
  background-color: #ff3b30;
}

.control-hint {
  font-size: 12px;
  color: var(--admin-text-muted, #888);
  margin: 8px 0 0;
  text-align: center;
}

.modules-card {
  min-height: 360px;
  min-inline-size: 0;
}

.modules-card .card-header {
  align-items: center;
}

.add-button {
  background-color: #007AFF;
  padding: 9px 16px;
  border-radius: 999px;
  border: none;
  color: #FFF;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  min-block-size: 40px;
  white-space: nowrap;
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.modules-list {
  margin-top: 16px;
}

.module-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background-color: #2C2C2E;
  border-radius: 12px;
  margin-bottom: 12px;
  min-inline-size: 0;
}

.module-info {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
  min-inline-size: 0;
}

.module-name {
  font-size: 16px;
  font-weight: 600;
  color: #FFF;
  margin: 0;
  overflow-wrap: break-word;
  min-inline-size: 0;
}

.module-badges {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.module-type-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 9px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  background: var(--admin-surface, #1E1E1E);
  border: 1px solid var(--admin-border, #3c3c40);
  color: var(--admin-text-muted, #c5c5ca);
}

.module-status-badge {
  display: inline-block;
  padding: 3px 8px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 700;
}

.module-status-badge.status-active {
  background-color: rgba(48, 209, 88, 0.2);
  color: #30D158;
}

.module-status-badge.status-inactive {
  background-color: rgba(136, 136, 136, 0.2);
  color: #888;
}

.module-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.module-btn {
  padding: 8px 13px;
  border-radius: 8px;
  border: none;
  font-size: 13px;
  font-weight: 600;
  min-block-size: 38px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.module-btn.config-btn {
  background-color: #333;
  color: #FFF;
}

.module-btn.host-btn {
  background-color: #5E5CE6;
  color: #FFF;
}

.module-btn.start-button {
  background-color: #28cd41;
  color: #FFF;
}

.module-btn.stop-button {
  background-color: #ff3b30;
  color: #FFF;
}

.module-btn.delete-btn {
  background-color: #1c1c1e;
  color: #ff453a;
  border: 1px solid #ff453a;
}

.empty-modules {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 16px;
  text-align: center;
}

.empty-modules-text {
  color: var(--admin-text-muted, #666);
  font-size: 14px;
  margin: 0;
}

@media (max-width: 960px) {
  .scroll-content {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .event-header {
    padding: 14px 16px;
  }

  .event-subnav {
    width: 100%;
  }

  .event-subnav > * {
    flex: 1 1 auto;
    min-block-size: 44px;
  }

  .scroll-content {
    padding: 16px 12px 64px;
    gap: 16px;
  }

  .event-sidebar {
    padding: 14px 16px;
    gap: 12px;
  }

  .overview-desktop-header {
    display: none;
  }

  .overview-primary-strip {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr) auto;
    align-items: center;
    gap: 10px;
  }

  .pin-group {
    gap: 2px;
  }

  .pin-group .label {
    font-size: 10px;
  }

  .pin-display {
    padding: 6px 10px;
  }

  .pin-text {
    font-size: 16px;
    letter-spacing: 2px;
  }

  .control-group {
    padding-top: 0;
    border-top: none;
  }

  .section-label,
  .control-hint {
    display: none;
  }

  .action-button {
    min-block-size: 44px;
    padding: 8px 12px;
    font-size: 13px;
  }

  .share-drawer-toggle {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    min-block-size: 44px;
    padding: 8px 12px;
    border: 1px solid var(--admin-border-strong, #444);
    border-radius: 8px;
    background: var(--admin-surface-subtle, #252528);
    color: var(--admin-text, #ffffff);
    font: inherit;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    white-space: nowrap;
  }

  .overview-collapsible {
    display: none;
    padding-top: 14px;
    border-top: 1px solid var(--admin-border, #333);
  }

  .overview-collapsible.is-open {
    display: flex;
  }

  .modules-card {
    padding: 18px 16px;
    min-height: 0;
  }

  .add-button {
    min-block-size: 44px;
  }

  .module-item {
    flex-direction: column;
    align-items: stretch;
  }

  .module-actions {
    width: 100%;
  }

  .module-btn {
    flex: 1 1 auto;
    min-block-size: 44px;
  }
}

@media (max-width: 520px) {
  .overview-primary-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .pin-group {
    grid-column: 1 / -1;
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
  }
}

@media (pointer: coarse) {
  .nav-button,
  .add-button,
  .module-btn,
  .copy-btn,
  .action-button,
  .share-drawer-toggle {
    min-block-size: 44px;
  }
}
</style>
