<template>
  <div class="login-page">
    <div class="language-switcher">
      <select v-model="$i18n.locale" :aria-label="$t('library.language_label')" @change="changeLanguage">
        <option value="en">English</option>
        <option value="ru">Русский</option>
        <option value="ro">Română</option>
      </select>
    </div>
    <main class="login-card">
      <div class="login-logo">
        <img src="/brand-mark.svg" width="48" height="48" alt="Event Operations" />
      </div>
      <h1 class="login-title">{{ $t('login.title') }}</h1>
      <p class="login-subtitle">{{ mode === 'google' ? $t('auth.googleSubtitle') : $t('login.subtitle') }}</p>
      <p v-if="error" class="error-msg" role="alert">{{ error }}</p>

      <a v-if="mode === 'google'" class="login-btn" href="/auth/google/start">{{ $t('auth.googleSignIn') }}</a>
      <form v-else-if="mode === 'dev'" @submit.prevent="handleLogin" class="login-form">
        <div class="input-group">
          <label for="password">{{ $t('login.password_label') }}</label>
          <input
            id="password"
            v-model="password"
            :type="showPassword ? 'text' : 'password'"
            :placeholder="$t('login.password_placeholder')"
            autocomplete="current-password"
            :disabled="loading"
          />
          <button
            type="button"
            class="toggle-pw"
            @click="showPassword = !showPassword"
            :aria-label="showPassword ? 'Hide password' : 'Show password'"
            :aria-pressed="showPassword"
          >
            <AppIcon :name="showPassword ? 'eyeOff' : 'eye'" :size="18" aria-hidden="true" />
          </button>
        </div>

        <button type="submit" class="login-btn" :disabled="loading || !password">
          <span v-if="loading" class="spinner"></span>
          <span v-else>{{ $t('login.signin_btn') }}</span>
        </button>
      </form>
      <p v-if="!mode && !error" class="loading-msg">{{ $t('auth.loading') }}</p>
    </main>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { API_BASE_URL } from '../utils/index.js'
import { loadAuthMode, loadSession } from '../session.js'
import AppIcon from '../components/AppIcon.vue'

const { t, locale } = useI18n()
const router = useRouter()
const password = ref('')
const error = ref('')
const loading = ref(false)
const showPassword = ref(false)
const mode = ref(null)

onMounted(async () => {
	localStorage.removeItem('adminToken')
	if (new URLSearchParams(window.location.search).get('error') === 'disabled') error.value = t('auth.disabled')
	if (new URLSearchParams(window.location.search).get('error') === 'cancelled') error.value = t('auth.cancelled')
	try { mode.value = await loadAuthMode() } catch { error.value = t('login.errors.failed') }
})

const changeLanguage = () => {
  localStorage.setItem('userLanguage', locale.value)
}

async function handleLogin() {
  error.value = ''
  loading.value = true
  try {
    const res = await fetch(`${API_BASE_URL}/auth/dev/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: password.value })
    })
    if (!res.ok) {
      error.value = t('login.errors.incorrect')
      return
    }
		await loadSession(true)
    router.push('/admin')
  } catch (e) {
    error.value = t('login.errors.failed')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: max(72px, calc(60px + env(safe-area-inset-top, 0px)))
           max(16px, env(safe-area-inset-right, 0px))
           max(84px, calc(76px + env(safe-area-inset-bottom, 0px)))
           max(16px, env(safe-area-inset-left, 0px));
  box-sizing: border-box;
  background: #0f0f13;
  background-image: radial-gradient(ellipse at 20% 50%, rgba(99, 102, 241, 0.12) 0%, transparent 60%),
                    radial-gradient(ellipse at 80% 20%, rgba(139, 92, 246, 0.08) 0%, transparent 50%);
  font-family: 'Inter', system-ui, sans-serif;
  position: relative;
}

.language-switcher {
  position: absolute;
  top: max(16px, env(safe-area-inset-top, 0px));
  right: max(16px, env(safe-area-inset-right, 0px));
}

.language-switcher select {
  background: rgba(255, 255, 255, 0.06);
  color: white;
  border: 1px solid rgba(255, 255, 255, 0.1);
  padding: 8px 12px;
  border-radius: 8px;
  cursor: pointer;
  outline: none;
  min-block-size: 36px;
}

.login-card {
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 20px;
  padding: clamp(24px, 6vw, 44px) clamp(20px, 5vw, 36px);
  width: 100%;
  max-width: 400px;
  box-sizing: border-box;
  backdrop-filter: blur(20px);
  box-shadow: 0 24px 64px rgba(0, 0, 0, 0.4);
  animation: fadeUp 0.4s ease;
}

@keyframes fadeUp {
  from { opacity: 0; transform: translateY(20px); }
  to   { opacity: 1; transform: translateY(0); }
}

.login-logo {
  display: flex;
  justify-content: center;
  margin-bottom: 24px;
}

.login-title {
  color: #fff;
  font-size: 1.6rem;
  font-weight: 700;
  text-align: center;
  margin: 0 0 8px;
  letter-spacing: -0.02em;
  text-wrap: balance;
}

.login-subtitle {
  color: rgba(255, 255, 255, 0.72);
  text-align: center;
  font-size: 0.9rem;
  margin: 0 0 28px;
  line-height: 1.45;
  text-wrap: pretty;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.input-group {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.input-group label {
  color: rgba(255,255,255,0.6);
  font-size: 0.8rem;
  font-weight: 500;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.input-group input {
  background: rgba(255,255,255,0.06);
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 10px;
  color: #fff;
  font-size: 1rem;
  line-height: 1.25;
  padding: 12px 44px 12px 16px;
  outline: none;
  transition: border-color 0.2s, background 0.2s;
  width: 100%;
  box-sizing: border-box;
}

.input-group input:focus {
  border-color: rgba(20, 127, 245, 0.7);
  background: rgba(255,255,255,0.08);
}

.input-group input::placeholder {
  color: rgba(255,255,255,0.25);
}

.toggle-pw {
  position: absolute;
  right: 6px;
  bottom: 5px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-block-size: 36px;
  min-inline-size: 36px;
  background: transparent;
  border: none;
  border-radius: 8px;
  color: var(--admin-text-muted, rgba(255, 255, 255, 0.75));
  cursor: pointer;
  font-size: 1rem;
  padding: 0;
  line-height: 1;
  transition: color 0.2s, opacity 0.2s;
}

.toggle-pw:hover {
  color: var(--admin-text, #fff);
}

:global(html[data-admin-appearance="true"]) .login-page .toggle-pw {
  background: transparent !important;
  border-color: transparent !important;
  color: var(--admin-text-muted) !important;
}

:global(html[data-admin-appearance="true"]) .login-page .toggle-pw:hover {
  color: var(--admin-text) !important;
}

.error-msg,
:global(html[data-admin-appearance="true"]) .login-page p.error-msg {
  color: var(--admin-danger, #f87171);
  font-size: 0.85rem;
  line-height: 1.4;
  margin: 0 0 16px;
  padding: 10px 14px;
  background: var(--admin-danger-soft, rgba(248, 113, 113, 0.1));
  border: 1px solid color-mix(in srgb, var(--admin-danger, #f87171) 30%, transparent);
  border-radius: 8px;
}

.loading-msg {
  text-align: center;
  margin: 0;
  font-size: 0.95rem;
}

.login-btn {
  width: 100%;
  box-sizing: border-box;
  background: linear-gradient(135deg, #2d8cff, #0a6cf5);
  border: none;
  border-radius: 10px;
  color: #fff;
  text-decoration: none;
  text-align: center;
  justify-content: center;
  cursor: pointer;
  font-size: 1rem;
  font-weight: 600;
  padding: 14px;
  transition: opacity 0.2s, transform 0.15s;
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
}

.login-btn:hover:not(:disabled) {
  opacity: 0.9;
  transform: translateY(-1px);
}

.login-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.spinner {
  width: 18px;
  height: 18px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: #fff;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
  display: inline-block;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

@media (pointer: coarse) {
  .language-switcher select {
    min-block-size: 44px;
    min-inline-size: 44px;
  }

  .input-group input {
    min-block-size: 44px;
    padding: 13px 48px 13px 16px;
  }

  .toggle-pw {
    min-block-size: 44px;
    min-inline-size: 44px;
    right: 2px;
    bottom: 2px;
  }

  .login-btn {
    min-block-size: 44px;
  }
}
</style>
