<template>
  <main class="access-page">
    <div class="access-utility-strip">
      <AdminThemeToggle variant="inline" />
    </div>
    <div class="access-card">
      <h1>{{ $t('auth.pendingTitle') }}</h1>
      <p>{{ $t('auth.pendingMessage') }}</p>
      <div class="access-actions">
        <button type="button" class="primary" :disabled="busy" @click="check">{{ $t('auth.checkAgain') }}</button>
        <button type="button" class="secondary" :disabled="busy" @click="leave">{{ $t('auth.signOut') }}</button>
      </div>
      <p v-if="error" class="error-msg" role="alert">{{ error }}</p>
    </div>
  </main>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { loadSession, signOut } from '../session'
import AdminThemeToggle from '../components/AdminThemeToggle.vue'

const router = useRouter()
const { t } = useI18n()
const error = ref('')
const busy = ref(false)

async function check() {
  if (busy.value) return
  error.value = ''
  busy.value = true
  try {
    const user = await loadSession(true)
    if (user?.status === 'active') router.push('/admin')
    else if (!user) router.push('/login')
    else if (user.status === 'disabled') router.push({ name: 'login', query: { error: 'disabled' } })
  } catch {
    error.value = t('auth.unavailable')
  } finally {
    busy.value = false
  }
}

async function leave() {
  if (busy.value) return
  error.value = ''
  busy.value = true
  try {
    await signOut()
    router.push('/login')
  } catch {
    error.value = t('auth.unavailable')
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.access-page {
  position: relative;
  min-height: 100vh;
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: max(72px, calc(60px + env(safe-area-inset-top, 0px)))
           max(16px, env(safe-area-inset-right, 0px))
           max(84px, calc(72px + env(safe-area-inset-bottom, 0px)))
           max(16px, env(safe-area-inset-left, 0px));
  box-sizing: border-box;
  background: #121212;
  color: white;
  text-align: center;
}

.access-utility-strip {
  position: absolute;
  top: max(16px, env(safe-area-inset-top, 0px));
  right: max(16px, env(safe-area-inset-right, 0px));
  left: max(16px, env(safe-area-inset-left, 0px));
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.access-card {
  max-width: 440px;
  width: 100%;
  margin: 0 auto;
  padding: clamp(24px, 5vw, 36px);
  border-radius: 16px;
  border: 1px solid var(--admin-border, #333);
  background: var(--admin-surface, #1E1E1E);
  box-shadow: var(--admin-shadow, 0 14px 32px rgba(0, 0, 0, 0.24));
  box-sizing: border-box;
}

h1 {
  margin: 0 0 12px;
  font-size: clamp(1.35rem, 1.1rem + 1.2vw, 1.75rem);
  font-weight: 700;
  line-height: 1.25;
  text-wrap: balance;
}

p {
  margin: 0;
  line-height: 1.5;
  text-wrap: pretty;
}

.access-actions {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 24px;
}

button {
  width: 100%;
  min-block-size: 44px;
  padding: 12px 16px;
  border-radius: 10px;
  font-size: 1rem;
  font-weight: 600;
  cursor: pointer;
  box-sizing: border-box;
  transition: background-color 0.2s, border-color 0.2s, opacity 0.2s;
}

button.primary {
  background: var(--admin-primary-solid, #0969da);
  border: 1px solid var(--admin-primary-solid, #0969da);
  color: #ffffff;
}

button.secondary,
:global(html[data-admin-appearance="true"] .access-card button.secondary) {
  background: var(--admin-surface-subtle, #2a2a2e) !important;
  border: 1px solid var(--admin-border-strong, #4b4b50) !important;
  color: var(--admin-text, #f5f5f7) !important;
}

:global(html[data-admin-appearance="true"] .access-card button.secondary:hover:not(:disabled)) {
  background: var(--admin-surface-hover) !important;
}

button:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.error-msg,
:global(html[data-admin-appearance="true"] .access-page p.error-msg) {
  margin: 16px 0 0;
  padding: 10px 14px;
  border-radius: 8px;
  font-size: 0.9rem;
  line-height: 1.4;
  text-align: left;
  background: var(--admin-danger-soft, rgba(248, 113, 113, 0.14));
  border: 1px solid color-mix(in srgb, var(--admin-danger, #f87171) 35%, var(--admin-border, #333));
  color: var(--admin-danger, #f87171);
}

@media (max-width: 768px), (pointer: coarse) {
  .access-utility-strip :deep(.admin-theme-option) {
    min-block-size: 44px;
  }
}
</style>
