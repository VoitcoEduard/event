<template>
  <Teleport to="body">
    <div class="teams-config-overlay" @click.self="$emit('close')">
      <section class="teams-config-modal" role="dialog" aria-modal="true" :aria-label="$t('teams_module.configure')">
        <header><h2>{{ module.name }}</h2><button @click="$emit('close')" :aria-label="$t('common.close')">×</button></header>
        <p>{{ $t('teams_module.configure') }}</p>
        <TeamsConfigFields v-model="config" :sources="sources" :source-locked="initialized" />
        <p v-if="error" role="alert" class="error">{{ error }}</p>
        <footer><button :disabled="saving" @click="save">{{ $t('teams_module.save_config') }}</button><button @click="openAdmin">{{ $t('teams_module.manage_teams') }}</button></footer>
      </section>
    </div>
  </Teleport>
</template>
<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { API_BASE_URL, getAuthHeaders } from '../utils'
import { teamsActivity } from '../services/teamsActivity'
import { defaultTeamsConfig } from '../utils/teamsActivity'
import TeamsConfigFields from './TeamsConfigFields.vue'
const props = defineProps({ module: Object, eventId: String, modules: { type: Array, default: () => [] } })
const emit = defineEmits(['close', 'update'])
const router = useRouter()
const index = props.modules.findIndex(module => module.id === props.module.id)
const sources = props.modules.slice(0, index).filter(module => module.type === 'teams')
const config = ref({ ...defaultTeamsConfig(), ...props.module.data?.teamsConfig })
const initialized = ref(false), saving = ref(false), error = ref('')
onMounted(async () => {
  try { const data = await teamsActivity.admin(props.eventId, props.module.id); if (data.activity) { initialized.value = true; config.value = data.activity.config } }
  catch (err) { error.value = err.message }
})
async function save() {
  saving.value = true; error.value = ''
  try {
    if (initialized.value) await teamsActivity.action(props.eventId, props.module.id, { action: 'configure', config: config.value })
    const response = await fetch(`${API_BASE_URL}/events/${props.eventId}/modules/${props.module.id}`, { method: 'PUT', headers: { 'Content-Type': 'application/json', ...getAuthHeaders() }, body: JSON.stringify({ data: { teamsConfig: config.value } }) })
    if (!response.ok) throw new Error(await response.text())
    emit('update'); emit('close')
  } catch (err) { error.value = err.message } finally { saving.value = false }
}
function openAdmin() { router.push({ name: 'teams-admin', params: { id: props.eventId }, query: { moduleId: props.module.id } }); emit('close') }
</script>
<style scoped>
.teams-config-overlay{position:fixed;inset:0;z-index:2000;display:grid;place-items:center;padding:16px;background:#000b}.teams-config-modal{background:#192235;color:#fff;border:1px solid #43516a;padding:24px;border-radius:18px;width:min(540px,100%);max-height:90vh;overflow:auto}header,footer{display:flex;justify-content:space-between;gap:12px;align-items:center}h2{overflow-wrap:anywhere}button{cursor:pointer;color:#fff;background:#264c91;border:0;border-radius:8px;padding:12px}footer{margin-top:24px;flex-wrap:wrap}.error{color:#ffb3b3}
</style>
