<template>
  <AdminLayout :breadcrumbs="breadcrumbs">
    <div class="teams-admin-wrapper">
      <div class="teams-admin-header">
        <div class="teams-admin-nav-content">
          <div class="teams-admin-context">
            <span v-if="eventName">{{ eventName }}</span>
            <h1>{{ selected?.name || $t('teams_module.teams') }}</h1>
          </div>
          <label v-if="teamModules.length > 1" class="teams-admin-picker">
            <span>{{ $t('teams_module.activity') }}</span>
            <select v-model="selectedId" :aria-label="$t('teams_module.activity')">
              <option value="">{{ $t('teams_module.select_activity') }}</option>
              <option v-for="module in teamModules" :key="module.id" :value="module.id">{{ module.name }}</option>
            </select>
          </label>
        </div>
      </div>

      <p v-if="error" class="teams-error teams-page-message" role="alert">{{ error }}</p>
      <TeamsActivityAdmin v-if="selected" :key="selected.id" :event-id="eventId" :module="selected" :modules="modules" />
      <p v-else class="teams-picker-message">{{ $t('teams_module.select_activity') }}</p>
    </div>
  </AdminLayout>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { API_BASE_URL } from '../utils'
import AdminLayout from '../components/AdminLayout.vue'
import TeamsActivityAdmin from '../components/TeamsActivityAdmin.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const eventId = computed(() => {
  const raw = route.params?.eventId ?? route.params?.id ?? route.query?.eventId
  if (Array.isArray(raw)) return raw[0] ? String(raw[0]) : ''
  return raw ? String(raw) : ''
})

const modules = ref([])
const eventName = ref('')
const error = ref('')
const selectedId = ref(route.query.moduleId || '')

const teamModules = computed(() => modules.value.filter(module => module.type === 'teams'))
const selected = computed(() => teamModules.value.find(module => module.id === selectedId.value))

const breadcrumbs = computed(() => [
  { label: t('admin_shell.events'), to: '/admin/events' },
  {
    label: eventName.value || t('event_details.event_fallback'),
    to: eventId.value ? `/admin/events/${encodeURIComponent(eventId.value)}` : '/admin/events'
  },
  { label: t('teams_module.teams') }
])

watch(selectedId, value => {
  router.replace({ query: { ...route.query, moduleId: value || undefined } })
})

async function loadEventTeams(id) {
  if (!id) return
  error.value = ''
  try {
    const response = await fetch(`${API_BASE_URL}/events/${encodeURIComponent(id)}`)
    if (!response.ok) throw new Error(await response.text())
    const event = await response.json()
    eventName.value = event.name || ''
    modules.value = event.modules || []
    if (!selectedId.value && teamModules.value.length === 1) {
      selectedId.value = teamModules.value[0].id
    }
  } catch (err) {
    error.value = err.message
  }
}

watch(eventId, (id) => {
  loadEventTeams(id)
}, { immediate: true })
</script>

<style scoped>
.teams-admin-wrapper {
  background: var(--admin-bg, #121212);
  min-height: 100%;
  color: var(--admin-text, #fff);
  box-sizing: border-box;
}

.teams-admin-wrapper * {
  box-sizing: border-box;
}

.teams-admin-header {
  padding: 18px clamp(16px, 3vw, 40px);
  background: var(--admin-surface, #1e1e1e);
  border-bottom: 1px solid var(--admin-border, #333);
}

.teams-admin-nav-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  max-width: 1400px;
  margin: 0 auto;
}

.teams-admin-context {
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow-wrap: break-word;
}

.teams-admin-context span {
  color: var(--admin-text-muted, #888);
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.8px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.teams-admin-context h1 {
  font-size: clamp(20px, 4.5vw, 26px);
  line-height: 1.2;
  margin: 0;
  overflow-wrap: break-word;
  word-break: break-word;
}

.teams-admin-picker {
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--admin-text-muted, #888);
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.6px;
  min-width: 0;
}

.teams-admin-picker span {
  overflow-wrap: break-word;
}

.teams-admin-picker select {
  min-width: 220px;
  max-width: 360px;
  min-height: 44px;
  background: var(--admin-surface, #252525);
  border: 1px solid var(--admin-border-strong, #444);
  color: var(--admin-text, #fff);
  border-radius: 8px;
  padding: 10px 36px 10px 12px;
  font: inherit;
  text-transform: none;
  letter-spacing: normal;
}

.teams-page-message,
.teams-picker-message {
  max-width: 1400px;
  margin: 24px auto;
  padding: 16px 24px;
}

.teams-picker-message {
  color: var(--admin-text-muted, #888);
  background: var(--admin-surface, #1e1e1e);
  border: 1px solid var(--admin-border, #333);
  border-radius: 12px;
}

@media (max-width: 760px) {
  .teams-admin-header {
    padding: 14px 16px;
  }

  .teams-admin-nav-content,
  .teams-admin-picker {
    align-items: stretch;
    flex-direction: column;
    gap: 12px;
  }

  .teams-admin-context span {
    white-space: normal;
    overflow: visible;
    text-overflow: clip;
  }

  .teams-admin-picker {
    gap: 6px;
  }

  .teams-admin-picker select {
    min-width: 0;
    max-width: none;
    width: 100%;
    font-size: 16px;
  }

  .teams-page-message,
  .teams-picker-message {
    margin: 16px max(16px, env(safe-area-inset-right, 0px)) calc(64px + env(safe-area-inset-bottom, 0px)) max(16px, env(safe-area-inset-left, 0px));
    padding: 16px;
  }
}

@media (pointer: coarse) {
  .teams-admin-picker select {
    min-block-size: 44px;
  }
}
</style>
