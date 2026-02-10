<template>
  <div class="app-container">
    <router-view />
    <Toast />
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { API_BASE_URL, getUserId } from './utils'
import Toast from './components/Toast.vue'

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
  background-color: #121212;
}
</style>
