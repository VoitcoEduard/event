<template>
  <main class="access-page">
    <nav><RouterLink to="/admin">{{ $t('auth.backToAdmin') }}</RouterLink></nav>
    <h1>{{ $t('auth.manageUsers') }}</h1>
    <p v-if="error">{{ error }}</p>
    <div v-for="user in users" :key="user.id" class="user-row">
      <div><strong>{{ user.name || user.email }}</strong><br>{{ user.email }}<br>{{ user.status }} · {{ user.role }}</div>
      <div v-if="user.role !== 'owner'">
        <button v-if="user.status !== 'active'" :disabled="busy" @click="changeStatus(user, 'active')">{{ $t('auth.approve') }}</button>
        <button v-if="user.status === 'active'" :disabled="busy" @click="changeStatus(user, 'disabled')">{{ $t('auth.disable') }}</button>
      </div>
    </div>
    <button v-if="hasMore" @click="nextPage">{{ $t('auth.loadMore') }}</button>
  </main>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { listAdminUsers, setAdminStatus } from '../session'

const { t } = useI18n()
const users = ref([])
const page = ref(0)
const hasMore = ref(false)
const error = ref('')
const busy = ref(false)

async function load() {
  error.value = ''
  try {
    const result = await listAdminUsers(page.value)
    users.value = page.value === 0 ? result.users : [...users.value, ...result.users]
    hasMore.value = result.hasMore
  } catch { error.value = t('auth.unavailable') }
}
async function nextPage() { page.value++; await load() }
async function changeStatus(user, status) {
  busy.value = true
  error.value = ''
  try {
    await setAdminStatus(user.id, status)
    user.status = status
  } catch { error.value = t('auth.unavailable') }
  finally { busy.value = false }
}
onMounted(load)
</script>

<style scoped>
.access-page { min-height: 100vh; padding: 2rem; background: #121212; color: white; }
.user-row { display: flex; justify-content: space-between; align-items: center; gap: 1rem; padding: 1rem; border-bottom: 1px solid #444; }
button { margin-left: .5rem; padding: .6rem 1rem; cursor: pointer; }
a { color: #a5b4fc; }
</style>
