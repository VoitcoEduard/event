<template>
  <div class="login-page">
    <div class="language-switcher">
      <select v-model="$i18n.locale" @change="changeLanguage">
        <option value="en">English</option>
        <option value="ru">Русский</option>
        <option value="ro">Română</option>
      </select>
    </div>
    <div class="login-card">
      <div class="login-logo">
        <svg width="40" height="40" viewBox="0 0 40 40" fill="none">
          <rect width="40" height="40" rx="12" fill="url(#grad)"/>
          <path d="M20 10L28 15V25L20 30L12 25V15L20 10Z" stroke="white" stroke-width="2" fill="none"/>
          <circle cx="20" cy="20" r="4" fill="white"/>
          <defs>
            <linearGradient id="grad" x1="0" y1="0" x2="40" y2="40">
              <stop offset="0%" stop-color="#6366f1"/>
              <stop offset="100%" stop-color="#8b5cf6"/>
            </linearGradient>
          </defs>
        </svg>
      </div>
      <h1 class="login-title">{{ $t('login.title') }}</h1>
	  <p class="login-subtitle">{{ mode === 'google' ? $t('auth.googleSubtitle') : $t('login.subtitle') }}</p>
	  <p v-if="error" class="error-msg">{{ error }}</p>

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
          <button type="button" class="toggle-pw" @click="showPassword = !showPassword" tabindex="-1">
            {{ showPassword ? '🙈' : '👁️' }}
          </button>
        </div>

        <button type="submit" class="login-btn" :disabled="loading || !password">
          <span v-if="loading" class="spinner"></span>
          <span v-else>{{ $t('login.signin_btn') }}</span>
        </button>
		</form>
		<p v-if="!mode && !error">{{ $t('auth.loading') }}</p>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { API_BASE_URL } from '../utils/index.js'
import { loadAuthMode, loadSession } from '../session.js'

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
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: #0f0f13;
  background-image: radial-gradient(ellipse at 20% 50%, rgba(99, 102, 241, 0.12) 0%, transparent 60%),
                    radial-gradient(ellipse at 80% 20%, rgba(139, 92, 246, 0.08) 0%, transparent 50%);
  font-family: 'Inter', system-ui, sans-serif;
  position: relative;
}

.language-switcher {
  position: absolute;
  top: 20px;
  right: 20px;
}

.language-switcher select {
  background: rgba(255, 255, 255, 0.06);
  color: white;
  border: 1px solid rgba(255, 255, 255, 0.1);
  padding: 8px 12px;
  border-radius: 8px;
  cursor: pointer;
  outline: none;
}

.login-card {
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 20px;
  padding: 48px 40px;
  width: 100%;
  max-width: 400px;
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
}

.login-subtitle {
  color: rgba(255,255,255,0.4);
  text-align: center;
  font-size: 0.9rem;
  margin: 0 0 32px;
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
  padding: 12px 44px 12px 16px;
  outline: none;
  transition: border-color 0.2s, background 0.2s;
  width: 100%;
  box-sizing: border-box;
}

.input-group input:focus {
  border-color: rgba(99, 102, 241, 0.6);
  background: rgba(255,255,255,0.08);
}

.input-group input::placeholder {
  color: rgba(255,255,255,0.25);
}

.toggle-pw {
  position: absolute;
  right: 12px;
  bottom: 10px;
  background: none;
  border: none;
  cursor: pointer;
  font-size: 1rem;
  padding: 0;
  line-height: 1;
  opacity: 0.6;
  transition: opacity 0.2s;
}

.toggle-pw:hover { opacity: 1; }

.error-msg {
  color: #f87171;
  font-size: 0.85rem;
  margin: 0;
  padding: 10px 14px;
  background: rgba(248, 113, 113, 0.1);
  border: 1px solid rgba(248, 113, 113, 0.2);
  border-radius: 8px;
}

.login-btn {
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
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
  justify-content: center;
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
</style>
