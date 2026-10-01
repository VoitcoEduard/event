<template>
  <section class="teams-projector">
    <header><div><p class="projector-kicker">{{ $t('teams_module.offline_activity') }}</p><h1>{{ state?.name || $t('teams_module.teams') }}</h1><p v-if="state?.status">{{ $t(`teams_module.${state.status}`) }} · {{ state.entrantCount }} {{ $t('teams_module.people') }}</p></div><div v-if="state?.status === 'setup'" class="teams-projector-qr"><QRCodeVue :value="joinUrl" :size="160" level="H" /><span>{{ $t('teams_module.join') }}</span></div></header>
    <p v-if="loading || !state?.initialized" class="projector-notice">{{ $t('teams_module.waiting_setup') }}</p><p v-if="disconnected" class="projector-notice">{{ $t('teams_module.disconnected') }}</p><p v-if="error" role="alert">{{ error }}</p>
    <div class="projector-team-grid"><article v-for="team in pageTeams" :key="team.id" :style="{ '--team-color': team.color }"><div class="projector-team-title"><span v-if="state.status !== 'setup'" class="rank">{{ team.rank }}</span><h2>{{ team.name }}</h2><strong>{{ team.score }}</strong></div><p class="projector-members" v-if="state.status === 'setup'">{{ team.members.slice(0, 12).map(person => person.name).join(' · ') }}<span v-if="team.members.length > 12"> · +{{ team.members.length - 12 }}</span></p><p v-else>{{ team.members.length }} {{ $t('teams_module.members') }}</p></article></div>
    <footer><span v-if="state?.unassigned?.length">{{ state.unassigned.length }} {{ $t('teams_module.waiting_assignment') }}</span><span v-if="pageCount > 1">{{ page + 1 }} / {{ pageCount }}</span></footer>
    <div v-if="celebration" class="projector-award" role="status"><strong>{{ celebration.points > 0 ? '+' : '' }}{{ celebration.points }}</strong><h2>{{ teamName(celebration.teamId) }}</h2><p>{{ celebration.reason || $t('teams_module.manual_award') }}</p></div>
  </section>
</template>
<script setup>
import { computed, ref, watch, onMounted, onUnmounted } from 'vue'
import QRCodeVue from 'qrcode.vue'
import { useTeamsActivity } from '../composables/useTeamsActivity'
import { teamAwardTracker } from '../utils/teamsActivity'
const props = defineProps({ eventId: String, moduleId: String })
const { state, loading, disconnected, error } = useTeamsActivity(props.eventId, props.moduleId, 'projector')
const page = ref(0), celebration = ref(null)
const pageCount = computed(() => Math.max(1, Math.ceil((state.value?.teams?.length || 0) / 8)))
const pageTeams = computed(() => (state.value?.teams || []).slice(page.value * 8, page.value * 8 + 8))
const joinUrl = `${window.location.origin}/teams/${props.eventId}/${props.moduleId}/player`
const teamName = id => state.value?.teams?.find(team => team.id === id)?.name || ''
const track = teamAwardTracker()
let cycle, timer
watch(state, value => {
  if (!value?.initialized) return
  if (page.value >= pageCount.value) page.value = 0
  const awards = track(value.awards || [])
  if (awards.length) { celebration.value = awards[awards.length - 1]; clearTimeout(timer); timer = setTimeout(() => { celebration.value = null }, 4500) }
})
onMounted(() => { cycle = setInterval(() => { page.value = (page.value + 1) % pageCount.value }, 10000) })
onUnmounted(() => { clearInterval(cycle); clearTimeout(timer) })
</script>
<style scoped>
.teams-projector{min-height:100vh;padding:3vh 4vw;background:radial-gradient(ellipse at top,#233c60,#0c1220 65%);color:#f7faff;box-sizing:border-box}.teams-projector header{display:flex;justify-content:space-between;gap:4vw;align-items:center;margin-bottom:3vh}.projector-kicker{text-transform:uppercase;letter-spacing:3px;font-size:clamp(14px,1.2vw,24px);color:#9dc3f3;margin:0}.teams-projector h1{font-size:clamp(32px,3.3vw,70px);margin:10px 0;overflow-wrap:anywhere}.teams-projector header p{font-size:clamp(18px,1.5vw,30px)}.teams-projector-qr{background:white;padding:14px;border-radius:16px;display:grid;gap:8px;text-align:center;color:#0c1220}.projector-team-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:1.7vh 2vw}.projector-team-grid article{border-left:10px solid var(--team-color);background:#1a2941;border-radius:14px;padding:1.5vh 1.4vw;min-height:12vh;box-sizing:border-box}.projector-team-title{display:flex;gap:1vw;align-items:center}.projector-team-title h2{font-size:clamp(20px,2.2vw,44px);flex:1;overflow-wrap:anywhere;margin:0;line-height:1.1}.projector-team-title strong{font-size:clamp(32px,3.4vw,65px);font-variant-numeric:tabular-nums}.rank{font-size:clamp(22px,2vw,40px);color:#9cc2ef;min-width:1.5em}.projector-team-grid p{color:#b8cce7;font-size:clamp(14px,1.1vw,22px);margin:8px 0 0;overflow-wrap:anywhere}.teams-projector footer{display:flex;justify-content:space-between;color:#b8cce7;font-size:1.4vw;margin-top:2vh}.projector-award{position:fixed;inset:24% 25%;border:2px solid #94c4ff;background:#285fafd9;backdrop-filter:blur(12px);padding:30px;border-radius:24px;text-align:center;display:grid;align-content:center;box-shadow:0 15px 90px #0009}.projector-award strong{font-size:8vw}.projector-award h2{font-size:3vw;overflow-wrap:anywhere;margin:0}.projector-award p{font-size:1.6vw}.projector-notice{font-size:24px}
</style>
