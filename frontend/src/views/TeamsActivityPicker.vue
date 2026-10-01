<template><main class="teams-module teams-player"><button @click="router.push({ name: 'event', query: { eventId } })">← {{ $t('teams_module.back_event') }}</button><h1>{{ $t('teams_module.select_activity') }}</h1><p v-if="error" class="teams-error">{{ error }}</p><p v-if="loading" class="teams-notice">{{ $t('teams_module.loading') }}</p><div class="player-choice"><button v-for="module in modules" :key="module.id" @click="open(module)"><strong>{{ module.name }}</strong>{{ $t('teams_module.enter') }}</button></div><p v-if="!loading && !modules.length">{{ $t('teams_module.no_activities') }}</p></main></template>
<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { API_BASE_URL } from '../utils'
import '../styles/teams-activity.css'
const route = useRoute(), router = useRouter(), modules = ref([]), error = ref(''), loading = ref(true)
const eventId = route.query.eventId || route.params.eventId
function open(module) { router.replace(`/teams/${eventId}/${module.id}/player`) }
onMounted(async () => {
  if (!eventId) { loading.value = false; router.replace('/'); return }
  try { const response = await fetch(`${API_BASE_URL}/events/${eventId}`); if (!response.ok) throw new Error(await response.text()); modules.value = (await response.json()).modules?.filter(module => module.type === 'teams' && module.isActive) || []; const requested = modules.value.find(module => module.id === route.query.moduleId); if (requested) open(requested); else if (modules.value.length === 1) open(modules.value[0]) }
  catch (err) { error.value = err.message } finally { loading.value = false }
})
</script>
