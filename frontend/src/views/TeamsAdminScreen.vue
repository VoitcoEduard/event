<template>
  <div class="teams-admin-wrapper">
    <nav class="teams-admin-nav"><button @click="router.push(`/admin/event/${eventId}`)">← {{ $t('teams_module.back_event') }}</button><select v-model="selectedId" :aria-label="$t('teams_module.activity')"><option value="">{{ $t('teams_module.select_activity') }}</option><option v-for="module in modules.filter(module => module.type === 'teams')" :key="module.id" :value="module.id">{{ module.name }}</option></select></nav>
    <p v-if="error" class="teams-error">{{ error }}</p>
    <TeamsActivityAdmin v-if="selected" :key="selected.id" :event-id="eventId" :module="selected" :modules="modules" />
    <p v-else class="teams-picker-message">{{ $t('teams_module.select_activity') }}</p>
  </div>
</template>
<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { API_BASE_URL } from '../utils'
import TeamsActivityAdmin from '../components/TeamsActivityAdmin.vue'
const route = useRoute(), router = useRouter()
const eventId = route.params.id || route.query.eventId
const modules = ref([]), error = ref(''), selectedId = ref(route.query.moduleId || '')
const selected = computed(() => modules.value.find(module => module.type === 'teams' && module.id === selectedId.value))
watch(selectedId, value => router.replace({ query: { ...route.query, moduleId: value || undefined } }))
onMounted(async () => {
  try { const response = await fetch(`${API_BASE_URL}/events/${eventId}`); if (!response.ok) throw new Error(await response.text()); modules.value = (await response.json()).modules || []; const teams = modules.value.filter(module => module.type === 'teams'); if (!selectedId.value && teams.length === 1) selectedId.value = teams[0].id }
  catch (err) { error.value = err.message }
})
</script>
<style scoped>
.teams-admin-wrapper{background:#0c1220;min-height:100vh;color:white}.teams-admin-nav{display:flex;gap:12px;padding:16px 28px;background:#162034;border-bottom:1px solid #32445e;flex-wrap:wrap}.teams-admin-nav button,.teams-admin-nav select{background:#24354e;border:1px solid #43516a;color:white;border-radius:8px;padding:10px;font:inherit;max-width:100%}.teams-picker-message{padding:28px}
</style>
