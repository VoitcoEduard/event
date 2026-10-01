<template>
  <div class="teams-admin-wrapper">
    <nav class="teams-admin-nav">
      <div class="teams-admin-nav-content">
        <div class="teams-admin-nav-left">
          <button class="teams-admin-back" @click="router.push(`/admin/event/${eventId}`)">← {{ $t('teams_module.back_event') }}</button>
          <div class="teams-admin-context">
            <span>{{ eventName }}</span>
            <strong>{{ $t('teams_module.teams') }}</strong>
          </div>
        </div>
        <label class="teams-admin-picker">
          <span>{{ $t('teams_module.activity') }}</span>
          <select v-model="selectedId" :aria-label="$t('teams_module.activity')">
            <option value="">{{ $t('teams_module.select_activity') }}</option>
            <option v-for="module in teamModules" :key="module.id" :value="module.id">{{ module.name }}</option>
          </select>
        </label>
      </div>
    </nav>
    <p v-if="error" class="teams-error teams-page-message">{{ error }}</p>
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
const modules = ref([]), eventName = ref(''), error = ref(''), selectedId = ref(route.query.moduleId || '')
const teamModules = computed(() => modules.value.filter(module => module.type === 'teams'))
const selected = computed(() => teamModules.value.find(module => module.id === selectedId.value))
watch(selectedId, value => router.replace({ query: { ...route.query, moduleId: value || undefined } }))
onMounted(async () => {
  try { const response = await fetch(`${API_BASE_URL}/events/${eventId}`); if (!response.ok) throw new Error(await response.text()); const event = await response.json(); eventName.value = event.name || ''; modules.value = event.modules || []; if (!selectedId.value && teamModules.value.length === 1) selectedId.value = teamModules.value[0].id }
  catch (err) { error.value = err.message }
})
</script>
<style scoped>
.teams-admin-wrapper{background:#121212;min-height:100vh;color:#fff}.teams-admin-nav{padding:20px 40px;background:#1e1e1e;border-bottom:1px solid #333;box-shadow:0 2px 4px #0003}.teams-admin-nav-content{display:flex;align-items:center;justify-content:space-between;gap:24px;max-width:1400px;margin:0 auto}.teams-admin-nav-left{display:flex;align-items:center;gap:20px;min-width:0}.teams-admin-back{border:0;border-radius:8px;padding:12px 20px;background:#607d8b;color:#fff;font:inherit;font-weight:600;cursor:pointer;white-space:nowrap}.teams-admin-back:hover{background:#546e7a}.teams-admin-context{display:flex;flex-direction:column;min-width:0}.teams-admin-context span{color:#888;font-size:12px;text-transform:uppercase;letter-spacing:.8px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.teams-admin-context strong{font-size:26px;line-height:1.2}.teams-admin-picker{display:flex;align-items:center;gap:12px;color:#888;font-size:12px;font-weight:600;text-transform:uppercase;letter-spacing:.6px}.teams-admin-picker select{min-width:220px;max-width:360px;background:#252525;border:1px solid #444;color:#fff;border-radius:8px;padding:11px 36px 11px 12px;font:inherit;text-transform:none;letter-spacing:normal}.teams-page-message,.teams-picker-message{max-width:1400px;margin:24px auto;padding:16px 24px}.teams-picker-message{color:#888;background:#1e1e1e;border:1px solid #333;border-radius:12px}@media(max-width:760px){.teams-admin-nav{padding:16px 18px}.teams-admin-nav-content,.teams-admin-picker{align-items:stretch;flex-direction:column}.teams-admin-nav-left{align-items:flex-start}.teams-admin-picker select{max-width:none;width:100%}}
</style>
