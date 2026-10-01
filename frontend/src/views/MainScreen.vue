<template>
  <div class="main-screen">
    <div class="bg-glow"></div>

    <div class="language-switcher">
      <select v-model="$i18n.locale" @change="changeLanguage">
        <option value="en">English</option>
        <option value="ru">Русский</option>
        <option value="ro">Română</option>
      </select>
    </div>

    <div class="content">
      <div class="brand">
        <div class="brand-icon"><img src="/brand-mark.svg" alt="" width="48" height="48" /></div>
        <h1 class="title">{{ $t('main.title') }}</h1>
        <p class="subtitle">{{ $t('main.subtitle') }}</p>
      </div>

      <div class="card">
        <h2 class="card-title">{{ $t('main.connect_title') }}</h2>
        <p class="card-text">{{ $t('main.connect_desc') }}</p>

        <div class="input-wrapper">
          <input
            v-model="pin"
            type="text"
            inputmode="numeric"
            pattern="[0-9]*"
            :placeholder="$t('main.pin_placeholder')"
            maxlength="6"
            class="input"
            @input="handlePinInput"
          />
          <div class="input-hint">{{ pin.length }}/6</div>
        </div>

        <p v-if="error" class="error-text">
          <AppIcon class="error-icon" name="alert" :size="18" /> {{ error }}
        </p>

        <button
          class="connect-button"
          @click="handleConnect"
          :disabled="loading"
          :class="{ 'is-loading': loading }"
        >
          <span v-if="loading" class="loading-spinner"></span>
          <span v-else class="btn-label">{{ $t('main.connect_btn') }} <AppIcon name="arrowRight" :size="18" /></span>
        </button>
      </div>

      <button class="admin-link" @click="goToAdmin">
        {{ $t('main.admin_link') }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

const { t, locale } = useI18n()
const router = useRouter()
const pin = ref('')
const loading = ref(false)
const error = ref('')

import { API_BASE_URL } from '../utils'
import AppIcon from '../components/AppIcon.vue'

onMounted(() => {
  const params = new URLSearchParams(window.location.search)
  const pinParam = params.get('pin')
  if (pinParam) {
    pin.value = pinParam
    // Auto-connect if PIN is valid length
    if (pin.value.length === 6) {
        handleConnect()
    }
  }
})

const handlePinInput = (e) => {
  pin.value = e.target.value.replace(/[^0-9]/g, '')
  error.value = ''
}

const changeLanguage = () => {
  localStorage.setItem('userLanguage', locale.value)
}

const handleConnect = async () => {
  error.value = ''
  if (!pin.value || pin.value.length !== 6) {
    error.value = t('main.errors.invalid_pin')
    return
  }

  loading.value = true
  try {
    const response = await fetch(`${API_BASE_URL}/events/connect`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pin: pin.value })
    })

    if (response.ok) {
      const event = await response.json()
      sessionStorage.setItem('currentEvent', JSON.stringify(event))
      router.push('/event')
    } else {
      if (response.status === 404) {
        error.value = t('main.errors.not_found')
      } else if (response.status === 403) {
        error.value = t('main.errors.not_active')
      } else {
        error.value = t('main.errors.failed')
      }
    }
  } catch (err) {
    console.error('Connection error:', err)
    error.value = t('main.errors.network')
  } finally {
    loading.value = false
  }
}

const goToAdmin = () => {
  router.push('/admin')
}
</script>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&display=swap');

* { font-family: 'Inter', sans-serif; box-sizing: border-box; }

.main-screen {
  min-height: 100vh;
  background: #0d0d0f;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

/* Subtle ambient glow — not overdone */
.bg-glow {
  position: absolute;
  top: -200px;
  left: 50%;
  transform: translateX(-50%);
  width: 600px;
  height: 600px;
  background: radial-gradient(ellipse at center, rgba(99, 102, 241, 0.12) 0%, transparent 70%);
  pointer-events: none;
  z-index: 0;
}

.language-switcher {
  position: absolute;
  top: 20px;
  right: 20px;
  z-index: 10;
}

.language-switcher select {
  background-color: #1a1a1f;
  color: #aaa;
  border: 1px solid #2a2a35;
  padding: 7px 12px;
  border-radius: 8px;
  font-size: 13px;
  cursor: pointer;
  outline: none;
  transition: border-color 0.2s;
}

.language-switcher select:hover {
  border-color: #444;
  color: #fff;
}

.content {
  position: relative;
  z-index: 1;
  width: 92%;
  max-width: 400px;
  display: flex;
  flex-direction: column;
  align-items: center;
}

/* Brand */
.brand {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 36px;
}

.brand-icon {
  margin-bottom: 14px;
  filter: drop-shadow(0 0 16px rgba(20, 127, 245, 0.42));
  animation: pulse-glow 2.5s ease-in-out infinite;
}

@keyframes pulse-glow {
  0%, 100% { filter: drop-shadow(0 0 12px rgba(20, 127, 245, 0.35)); }
  50%       { filter: drop-shadow(0 0 22px rgba(20, 127, 245, 0.6)); }
}

.title {
  font-size: 34px;
  font-weight: 900;
  color: #fff;
  margin-bottom: 6px;
  letter-spacing: -1px;
  text-align: center;
}

.subtitle {
  font-size: 12px;
  color: #555;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 2px;
  text-align: center;
}

/* Card */
.card {
  background: #161619;
  padding: 32px 28px;
  border-radius: 24px;
  width: 100%;
  border: 1px solid #2a2a35;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 20px;
}

.card-title {
  font-size: 20px;
  font-weight: 800;
  color: #fff;
  margin-bottom: 6px;
  text-align: center;
}

.card-text {
  font-size: 14px;
  color: #666;
  text-align: center;
  margin-bottom: 28px;
  line-height: 1.6;
}

/* Input */
.input-wrapper {
  position: relative;
  width: 100%;
  margin-bottom: 20px;
}

.input {
  width: 100%;
  background: #0d0d0f;
  border-radius: 14px;
  padding: 18px 16px;
  font-size: 28px;
  font-weight: 800;
  text-align: center;
  letter-spacing: 10px;
  color: #fff;
  border: 2px solid #2a2a35;
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s;
  caret-color: #147ff5;
}

.input:focus {
  border-color: #147ff5;
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.15);
}

.input::placeholder {
  color: #363640;
  letter-spacing: 4px;
  font-size: 20px;
}

.input-hint {
  position: absolute;
  bottom: -20px;
  right: 0;
  font-size: 11px;
  color: #444;
  font-weight: 600;
}

/* Error */
.error-text {
  color: #f87171;
  font-size: 13px;
  margin-top: 4px;
  margin-bottom: 16px;
  text-align: center;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 5px;
}

.error-icon {
  font-style: normal;
}

/* Connect Button */
.connect-button {
  background: linear-gradient(135deg, #2d8cff 0%, #0a6cf5 100%);
  padding: 17px;
  border-radius: 14px;
  width: 100%;
  color: white;
  font-size: 16px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  cursor: pointer;
  transition: transform 0.15s, box-shadow 0.2s, opacity 0.2s;
  box-shadow: 0 4px 20px rgba(99, 102, 241, 0.35);
  margin-top: 8px;
}

.connect-button:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 6px 28px rgba(99, 102, 241, 0.5);
}

.connect-button:active:not(:disabled) {
  transform: translateY(0);
}

.connect-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-label {
  letter-spacing: 0.3px;
}

/* Admin link */
.admin-link {
  padding: 12px 20px;
  background: none;
  border: none;
  color: #444;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: color 0.2s;
}

.admin-link:hover {
  color: #888;
}

/* Spinner */
.loading-spinner {
  width: 22px;
  height: 22px;
  border: 2.5px solid rgba(255, 255, 255, 0.25);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
