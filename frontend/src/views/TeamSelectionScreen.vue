<template>
  <div class="team-selection-screen">
    <div class="header">
      <h1 class="title">{{ $t('team_selection.title') }}</h1>
      <p class="subtitle">{{ $t('team_selection.subtitle') }}</p>
    </div>

    <div class="teams-grid">
      <div v-if="teams.length === 0" class="loading">{{ $t('common.loading') }}</div>
      <div
        v-for="team in teams"
        :key="team.id"
        class="team-card"
        :style="{ borderColor: team.color }"
        @click="joinTeam(team)"
      >
        <div class="team-color-bar" :style="{ backgroundColor: team.color }"></div>
        <div class="team-info">
          <h3 class="team-name">{{ team.name }}</h3>
          <p class="team-members">{{ $t('teams_admin.members_count', { count: getMemberCount(team.id) }) }}</p>
        </div>
        <button class="join-btn" :style="{ backgroundColor: team.color }">
          {{ $t('team_selection.join_team') }}
        </button>
      </div>
    </div>

    <div v-if="currentTeam" class="current-team-banner">
      <p>{{ $t('team_selection.already_in_team') }}</p>
      <button class="btn-secondary" @click="goToMyTeam">{{ $t('team_selection.go_to_team') }}</button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { teamsService } from '../services/teams'
import { quizWS } from '../services/websocket'
import { getUserId, getAuthHeaders } from '../utils'

const router = useRouter()
const route = useRoute()

const teams = ref([])
const participants = ref([])
const currentTeam = ref(null)
const eventId = computed(() => route.query.eventId || route.params.id || route.params.eventId)

const getMemberCount = (teamId) => {
  return participants.value.filter(p => p.teamId === teamId).length
}

const loadData = async () => {
  try {
    const userId = getUserId()
    const [teamsData, participantsData] = await Promise.all([
      teamsService.getTeams(eventId.value),
      fetch(`${import.meta.env.VITE_API_URL || ''}/events/${eventId.value}/participants`, {
        headers: getAuthHeaders()
      }).then(r => r.json())
    ])
    teams.value = teamsData
    participants.value = Array.isArray(participantsData) ? participantsData : []

    const user = participants.value.find(p => p.id === userId)
    if (user?.teamId) {
      currentTeam.value = teams.value.find(t => t.id === user.teamId)
    }
  } catch (error) {
    console.error('Failed to load data:', error)
  }
}

const joinTeam = async (team) => {
  try {
    const userId = getUserId()
    await teamsService.assignParticipant(eventId.value, userId, team.id)
    currentTeam.value = team
    router.push({ name: 'my-team', query: { eventId: eventId.value } })
  } catch (error) {
    console.error('Failed to join team:', error)
  }
}

const goToMyTeam = () => {
  router.push({ name: 'my-team', query: { eventId: eventId.value } })
}

onMounted(() => {
  quizWS.connect(null, null, false, false, eventId.value)
  quizWS.send('participant:join', {
    eventId: eventId.value,
    userId: getUserId(),
    displayName: localStorage.getItem('userName') || 'User',
  })

  quizWS.on('teams:available', (data) => {
    teams.value = data.teams
  })

  loadData()
})
</script>

<style scoped>
.team-selection-screen {
  min-height: 100vh;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
  color: #fff;
  padding: 40px 20px;
}

.header {
  text-align: center;
  margin-bottom: 40px;
}

.title {
  font-size: 32px;
  font-weight: 700;
  margin-bottom: 8px;
}

.subtitle {
  color: #888;
  font-size: 18px;
}

.teams-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 20px;
  max-width: 900px;
  margin: 0 auto;
}

.loading {
  text-align: center;
  color: #888;
  grid-column: 1 / -1;
}

.team-card {
  background: #1a1a2e;
  border-radius: 16px;
  padding: 24px;
  border: 2px solid transparent;
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;
}

.team-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.4);
}

.team-color-bar {
  height: 8px;
  border-radius: 4px;
  margin-bottom: 16px;
}

.team-info {
  margin-bottom: 20px;
}

.team-name {
  font-size: 24px;
  font-weight: 600;
  margin-bottom: 4px;
}

.team-members {
  color: #888;
  font-size: 14px;
}

.join-btn {
  width: 100%;
  padding: 12px;
  border: none;
  border-radius: 8px;
  color: #fff;
  font-weight: 600;
  font-size: 16px;
  cursor: pointer;
  transition: opacity 0.2s;
}

.join-btn:hover {
  opacity: 0.9;
}

.current-team-banner {
  position: fixed;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  background: #252530;
  padding: 16px 24px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

.current-team-banner p {
  color: #888;
}

.btn-secondary {
  background: #3b82f6;
  color: #fff;
  border: none;
  padding: 10px 20px;
  border-radius: 8px;
  cursor: pointer;
}
</style>
