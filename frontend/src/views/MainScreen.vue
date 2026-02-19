<template>
  <div class="main-screen">
    <div class="language-switcher">
      <select v-model="$i18n.locale" @change="changeLanguage">
        <option value="en">English</option>
        <option value="ru">Русский</option>
        <option value="ro">Română</option>
      </select>
    </div>
    <div class="content">
      <h1 class="title">{{ $t('main.title') }}</h1>
      <p class="subtitle">{{ $t('main.subtitle') }}</p>

      <div class="card">
        <h2 class="card-title">{{ $t('main.connect_title') }}</h2>
        <p class="card-text">{{ $t('main.connect_desc') }}</p>
        
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
        
        <p v-if="error" class="error-text">{{ error }}</p>
        
        <button 
          class="connect-button" 
          @click="handleConnect" 
          :disabled="loading"
        >
          <span v-if="loading" class="loading-spinner"></span>
          <span v-else>{{ $t('main.connect_btn') }}</span>
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
.main-screen {
  min-height: 100vh;
  background-color: #121212;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
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

.content {
  width: 90%;
  max-width: 400px;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.title {
  font-size: 32px;
  font-weight: 800;
  color: #ffffff;
  margin-bottom: 8px;
  letter-spacing: -0.5;
}

.subtitle {
  font-size: 16px;
  color: #888888;
  margin-bottom: 40px;
  font-weight: 500;
  letter-spacing: 0.5;
  text-transform: uppercase;
}

.card {
  background-color: #1E1E1E;
  padding: 32px;
  border-radius: 24px;
  width: 100%;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
  margin-bottom: 32px;
  border: 1px solid #333333;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.card-title {
  font-size: 20px;
  font-weight: 700;
  color: #ffffff;
  margin-bottom: 8px;
}

.card-text {
  font-size: 15px;
  color: #888888;
  text-align: center;
  margin-bottom: 24px;
  line-height: 22px;
}

.input {
  background-color: #252525;
  border-radius: 12px;
  padding: 16px;
  font-size: 24px;
  font-weight: 700;
  text-align: center;
  letter-spacing: 8px;
  margin-bottom: 24px;
  color: #ffffff;
  border: 1px solid #333333;
}

.input::placeholder {
  color: #666666;
}

.connect-button {
  background-color: #007AFF;
  padding: 16px;
  border-radius: 12px;
  width: 100%;
  color: white;
  font-size: 18px;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
}

.connect-button:hover {
  background-color: #0056b3;
}

.admin-link {
  padding: 16px;
  background: none;
  color: #888888;
  font-size: 14px;
  font-weight: 500;
}

.admin-link:hover {
  color: #ffffff;
}

.error-text {
  color: #FF6B6B;
  font-size: 14px;
  margin-bottom: 16px;
  text-align: center;
  font-weight: 500;
}

.loading-spinner {
  width: 24px;
  height: 24px;
  border: 3px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
