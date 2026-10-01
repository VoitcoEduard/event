<template>
  <main class="teams-module teams-admin">
    <header class="teams-header">
      <div class="teams-summary">
        <p class="teams-eyebrow">{{ $t('teams_module.offline_activity') }}</p>
        <div v-if="activity" class="teams-summary-counts">
          <span class="teams-status" :class="activity.status">{{ $t(`teams_module.${activity.status}`) }}</span>
          <span>{{ $t('teams_module.participant_count', activity.entrants.length) }}</span>
          <span>{{ $t('teams_module.unassigned_count', unassigned.length) }}</span>
        </div>
        <div class="teams-visibility">
          <button role="switch" :aria-checked="visibleToPlayers" :aria-label="$t('teams_module.player_visibility')" :disabled="busy || loading" @click="toggleVisibility">
            <span class="visibility-track" :class="{ visible: visibleToPlayers }" aria-hidden="true"><span></span></span>
            {{ $t(visibleToPlayers ? 'teams_module.visible_players' : 'teams_module.hidden_players') }}
          </button>
          <p class="teams-muted">{{ $t('teams_module.visibility_help') }}</p>
        </div>
        <button v-if="activity && activity.status !== 'completed'" :disabled="busy" @click="showParticipants">{{ $t('teams_module.add_participants') }}</button>
      </div>
      <div class="teams-actions">
        <button v-if="activity?.status === 'setup'" class="primary start-action" :disabled="busy" @click="act({ action: 'start' })">{{ $t('teams_module.start') }}</button>
        <button v-if="activity?.status === 'live'" class="primary" @click="manageScores">{{ $t('teams_module.manage_scores') }}</button>
        <button v-if="activity?.status === 'completed'" class="primary reopen-action" :disabled="busy" @click="reopen">{{ $t('teams_module.reopen') }}</button>
        <button class="projector-action" @click="showProjector">{{ $t('teams_module.projector') }}</button>
        <button v-if="activity?.status === 'live'" class="complete-action" :disabled="busy" @click="complete">{{ $t('teams_module.complete') }}</button>
      </div>
    </header>
    <p v-if="loading" class="teams-notice">{{ $t('teams_module.loading') }}</p>
    <p v-if="disconnected" class="teams-notice">{{ $t('teams_module.disconnected') }}</p>
    <p v-if="message || error" class="teams-error" role="alert">{{ message || error }}</p>
    <div v-if="!loading && !activity" class="teams-panel setup-panel">
      <h2>{{ $t('teams_module.initialize') }}</h2><p>{{ $t('teams_module.initialize_help') }}</p>
      <TeamsConfigFields v-model="config" :sources="sources" />
      <div class="teams-actions"><button class="primary" :disabled="busy" @click="act({ action: 'initialize', config })">{{ $t('teams_module.setup_new') }}</button><button v-if="state?.legacyTeams?.length" :disabled="busy || !!config.sourceModuleId" @click="act({ action: 'initialize', config, importLegacy: true })">{{ $t('teams_module.import_legacy') }}</button></div>
    </div>
    <template v-if="activity">
      <details v-if="activity.status === 'setup'" class="teams-panel">
        <summary>{{ $t('teams_module.configure') }}</summary><TeamsConfigFields v-model="config" :sources="sources" source-locked /><button :disabled="busy" @click="act({ action: 'configure', config })">{{ $t('teams_module.save_config') }}</button>
      </details>
      <section v-if="activity.status !== 'completed'" class="teams-panel teams-toolbar">
        <div><strong>{{ $t('teams_module.full_points_help') }}</strong><label>{{ $t('teams_module.reason') }}<input v-model="reason" maxlength="500" :placeholder="$t('teams_module.reason_placeholder')" /></label></div>
        <div class="teams-actions"><button :disabled="busy" @click="editTeam()">{{ $t('teams_module.create_team') }}</button><template v-if="activity.status === 'setup' && activity.config.method === 'random'"><button :disabled="busy" @click="act({ action: 'distribute', reshuffle: false })">{{ $t('teams_module.distribute') }}</button><button :disabled="busy" @click="reshuffle">{{ $t('teams_module.reshuffle') }}</button></template></div>
      </section>
      <div class="teams-admin-grid">
        <section ref="scoringPanel" tabindex="-1" class="teams-cards" :aria-label="$t('teams_module.teams')">
          <p v-if="!activity.teams.length" class="teams-notice">{{ $t('teams_module.no_teams') }}</p>
          <article v-for="team in orderedTeams" :key="team.id" class="teams-panel team-card" :style="{ '--team-color': team.color }">
            <header><div><span class="team-dot"></span><h2>{{ team.name }}</h2><p>{{ $t('teams_module.member_count', members(team.id).length) }}<span v-if="team.capacity || activity.config.capacity"> / {{ team.capacity || activity.config.capacity }}</span></p></div><strong class="team-score">{{ totals.teams.get(team.id) || 0 }}</strong></header>
            <p v-if="activity.status === 'live' && members(team.id).length">{{ $t('teams_module.recipients', members(team.id).length) }}</p>
            <div v-if="activity.status === 'live' && members(team.id).length" class="award-buttons"><button v-for="points in activity.config.presets" :key="points" class="primary" :disabled="busy || !members(team.id).length" @click="award(team, points)">{{ points > 0 ? '+' : '' }}{{ points }}</button></div>
            <div v-if="activity.status === 'live' && members(team.id).length" class="custom-award"><input v-model="customPoints[team.id]" type="number" step="1" min="-1000000" max="1000000" :aria-label="$t('teams_module.custom_points')" :placeholder="$t('teams_module.custom_points')" /><button :disabled="busy || !members(team.id).length" @click="award(team, Number(customPoints[team.id]))">{{ $t('teams_module.award') }}</button></div>
            <div v-if="!members(team.id).length" class="team-empty">
              <p class="teams-muted">{{ $t('teams_module.empty_team') }}</p>
              <button v-if="activity.status !== 'completed'" :disabled="busy" @click="assignTeam = team; assignParticipantId = ''">{{ $t('teams_module.assign_participant') }}</button>
            </div>
            <ul v-else class="teams-roster"><li v-for="person in members(team.id)" :key="person.participantId"><span>{{ person.name }}</span><span v-if="activity.status === 'completed'">{{ totals.people.get(person.participantId) || 0 }} {{ $t('teams_module.points') }}</span><select v-else :value="person.teamId" :disabled="busy" :aria-label="$t('teams_module.assign_person', { name: person.name })" @change="act({ action: 'assign', participantId: person.participantId, teamId: $event.target.value })"><option value="">{{ $t('teams_module.unassigned') }}</option><option v-for="target in activity.teams" :key="target.id" :value="target.id">{{ target.name }}</option></select></li></ul>
            <div v-if="activity.status !== 'completed'" class="teams-actions"><button :disabled="busy" @click="editTeam(team)">{{ $t('teams_module.edit') }}</button><button :disabled="busy || hasAwards(team.id)" @click="deleteTeam(team)">{{ $t('teams_module.delete') }}</button></div>
          </article>
        </section>
        <aside ref="participantPanel" tabindex="-1" class="teams-panel entrant-panel">
          <h2>{{ $t('teams_module.unassigned') }} · {{ unassigned.length }}</h2><p v-if="!unassigned.length" class="teams-muted">{{ $t('teams_module.all_assigned') }}</p>
          <div v-for="person in unassigned" :key="person.participantId" class="entrant-row"><span>{{ person.name }}</span><select v-if="activity.status !== 'completed'" :disabled="busy" :aria-label="$t('teams_module.assign_person', { name: person.name })" @change="act({ action: 'assign', participantId: person.participantId, teamId: $event.target.value })"><option value="">{{ $t('teams_module.assign') }}</option><option v-for="team in activity.teams" :key="team.id" :value="team.id">{{ team.name }}</option></select></div>
          <template v-if="activity.status !== 'completed'">
            <h3>{{ $t('teams_module.invited') }} · {{ invited.length }}</h3><p class="teams-muted">{{ $t('teams_module.invited_help') }}</p><div v-for="person in invited" :key="person.participantId" class="entrant-row"><span>{{ person.name }}</span><button :disabled="busy" @click="act({ action: 'add_participant', participantId: person.participantId })">{{ $t('teams_module.add') }}</button></div>
            <h3>{{ $t('teams_module.add_participants') }}</h3><input ref="participantSearch" v-model="search" :placeholder="$t('teams_module.search')" :aria-label="$t('teams_module.search')" /><div class="participant-candidates"><div v-for="person in candidates" :key="person.participantId" class="entrant-row"><span>{{ person.name }}</span><button :disabled="busy" @click="act({ action: 'add_participant', participantId: person.participantId })">{{ $t('teams_module.add') }}</button></div></div>
          </template>
        </aside>
      </div>
      <section class="teams-panel award-history">
        <h2>{{ $t('teams_module.history') }} <span class="teams-muted">({{ history.length }})</span></h2>
        <p v-if="activity.status === 'completed' && history.length" class="teams-muted">{{ $t('teams_module.reopen_awards') }}</p>
        <p v-if="!history.length" class="teams-muted">{{ $t('teams_module.no_awards') }}</p>
        <ol>
          <li v-for="entry in visibleHistory" :key="entry.id">
            <div><strong>{{ entry.points > 0 ? '+' : '' }}{{ entry.points }} · {{ teamName(entry.teamId) }}</strong><p>{{ entry.reason || $t('teams_module.manual_award') }} · {{ $t('teams_module.member_count', entry.recipients.length) }}</p><small>{{ new Date(entry.createdAt).toLocaleString() }}</small></div>
            <span v-if="entry.reversesId || reversed.has(entry.id)">{{ $t('teams_module.reversed') }}</span>
            <button v-else-if="activity.status === 'live'" :disabled="busy" @click="act({ action: 'undo', awardId: entry.id })">{{ $t('teams_module.undo') }}</button>
          </li>
        </ol>
        <button v-if="history.length > 5" :aria-expanded="showAllHistory" @click="showAllHistory = !showAllHistory">{{ $t(showAllHistory ? 'teams_module.show_latest' : 'teams_module.show_all') }}</button>
      </section>
    </template>
    <div v-if="assignTeam" class="teams-overlay" @click.self="assignTeam = null">
      <form class="teams-panel team-editor" role="dialog" aria-modal="true" :aria-label="$t('teams_module.assign_to_team', { name: assignTeam.name })" @submit.prevent="assignToTeam">
        <h2>{{ $t('teams_module.assign_to_team', { name: assignTeam.name }) }}</h2>
        <label v-if="assignableParticipants.length">{{ $t('teams_module.participant') }}
          <select v-model="assignParticipantId" required :disabled="busy"><option value="" disabled>{{ $t('teams_module.select_participant') }}</option><option v-for="person in assignableParticipants" :key="person.participantId" :value="person.participantId">{{ person.name }}{{ person.teamId ? ` · ${teamName(person.teamId)}` : '' }}</option></select>
        </label>
        <p v-else class="teams-muted">{{ $t('teams_module.no_assignable_participants') }}</p>
        <p v-if="message" class="teams-error" role="alert">{{ message }}</p>
        <div class="teams-actions">
          <button v-if="assignableParticipants.length" class="primary" :disabled="busy || !assignParticipantId">{{ $t('teams_module.assign_participant') }}</button>
          <button v-else type="button" @click="assignTeam = null; showParticipants()">{{ $t('teams_module.add_participants') }}</button>
          <button type="button" :disabled="busy" @click="assignTeam = null">{{ $t('teams_module.cancel') }}</button>
        </div>
      </form>
    </div>
    <div v-if="teamForm" class="teams-overlay" @click.self="teamForm = null"><form class="teams-panel team-editor" @submit.prevent="saveTeam"><h2>{{ $t('teams_module.manage_teams') }}</h2><label>{{ $t('teams_module.team_name') }}<input v-model="teamForm.name" required maxlength="120" /></label><label>{{ $t('teams_module.team_color') }}<input v-model="teamForm.color" type="color" /></label><label>{{ $t('teams_module.capacity') }}<input v-model.number="teamForm.capacity" type="number" min="0" /></label><p v-if="message" class="teams-error" role="alert">{{ message }}</p><div class="teams-actions"><button :disabled="busy" class="primary">{{ $t('teams_module.save') }}</button><button type="button" @click="teamForm = null">{{ $t('teams_module.cancel') }}</button></div></form></div>
  </main>
</template>
<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTeamsActivity } from '../composables/useTeamsActivity'
import { teamsActivity } from '../services/teamsActivity'
import { activityTotals, defaultTeamsConfig } from '../utils/teamsActivity'
import TeamsConfigFields from './TeamsConfigFields.vue'
import '../styles/teams-activity.css'
const props = defineProps({ eventId: String, module: Object, modules: Array })
const { t } = useI18n()
const { state, loading, disconnected, error, load } = useTeamsActivity(props.eventId, props.module.id, 'admin')
const activity = computed(() => state.value?.activity)
const visibleToPlayers = computed(() => !!(state.value?.module || props.module).isActive)
const scoringPanel = ref(null), participantPanel = ref(null), participantSearch = ref(null)
const showAllHistory = ref(false), assignTeam = ref(null), assignParticipantId = ref('')
const assignableParticipants = computed(() => activity.value?.entrants.filter(person => person.teamId !== assignTeam.value?.id) || [])
const config = ref({ ...defaultTeamsConfig(), ...props.module.data?.teamsConfig })
let configLoaded = false
watch(activity, value => { if (value && !configLoaded) { config.value = JSON.parse(JSON.stringify(value.config)); configLoaded = true } })
const sources = computed(() => props.modules.slice(0, props.modules.findIndex(module => module.id === props.module.id)).filter(module => module.type === 'teams'))
const busy = ref(false), message = ref(''), reason = ref(''), search = ref(''), teamForm = ref(null), customPoints = ref({})
const totals = computed(() => activityTotals(activity.value))
const orderedTeams = computed(() => [...activity.value.teams].sort((a, b) => (totals.value.teams.get(b.id) || 0) - (totals.value.teams.get(a.id) || 0) || a.name.localeCompare(b.name)))
const unassigned = computed(() => activity.value.entrants.filter(person => !person.teamId))
const entrantIds = computed(() => new Set(activity.value.entrants.map(person => person.participantId)))
const invited = computed(() => activity.value.sourceMembers.filter(person => !entrantIds.value.has(person.participantId)))
const candidates = computed(() => (state.value.participants || []).filter(person => !entrantIds.value.has(person.participantId) && person.name.toLowerCase().includes(search.value.toLowerCase())))
const history = computed(() => [...activity.value.awards].reverse())
const visibleHistory = computed(() => showAllHistory.value ? history.value : history.value.slice(0, 5))
const reversed = computed(() => new Set(activity.value.awards.map(entry => entry.reversesId).filter(Boolean)))
const members = id => activity.value.entrants.filter(person => person.teamId === id)
const teamName = id => activity.value.teams.find(team => team.id === id)?.name || id
const hasAwards = id => activity.value.awards.some(entry => entry.teamId === id)
const pending = new Map()
async function act(payload) {
  if (busy.value) return false
  const key = JSON.stringify(payload)
  if (['award', 'undo'].includes(payload.action)) {
    if (!pending.has(key)) pending.set(key, { ...payload, requestId: crypto.randomUUID() })
    payload = pending.get(key)
  }
  busy.value = true; message.value = ''
  try { await teamsActivity.action(props.eventId, props.module.id, payload); pending.delete(key); await load(); return true }
  catch (err) { message.value = err.message; return false }
  finally { busy.value = false }
}
function award(team, points) { return act({ action: 'award', teamId: team.id, points, reason: reason.value }) }
function editTeam(team) { teamForm.value = team ? { ...team } : { name: '', color: '#3B82F6', capacity: 0 } }
async function saveTeam() { if (await act({ action: 'save_team', team: { ...teamForm.value } })) teamForm.value = null }
function deleteTeam(team) { if (window.confirm(t('teams_module.confirm_delete', { name: team.name }))) act({ action: 'delete_team', teamId: team.id }) }
function reshuffle() { if (window.confirm(t('teams_module.confirm_reshuffle'))) act({ action: 'distribute', reshuffle: true }) }
async function toggleVisibility() {
  if (busy.value || loading.value) return
  busy.value = true; message.value = ''
  try { await teamsActivity.setVisibility(props.eventId, props.module.id, !visibleToPlayers.value); await load() }
  catch (err) { message.value = err.message }
  finally { busy.value = false }
}
function focusPanel(panel) {
  panel?.scrollIntoView({ behavior: window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth', block: 'start' })
  panel?.focus({ preventScroll: true })
}
function manageScores() { focusPanel(scoringPanel.value) }
async function showParticipants() { await nextTick(); focusPanel(participantPanel.value); participantSearch.value?.focus({ preventScroll: true }) }
async function assignToTeam() {
  if (await act({ action: 'assign', participantId: assignParticipantId.value, teamId: assignTeam.value.id })) assignTeam.value = null
}
function reopen() { if (window.confirm(t('teams_module.confirm_reopen'))) act({ action: 'reopen' }) }
function complete() { if (window.confirm(t('teams_module.confirm_complete'))) act({ action: 'complete' }) }
function showProjector() { window.open(`/projector/${props.eventId}`, '_blank', 'noopener') }
</script>
