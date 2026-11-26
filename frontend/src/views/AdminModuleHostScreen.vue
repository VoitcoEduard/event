<template>
  <QuizHostScreen
    v-if="moduleType === 'quiz' || moduleType === 'wrong_pictures'"
    :key="`quiz-host-${eventId}-${moduleId}`"
    :event-name="eventName"
    :module-name="moduleName"
  />
  <SimpleQuizHostScreen
    v-else-if="moduleType === 'simple_quiz'"
    :key="`simple-quiz-host-${eventId}-${moduleId}`"
    :event-name="eventName"
    :module-name="moduleName"
  />
  <AdminLayout v-else :breadcrumbs="breadcrumbs">
    <main class="host-loader-page">
      <div v-if="loading" class="host-state-card" role="status" aria-live="polite">
        <p class="host-state-message">{{ $t('host.loading') }}</p>
      </div>

      <div v-else-if="errorMessage" class="host-state-card host-state-card--error" role="alert">
        <AppIcon name="alert" :size="28" aria-hidden="true" />
        <h1 class="host-state-title">{{ errorMessage }}</h1>
        <div class="host-state-actions">
          <button type="button" class="host-action-btn host-action-btn--primary" @click="loadHostContext">
            {{ $t('host.retry') }}
          </button>
          <RouterLink :to="`/admin/events/${eventId}`" class="host-action-btn">
            <AppIcon name="arrowLeft" :size="16" aria-hidden="true" />
            <span>{{ $t('host.back_to_event') }}</span>
          </RouterLink>
        </div>
      </div>

      <div v-else class="host-state-card">
        <h1 class="host-state-title">{{ moduleName || moduleType }}</h1>
        <p class="host-state-message">{{ $t('host.unsupported_module') }}</p>
        <div class="host-state-actions">
          <RouterLink
            :to="`/admin/events/${eventId}/modules/${moduleId}/edit`"
            class="host-action-btn host-action-btn--primary"
          >
            {{ $t('host.edit_module') }}
          </RouterLink>
          <RouterLink :to="`/admin/events/${eventId}`" class="host-action-btn">
            <AppIcon name="arrowLeft" :size="16" aria-hidden="true" />
            <span>{{ $t('host.back_to_event') }}</span>
          </RouterLink>
        </div>
      </div>
    </main>
  </AdminLayout>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AdminLayout from '../components/AdminLayout.vue'
import AppIcon from '../components/AppIcon.vue'
import QuizHostScreen from './QuizHostScreen.vue'
import SimpleQuizHostScreen from './SimpleQuizHostScreen.vue'
import { API_BASE_URL, getAuthHeaders } from '../utils'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const eventId = computed(() => String(route.params.eventId || ''))
const moduleId = computed(() => String(route.params.moduleId || ''))

const loading = ref(true)
const errorMessage = ref('')
const eventName = ref('')
const moduleName = ref('')
const moduleType = ref('')

let loadSeq = 0

const breadcrumbs = computed(() => {
  const crumbs = [{ label: t('admin_shell.events'), to: '/admin/events' }]
  if (eventId.value) {
    crumbs.push({
      label: eventName.value || eventId.value,
      to: `/admin/events/${eventId.value}`
    })
  }
  if (moduleName.value) {
    crumbs.push({ label: moduleName.value })
  }
  return crumbs
})

async function loadHostContext() {
  const seq = ++loadSeq
  const currentEventId = eventId.value
  const currentModuleId = moduleId.value

  loading.value = true
  errorMessage.value = ''
  moduleType.value = ''
  moduleName.value = ''
  eventName.value = ''

  if (!currentEventId || !currentModuleId) {
    loading.value = false
    errorMessage.value = t('host.load_error')
    return
  }

  try {
    const headers = getAuthHeaders()
    const [moduleRes, eventRes] = await Promise.allSettled([
      fetch(
        `${API_BASE_URL}/events/${encodeURIComponent(currentEventId)}/modules/${encodeURIComponent(currentModuleId)}`,
        { headers }
      ),
      fetch(`${API_BASE_URL}/events/${encodeURIComponent(currentEventId)}`, { headers })
    ])

    if (seq !== loadSeq) return

    if (eventRes.status === 'fulfilled' && eventRes.value.ok) {
      const eventData = await eventRes.value.json()
      if (seq !== loadSeq) return
      eventName.value = eventData?.name || ''
    }

    if (moduleRes.status !== 'fulfilled' || !moduleRes.value.ok) {
      errorMessage.value = t('host.load_error')
      return
    }

    const mod = await moduleRes.value.json()
    if (seq !== loadSeq) return

    moduleName.value = mod?.name || ''
    const resolvedType = mod?.type || ''

    if (resolvedType === 'teams') {
      await router.replace({
        name: 'teams-admin',
        params: { eventId: currentEventId },
        query: { moduleId: currentModuleId }
      })
      return
    }

    moduleType.value = resolvedType
  } catch {
    if (seq === loadSeq) {
      errorMessage.value = t('host.load_error')
    }
  } finally {
    if (seq === loadSeq) {
      loading.value = false
    }
  }
}

watch([eventId, moduleId], loadHostContext, { immediate: true })
</script>

<style scoped>
.host-loader-page {
  max-width: 560px;
  margin: 48px auto;
  padding: 0 20px;
  width: 100%;
  box-sizing: border-box;
}

.host-state-card {
  background: var(--admin-surface, #ffffff);
  border: 1px solid var(--admin-border, #dde1e7);
  border-radius: 16px;
  padding: 28px 24px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: 14px;
  box-shadow: var(--admin-shadow, 0 1px 2px rgba(16, 24, 40, 0.06));
}

.host-state-card--error {
  border-color: color-mix(in srgb, var(--admin-danger, #b42318) 35%, var(--admin-border, #dde1e7));
  color: var(--admin-danger, #b42318);
}

.host-state-title {
  margin: 0;
  font-size: 1.2rem;
  color: var(--admin-text, #101828);
}

.host-state-message {
  margin: 0;
  color: var(--admin-text-muted, #475467);
}

.host-state-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 10px;
  margin-top: 6px;
}

.host-action-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-block-size: 44px;
  padding: 10px 16px;
  border-radius: 10px;
  border: 1px solid var(--admin-border-strong, #c8ced8);
  background: var(--admin-surface, #ffffff);
  color: var(--admin-text, #101828);
  font-weight: 600;
  font-size: 0.9375rem;
  text-decoration: none;
  cursor: pointer;
}

.host-action-btn--primary {
  background: var(--admin-primary-solid, #0969da);
  border-color: var(--admin-primary-solid, #0969da);
  color: #ffffff;
}
</style>
