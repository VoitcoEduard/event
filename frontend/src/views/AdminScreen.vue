<template>
  <div class="admin-screen">
    <div class="header">
      <div class="language-switcher">
        <select v-model="$i18n.locale" @change="changeLanguage">
          <option value="en">English</option>
          <option value="ru">Русский</option>
          <option value="ro">Română</option>
        </select>
      </div>
      <h1 class="header-title">{{ $t('admin.title') }}</h1>
      <div class="header-actions">
          <button class="library-btn" @click="goToLibrary">{{ $t('admin.library_btn') }}</button>
		  <RouterLink v-if="session.user?.role === 'owner'" class="library-btn" to="/admin/users">{{ $t('auth.manageUsers') }}</RouterLink>
		  <button class="library-btn" @click="leave">{{ $t('auth.signOut') }}</button>
      </div>
      <p class="header-subtitle">{{ $t('admin.subtitle') }}</p>
    </div>

    <div class="scroll-content">
      <div class="form-card">
        <h2 class="form-title">{{ editingId ? $t('admin.edit_event') : $t('admin.new_event') }}</h2>
        
        <input
          v-model="name"
          type="text"
          :placeholder="$t('admin.event_name_placeholder')"
          class="input"
        />
        
        <textarea
          v-model="description"
          :placeholder="$t('admin.description_placeholder')"
          class="input text-area"
          rows="3"
        ></textarea>

        <select v-model="eventLanguage" class="input lang-select">
          <option value="en">English</option>
          <option value="ru">Русский</option>
          <option value="ro">Română</option>
        </select>
        
        <div class="form-actions">
          <button 
            v-if="editingId" 
            class="cancel-button" 
            @click="cancelEdit"
          >
            {{ $t('admin.cancel_btn') }}
          </button>
          <button class="submit-button" @click="handleSubmit">
            {{ editingId ? $t('admin.update_btn') : $t('admin.create_btn') }}
          </button>
        </div>
      </div>

      <h2 class="section-title">{{ $t('admin.upcoming_events') }}</h2>

      <div class="events-grid">
        <p v-if="isLoading" class="empty-text">{{ $t('admin.loading_events') }}</p>
        <template v-else>
          <div v-for="event in events" :key="event.id" class="event-card">
            <div class="card-header" @click="viewDetails(event)">
              <div>
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
              <button class="card-action view-action" @click="viewDetails(event)">
                {{ $t('admin.manage_btn') }}
              </button>
              <button class="card-action edit-action" @click="editEvent(event)">
                {{ $t('admin.edit_btn') }}
              </button>
              <button class="card-action delete-action" @click="deleteEvent(event.id)">
                {{ $t('admin.delete_btn') }}
              </button>
            </div>
          </div>
          
          <p v-if="events.length === 0" class="empty-text">
            {{ $t('admin.no_events') }}
          </p>
        </template>
      </div>

      <button class="back-button" @click="goToMain">
        {{ $t('admin.back_to_main') }}
      </button>
    </div>
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
  localStorage.setItem('userLanguage', locale.value)
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
  sessionStorage.setItem('currentEvent', JSON.stringify(event))
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
  min-height: 100vh;
  background-color: #121212;
}

.header {
  padding-top: 60px;
  padding-bottom: 20px;
  padding-left: 24px;
  padding-right: 24px;
  background-color: #1E1E1E;
  border-bottom: 1px solid #2C2C2C;
  position: relative;
}

.language-switcher {
  position: absolute;
  top: 20px;
  right: 20px;
}

.language-switcher select {
  background-color: #252525;
  color: white;
  border: 1px solid #333333;
  padding: 8px 12px;
  border-radius: 8px;
  cursor: pointer;
  outline: none;
}

.header-title {
  font-size: 28px;
  font-weight: 800;
  color: #FFFFFF;
  letter-spacing: -0.5;
  margin-bottom: 4px;
}

.header-actions {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
}

.library-btn {
    background-color: #333;
    border: 1px solid #555;
    color: white;
    padding: 8px 16px;
    border-radius: 8px;
    cursor: pointer;
    font-weight: 600;
    transition: background 0.2s;
}
.library-btn:hover { background-color: #444; }

.header-subtitle {
  font-size: 14px;
  color: #888888;
  text-transform: uppercase;
  letter-spacing: 1px;
  font-weight: 600;
}

.scroll-content {
  padding: 24px;
  padding-bottom: 40px;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.form-card {
  background-color: #1E1E1E;
  border-radius: 12px;
  padding: 16px;
  margin-bottom: 20px;
  border: 1px solid #2C2C2C;
  width: 100%;
  max-width: 500px;
}

.form-title {
  font-size: 18px;
  font-weight: 700;
  color: #FFF;
  margin-bottom: 16px;
}

.input {
  margin-bottom: 12px;
}

.text-area {
  min-height: 80px;
  resize: vertical;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.submit-button {
  background-color: #007AFF;
  padding: 12px 20px;
  border-radius: 8px;
  color: #FFF;
  font-weight: 600;
  font-size: 15px;
}

.cancel-button {
  padding: 12px 20px;
  border-radius: 8px;
  border: 1px solid #333;
  color: #AAA;
  font-weight: 600;
  font-size: 15px;
  background: none;
}

.section-title {
  font-size: 20px;
  font-weight: 700;
  color: #FFFFFF;
  margin-bottom: 20px;
  margin-top: 10px;
  width: 100%;
  max-width: 1200px;
  text-align: left;
}

.events-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  width: 100%;
  max-width: 1200px;
}

.event-card {
  background-color: #1E1E1E;
  border-radius: 12px;
  padding: 16px;
  min-width: 250px;
  flex: 1;
  max-width: 350px;
  border: 1px solid #2C2C2C;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
}

.event-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 8px;
  cursor: pointer;
}

.event-name {
  font-size: 20px;
  font-weight: 600;
  color: #FFFFFF;
  flex: 1;
  margin-right: 10px;
  margin-bottom: 8px;
}

.event-id {
  font-size: 12px;
  color: #666;
  font-family: monospace;
}

.status-badge {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 700;
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
  margin-bottom: 12px;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.card-actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.card-action {
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 500;
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
  color: #666;
  font-style: italic;
  text-align: center;
  margin-top: 20px;
  width: 100%;
}

.back-button {
  background-color: #FFFFFF;
  padding: 18px;
  border-radius: 14px;
  align-items: center;
  margin-top: 10px;
  width: 100%;
  max-width: 1200px;
  color: #000000;
  font-size: 16px;
  font-weight: 700;
  letter-spacing: 0.5;
}
</style>
