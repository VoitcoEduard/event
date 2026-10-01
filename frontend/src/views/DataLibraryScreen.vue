<template>
  <main class="data-page">
    <header>
      <button @click="router.push(`/admin/event/${eventId}`)"><AppIcon name="arrowLeft" :size="18" /> Back to event</button>
      <h1>{{ event?.name || 'Event' }} · Data</h1>
      <button :disabled="busy || loading || dirty" @click="load">Refresh</button>
    </header>
    <p class="scoring-note">Only the latest completed run of each game counts. Manual adjustments are added separately. Teams activity awards give the full award to each recorded member; legacy team adjustments stay with the team.</p>
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
        <span class="record-count">{{ rows.length }} {{ target }}</span>
      </section>
      <p class="sheet-hint" v-if="!selectedModule">Choose a module to enter or correct points.</p>
      <p class="sheet-hint" v-else-if="!allowsManual(selectedModule, target)">Manual entry for {{ target }} is disabled for this module. Previously saved points still count.</p>
      <div class="table-wrap" role="region" aria-label="Event points spreadsheet" tabindex="0">
        <table>
          <thead><tr><th class="row-number" scope="col">#</th><th class="name-cell" scope="col">{{ target === 'players' ? 'Player' : 'Team' }}</th>
            <template v-for="module in visibleModules" :key="module.id">
              <th v-if="target === 'players' && isGame(module)" scope="col" class="score-heading">{{ module.name }}<span>Latest game</span></th>
              <th v-if="module.type === 'teams'" scope="col" class="score-heading">{{ module.name }}<span>{{ $t(target === 'players' ? 'teams_module.earned' : 'teams_module.team_score') }}</span></th>
              <th scope="col" class="score-heading">{{ module.name }}<span>Manual adjustments</span></th>
            </template><th scope="col" class="total-cell">Total points</th><th v-if="selectedModule" scope="col" class="actions-cell">Actions</th>
          </tr></thead>
          <tbody><tr v-for="(row, index) in rows" :key="row.id" :class="{ edited: hasDraft(row) }">
            <td class="row-number">{{ index + 1 }}</td>
            <th scope="row" class="name-cell" :title="row.id"><span>{{ row.name }}</span><small>{{ row.id }}</small></th>
            <template v-for="module in visibleModules" :key="module.id">
              <td v-if="target === 'players' && isGame(module)">
                <input v-if="selectedModule && row.game[module.id] !== undefined" :aria-label="`${row.name} game points`" type="number" step="1" :disabled="busy" :value="draft(row, 'game')" @input="setDraft(row, 'game', $event.target.value)" />
                <span v-else>{{ row.game[module.id] ?? '—' }}</span>
              </td>
              <td v-if="module.type === 'teams'">{{ row.awards[module.id] ?? 0 }}</td>
              <td><input v-if="selectedModule && row.registered && allowsManual(module, target)" :aria-label="`${row.name} manual points`" type="number" step="any" :disabled="busy" :value="draft(row, 'manual')" @input="setDraft(row, 'manual', $event.target.value)" /><span v-else>{{ row.manual[module.id] ?? 0 }}</span></td>
            </template>
            <td class="total-cell"><strong>{{ row.total }}</strong></td>
            <td v-if="selectedModule" class="actions-cell"><button :disabled="busy || !hasDraft(row)" @click="saveRow(row)">Save</button><button v-if="hasDraft(row)" :disabled="busy" @click="delete drafts[row.id]">Cancel</button></td>
          </tr></tbody>
        </table>
      </div>
      <div class="sheet-status"><span>{{ selectedModule?.name || 'All points' }} · {{ target === 'players' ? 'Player scores' : 'Team scores' }}</span><span>{{ busy ? 'Saving…' : dirty ? 'Unsaved changes — save each edited row' : 'Up to date' }}</span></div>
      <p v-if="!rows.length">No {{ target }} yet. Add them to the event before entering points.</p>
      <section v-if="selectedModule?.type === 'teams'">
        <h2>{{ $t('teams_module.standings') }}</h2>
        <p><router-link :to="{ name: 'teams-admin', params: { id: eventId }, query: { moduleId: selectedModule.id } }">{{ $t('teams_module.manage_teams') }}</router-link></p>
        <article v-for="(result, resultIndex) in responses" :key="resultIndex">
          <p v-if="result.status">{{ $t(`teams_module.${result.status}`) }}</p>
          <div class="table-wrap"><table><thead><tr><th>{{ $t('teams_module.rank') }}</th><th>{{ $t('teams_module.team_name') }}</th><th>{{ $t('teams_module.team_score') }}</th><th>{{ $t('teams_module.members') }}</th></tr></thead><tbody><tr v-for="team in result.teams || []" :key="team.id"><td>{{ team.rank ?? '—' }}</td><td>{{ team.name }}</td><td>{{ team.score ?? '—' }}</td><td>{{ team.members?.map(member => member.nickname).join(', ') }}</td></tr></tbody></table></div>
          <h3>{{ $t('teams_module.award_history') }}</h3><div v-for="award in result.awards || []" :key="award.id" class="result-award"><strong>{{ award.points > 0 ? '+' : '' }}{{ award.points }}</strong> · {{ result.teams?.find(team => team.id === award.teamId)?.name }} · {{ award.reason }} · {{ award.recipients.length }} {{ $t('teams_module.members') }}</div>
        </article>
      </section>
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
import AppIcon from '../components/AppIcon.vue'
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
  if (!['form', 'teams'].includes(selectedModule.value?.type)) return
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
.data-page {
  --page: #292c30;
  --sheet: #33373b;
  --line: #454a50;
  --ink: #e0e3e6;
  --muted: #adb5bd;
  box-sizing: border-box;
  min-height: 100vh;
  padding: 24px 28px;
  background: var(--page);
  color: var(--ink);
  color-scheme: dark;
  font-size: 13px;
  line-height: 1.45;
}
.data-page *, .data-page *::before, .data-page *::after { box-sizing: border-box; }
header { display: flex; align-items: center; gap: 16px; margin-bottom: 18px; }
h1 { flex: 1; margin: 0; font-size: 22px; font-weight: 650; letter-spacing: -.4px; }
p { color: var(--muted); margin: 12px 0; }
.scoring-note { font-size: 12px; margin: 0 0 20px; max-width: 1000px; }
.data-page button, .data-page select, .data-page input {
  font: inherit;
  color: var(--ink);
  background: #383d42;
  border: 1px solid #50575f;
  border-radius: 4px;
  box-shadow: none;
  min-height: 30px;
  padding: 5px 10px;
  margin: 0;
}
.data-page button { cursor: pointer; white-space: nowrap; }
.data-page button:hover:not(:disabled) { background: #444b52; border-color: #68737d; }
.data-page button:disabled { color: #8e979f; background: transparent; border-color: #454a50; cursor: default; }
.data-page :is(button, select, input, .table-wrap):focus-visible { outline: 2px solid #91acbf; outline-offset: 2px; }
nav { display: flex; align-items: end; gap: 3px; overflow-x: auto; border-bottom: 1px solid #50575f; }
.data-page nav button { background: transparent; border-color: transparent; border-radius: 5px 5px 0 0; padding: 9px 16px; color: #b4bcc4; }
.data-page nav button.active { background: #383d42; color: #d8e4ec; border-color: #50575f; border-bottom-color: #383d42; box-shadow: inset 0 2px #90a8b8; font-weight: 600; }
.toolbar { display: flex; align-items: center; flex-wrap: wrap; gap: 12px 24px; padding: 12px 14px; background: #31363b; border: 1px solid #4b525a; border-top: 0; }
label { display: inline-flex; align-items: center; gap: 9px; color: #b8c0c8; font-size: 12px; }
.data-page select { max-width: 100%; }
.record-count { margin-left: auto; color: var(--muted); font-size: 12px; font-variant-numeric: tabular-nums; }
.sheet-hint { font-size: 12px; margin: 10px 1px; }
.table-wrap { margin-top: 12px; overflow: auto; max-height: 65vh; min-height: 160px; background: var(--sheet); border: 1px solid #50575e; scrollbar-color: #626c75 #30353a; }
table { border-collapse: separate; border-spacing: 0; width: 100%; table-layout: fixed; font-variant-numeric: tabular-nums; }
th, td { height: 42px; padding: 5px 12px; border-right: 1px solid var(--line); border-bottom: 1px solid var(--line); text-align: right; background: var(--sheet); vertical-align: middle; }
th:last-child, td:last-child { border-right: 0; }
thead th { position: sticky; top: 0; z-index: 3; height: 52px; background: #3c4248; color: #d1d8de; font-size: 12px; font-weight: 600; border-bottom-color: #59626b; }
.score-heading { width: 166px; overflow-wrap: anywhere; }
.score-heading span { display: block; font-size: 11px; font-weight: 400; color: var(--muted); margin-top: 2px; }
.row-number { width: 42px; min-width: 42px; padding: 5px 8px; text-align: center; position: sticky; left: 0; z-index: 2; background: #363c42; color: #aeb7bf; font-size: 11px; }
.name-cell { position: sticky; left: 42px; z-index: 2; width: 210px; text-align: left; border-right-color: #59626b; font-weight: 500; }
.name-cell > span { display: block; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
.name-cell small { display: block; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; font-weight: 400; color: #a4afb8; font-size: 10px; margin-top: 1px; }
thead .row-number, thead .name-cell { z-index: 4; background: #3b4249; }
.total-cell { width: 130px; background: #3a4249; }
.actions-cell { width: 152px; text-align: left; white-space: nowrap; }
.data-page .actions-cell button { min-height: 25px; font-size: 11px; padding: 3px 8px; margin-right: 5px; }
tbody tr:nth-child(even) > td:not(.row-number):not(.total-cell), tbody tr:nth-child(even) > .name-cell { background: #363a3f; }
tbody tr:hover > td:not(.row-number), tbody tr:hover > .name-cell { background: #404951; }
tbody tr.edited > td:not(.row-number), tbody tr.edited > .name-cell { background: #414a40; }
.data-page td input[type="number"] { width: 100%; min-width: 0; min-height: 28px; height: 28px; padding: 3px 5px; text-align: right; background: transparent; border: 1px solid transparent; border-radius: 2px; font-variant-numeric: tabular-nums; appearance: textfield; }
.data-page td input[type="number"]::-webkit-inner-spin-button, .data-page td input[type="number"]::-webkit-outer-spin-button { appearance: none; margin: 0; }
.data-page td input:hover:not(:disabled) { border-color: #71818e; background: #3f474e; }
.data-page td input:focus { border-color: #92b0c5; outline: 1px solid #92b0c5; outline-offset: 0; background: #29343e; }
.data-page td input:disabled { color: var(--muted); }
.sheet-status { display: flex; justify-content: space-between; gap: 12px; padding: 7px 12px; font-size: 11px; color: var(--muted); background: #363c42; border: 1px solid #50575e; border-top: 0; }
.error, .notice { padding: 9px 12px; border: 1px solid; border-radius: 4px; font-size: 12px; }
.error { color: #edc0b9; background: #493b3b; border-color: #755751; }
.notice { color: #bdd8c5; background: #35473d; border-color: #536f5d; }
.result-award{padding:8px 0;border-bottom:1px solid var(--line)}a{color:#afd3fb}h2 { font-size: 16px; margin-top: 24px; } h3 { font-size: 13px; margin: 0 0 10px; }
article { padding: 14px; border: 1px solid var(--line); background: #343a40; }
article label { margin: 4px 16px 4px 0; }
.data-page article input { width: 240px; max-width: 100%; }
@media (max-width: 700px) {
  .data-page { padding: 14px 12px; }
  header { gap: 10px; flex-wrap: wrap; } h1 { font-size: 18px; }
  header button { font-size: 11px; }
  .toolbar { gap: 10px; } .record-count { margin-left: 0; }
  .name-cell { width: 155px; } .score-heading { width: 135px; }
  .sheet-status { flex-wrap: wrap; }
}
</style>
