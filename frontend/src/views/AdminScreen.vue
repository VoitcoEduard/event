<template>
  <div class="admin-screen">
    <header class="header">
      <div class="header-brand">
        <h1 class="header-title">{{ $t('admin.title') }}</h1>
        <p class="header-subtitle">{{ $t('admin.subtitle') }}</p>
      </div>
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
      <div class="header-actions">
        <button type="button" class="library-btn" @click="goToLibrary">{{ $t('admin.library_btn') }}</button>
        <RouterLink v-if="session.user?.role === 'owner'" class="library-btn" to="/admin/users">{{ $t('auth.manageUsers') }}</RouterLink>
        <button type="button" class="library-btn" @click="leave">{{ $t('auth.signOut') }}</button>
      </div>
    </header>

    <main class="scroll-content">
      <form class="form-card" @submit.prevent="handleSubmit">
        <h2 class="form-title">{{ editingId ? $t('admin.edit_event') : $t('admin.new_event') }}</h2>
        
        <label class="field-label" for="admin-event-name">{{ $t('admin.event_name_placeholder') }}</label>
        <input
          id="admin-event-name"
          v-model="name"
          type="text"
          :placeholder="$t('admin.event_name_placeholder')"
          class="input"
        />
        
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
            v-if="editingId" 
            type="button"
            class="cancel-button" 
            @click="cancelEdit"
          >
            {{ $t('admin.cancel_btn') }}
          </button>
          <button type="submit" class="submit-button">
            {{ editingId ? $t('admin.update_btn') : $t('admin.create_btn') }}
          </button>
        </div>
      </form>

      <h2 class="section-title">{{ $t('admin.upcoming_events') }}</h2>

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
            
            <p class="event-description">{{ event.description }}</p>
            
            <div class="card-actions">
              <button type="button" class="card-action view-action" @click="viewDetails(event)">
                {{ $t('admin.manage_btn') }}
              </button>
              <button type="button" class="card-action edit-action" @click="editEvent(event)">
                {{ $t('admin.edit_btn') }}
              </button>
              <button type="button" class="card-action delete-action" @click="deleteEvent(event.id)">
                {{ $t('admin.delete_btn') }}
              </button>
            </div>
          </div>
          
          <p v-if="events.length === 0" class="empty-text">
            {{ $t('admin.no_events') }}
          </p>
        </template>
      </div>

      <button type="button" class="back-button" @click="goToMain">
        {{ $t('admin.back_to_main') }}
      </button>
    </main>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { API_BASE_URL, getAuthHeaders } from '../utils'
import { session, signOut } from '../session'
import { useToast } from '../composables/useToast'

const { t, locale } = useI18n()
const router = useRouter()

async function leave() { await signOut(); router.push('/login') }
const events = ref([])
const name = ref('')
const description = ref('')
const eventLanguage = ref(locale.value)
const editingId = ref(null)
const isLoading = ref(false)

const { success, error, info } = useToast()

const changeLanguage = () => {
  try { localStorage.setItem('userLanguage', locale.value) } catch { /* storage unavailable */ }
  if (!editingId.value) {
    eventLanguage.value = locale.value
  }
}

const fetchEvents = async () => {
  isLoading.value = true
  try {
    const response = await fetch(`${API_BASE_URL}/events`)
    const data = await response.json()
    if (Array.isArray(data)) {
      events.value = data
    } else {
      events.value = []
    }
  } catch (err) {
    console.error('Error fetching events:', err)
  } finally {
    isLoading.value = false
  }
}

onMounted(() => {
  fetchEvents()
})

const handleSubmit = async () => {
  if (!name.value || !description.value) {
    info(t('admin.messages.fill_fields'))
    return
  }

  try {
    const url = editingId.value 
      ? `${API_BASE_URL}/events/${editingId.value}` 
      : `${API_BASE_URL}/events`
    const method = editingId.value ? 'PUT' : 'POST'

    const response = await fetch(url, {
      method,
      headers: { 'Content-Type': 'application/json', ...getAuthHeaders() },
      body: JSON.stringify({ name: name.value, description: description.value, language: eventLanguage.value })
    })

    if (response.ok) {
      name.value = ''
      description.value = ''
      eventLanguage.value = locale.value
      const wasEditing = !!editingId.value
      editingId.value = null
      fetchEvents()
      success(t(`admin.messages.${wasEditing ? 'updated' : 'created'}`))
    } else {
      error(t('admin.messages.save_failed'))
    }
  } catch (err) {
    console.error('Error saving event:', err)
    error(t('admin.messages.save_failed'))
  }
}

const deleteEvent = async (id) => {
  if (!confirm(t('admin.confirm_delete'))) return
  
  try {
    const response = await fetch(`${API_BASE_URL}/events/${id}`, { method: 'DELETE', headers: { ...getAuthHeaders() } })
    if (response.ok) {
      fetchEvents()
    } else {
      error(t('admin.messages.delete_failed'))
    }
  } catch (err) {
    console.error('Error deleting event:', err)
    error(t('admin.messages.delete_failed'))
  }
}

const editEvent = (event) => {
  name.value = event.name
  description.value = event.description
  eventLanguage.value = event.language || locale.value
  editingId.value = event.id
}

const cancelEdit = () => {
  name.value = ''
  description.value = ''
  editingId.value = null
  eventLanguage.value = locale.value
}

const viewDetails = (event) => {
  try { sessionStorage.setItem('currentEvent', JSON.stringify(event)) } catch { /* storage unavailable */ }
  router.push(`/admin/event/${event.id}`)
}

const goToMain = () => {
  router.push('/')
}

const goToLibrary = () => {
  router.push('/quiz/library')
}
</script>

<style scoped>
.admin-screen {
  container: admin-dashboard / inline-size;
  width: 100%;
  min-height: 100vh;
  min-height: 100dvh;
  background-color: #121212;
  scroll-padding-bottom: calc(84px + env(safe-area-inset-bottom, 0px));
}

.admin-screen,
.admin-screen * {
  box-sizing: border-box;
}

.header {
  padding: 24px clamp(16px, 4vw, 64px);
  background-color: #1E1E1E;
  border-bottom: 1px solid #2C2C2C;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  grid-template-areas:
    "title actions lang"
    "subtitle actions lang";
  align-items: center;
  gap: 2px 20px;
}

.header-brand {
  display: contents;
}

.language-switcher {
  grid-area: lang;
  position: static;
  margin-inline-start: auto;
  flex-shrink: 0;
}

.language-switcher select {
  background-color: #252525;
  color: white;
  border: 1px solid #333333;
  padding: 8px 12px;
  border-radius: 8px;
  cursor: pointer;
  outline: none;
  font-size: 14px;
  min-block-size: 40px;
  max-inline-size: 100%;
}

.header-title {
  grid-area: title;
  font-size: clamp(1.375rem, 1.1rem + 1.4vw, 1.75rem);
  font-weight: 800;
  color: #FFFFFF;
  letter-spacing: -0.02em;
  line-height: 1.2;
  margin: 0;
  min-inline-size: 0;
  overflow-wrap: break-word;
  text-wrap: balance;
}

.header-actions {
  grid-area: actions;
  display: flex;
  justify-content: flex-end;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

.library-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background-color: #333;
  border: 1px solid #555;
  color: white;
  padding: 8px 16px;
  border-radius: 8px;
  cursor: pointer;
  font-family: inherit;
  font-size: 14px;
  font-weight: 600;
  line-height: 1.2;
  text-decoration: none;
  text-align: center;
  min-block-size: 40px;
  transition: background 0.2s;
}
.library-btn:hover { background-color: #444; }

:global(html[data-admin-appearance="true"] .admin-screen .header .header-actions .library-btn) {
  min-block-size: max(40px, var(--target-size, 24px));
}

.header-subtitle {
  grid-area: subtitle;
  font-size: 14px;
  color: #888888;
  text-transform: uppercase;
  letter-spacing: 1px;
  font-weight: 600;
  margin: 0;
  min-inline-size: 0;
  overflow-wrap: break-word;
  text-wrap: pretty;
}

.scroll-content,
:global(html[data-admin-appearance="true"] .admin-screen > main.scroll-content) {
  padding: 40px max(clamp(24px, 4vw, 64px), env(safe-area-inset-right, 0px)) calc(84px + env(safe-area-inset-bottom, 0px)) max(clamp(24px, 4vw, 64px), env(safe-area-inset-left, 0px));
  display: flex;
  flex-direction: column;
  align-items: center;
}

.form-card {
  background-color: #1E1E1E;
  border-radius: 12px;
  padding: 28px;
  margin-bottom: 38px;
  border: 1px solid #2C2C2C;
  width: min(100%, 540px);
  max-width: 540px;
}

.form-title {
  font-size: 24px;
  font-weight: 700;
  color: #FFF;
  margin: 0 0 16px;
  text-align: center;
  text-wrap: balance;
  overflow-wrap: break-word;
}

.input {
  width: 100%;
  min-inline-size: 0;
  font-size: 16px;
  padding: 12px 16px;
  border-radius: 8px;
  border: 1px solid #333;
  background-color: #121212;
  color: #fff;
  margin-bottom: 12px;
  min-block-size: 44px;
}

.field-label {
  display: block;
  margin: 2px 0 7px;
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
  margin-top: 4px;
}

.submit-button,
.cancel-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 12px 20px;
  border-radius: 8px;
  font-weight: 600;
  font-size: 15px;
  line-height: 1.25;
  min-block-size: 40px;
}

.submit-button {
  background-color: #007AFF;
  border: 1px solid #007AFF;
  color: #FFF;
}

.cancel-button {
  border: 1px solid #333;
  color: #AAA;
  background: none;
}

.section-title {
  font-size: 20px;
  font-weight: 700;
  color: #FFFFFF;
  margin: 10px 0 20px;
  width: 100%;
  max-width: 1200px;
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
  background-color: #1E1E1E;
  border-radius: 12px;
  padding: 20px;
  min-width: 0;
  min-inline-size: 0;
  max-width: none;
  border: 1px solid #2C2C2C;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
  display: flex;
  flex-direction: column;
}

.event-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
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
  font-size: 20px;
  font-weight: 600;
  color: #FFFFFF;
  margin: 0 0 6px;
  text-wrap: balance;
  overflow-wrap: break-word;
}

.event-id {
  font-size: 12px;
  color: #666;
  font-family: monospace;
  margin: 0;
  overflow-wrap: break-word;
}

.status-badge {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 10px;
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
  color: #AAAAAA;
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
  gap: 10px;
  flex-wrap: wrap;
  margin-top: auto;
}

.card-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 500;
  min-block-size: 36px;
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

.empty-text {
  grid-column: 1 / -1;
  color: #666;
  font-style: italic;
  text-align: center;
  margin-top: 20px;
  width: 100%;
}

.back-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background-color: #FFFFFF;
  padding: 14px 18px;
  border-radius: 14px;
  margin-top: 24px;
  width: 100%;
  max-width: 1200px;
  min-block-size: 48px;
  color: #000000;
  font-size: 16px;
  font-weight: 700;
  letter-spacing: 0.5px;
  text-align: center;
}

@container admin-dashboard (inline-size <= 900px) {
  .header {
    grid-template-columns: minmax(0, 1fr) auto;
    grid-template-areas:
      "title lang"
      "subtitle subtitle"
      "actions actions";
    gap: 4px 12px;
    padding: max(16px, env(safe-area-inset-top, 0px)) max(16px, env(safe-area-inset-right, 0px)) 20px max(16px, env(safe-area-inset-left, 0px));
  }

  .header-actions {
    justify-content: flex-start;
    margin-top: 12px;
  }

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
    padding: 20px;
  }
}

@container admin-dashboard (inline-size <= 560px) {
  .scroll-content,
  :global(html[data-admin-appearance="true"] .admin-screen > main.scroll-content) {
    padding: 16px max(16px, env(safe-area-inset-right, 0px)) calc(84px + env(safe-area-inset-bottom, 0px)) max(16px, env(safe-area-inset-left, 0px));
  }

  .language-switcher select {
    font-size: 16px;
  }

  .input {
    width: 100%;
    min-inline-size: 0;
    font-size: 16px;
  }

  .form-actions {
    flex-wrap: wrap;
  }

  .form-actions > button {
    flex: 1 1 140px;
    min-block-size: 44px;
  }

  .card-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .card-action {
    flex: 1 1 auto;
    min-block-size: 44px;
    text-align: center;
    justify-content: center;
  }
}

@media (pointer: coarse) {
  .library-btn,
  .language-switcher select,
  .submit-button,
  .cancel-button,
  .card-action,
  .back-button,
  .input,
  input,
  select,
  textarea {
    min-block-size: 44px;
    min-inline-size: 44px;
  }

  .language-switcher select {
    font-size: 16px;
  }
}
</style>
