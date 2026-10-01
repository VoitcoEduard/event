<template>
  <div class="teams-player-shell"><main class="teams-module teams-player">
    <header class="teams-header"><div><button @click="router.push({ name: 'event', query: { eventId } })">← {{ $t('teams_module.back_event') }}</button><p class="teams-eyebrow">{{ $t('teams_module.offline_activity') }}</p><h1>{{ state?.name || $t('teams_module.teams') }}</h1><span v-if="state?.status" class="teams-status">{{ $t(`teams_module.${state.status}`) }}</span></div></header>
    <p v-if="loading" class="teams-notice">{{ $t('teams_module.loading') }}</p><p v-if="disconnected" class="teams-notice">{{ $t('teams_module.disconnected') }}</p><p v-if="message || error" role="alert" class="teams-error">{{ message || error }}</p>
    <p v-if="state && !state.initialized" class="teams-notice">{{ $t('teams_module.waiting_setup') }}</p>
    <form v-if="state?.initialized && !state.joined && state.status !== 'completed'" class="teams-panel" @submit.prevent="join"><h2>{{ $t('teams_module.enter') }}</h2><label>{{ $t('teams_module.your_name') }}<input v-model="name" required maxlength="120" autocomplete="given-name" /></label><button class="primary" :disabled="busy || !name.trim()">{{ $t('teams_module.enter') }}</button></form>
    <p v-if="state?.status === 'completed' && !state.joined" class="teams-notice">{{ $t('teams_module.completed') }}</p>
    <template v-if="state?.joined">
      <section class="teams-panel player-team" :style="{ '--team-color': myTeam?.color || '#43516a' }"><p class="teams-eyebrow">{{ $t('teams_module.your_team') }}</p><h2>{{ myTeam?.name || $t('teams_module.unassigned') }}</h2><p v-if="!myTeam">{{ $t('teams_module.waiting_assignment') }}</p>
        <div class="player-stats"><div><strong>{{ myTeam?.score || 0 }}</strong><span>{{ $t('teams_module.team_score') }}</span></div><div><strong>{{ state.myPoints }}</strong><span>{{ $t('teams_module.my_activity_points') }}</span></div><div><strong>{{ state.totalPoints }}</strong><span>{{ $t('teams_module.event_total') }}</span></div></div>
        <ul v-if="myTeam" class="teams-roster"><li v-for="(person, index) in myTeam.members" :key="index"><span>{{ person.name }}</span></li></ul>
      </section>
      <section v-if="state.status === 'setup' && state.method === 'choice'" class="teams-panel"><h2>{{ $t('teams_module.choose_team') }}</h2><div class="player-choice"><button v-for="team in state.teams" :key="team.id" :class="{ selected: team.id === state.myTeamId }" :style="{ '--team-color': team.color }" :disabled="busy || (team.full && team.id !== state.myTeamId)" @click="choose(team.id)"><strong>{{ team.name }}</strong><span>{{ team.members.length }} {{ $t('teams_module.members') }} · {{ team.full ? $t('teams_module.full') : $t('teams_module.join') }}</span></button></div><button v-if="myTeam" style="margin-top:14px" :disabled="busy" @click="choose('')">{{ $t('teams_module.leave_team') }}</button></section>
      <section class="teams-panel"><h2>{{ $t('teams_module.your_awards') }}</h2><p v-if="!personalAwards.length" class="teams-muted">{{ $t('teams_module.no_awards') }}</p><ul class="teams-roster"><li v-for="award in personalAwards" :key="award.id"><span>{{ award.reason || $t('teams_module.manual_award') }}</span><strong>{{ award.points > 0 ? '+' : '' }}{{ award.points }}</strong></li></ul></section>
    </template>
    <div v-if="celebration" class="teams-celebration" role="status"><strong>{{ celebration.points > 0 ? '+' : '' }}{{ celebration.points }}</strong><p>{{ celebration.reason || $t('teams_module.manual_award') }}</p></div>
  </main></div>
</template>
<script setup>
import { ref, computed, watch, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getUserName, setUserName } from '../utils'
import { useTeamsActivity } from '../composables/useTeamsActivity'
import { teamsActivity } from '../services/teamsActivity'
import { teamAwardTracker } from '../utils/teamsActivity'
import '../styles/teams-activity.css'
const route = useRoute(), router = useRouter()
const eventId = route.params.eventId, moduleId = route.params.moduleId
const { state, loading, disconnected, error } = useTeamsActivity(eventId, moduleId)
const name = ref(getUserName()), busy = ref(false), message = ref(''), celebration = ref(null)
const myTeam = computed(() => state.value?.teams?.find(team => team.id === state.value.myTeamId))
const personalAwards = computed(() => [...(state.value?.awards || [])].filter(award => award.forYou).reverse())
const track = teamAwardTracker()
let timer, attempted = false
watch(state, async value => {
  if (!value?.initialized) return
  const awards = track(value.awards || []).filter(award => award.forYou)
  if (awards.length) {
    const award = awards[awards.length - 1]
    celebration.value = { ...award, points: awards.reduce((sum, item) => sum + item.points, 0) }
    clearTimeout(timer); timer = setTimeout(() => { celebration.value = null }, 4000)
  }
  if (!value.joined && value.status !== 'completed' && name.value && !attempted) { attempted = true; await join() }
})
async function join() {
  if (busy.value) return
  busy.value = true; message.value = ''
  try { state.value = await teamsActivity.join(eventId, moduleId, name.value); setUserName(name.value.trim()) }
  catch (err) { message.value = err.message } finally { busy.value = false }
}
async function choose(teamId) {
  busy.value = true; message.value = ''
  try { state.value = await teamsActivity.choose(eventId, moduleId, teamId) }
  catch (err) { message.value = err.message } finally { busy.value = false }
}
onUnmounted(() => clearTimeout(timer))
</script>
