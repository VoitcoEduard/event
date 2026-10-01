<template>
  <div class="app-container" :class="{ 'admin-app-container': adminAppearanceActive }">
    <router-view />
    <AdminThemeToggle v-if="adminAppearanceActive" />
    <Toast />
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { API_BASE_URL, getUserId } from './utils'
import Toast from './components/Toast.vue'
import AdminThemeToggle from './components/AdminThemeToggle.vue'
import { useAdminTheme } from './composables/useAdminTheme'

const { adminAppearanceActive } = useAdminTheme()

onMounted(() => {
  if (!sessionStorage.getItem('currentScreen')) {
    sessionStorage.setItem('currentScreen', 'main')
  }

  let uid = localStorage.getItem('userId')
  if (!uid) {
    uid = 'user_' + Math.random().toString(36).substr(2, 9)
    localStorage.setItem('userId', uid)
  }
})

defineExpose({ getUserId })
</script>

<style>
.app-container {
  min-height: 100vh;
  background-color: var(--ops-bg);
}

.admin-app-container { background-color: var(--admin-bg); }
</style>
