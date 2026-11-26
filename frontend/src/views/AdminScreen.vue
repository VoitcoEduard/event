<template>
  <AdminLayout>
    <template #actions>
      <RouterLink to="/" class="portal-link">
        <AppIcon name="external" :size="15" aria-hidden="true" />
        <span>{{ $t('admin.public_portal_link') }}</span>
      </RouterLink>
      <button
        type="button"
        class="submit-button create-toggle-btn"
        :aria-expanded="showForm ? 'true' : 'false'"
        @click="toggleNewEventForm"
      >
        <AppIcon :name="showForm && !editingId ? 'close' : 'plus'" :size="16" aria-hidden="true" />
        <span>{{ showForm && !editingId ? $t('admin.close_form_btn') : $t('admin.new_event_btn') }}</span>
      </button>
    </template>

    <div class="admin-screen">
      <main class="scroll-content">
        <form
          v-if="showForm"
          ref="formCardRef"
          class="form-card"
          novalidate
          @submit.prevent="handleSubmit"
        >
          <div class="form-header">
            <h2 class="form-title">{{ editingId ? $t('admin.edit_event') : $t('admin.new_event') }}</h2>
            <button
              type="button"
              class="form-close-btn"
              :aria-label="$t('common.close')"
              @click="cancelEdit"
            >
              <AppIcon name="close" :size="18" aria-hidden="true" />
            </button>
          </div>

          <label class="field-label" for="admin-event-name">{{ $t('admin.event_name_placeholder') }}</label>
          <input
            id="admin-event-name"
            ref="nameInputRef"
            v-model="name"
            type="text"
            :placeholder="$t('admin.event_name_placeholder')"
            class="input"
            :class="{ 'input-error': Boolean(nameError) }"
            :aria-invalid="nameError ? 'true' : undefined"
            :aria-describedby="nameError ? 'admin-event-name-error' : undefined"
            @input="nameError = ''"
          />
          <p v-if="nameError" id="admin-event-name-error" class="field-error" role="alert">
            {{ nameError }}
          </p>

          <label class="field-label" for="admin-event-description">{{ $t('admin.description_placeholder') }}</label>
          <textarea
            id="admin-event-description"
            v-model="description"
            :placeholder="$t('admin.description_placeholder')"
            class="input text-area"
            rows="3"
          ></textarea>

          <label class="field-label" for="admin-event-language">{{ $t('appearance.language') }}</label>
          <select id="admin-event-language" v-model="eventLanguage" class="input lang-select">
            <option value="en">English</option>
            <option value="ru">Русский</option>
            <option value="ro">Română</option>
          </select>

          <div class="form-actions">
            <button
              type="button"
              class="cancel-button"
              :disabled="isSubmitting"
              @click="cancelEdit"
            >
              {{ $t('admin.cancel_btn') }}
            </button>
            <button type="submit" class="submit-button" :disabled="isSubmitting">
              {{ editingId ? $t('admin.update_btn') : $t('admin.create_btn') }}
            </button>
          </div>
        </form>

        <div class="section-header">
          <h1 class="section-title">{{ $t('admin.events_heading') }}</h1>
        </div>

        <div class="events-grid">
          <p v-if="isLoading" class="empty-text">{{ $t('admin.loading_events') }}</p>
          <template v-else>
            <div v-for="event in events" :key="event.id" class="event-card">
              <div class="card-header" @click="viewDetails(event)">
                <div class="card-title-group">
                  <p class="event-name">{{ event.name }}</p>
                  <p class="event-id">ID: {{ event.id }}</p>
                </div>
                <span
                  class="status-badge"
                  :class="event.isActive ? 'status-active' : 'status-inactive'"
                >
                  {{ event.isActive ? $t('admin.status_active') : $t('admin.status_inactive') }}
                </span>
              </div>

              <p v-if="event.description" class="event-description">{{ event.description }}</p>

              <div class="card-actions">
                <button type="button" class="card-action view-action" @click="viewDetails(event)">
                  {{ $t('admin.manage_btn') }}
                </button>
                <button type="button" class="card-action edit-action" @click="editEvent(event)">
                  {{ $t('admin.edit_btn') }}
                </button>
                <button type="button" class="card-action delete-action" @click="deleteEvent(event)">
                  {{ $t('admin.delete_btn') }}
                </button>
              </div>
            </div>

            <div v-if="events.length === 0" class="empty-state-box">
              <p class="empty-text">{{ $t('admin.no_events') }}</p>
            </div>
          </template>
        </div>
      </main>
    </div>
  </AdminLayout>
</template>

<script setup>
import { computed, nextTick, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AdminLayout from '../components/AdminLayout.vue'
import AppIcon from '../components/AppIcon.vue'
import { API_BASE_URL, getAuthHeaders } from '../utils'
import { useToast } from '../composables/useToast'
import { useDialog } from '../composables/useDialog'
import { useUnsavedChanges } from '../composables/useUnsavedChanges'

const { t, locale } = useI18n()
const router = useRouter()
const { success, error } = useToast()
const { confirm } = useDialog()

const events = ref([])
const name = ref('')
const description = ref('')
const eventLanguage = ref(locale.value)
const editingId = ref(null)
const showForm = ref(false)
const isLoading = ref(false)
const isSubmitting = ref(false)
const nameError = ref('')

const formCardRef = ref(null)
const nameInputRef = ref(null)

// Dirty tracking for the create/edit event form (L1).
const formInitialSnapshot = ref('')
const serializeEventForm = () => JSON.stringify({
  name: name.value,
  description: description.value,
  language: eventLanguage.value
})
const isFormDirty = computed(() => (
  showForm.value &&
  formInitialSnapshot.value !== '' &&
  serializeEventForm() !== formInitialSnapshot.value
))
useUnsavedChanges(isFormDirty)

const fetchEvents = async () => {
  isLoading.value = true
  try {
    const response = await fetch(`${API_BASE_URL}/events`)
    const data = await response.json()
    events.value = Array.isArray(data) ? data : []
  } catch (err) {
    console.error('Error fetching events:', err)
  } finally {
    isLoading.value = false
  }
}

onMounted(() => {
  fetchEvents()
})

const focusForm = async () => {
  await nextTick()
  formCardRef.value?.scrollIntoView?.({ behavior: 'smooth', block: 'nearest' })
  nameInputRef.value?.focus?.()
}

const toggleNewEventForm = async () => {
  if (showForm.value && !editingId.value) {
    cancelEdit()
    return
  }
  editingId.value = null
  name.value = ''
  description.value = ''
  eventLanguage.value = locale.value
  nameError.value = ''
  showForm.value = true
  formInitialSnapshot.value = serializeEventForm()
  await focusForm()
}

const handleSubmit = async () => {
  if (isSubmitting.value) return
  const trimmedName = name.value.trim()
  if (!trimmedName) {
    nameError.value = t('admin.name_required')
    nameInputRef.value?.focus?.()
    return
  }

  isSubmitting.value = true
  nameError.value = ''
  try {
    const wasEditing = Boolean(editingId.value)
    const url = wasEditing
      ? `${API_BASE_URL}/events/${encodeURIComponent(editingId.value)}`
      : `${API_BASE_URL}/events`
    const method = wasEditing ? 'PUT' : 'POST'

    const response = await fetch(url, {
      method,
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify({
        name: trimmedName,
        description: description.value.trim(),
        language: eventLanguage.value
      })
    })

    if (response.ok) {
      cancelEdit()
      await fetchEvents()
      success(t(`admin.messages.${wasEditing ? 'updated' : 'created'}`))
    } else {
      error(t('admin.messages.save_failed'))
    }
  } catch (err) {
    console.error('Error saving event:', err)
    error(t('admin.messages.save_failed'))
  } finally {
    isSubmitting.value = false
  }
}

const deleteEvent = async (event) => {
  const confirmed = await confirm({
    title: t('admin.delete_event_title'),
    message: t('admin.confirm_delete_named', { name: event.name }),
    confirmLabel: t('admin.delete_btn'),
    cancelLabel: t('common.cancel'),
    destructive: true
  })
  if (!confirmed) return

  try {
    const response = await fetch(`${API_BASE_URL}/events/${encodeURIComponent(event.id)}`, {
      method: 'DELETE',
      headers: { ...getAuthHeaders() }
    })
    if (response.ok) {
      if (editingId.value === event.id) {
        cancelEdit()
      }
      await fetchEvents()
      success(t('admin.messages.deleted'))
    } else {
      error(t('admin.messages.delete_failed'))
    }
  } catch (err) {
    console.error('Error deleting event:', err)
    error(t('admin.messages.delete_failed'))
  }
}

const editEvent = async (event) => {
  name.value = event.name || ''
  description.value = event.description || ''
  eventLanguage.value = event.language || locale.value
  editingId.value = event.id
  nameError.value = ''
  showForm.value = true
  formInitialSnapshot.value = serializeEventForm()
  await focusForm()
}

const cancelEdit = () => {
  name.value = ''
  description.value = ''
  editingId.value = null
  eventLanguage.value = locale.value
  nameError.value = ''
  showForm.value = false
  formInitialSnapshot.value = ''
}

const viewDetails = (event) => {
  try { sessionStorage.setItem('currentEvent', JSON.stringify(event)) } catch { /* storage unavailable */ }
  router.push(`/admin/events/${encodeURIComponent(event.id)}`)
}
</script>

<style scoped>
.portal-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 38px;
  padding: 6px 12px;
  border-radius: 8px;
  color: var(--admin-text-muted, #c5c5ca);
  font-size: 13px;
  font-weight: 600;
  text-decoration: none;
  white-space: nowrap;
}

.portal-link:hover {
  background: var(--admin-surface-hover, #303034);
  color: var(--admin-text, #ffffff);
}

.create-toggle-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 38px;
  padding: 8px 14px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.admin-screen {
  container: admin-dashboard / inline-size;
  width: 100%;
  background: var(--admin-bg, #121212);
  scroll-padding-bottom: calc(84px + env(safe-area-inset-bottom, 0px));
}

.admin-screen,
.admin-screen * {
  box-sizing: border-box;
}

.scroll-content,
:global(html[data-admin-appearance="true"] .admin-screen > main.scroll-content) {
  padding: 28px max(clamp(16px, 4vw, 48px), env(safe-area-inset-right, 0px)) calc(64px + env(safe-area-inset-bottom, 0px)) max(clamp(16px, 4vw, 48px), env(safe-area-inset-left, 0px));
  display: flex;
  flex-direction: column;
  align-items: center;
}

.form-card {
  background-color: var(--admin-surface, #1E1E1E);
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 28px;
  border: 1px solid var(--admin-border, #2C2C2C);
  width: min(100%, 600px);
  max-width: 600px;
}

.form-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}

.form-title {
  font-size: 20px;
  font-weight: 700;
  color: var(--admin-text, #FFF);
  margin: 0;
  text-wrap: balance;
  overflow-wrap: break-word;
}

.form-close-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 40px;
  min-height: 40px;
  padding: 0;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--admin-text-muted, #aaa);
  cursor: pointer;
}

.input {
  width: 100%;
  min-inline-size: 0;
  font-size: 16px;
  padding: 12px 14px;
  border-radius: 8px;
  border: 1px solid var(--admin-border-strong, #333);
  background-color: var(--admin-surface, #121212);
  color: var(--admin-text, #fff);
  margin-bottom: 12px;
  min-block-size: 44px;
}

.input.input-error {
  border-color: var(--admin-danger, #ff453a) !important;
  margin-bottom: 6px;
}

.field-error {
  margin: 0 0 12px;
  font-size: 13px;
  font-weight: 600;
  color: var(--admin-danger, #ff453a);
}

.field-label {
  display: block;
  margin: 2px 0 6px;
  color: var(--admin-text, #fff);
  font-size: 13px;
  font-weight: 700;
}

.text-area {
  min-height: 80px;
  resize: vertical;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 6px;
}

.submit-button,
.cancel-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 10px 18px;
  border-radius: 8px;
  font-weight: 600;
  font-size: 14px;
  line-height: 1.25;
  min-block-size: 40px;
  cursor: pointer;
}

.submit-button {
  background-color: var(--admin-primary-solid, #007AFF);
  border: 1px solid var(--admin-primary-solid, #007AFF);
  color: #FFF;
}

.cancel-button {
  border: 1px solid var(--admin-border-strong, #333);
  color: var(--admin-text-muted, #AAA);
  background: transparent;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  max-width: 1200px;
  margin-bottom: 16px;
}

.section-title {
  font-size: 22px;
  font-weight: 700;
  color: var(--admin-text, #FFFFFF);
  margin: 0;
  text-align: left;
  text-wrap: balance;
  overflow-wrap: break-word;
}

.events-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  width: 100%;
  max-width: 1200px;
}

.event-card {
  background-color: var(--admin-surface, #1E1E1E);
  border-radius: 12px;
  padding: 20px;
  min-width: 0;
  min-inline-size: 0;
  max-width: none;
  border: 1px solid var(--admin-border, #2C2C2C);
  box-shadow: var(--admin-shadow, 0 1px 3px rgba(0, 0, 0, 0.3));
  display: flex;
  flex-direction: column;
}

.event-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.14);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 10px;
  margin-bottom: 8px;
  cursor: pointer;
}

.card-title-group {
  flex: 1;
  min-inline-size: 0;
}

.event-name {
  font-size: 18px;
  font-weight: 700;
  color: var(--admin-text, #FFFFFF);
  margin: 0 0 4px;
  text-wrap: balance;
  overflow-wrap: break-word;
}

.event-id {
  font-size: 12px;
  color: var(--admin-text-dim, #666);
  font-family: monospace;
  margin: 0;
  overflow-wrap: break-word;
}

.status-badge {
  padding: 4px 8px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 700;
  flex-shrink: 0;
  white-space: nowrap;
}

.status-active {
  background-color: rgba(48, 209, 88, 0.2);
  color: #30D158;
}

.status-inactive {
  background-color: rgba(255, 69, 58, 0.2);
  color: #FF453A;
}

.event-description {
  font-size: 14px;
  color: var(--admin-text-muted, #AAAAAA);
  line-height: 1.5;
  margin: 0 0 12px;
  text-wrap: pretty;
  overflow-wrap: break-word;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.card-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: auto;
  padding-top: 8px;
}

.card-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 600;
  min-block-size: 38px;
  cursor: pointer;
}

.view-action {
  background-color: #2A2A2A;
  border: 1px solid #333;
  color: #FFF;
}

.edit-action {
  background-color: #2A2A2A;
  border: 1px solid #333;
  color: #FFF;
}

.delete-action {
  background-color: rgba(255, 59, 48, 0.1);
  border: 1px solid rgba(255, 59, 48, 0.3);
  color: #FF453A;
}

.empty-state-box {
  grid-column: 1 / -1;
  padding: 36px 20px;
  border: 1px dashed var(--admin-border, #333);
  border-radius: 12px;
  text-align: center;
}

.empty-text {
  grid-column: 1 / -1;
  color: var(--admin-text-muted, #666);
  margin: 0;
  width: 100%;
}

@container admin-dashboard (inline-size <= 900px) {
  .events-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@container admin-dashboard (inline-size <= 600px) {
  .events-grid,
  :global(html[data-admin-appearance="true"] .admin-screen > main.scroll-content .events-grid) {
    grid-template-columns: minmax(0, 1fr);
  }

  .form-card {
    padding: 18px;
  }
}

@container admin-dashboard (inline-size <= 560px) {
  .scroll-content,
  :global(html[data-admin-appearance="true"] .admin-screen > main.scroll-content) {
    padding: 16px max(16px, env(safe-area-inset-right, 0px)) calc(64px + env(safe-area-inset-bottom, 0px)) max(16px, env(safe-area-inset-left, 0px));
  }

  .input {
    font-size: 16px;
  }

  .form-actions > button {
    flex: 1 1 140px;
    min-block-size: 44px;
  }

  .card-action {
    flex: 1 1 auto;
    min-block-size: 44px;
  }
}

@media (max-width: 768px) {
  .portal-link,
  .create-toggle-btn,
  .form-close-btn,
  .card-action,
  .submit-button,
  .cancel-button {
    min-block-size: 44px;
    min-inline-size: 44px;
  }

  .portal-link,
  .create-toggle-btn {
    width: 100%;
    justify-content: center;
  }
}

@media (pointer: coarse) {
  .submit-button,
  .cancel-button,
  .card-action,
  .form-close-btn,
  .portal-link,
  .input,
  input,
  select,
  textarea {
    min-block-size: 44px;
    min-inline-size: 44px;
  }
}
</style>
