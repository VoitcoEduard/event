<template>
  <main class="access-page">
    <h1>{{ $t('auth.pendingTitle') }}</h1>
    <p>{{ $t('auth.pendingMessage') }}</p>
    <button @click="check">{{ $t('auth.checkAgain') }}</button>
    <button @click="leave">{{ $t('auth.signOut') }}</button>
    <p v-if="error">{{ error }}</p>
  </main>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { loadSession, signOut } from '../session'

const router = useRouter()
const { t } = useI18n()
const error = ref('')
async function check() {
  error.value = ''
  try {
    const user = await loadSession(true)
    if (user?.status === 'active') router.push('/admin')
    else if (!user) router.push('/login')
    else if (user.status === 'disabled') router.push({ name: 'login', query: { error: 'disabled' } })
  } catch { error.value = t('auth.unavailable') }
}
async function leave() {
  try { await signOut(); router.push('/login') } catch { error.value = t('auth.unavailable') }
}
</script>

<style scoped>
.access-page { min-height: 100vh; display: grid; place-content: center; gap: 1rem; padding: 2rem; background: #121212; color: white; text-align: center; }
button { padding: .8rem 1rem; border-radius: .5rem; cursor: pointer; }
</style>
