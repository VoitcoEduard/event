<template>
  <main class="data-page">
    <header>
      <button type="button" class="back-btn" @click="router.push(`/admin/event/${eventId}`)"><AppIcon name="arrowLeft" :size="18" aria-hidden="true" /> {{ t('data_library.back_to_event') }}</button>
      <h1>{{ event?.name || 'Event' }} · Data</h1>
      <button type="button" class="refresh-btn" :disabled="busy || loading || dirty" @click="load">{{ t('data_library.refresh_btn') }}</button>
    </header>
    <p class="scoring-note">{{ t('data_library.scoring_note') }}</p>
    <p v-if="error" role="alert" class="error">{{ error }}</p>
    <p v-if="notice" role="status" class="notice">{{ notice }}</p>
    <p v-if="loading">{{ t('data_library.loading') }}</p>
    <template v-else-if="event">
      <nav :aria-label="t('data_library.modules')">
        <button type="button" :class="{ active: activeTab === 'unified' }" :disabled="busy" @click="selectTab('unified')">{{ t('data_library.all_points') }}</button>
        <button v-for="module in event.modules || []" :key="module.id" type="button" :disabled="busy" :class="{ active: activeTab === module.id }" @click="selectTab(module.id)">{{ module.name }}</button>
      </nav>
      <section class="toolbar">
        <label>{{ t('data_library.show') }} <select v-model="target" :disabled="busy || dirty"><option value="players">{{ t('data_library.players') }}</option><option value="teams">{{ t('data_library.teams') }}</option></select></label>
        <label v-if="selectedModule">{{ t('data_library.manual_points_for') }}
          <select :value="manualTarget(selectedModule)" :disabled="busy || dirty" @change="saveTarget($event.target.value)">
            <option value="players">{{ t('data_library.players') }}</option><option value="teams">{{ t('data_library.teams') }}</option><option value="both">{{ t('data_library.players_and_teams') }}</option>
          </select>
        </label>
        <span class="record-count">{{ rows.length }} {{ target }}</span>
      </section>
      <p class="sheet-hint" v-if="!selectedModule">{{ t('data_library.edit_hint') }}</p>
      <p class="sheet-hint" v-else-if="!allowsManual(selectedModule, target)">{{ t('data_library.manual_disabled', { target: target === 'players' ? t('data_library.players') : t('data_library.teams') }) }}</p>
      <div class="table-shell">
        <div class="table-wrap" role="region" :aria-label="t('data_library.spreadsheet')" tabindex="0">
          <table>
            <thead><tr><th class="row-number" scope="col">#</th><th class="name-cell" scope="col">{{ target === 'players' ? t('data_library.player') : t('data_library.team') }}</th>
              <template v-for="module in visibleModules" :key="module.id">
                <th v-if="target === 'players' && isGame(module)" scope="col" class="score-heading">{{ module.name }}<span>{{ t('data_library.latest_game') }}</span></th>
                <th v-if="module.type === 'teams'" scope="col" class="score-heading">{{ module.name }}<span>{{ t(target === 'players' ? 'teams_module.earned' : 'teams_module.team_score') }}</span></th>
                <th scope="col" class="score-heading">{{ module.name }}<span>{{ t('data_library.manual_adjustments') }}</span></th>
              </template><th scope="col" class="total-cell">{{ t('data_library.total_points') }}</th><th scope="col" class="actions-cell">{{ t('data_library.actions') }}</th>
            </tr></thead>
            <tbody><tr v-for="(row, index) in rows" :key="row.id" :class="{ edited: hasDraft(row) }">
              <td class="row-number" data-label="#">{{ index + 1 }}</td>
              <th scope="row" class="name-cell" :title="row.id"><input v-if="target === 'players' && row.registered" :aria-label="`${row.name} player name`" :disabled="busy" :value="drafts[row.id]?.name ?? row.name" @input="setDraft(row, 'name', $event.target.value)" /><span v-else>{{ row.name }}</span><small>{{ row.id }}</small></th>
              <template v-for="module in visibleModules" :key="module.id">
                <td v-if="target === 'players' && isGame(module)" :data-label="`${module.name} · ${t('data_library.latest_game')}`">
                  <input v-if="selectedModule && isEditableGameScore(module) && row.game[module.id] !== undefined" :aria-label="`${row.name} game points`" type="number" step="1" :disabled="busy" :value="draft(row, 'game')" @input="setDraft(row, 'game', $event.target.value)" />
                  <span v-else>{{ row.game[module.id] ?? '—' }}</span>
                </td>
                <td v-if="module.type === 'teams'" :data-label="`${module.name} · ${t(target === 'players' ? 'teams_module.earned' : 'teams_module.team_score')}`">{{ row.awards[module.id] ?? 0 }}</td>
                <td :data-label="`${module.name} · ${t('data_library.manual_adjustments')}`"><input v-if="selectedModule && row.registered && allowsManual(module, target)" :aria-label="`${row.name} manual points`" type="number" step="any" :disabled="busy" :value="draft(row, 'manual')" @input="setDraft(row, 'manual', $event.target.value)" /><span v-else>{{ row.manual[module.id] ?? 0 }}</span></td>
              </template>
              <td class="total-cell" :data-label="t('data_library.total_points')"><strong>{{ row.total }}</strong></td>
              <td class="actions-cell" :data-label="t('data_library.actions')"><button type="button" :disabled="busy || !hasDraft(row)" @click="saveRow(row)">{{ t('data_library.save') }}</button><button v-if="hasDraft(row)" type="button" :disabled="busy" @click="delete drafts[row.id]">{{ t('common.cancel') }}</button></td>
            </tr></tbody>
          </table>
        </div>
      </div>
      <div class="sheet-status"><span>{{ selectedModule?.name || t('data_library.all_points') }} · {{ target === 'players' ? t('data_library.player_scores') : t('data_library.team_scores') }}</span><span>{{ busy ? t('data_library.saving') : dirty ? t('data_library.unsaved') : t('data_library.up_to_date') }}</span></div>
      <p v-if="!rows.length">{{ t('data_library.no_rows', { target: target === 'players' ? t('data_library.players') : t('data_library.teams') }) }}</p>
      <section v-if="selectedModule?.type === 'teams'">
        <h2>{{ t('teams_module.standings') }}</h2>
        <p><router-link :to="{ name: 'teams-admin', params: { id: eventId }, query: { moduleId: selectedModule.id } }">{{ t('teams_module.manage_teams') }}</router-link></p>
        <article v-for="(result, resultIndex) in responses" :key="resultIndex">
          <p v-if="result.status">{{ t(`teams_module.${result.status}`) }}</p>
          <div class="table-shell">
            <div class="table-wrap"><table><thead><tr><th>{{ t('teams_module.rank') }}</th><th>{{ t('teams_module.team_name') }}</th><th>{{ t('teams_module.team_score') }}</th><th>{{ t('teams_module.members') }}</th></tr></thead><tbody><tr v-for="team in result.teams || []" :key="team.id"><td :data-label="t('teams_module.rank')">{{ team.rank ?? '—' }}</td><td :data-label="t('teams_module.team_name')">{{ team.name }}</td><td :data-label="t('teams_module.team_score')">{{ team.score ?? '—' }}</td><td :data-label="t('teams_module.members')">{{ team.members?.map(member => member.nickname).join(', ') }}</td></tr></tbody></table></div>
          </div>
          <h3>{{ t('teams_module.award_history') }}</h3><div v-for="award in result.awards || []" :key="award.id" class="result-award"><strong>{{ award.points > 0 ? '+' : '' }}{{ award.points }}</strong> · {{ result.teams?.find(team => team.id === award.teamId)?.name }} · {{ award.reason }} · {{ award.recipients.length }} {{ t('teams_module.members') }}</div>
        </article>
      </section>
      <section v-if="selectedModule?.type === 'form'">
        <h2>{{ t('data_library.form_responses') }}</h2>
        <p v-if="!responses.length">{{ t('data_library.no_responses') }}</p>
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
import { ref, computed, watch, getCurrentInstance } from 'vue'
import { useRoute, useRouter, onBeforeRouteLeave, onBeforeRouteUpdate } from 'vue-router'
import { API_BASE_URL, getAuthHeaders } from '../utils'
import AppIcon from '../components/AppIcon.vue'
import { extractLabel } from '../utils/labelExtractor'
import { isGame, isEditableGameScore, manualTarget, allowsManual, scoreRows, parsePoints } from '../utils/eventScores'
const route = useRoute(), router = useRouter()
const instance = getCurrentInstance()
const t = (...args) => instance?.appContext.config.globalProperties.$t?.(...args) ?? args[0]
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
  if (Object.keys(drafts.value).length) { error.value = 'Save or cancel your edits before changing modules.'; return }
  activeTab.value = id; error.value = ''; notice.value = ''
  try { await loadResponses() } catch (err) { error.value = err.message }
}
function draft(row, kind) { return drafts.value[row.id]?.[kind] ?? row[kind][activeTab.value] ?? 0 }
function setDraft(row, kind, value) { drafts.value[row.id] = { ...drafts.value[row.id], [kind]: value } }
function hasDraft(row) { return !!drafts.value[row.id] }
async function refreshScores() {
  const [updatedScores, updatedParticipants] = await Promise.all([request(path('/scores')), request(path('/participants'))])
  scores.value = updatedScores
  participants.value = updatedParticipants || []
}
async function saveRow(row) {
  busy.value = true; error.value = ''; notice.value = ''
  try {
    const values = drafts.value[row.id]
    const game = values.game === undefined || !isEditableGameScore(selectedModule.value) ? undefined : parsePoints(values.game, true)
    const manual = values.manual === undefined ? undefined : parsePoints(values.manual)
    const name = values.name === undefined ? undefined : values.name.trim()
    if (name !== undefined && (!name || name.length > 120)) throw new Error('Enter a player name (up to 120 characters).')
    if (name !== undefined) {
      await request(path(`/participants/${encodeURIComponent(row.id)}/name`), 'POST', { displayName: name })
      delete values.name
    }
    if (game !== undefined) {
      await request(modulePath() + `/quiz-scores/${encodeURIComponent(row.id)}`, 'PUT', { score: game })
      delete values.game
    }
    if (manual !== undefined) {
      await request(modulePath() + `/manual-scores/${target.value}/${encodeURIComponent(row.id)}`, 'PUT', { points: manual })
      delete values.manual
    }
    delete drafts.value[row.id]
    await refreshScores()
    await loadResponses()
    notice.value = `Changes saved for ${name ?? row.name}.`
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
    if (key === 'step_welcome_name') {
      await request(path(`/participants/${encodeURIComponent(response.userId)}/name`), 'POST', { displayName: value })
      data[key] = value.trim()
      await refreshScores()
    }
    await request(modulePath() + `/responses/${encodeURIComponent(response.id)}`, 'PUT', { data })
    response.data = data; notice.value = 'Response saved.'
  } catch (err) { error.value = `Response was not saved: ${err.message}`; try { await loadResponses() } catch { /* Preserve save error. */ } }
  finally { busy.value = false }
}
function canNavigate() {
  if (busy.value || dirty.value) { error.value = 'Save or cancel your edits before leaving.'; return false }
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
  container: data-page / inline-size;
  box-sizing: border-box;
  width: 100%;
  min-height: 100vh;
  min-height: 100dvh;
  scroll-padding-bottom: calc(84px + env(safe-area-inset-bottom, 0px));
  padding: 24px max(28px, env(safe-area-inset-right, 0px)) calc(84px + env(safe-area-inset-bottom, 0px)) max(28px, env(safe-area-inset-left, 0px));
  background: var(--page);
  color: var(--ink);
  color-scheme: dark;
  font-size: 13px;
  line-height: 1.45;
}
.data-page *, .data-page *::before, .data-page *::after { box-sizing: border-box; }
header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 18px;
  padding: 12px 16px;
  background: var(--sheet);
  border: 1px solid var(--line);
  border-radius: 10px;
}
header button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  flex-shrink: 0;
}
header button svg { flex-shrink: 0; }
h1 {
  flex: 1;
  min-inline-size: 0;
  margin: 0;
  font-size: 22px;
  font-weight: 650;
  letter-spacing: -.4px;
  overflow-wrap: break-word;
  text-wrap: balance;
}
p { color: var(--muted); margin: 12px 0; text-wrap: pretty; overflow-wrap: break-word; }
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
nav {
  display: flex;
  align-items: end;
  flex-wrap: nowrap;
  gap: 4px;
  overflow-x: auto;
  overscroll-behavior-x: contain;
  scrollbar-width: thin;
  padding: 6px 6px 0;
  border: 1px solid #50575f;
  border-radius: 8px 8px 0 0;
}
.data-page nav button {
  flex: none;
  white-space: nowrap;
  min-block-size: 44px;
  background: transparent;
  border-color: transparent;
  border-radius: 5px 5px 0 0;
  padding: 9px 16px;
  color: #b4bcc4;
}
.data-page nav button.active { background: #383d42; color: #d8e4ec; border-color: #50575f; border-bottom-color: #383d42; box-shadow: inset 0 2px #90a8b8; font-weight: 600; }
:global(html[data-admin-appearance="true"] main.data-page nav button) {
  background: transparent !important;
  border-color: transparent !important;
  color: var(--admin-text-muted) !important;
}
:global(html[data-admin-appearance="true"] main.data-page nav button:hover:not(:disabled)) {
  background: var(--admin-surface-hover) !important;
  color: var(--admin-text) !important;
}
:global(html[data-admin-appearance="true"] main.data-page nav button.active) {
  background: var(--admin-primary-soft) !important;
  color: var(--admin-primary) !important;
  border-color: color-mix(in srgb, var(--admin-primary) 30%, var(--admin-border)) !important;
  border-bottom-color: transparent !important;
  box-shadow: inset 0 2px 0 var(--admin-primary) !important;
  font-weight: 600;
}
.toolbar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px 24px;
  padding: 12px 14px;
  background: #31363b;
  border: 1px solid #4b525a;
  border-top: 0;
  border-radius: 0 0 8px 8px;
}
label { display: inline-flex; align-items: center; gap: 9px; color: var(--muted, #b8c0c8); font-size: 12px; }
.data-page select { max-width: 100%; }
.record-count { margin-left: auto; color: var(--muted); font-size: 12px; font-variant-numeric: tabular-nums; }
.sheet-hint { font-size: 12px; margin: 10px 1px; }
.table-shell {
  container: data-table-shell / inline-size;
  margin-top: 12px;
}
.table-wrap {
  container: data-table / inline-size;
  overflow: auto;
  max-height: 65vh;
  min-height: 160px;
  background: var(--sheet);
  border: 1px solid #50575e;
  scrollbar-color: #626c75 #30353a;
}
table { border-collapse: separate; border-spacing: 0; width: 100%; table-layout: fixed; font-variant-numeric: tabular-nums; }
th, td { height: 42px; padding: 5px 12px; border-right: 1px solid var(--line); border-bottom: 1px solid var(--line); text-align: right; background: var(--sheet); vertical-align: middle; }
th:last-child, td:last-child { border-right: 0; }
thead th { position: sticky; top: 0; z-index: 3; height: 52px; background: #3c4248; color: #d1d8de; font-size: 12px; font-weight: 600; border-bottom-color: #59626b; }
.score-heading { width: 166px; overflow-wrap: anywhere; }
.score-heading span { display: block; font-size: 11px; font-weight: 400; color: var(--muted); margin-top: 2px; }
.row-number { width: 42px; min-width: 42px; padding: 5px 8px; text-align: center; position: sticky; left: 0; z-index: 2; background: #363c42; color: #aeb7bf; font-size: 11px; }
.name-cell { position: sticky; left: 42px; z-index: 2; width: 210px; text-align: left; border-right-color: #59626b; font-weight: 500; }
.name-cell > span { display: block; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
.name-cell small { display: block; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; font-weight: 400; color: var(--muted, #a4afb8); font-size: 10px; margin-top: 1px; }
thead .row-number, thead .name-cell { z-index: 4; background: #3b4249; }
.total-cell { width: 130px; background: #3a4249; }
.actions-cell { width: 152px; text-align: left; white-space: nowrap; }
.data-page .actions-cell button { min-height: 25px; font-size: 11px; padding: 3px 8px; margin-right: 5px; }
tbody tr:nth-child(even) > td:not(.row-number):not(.total-cell), tbody tr:nth-child(even) > .name-cell { background: #363a3f; }
tbody tr:hover > td:not(.row-number), tbody tr:hover > .name-cell { background: #404951; }
tbody tr.edited > td:not(.row-number), tbody tr.edited > .name-cell { background: #414a40; }
:global(html[data-admin-appearance="true"] main.data-page table tbody tr.edited > :is(td, th)) {
  background: color-mix(in srgb, var(--admin-primary) 12%, var(--admin-surface)) !important;
}
.data-page td input[type="number"] { width: 100%; min-width: 0; min-height: 28px; height: 28px; padding: 3px 5px; text-align: right; background: transparent; border: 1px solid transparent; border-radius: 2px; font-variant-numeric: tabular-nums; appearance: textfield; }
.data-page td input[type="number"]::-webkit-inner-spin-button, .data-page td input[type="number"]::-webkit-outer-spin-button { appearance: none; margin: 0; }
.data-page td input:hover:not(:disabled) { border-color: #71818e; background: #3f474e; }
.data-page td input:focus { border-color: #92b0c5; outline: 1px solid #92b0c5; outline-offset: 0; background: #29343e; }
.data-page td input:disabled { color: var(--muted); }
.data-page .name-cell input { width: 100%; min-width: 0; background: transparent; font-weight: 500; }
.sheet-status { display: flex; justify-content: space-between; gap: 12px; padding: 7px 12px; font-size: 11px; color: var(--muted); background: #363c42; border: 1px solid #50575e; border-top: 0; }
.error, .notice { padding: 9px 12px; border: 1px solid; border-radius: 4px; font-size: 12px; overflow-wrap: break-word; }
.error { color: #edc0b9; background: #493b3b; border-color: #755751; }
.notice { color: #bdd8c5; background: #35473d; border-color: #536f5d; }
:global(html[data-admin-appearance="true"] main.data-page > section:not(.toolbar)) {
  background: transparent !important;
  border: none !important;
  box-shadow: none !important;
  margin-top: 24px;
  padding: 0;
}
.result-award { padding: 8px 0; border-bottom: 1px solid var(--line); overflow-wrap: break-word; }
a, :global(html[data-admin-appearance="true"] main.data-page a) { color: var(--admin-primary, #afd3fb); }
h2 { font-size: 16px; margin-top: 24px; text-wrap: balance; }
h3 { font-size: 13px; margin: 0 0 10px; overflow-wrap: anywhere; }
article { min-inline-size: 0; padding: 14px; border: 1px solid var(--line); border-radius: 10px; background: #343a40; margin-top: 12px; }
article h3:not(:first-child) { margin-top: 18px; }
article label { margin: 4px 16px 4px 0; }
.data-page article input { width: 240px; max-width: 100%; min-inline-size: 0; }

@media (max-width: 700px) {
  .data-page {
    padding: max(14px, env(safe-area-inset-top, 0px)) max(12px, env(safe-area-inset-right, 0px)) calc(84px + env(safe-area-inset-bottom, 0px)) max(12px, env(safe-area-inset-left, 0px));
  }
}

@container data-page (inline-size <= 700px) {
  header {
    gap: 10px;
    flex-wrap: wrap;
    padding: 12px;
  }
  header button { min-block-size: 44px; }
  .refresh-btn { margin-left: auto; }
  h1 {
    order: 3;
    flex: 1 0 100%;
    font-size: clamp(18px, 4.8vw, 22px);
  }
  .name-cell { width: 155px; }
  .score-heading { width: 135px; }
  .sheet-status { flex-wrap: wrap; }
  article label {
    display: grid;
    gap: 6px;
    margin: 10px 0;
  }
  .data-page article input {
    width: 100%;
    min-block-size: 44px;
    font-size: 16px;
  }
}

@container data-page (inline-size <= 640px) {
  .toolbar {
    flex-direction: column;
    align-items: stretch;
    gap: 10px;
  }
  .toolbar label {
    display: flex;
    flex-direction: column;
    align-items: stretch;
    gap: 6px;
    width: 100%;
  }
  .toolbar select {
    width: 100%;
    min-block-size: 44px;
    font-size: 16px;
  }
  .record-count { margin-left: 0; }
  .sheet-status {
    border-top: 1px solid var(--admin-border, #50575e);
    border-radius: 8px;
    margin-top: 4px;
  }
}

/* A container cannot query itself, so .table-shell queries the wrapper width for .table-wrap. */
@container data-table-shell (inline-size < 640px) {
  .table-wrap,
  :global(html[data-admin-appearance="true"] main.data-page .table-shell .table-wrap) {
    max-height: none;
    min-height: 0;
    overflow: visible;
    border: none;
    background: transparent !important;
    box-shadow: none !important;
  }
}

@container data-table (inline-size < 640px) {
  table, tbody, tr, th, td {
    display: block;
    width: 100%;
  }
  table,
  :global(html[data-admin-appearance="true"] main.data-page .table-wrap table) {
    background: transparent !important;
  }
  thead { display: none; }
  tbody tr {
    margin-block-end: 14px;
    border: 1px solid var(--admin-border, #50575e);
    border-radius: 12px;
    overflow: hidden;
    background: var(--admin-surface, var(--sheet));
  }
  tbody tr:last-child { margin-block-end: 0; }
  th, td {
    position: static;
    left: auto;
    width: 100%;
    min-width: 0;
    height: auto;
    border-right: 0;
    border-bottom: 1px solid var(--admin-border, var(--line));
  }
  td {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
    min-height: 44px;
    padding: 10px 14px;
    text-align: right;
    overflow-wrap: break-word;
    background: transparent;
  }
  td::before {
    content: attr(data-label);
    font-size: 12px;
    font-weight: 600;
    color: var(--admin-text-muted, #adb5bd);
    text-align: left;
    flex: 1 1 auto;
    overflow-wrap: break-word;
  }
  tbody tr > td:last-child { border-bottom: 0; }
  tbody tr:nth-child(even) > td:not(.row-number):not(.total-cell),
  tbody tr:hover > td:not(.row-number) {
    background: transparent;
  }
  :global(html[data-admin-appearance="true"] main.data-page .table-wrap table tbody tr > td),
  :global(html[data-admin-appearance="true"] main.data-page .table-wrap tbody tr:nth-child(even) > td),
  :global(html[data-admin-appearance="true"] main.data-page .table-wrap tbody tr:hover > td) {
    background: transparent !important;
  }
  .row-number {
    position: static;
    left: auto;
    z-index: auto;
    width: 100%;
    min-width: 0;
    justify-content: flex-start;
    gap: 2px;
    min-height: 0;
    padding: 10px 14px 2px;
    border-bottom: 0;
    text-align: left;
    font-size: 11px;
    font-weight: 600;
    color: var(--admin-text-muted, #aeb7bf);
    background: var(--admin-surface-subtle, #3b4249);
  }
  td.row-number::before {
    flex: 0 0 auto;
    font-size: 11px;
  }
  .name-cell {
    position: static;
    left: auto;
    z-index: auto;
    width: 100%;
    padding: 4px 14px 12px;
    text-align: left;
    background: var(--admin-surface-subtle, #3b4249);
  }
  .name-cell > span {
    white-space: normal;
    overflow-wrap: break-word;
    font-size: 15px;
    font-weight: 600;
  }
  .name-cell small {
    white-space: normal;
    overflow-wrap: anywhere;
    margin-top: 3px;
  }
  .data-page .name-cell input {
    width: 100%;
    min-height: 44px;
    font-size: 16px;
    padding: 8px 10px;
    background: var(--admin-surface, #29343e);
    border-color: var(--admin-border-strong, #50575f);
  }
  :global(html[data-admin-appearance="true"] main.data-page .table-wrap tbody tr > td.row-number),
  :global(html[data-admin-appearance="true"] main.data-page .table-wrap tbody tr > th.name-cell),
  :global(html[data-admin-appearance="true"] main.data-page .table-wrap tbody tr:nth-child(even) > td.row-number),
  :global(html[data-admin-appearance="true"] main.data-page .table-wrap tbody tr:nth-child(even) > th.name-cell),
  :global(html[data-admin-appearance="true"] main.data-page .table-wrap tbody tr:hover > td.row-number),
  :global(html[data-admin-appearance="true"] main.data-page .table-wrap tbody tr:hover > th.name-cell) {
    background: var(--admin-surface-subtle) !important;
  }
  .total-cell {
    width: 100%;
    background: var(--admin-surface-subtle, #3a4249);
  }
  :global(html[data-admin-appearance="true"] main.data-page .table-wrap tbody tr > td.total-cell),
  :global(html[data-admin-appearance="true"] main.data-page .table-wrap tbody tr:nth-child(even) > td.total-cell),
  :global(html[data-admin-appearance="true"] main.data-page .table-wrap tbody tr:hover > td.total-cell) {
    background: var(--admin-surface-subtle) !important;
  }
  tbody tr.edited,
  :global(html[data-admin-appearance="true"] main.data-page .table-wrap tbody tr.edited) {
    border-color: var(--admin-primary, #90a8b8) !important;
    background: color-mix(in srgb, var(--admin-primary, #90a8b8) 8%, var(--admin-surface, var(--sheet))) !important;
  }
  :global(html[data-admin-appearance="true"] main.data-page .table-wrap tbody tr.edited > td.row-number),
  :global(html[data-admin-appearance="true"] main.data-page .table-wrap tbody tr.edited > th.name-cell),
  :global(html[data-admin-appearance="true"] main.data-page .table-wrap tbody tr.edited > td.total-cell) {
    background: color-mix(in srgb, var(--admin-primary) 16%, var(--admin-surface-subtle)) !important;
  }
  .data-page td input[type="number"] {
    width: min(140px, 45%);
    height: auto;
    min-height: 40px;
    font-size: 16px;
    padding: 6px 10px;
    border-color: var(--admin-border-strong, #50575f);
    background: var(--admin-surface-subtle, #383d42);
  }
  .actions-cell {
    width: 100%;
    white-space: normal;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: 8px;
  }
  .data-page .actions-cell button {
    min-block-size: 40px;
    min-height: 40px;
    font-size: 13px;
    padding: 6px 14px;
    margin-right: 0;
  }
}

@media (pointer: coarse) {
  .data-page button,
  .data-page select,
  .data-page input,
  .data-page .actions-cell button,
  .data-page td input[type="number"],
  .data-page .name-cell input {
    min-block-size: 44px;
    min-inline-size: 44px;
  }
  .data-page td input[type="number"] {
    min-height: 44px;
    height: auto;
  }
  .data-page a {
    display: inline-flex;
    align-items: center;
    min-block-size: 44px;
  }
}
</style>
