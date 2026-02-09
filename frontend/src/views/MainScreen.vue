<template>
  <div class="main-screen">
    <div class="content">
      <h1 class="title">Hello from Expo</h1>
      <p class="subtitle">Minimalist Design</p>

      <div class="card">
        <h2 class="card-title">Connect to Event</h2>
        <p class="card-text">Enter the 6-digit PIN code to join the live event.</p>
        
        <input
          v-model="pin"
          type="text"
          placeholder="000000"
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
          <span v-else>Connect</span>
        </button>
      </div>

      <button class="admin-link" @click="goToAdmin">
        Go to Admin Dashboard
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'

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
  }
})

const handlePinInput = (e) => {
  pin.value = e.target.value.replace(/[^0-9]/g, '')
  error.value = ''
}

const handleConnect = async () => {
  error.value = ''
  if (!pin.value || pin.value.length !== 6) {
    error.value = 'Please enter a valid 6-digit PIN'
    return
  }

  loading.value = true
  try {
    const response = await fetch(`${API_URL}/events/connect`, {
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
        error.value = 'Event not found. Check your PIN.'
      } else if (response.status === 403) {
        error.value = 'This event is not active currently.'
      } else {
        error.value = 'Failed to connect to event.'
      }
    }
  } catch (err) {
    console.error('Connection error:', err)
    error.value = 'Network error. Please try again.'
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
  background-color: #FAFAFA;
  display: flex;
  align-items: center;
  justify-content: center;
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
  color: #1A1A1A;
  margin-bottom: 8px;
  letter-spacing: -0.5;
}

.subtitle {
  font-size: 16px;
  color: #666666;
  margin-bottom: 40px;
  font-weight: 500;
  letter-spacing: 0.5;
  text-transform: uppercase;
}

.card {
  background-color: #FFFFFF;
  padding: 32px;
  border-radius: 24px;
  width: 100%;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
  margin-bottom: 32px;
  border: 1px solid #F0F0F0;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.card-title {
  font-size: 20px;
  font-weight: 700;
  color: #1A1A1A;
  margin-bottom: 8px;
}

.card-text {
  font-size: 15px;
  color: #666;
  text-align: center;
  margin-bottom: 24px;
  line-height: 22px;
}

.input {
  background-color: #F5F5F7;
  border-radius: 12px;
  padding: 16px;
  font-size: 24px;
  font-weight: 700;
  text-align: center;
  letter-spacing: 8px;
  margin-bottom: 24px;
  color: #1A1A1A;
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

.admin-link {
  padding: 16px;
  background: none;
  color: #666;
  font-size: 14px;
  font-weight: 500;
}

.error-text {
  color: #FF3B30;
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
