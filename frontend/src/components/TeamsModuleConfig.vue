<template>
  <Teleport to="body">
    <div class="teams-config-overlay" @click.self="$emit('close')">
      <section
        ref="modalRef"
        class="teams-config-modal"
        role="dialog"
        aria-modal="true"
        :aria-label="$t('teams_module.configure')"
        tabindex="-1"
        @keydown.escape.stop.prevent="$emit('close')"
      >
        <header><h2>{{ module.name }}</h2><button type="button" @click="$emit('close')" :aria-label="$t('common.close')">×</button></header>
        <p>{{ $t('teams_module.configure') }}</p>
        <TeamsConfigFields v-model="config" :sources="sources" :source-locked="initialized" />
        <p v-if="error" role="alert" class="error">{{ error }}</p>
        <footer><button type="button" :disabled="saving" @click="save">{{ $t('teams_module.save_config') }}</button><button type="button" @click="openAdmin">{{ $t('teams_module.manage_teams') }}</button></footer>
      </section>
    </div>
  </Teleport>
</template>
<script setup>
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
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
const modalRef = ref(null)
let previouslyFocusedEl = null

function handleKeydown(event) {
  if (event.key === 'Escape' && !event.defaultPrevented) {
    event.preventDefault()
    emit('close')
  }
}

onMounted(async () => {
  if (typeof window !== 'undefined') {
    window.addEventListener('keydown', handleKeydown)
  }
  if (typeof document !== 'undefined') {
    previouslyFocusedEl = document.activeElement
    nextTick(() => {
      if (modalRef.value && !modalRef.value.contains(document.activeElement)) {
        modalRef.value.focus?.()
      }
    })
  }
  try { const data = await teamsActivity.admin(props.eventId, props.module.id); if (data.activity) { initialized.value = true; config.value = data.activity.config } }
  catch (err) { error.value = err.message }
})

onUnmounted(() => {
  if (typeof window !== 'undefined') {
    window.removeEventListener('keydown', handleKeydown)
  }
  if (previouslyFocusedEl && typeof previouslyFocusedEl.focus === 'function' && previouslyFocusedEl.isConnected) {
    previouslyFocusedEl.focus()
  }
  previouslyFocusedEl = null
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
.teams-config-overlay {
  position: fixed;
  inset: 0;
  z-index: 2000;
  display: grid;
  place-items: center;
  padding: 16px;
  background: #000b;
}
.teams-config-modal {
  background: #192235;
  color: #fff;
  border: 1px solid #43516a;
  padding: 24px;
  border-radius: 18px;
  width: min(540px, 100%);
  max-height: 92vh;
  max-height: 92dvh;
  overflow: auto;
  overscroll-behavior: contain;
  box-sizing: border-box;
}
header, footer {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
}
h2 {
  overflow-wrap: anywhere;
  min-inline-size: 0;
}
header button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
button {
  cursor: pointer;
  color: #fff;
  background: #264c91;
  border: 0;
  border-radius: 8px;
  padding: 12px;
}
footer {
  margin-top: 24px;
  flex-wrap: wrap;
}
.error { color: #ffb3b3; }

@media (pointer: coarse) {
  button {
    min-block-size: var(--target-size, 44px);
    min-inline-size: var(--target-size, 44px);
  }
}

@media (max-width: 480px) {
  .teams-config-overlay {
    padding: max(12px, env(safe-area-inset-top, 0px)) max(12px, env(safe-area-inset-right, 0px)) max(12px, env(safe-area-inset-bottom, 0px)) max(12px, env(safe-area-inset-left, 0px));
  }
  .teams-config-modal {
    padding: 18px;
  }
  footer {
    flex-direction: column-reverse;
    align-items: stretch;
  }
  footer button {
    width: 100%;
  }
}
</style>
