<template>
  <main class="data-page">
    <header>
      <button @click="router.push(`/admin/event/${eventId}`)">← Back to event</button>
      <h1>{{ event?.name || 'Event' }} · Data</h1>
      <button :disabled="busy || loading || dirty" @click="load">Refresh</button>
    </header>
    <p>Only the latest completed run of each game counts. Manual points are added separately. Team points stay with the team.</p>
    <p v-if="error" role="alert" class="error">{{ error }}</p>
    <p v-if="notice" role="status" class="notice">{{ notice }}</p>
    <p v-if="loading">Loading event data…</p>
    <template v-else-if="event">
      <nav aria-label="Modules">
        <button :class="{ active: activeTab === 'unified' }" :disabled="busy" @click="selectTab('unified')">All points</button>
        <button v-for="module in event.modules || []" :key="module.id" :disabled="busy" :class="{ active: activeTab === module.id }" @click="selectTab(module.id)">{{ module.name }}</button>
      </nav>
      <section class="toolbar">
        <label>Show <select v-model="target" :disabled="busy || dirty"><option value="players">Players</option><option value="teams">Teams</option></select></label>
        <label v-if="selectedModule">Manual points for
          <select :value="manualTarget(selectedModule)" :disabled="busy || dirty" @change="saveTarget($event.target.value)">
            <option value="players">Players</option><option value="teams">Teams</option><option value="both">Players and teams</option>
          </select>
        </label>
        <span>{{ rows.length }} {{ target }}</span>
      </section>
      <p v-if="!selectedModule">Choose a module to enter or correct points.</p>
      <p v-else-if="!allowsManual(selectedModule, target)">Manual entry for {{ target }} is disabled for this module. Previously saved points still count.</p>
      <div class="table-wrap">
        <table>
          <thead><tr><th>{{ target === 'players' ? 'Player' : 'Team' }}</th>
            <template v-for="module in visibleModules" :key="module.id">
              <th v-if="target === 'players' && isGame(module)">{{ module.name }} · latest game</th>
              <th>{{ module.name }} · manual</th>
            </template><th>Total points</th><th v-if="selectedModule">Actions</th>
          </tr></thead>
          <tbody><tr v-for="row in rows" :key="row.id">
            <th scope="row">{{ row.name }}<small>{{ row.id }}</small></th>
            <template v-for="module in visibleModules" :key="module.id">
              <td v-if="target === 'players' && isGame(module)">
                <input v-if="selectedModule && row.game[module.id] !== undefined" :aria-label="`${row.name} game points`" type="number" step="1" :disabled="busy" :value="draft(row, 'game')" @input="setDraft(row, 'game', $event.target.value)" />
                <span v-else>{{ row.game[module.id] ?? '—' }}</span>
              </td>
              <td><input v-if="selectedModule && row.registered && allowsManual(module, target)" :aria-label="`${row.name} manual points`" type="number" step="any" :disabled="busy" :value="draft(row, 'manual')" @input="setDraft(row, 'manual', $event.target.value)" /><span v-else>{{ row.manual[module.id] ?? 0 }}</span></td>
            </template>
            <td><strong>{{ row.total }}</strong></td>
            <td v-if="selectedModule"><button :disabled="busy || !hasDraft(row)" @click="saveRow(row)">Save</button><button v-if="hasDraft(row)" :disabled="busy" @click="delete drafts[row.id]">Cancel</button></td>
          </tr></tbody>
        </table>
      </div>
      <p v-if="!rows.length">No {{ target }} yet. Add them to the event before entering points.</p>
      <section v-if="selectedModule?.type === 'form'">
        <h2>Form responses</h2>
        <p v-if="!responses.length">No responses yet.</p>
        <article v-for="response in responses" :key="response.id">
          <h3>{{ response.userId }}</h3>
          <label v-for="(value, key) in response.data" :key="key">{{ fieldLabel(key) }}
            <input :value="typeof value === 'object' ? JSON.stringify(value) : value" :disabled="busy || typeof value === 'object'" @change="saveResponse(response, key, $event.target.value)" />
          </label>
        </article>
      </section>
    </template>
  </main>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter, onBeforeRouteLeave, onBeforeRouteUpdate } from 'vue-router'
import { API_BASE_URL, getAuthHeaders } from '../utils'
import { extractLabel } from '../utils/labelExtractor'
import { isGame, manualTarget, allowsManual, scoreRows, parsePoints } from '../utils/eventScores'
const route = useRoute(), router = useRouter()
const eventId = computed(() => route.query.eventId)
const event = ref(null), participants = ref([]), teams = ref([]), scores = ref({}), responses = ref([])
const loading = ref(false), busy = ref(false), error = ref(''), notice = ref(''), activeTab = ref('unified'), target = ref('players'), drafts = ref({})
const selectedModule = computed(() => event.value?.modules?.find(m => m.id === activeTab.value))
const visibleModules = computed(() => selectedModule.value ? [selectedModule.value] : event.value?.modules || [])
const rows = computed(() => event.value ? scoreRows(event.value, participants.value, teams.value, scores.value, target.value, activeTab.value) : [])
const dirty = computed(() => Object.keys(drafts.value).length > 0)
let generation = 0
const path = suffix => `/events/${encodeURIComponent(eventId.value)}${suffix}`
const modulePath = () => path(`/modules/${encodeURIComponent(activeTab.value)}`)
async function request(url, method = 'GET', body) {
  const response = await fetch(`${API_BASE_URL}${url}`, { method, headers: { ...getAuthHeaders(), ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}) }, ...(body !== undefined ? { body: JSON.stringify(body) } : {}) })
  if (!response.ok) throw new Error((await response.text()) || `Request failed (${response.status})`)
  return response.status === 204 ? null : response.json()
}
async function load() {
  const current = ++generation
  loading.value = true; error.value = ''; drafts.value = {}; event.value = null
  try {
    if (!eventId.value) throw new Error('Open Data from an event to continue.')
    const [e, p, t, s] = await Promise.all([request(path('')), request(path('/participants')), request(path('/teams')), request(path('/scores'))])
    if (current !== generation) return
    event.value = e; participants.value = p || []; teams.value = t || []; scores.value = s
    if (!e.modules?.some(m => m.id === activeTab.value)) activeTab.value = 'unified'
    await loadResponses()
  } catch (err) { if (current === generation) error.value = err.message }
  finally { if (current === generation) loading.value = false }
}
async function loadResponses() {
  const tab = activeTab.value
  responses.value = []
  if (selectedModule.value?.type !== 'form') return
  const result = await request(modulePath() + '/responses')
  if (activeTab.value === tab) responses.value = result.responses || []
}
async function selectTab(id) {
  if (Object.keys(drafts.value).length) { error.value = 'Save or cancel your point edits before changing modules.'; return }
  activeTab.value = id; error.value = ''; notice.value = ''
  try { await loadResponses() } catch (err) { error.value = err.message }
}
function draft(row, kind) { return drafts.value[row.id]?.[kind] ?? row[kind][activeTab.value] ?? 0 }
function setDraft(row, kind, value) { drafts.value[row.id] = { ...drafts.value[row.id], [kind]: value } }
function hasDraft(row) { return !!drafts.value[row.id] }
async function refreshScores() { scores.value = await request(path('/scores')) }
async function saveRow(row) {
  busy.value = true; error.value = ''; notice.value = ''
  try {
    const values = drafts.value[row.id]
    const game = values.game === undefined ? undefined : parsePoints(values.game, true)
    const manual = values.manual === undefined ? undefined : parsePoints(values.manual)
    if (game !== undefined) {
      await request(modulePath() + `/quiz-scores/${encodeURIComponent(row.id)}`, 'PUT', { score: game })
      delete values.game
    }
    if (manual !== undefined) {
      await request(modulePath() + `/manual-scores/${target.value}/${encodeURIComponent(row.id)}`, 'PUT', { points: manual })
      delete values.manual
    }
    delete drafts.value[row.id]
    await refreshScores(); notice.value = `Points saved for ${row.name}.`
  } catch (err) {
    error.value = `Could not finish saving: ${err.message}`
    try { await refreshScores() } catch { /* Keep the original save error visible. */ }
  } finally { busy.value = false }
}
async function saveTarget(value) {
  busy.value = true; error.value = ''
  try {
    await request(modulePath() + '/manual-score-target', 'PUT', { target: value })
    selectedModule.value.data = { ...selectedModule.value.data, manualScoreTarget: value }; drafts.value = {}
    notice.value = 'Manual entry setting saved. Existing points are preserved.'
  } catch (err) { error.value = err.message } finally { busy.value = false }
}
function fieldLabel(key) {
  const field = selectedModule.value?.data?.timeline?.find(step => step.id === key)
  return field?.data?.label ? extractLabel(field.data.label) : key
}
async function saveResponse(response, key, value) {
  busy.value = true; error.value = ''
  try {
    const data = { ...response.data, [key]: value }
    await request(modulePath() + `/responses/${encodeURIComponent(response.id)}`, 'PUT', { data })
    response.data = data; notice.value = 'Response saved.'
  } catch (err) { error.value = `Response was not saved: ${err.message}`; try { await loadResponses() } catch { /* Preserve save error. */ } }
  finally { busy.value = false }
}
function canNavigate() {
  if (busy.value || dirty.value) { error.value = 'Save or cancel your point edits before leaving.'; return false }
  return true
}
onBeforeRouteLeave(canNavigate)
onBeforeRouteUpdate(canNavigate)
watch(target, () => { drafts.value = {}; notice.value = '' })
watch(eventId, load, { immediate: true })
</script>

<style scoped>
.data-page { min-height:100vh; padding:24px; background:#121212; color:#eee; }
header, nav, .toolbar { display:flex; align-items:center; gap:12px; flex-wrap:wrap; margin-bottom:20px; }
h1 { flex:1; font-size:24px; } p, small { color:#aaa; } small { display:block; font-size:11px; margin-top:5px; font-weight:normal; }
button, select, input { color:#eee; background:#252525; border:1px solid #555; border-radius:6px; padding:8px 12px; }
button { cursor:pointer; } button.active { background:#155cac; border-color:#479afa; } button:disabled, input:disabled { opacity:.5; cursor:default; }
input { width:110px; } label { display:inline-flex; align-items:center; gap:10px; margin-right:18px; }
.table-wrap { overflow:auto; border:1px solid #444; border-radius:8px; } table { width:100%; border-collapse:collapse; }
th, td { padding:14px; text-align:left; border-bottom:1px solid #333; white-space:nowrap; } thead { background:#252525; } td button { margin-right:6px; }
.error { color:#ffb4ab; background:#431b1b; padding:12px; } .notice { color:#a8e2b8; } article { padding:16px; border-bottom:1px solid #444; } article input { width:240px; }
</style>
