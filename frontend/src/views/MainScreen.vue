<template>
  <div class="main-screen brutalist">
    <div class="language-switcher">
      <select v-model="$i18n.locale" @change="changeLanguage" class="b-select">
        <option value="en">English</option>
        <option value="ru">Русский</option>
        <option value="ro">Română</option>
      </select>
    </div>

    <div class="content">
      <div class="brand">
        <div class="brand-icon-box">
          <Zap :size="28" :stroke-width="2.5" />
        </div>
        <h1 class="title">{{ $t('main.title') }}</h1>
        <p class="subtitle">{{ $t('main.subtitle') }}</p>
      </div>

      <div class="card b-card b-card-accent">
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
            class="input b-input"
            @input="handlePinInput"
          />
          <div class="input-hint b-mono">{{ pin.length }}/6</div>
        </div>

        <p v-if="error" class="error-text b-mono">
          {{ error }}
        </p>

        <button
          class="connect-button b-btn b-btn-accent"
          @click="handleConnect"
          :disabled="loading"
          :class="{ 'is-loading': loading }"
        >
          <span v-if="loading" class="loading-blocks b-loader"><span></span><span></span><span></span></span>
          <template v-else>
            <span class="btn-label">{{ $t('main.connect_btn') }}</span>
            <ArrowRight :size="18" :stroke-width="3" />
          </template>
        </button>
      </div>

      <button class="admin-link b-label" @click="goToAdmin">
        {{ $t('main.admin_link') }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Zap, ArrowRight } from 'lucide-vue-next'

const { t, locale } = useI18n()
const router = useRouter()
const pin = ref('')
const loading = ref(false)
const error = ref('')

import { API_BASE_URL } from '../utils'

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
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&family=JetBrains+Mono:wght@400;700&display=swap');

.main-screen {
  min-height: 100vh;
  min-height: 100dvh;
  background: var(--b-bg);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  padding: 20px;
}

.language-switcher {
  position: absolute;
  top: 20px;
  right: 20px;
  z-index: 10;
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

.brand-icon-box {
  width: 64px;
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--b-bg);
  border: var(--b-border-w) solid var(--b-text);
  margin-bottom: 16px;
  color: var(--b-accent-cyan);
  box-shadow: var(--b-offset) var(--b-offset) 0 0 var(--b-accent-cyan);
}

.title {
  font-size: 32px;
  font-weight: 900;
  color: var(--b-text);
  margin-bottom: 8px;
  letter-spacing: -1px;
  text-align: center;
  text-transform: uppercase;
}

.subtitle {
  font-size: 11px;
  color: var(--b-text-dim);
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.25em;
  text-align: center;
  font-family: 'JetBrains Mono', ui-monospace, monospace;
}

/* Card */
.card {
  background: var(--b-surface);
  padding: 32px 24px;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 24px;
}

.card-title {
  font-size: 20px;
  font-weight: 900;
  color: var(--b-text);
  margin-bottom: 6px;
  text-align: center;
  text-transform: uppercase;
  letter-spacing: -0.3px;
}

.card-text {
  font-size: 13px;
  color: var(--b-text-dim);
  text-align: center;
  margin-bottom: 28px;
  line-height: 1.6;
}

/* Input */
.input-wrapper {
  position: relative;
  width: 100%;
  margin-bottom: 24px;
}

.input {
  text-align: center;
  letter-spacing: 10px;
  font-size: 26px;
  padding: 18px 16px;
}

.input-hint {
  position: absolute;
  bottom: -20px;
  right: 0;
  font-size: 11px;
  color: var(--b-text-faint);
  font-weight: 700;
}

/* Error */
.error-text {
  color: var(--b-accent-rose);
  font-size: 12px;
  margin-top: 4px;
  margin-bottom: 16px;
  text-align: center;
  font-weight: 700;
  border: var(--b-border-w) solid var(--b-accent-rose);
  padding: 8px 12px;
  width: 100%;
}

/* Connect Button */
.connect-button {
  width: 100%;
  margin-top: 8px;
}

/* Admin link */
.admin-link {
  padding: 12px 20px;
  background: none;
  border: none;
  cursor: pointer;
  transition: color 0.2s;
}

.admin-link:hover {
  color: var(--b-text);
}

/* Loading blocks */
.loading-blocks span {
  width: 8px;
  height: 8px;
  background: #000;
  animation-name: b-loader-pulse-dark;
}

@keyframes b-loader-pulse-dark {
  0%, 100% { opacity: 1; transform: translateY(0); }
  50%      { opacity: 0.3; transform: translateY(-4px); }
}
</style>
