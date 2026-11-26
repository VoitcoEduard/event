<template>
  <div class="app-container" :class="{ 'admin-app-container': adminAppearanceActive }">
    <router-view />
    <Toast />
    <AppDialogHost />
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { API_BASE_URL, getUserId } from './utils'
import { safeSessionGet, safeSessionSet } from './utils/storage'
import Toast from './components/Toast.vue'
import AppDialogHost from './components/AppDialogHost.vue'
import { useAdminTheme } from './composables/useAdminTheme'

const { adminAppearanceActive } = useAdminTheme()

onMounted(() => {
  if (!safeSessionGet('currentScreen')) {
    safeSessionSet('currentScreen', 'main')
  }

  getUserId()
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
