<template>
  <AdminLayout :breadcrumbs="breadcrumbs">
    <div class="access-page">
      <main class="users-content">
        <h1 class="users-title">{{ $t('auth.manageUsers') }}</h1>

        <p v-if="error" class="error-msg" role="alert">{{ error }}</p>

        <div v-if="loading && !users.length" class="state-card loading-state" aria-live="polite">
          <p>{{ $t('common.loading') }}</p>
        </div>

        <div v-else-if="!users.length && !error" class="state-card empty-state">
          <p>{{ $t('auth.noUsers') }}</p>
        </div>

        <div v-else-if="users.length" class="users-list">
          <div v-for="user in users" :key="user.id" class="user-row">
            <div class="user-info">
              <strong class="user-name">{{ user.name || user.email }}</strong>
              <span class="user-email">{{ user.email }}</span>
              <span class="user-meta">
                <span class="status-pill" :class="`status-${user.status}`">{{ user.status }}</span>
                <span class="role-label">{{ user.role }}</span>
              </span>
            </div>
            <div v-if="user.role !== 'owner'" class="user-actions">
              <button
                v-if="user.status !== 'active'"
                type="button"
                class="primary approve-btn"
                :disabled="busy"
                @click="changeStatus(user, 'active')"
              >{{ $t('auth.approve') }}</button>
              <button
                v-if="user.status === 'active'"
                type="button"
                class="danger disable-btn"
                :disabled="busy"
                @click="changeStatus(user, 'disabled')"
              >{{ $t('auth.disable') }}</button>
            </div>
          </div>
        </div>

        <div v-if="hasMore" class="load-more-wrap">
          <button type="button" class="load-more-btn" :disabled="loading || busy" @click="nextPage">
            {{ loading ? $t('common.loading') : $t('auth.loadMore') }}
          </button>
        </div>
      </main>
    </div>
  </AdminLayout>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { listAdminUsers, setAdminStatus } from '../session'
import AdminLayout from '../components/AdminLayout.vue'

const { t } = useI18n()
const users = ref([])
const page = ref(0)
const hasMore = ref(false)
const error = ref('')
const busy = ref(false)
const loading = ref(true)

const breadcrumbs = computed(() => [
  { label: t('admin_shell.users') }
])

async function load(targetPage = page.value) {
  error.value = ''
  loading.value = true
  try {
    const result = await listAdminUsers(targetPage)
    const list = Array.isArray(result?.users) ? result.users : []
    users.value = targetPage === 0 ? list : [...users.value, ...list]
    hasMore.value = Boolean(result?.hasMore)
    page.value = targetPage
  } catch {
    error.value = t('auth.unavailable')
  } finally {
    loading.value = false
  }
}

async function nextPage() {
  if (loading.value) return
  await load(page.value + 1)
}

async function changeStatus(user, status) {
  busy.value = true
  error.value = ''
  try {
    await setAdminStatus(user.id, status)
    user.status = status
  } catch {
    error.value = t('auth.unavailable')
  } finally {
    busy.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.access-page {
  container: users-page / inline-size;
  min-height: 100%;
  padding: 24px max(16px, env(safe-area-inset-right, 0px)) max(64px, calc(48px + env(safe-area-inset-bottom, 0px))) max(16px, env(safe-area-inset-left, 0px));
  box-sizing: border-box;
  background: var(--admin-bg, #121212);
  color: var(--admin-text, white);
}

.access-page,
.access-page * {
  box-sizing: border-box;
}

.users-content {
  width: 100%;
  max-width: 820px;
  margin: 0 auto;
}

.users-title {
  margin: 0 0 20px;
  font-size: clamp(1.35rem, 1.1rem + 1.2vw, 1.75rem);
  font-weight: 700;
  line-height: 1.2;
  text-wrap: balance;
}

.error-msg,
:global(html[data-admin-appearance="true"] .access-page p.error-msg) {
  margin: 0 0 16px;
  padding: 12px 16px;
  border-radius: 10px;
  font-size: 0.95rem;
  line-height: 1.4;
  background: var(--admin-danger-soft, rgba(248, 113, 113, 0.14));
  border: 1px solid color-mix(in srgb, var(--admin-danger, #f87171) 35%, var(--admin-border, #333));
  color: var(--admin-danger, #f87171);
}

.state-card {
  padding: 28px 20px;
  border-radius: 14px;
  border: 1px solid var(--admin-border, #333);
  background: var(--admin-surface, #1E1E1E);
  box-shadow: var(--admin-shadow, 0 2px 8px rgba(0, 0, 0, 0.28));
  text-align: center;
}

.state-card p {
  margin: 0;
  color: var(--admin-text-muted, #c5c5ca);
  text-wrap: pretty;
}

.users-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.user-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  padding: 16px;
  border: 1px solid var(--admin-border, #333);
  border-radius: 12px;
  background: var(--admin-surface, #1E1E1E);
}

.user-info {
  min-inline-size: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.user-name {
  font-size: 1rem;
  font-weight: 650;
  line-height: 1.3;
  overflow-wrap: break-word;
}

.user-email {
  font-size: 0.9rem;
  color: var(--admin-text-muted, #c5c5ca);
  overflow-wrap: break-word;
}

.user-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
  font-size: 0.8rem;
}

.status-pill {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 999px;
  font-weight: 650;
  text-transform: capitalize;
  background: var(--admin-surface-subtle, #252528);
  color: var(--admin-text-muted, #c5c5ca);
}

.status-pill.status-active {
  background: var(--admin-success-soft, #173624);
  color: var(--admin-success, #4bd176);
}

.status-pill.status-pending {
  background: var(--admin-warning-soft, #3d3018);
  color: var(--admin-warning, #ffc15c);
}

.status-pill.status-disabled {
  background: var(--admin-danger-soft, #3b2020);
  color: var(--admin-danger, #ff6961);
}

.role-label {
  color: var(--admin-text-muted, #c5c5ca);
  font-weight: 500;
  text-transform: capitalize;
}

.user-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

button {
  min-block-size: 40px;
  padding: 10px 16px;
  border-radius: 8px;
  font-size: 0.9rem;
  font-weight: 600;
  cursor: pointer;
  border: 1px solid var(--admin-border-strong, #4b4b50);
  background: var(--admin-surface, #252528);
  color: var(--admin-text, #f5f5f7);
  transition: background-color 0.2s, opacity 0.2s;
}

button.primary {
  background: var(--admin-primary-solid, #0969da);
  border-color: var(--admin-primary-solid, #0969da);
  color: #ffffff;
}

button.danger,
:global(html[data-admin-appearance="true"] .access-page button.danger) {
  background: var(--admin-danger-soft, rgba(248, 113, 113, 0.14)) !important;
  border-color: color-mix(in srgb, var(--admin-danger, #f87171) 35%, var(--admin-border, #333)) !important;
  color: var(--admin-danger, #f87171) !important;
}

button:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.load-more-wrap {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}

@container users-page (inline-size <= 520px) {
  .user-row {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }

  .user-actions {
    width: 100%;
  }

  .user-actions button,
  .load-more-btn {
    width: 100%;
    min-block-size: 44px;
    margin-left: 0;
  }
}

@media (pointer: coarse) {
  .user-actions button,
  .load-more-btn {
    min-block-size: 44px;
    min-inline-size: 44px;
  }
}
</style>
