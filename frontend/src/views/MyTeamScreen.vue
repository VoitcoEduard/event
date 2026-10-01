<template>
  <div class="my-team-screen">
    <div class="header">
      <button class="back-btn" @click="goBack">{{ $t('my_team.go_back') }}</button>
      <h1 class="title">{{ $t('my_team.title') }}</h1>
    </div>

    <div v-if="team" class="team-display">
      <div class="team-color-bar" :style="{ backgroundColor: team.color }"></div>
      <div class="team-info">
        <h2 class="team-name">{{ team.name }}</h2>
        <p class="members-count">{{ $t('my_team.members_count', { count: members.length }) }}</p>
      </div>

      <div class="members-list">
        <div v-for="member in members" :key="member.id" class="member-item">
          <span class="member-name">{{ member.displayName }}</span>
          <span v-if="member.id === userId" class="you-badge">{{ $t('my_team.you') }}</span>
        </div>
      </div>

      <button class="btn-secondary" @click="goToTeamSelection">{{ $t('my_team.change_team') }}</button>
    </div>

    <div v-else class="no-team">
      <p>{{ $t('my_team.no_team') }}</p>
      <button class="btn-primary" @click="goToTeamSelection">{{ $t('team_selection.title') }}</button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { teamsService } from '../services/teams'
import { getUserId, getAuthHeaders } from '../utils'

const router = useRouter()
const route = useRoute()

const teams = ref([])
const participants = ref([])
const eventId = computed(() => route.query.eventId || route.params.id || route.params.eventId)
const userId = getUserId()

const team = computed(() => {
  const user = participants.value.find(p => p.id === userId)
  if (!user?.teamId) return null
  return teams.value.find(t => t.id === user.teamId)
})

const members = computed(() => {
  if (!team.value) return []
  return participants.value.filter(p => p.teamId === team.value.id)
})

const loadData = async () => {
  try {
    const [teamsData, participantsData] = await Promise.all([
      teamsService.getTeams(eventId.value),
      fetch(`${import.meta.env.VITE_API_URL || ''}/events/${eventId.value}/participants`, {
        headers: getAuthHeaders()
      }).then(r => r.json())
    ])
    teams.value = teamsData
    participants.value = Array.isArray(participantsData) ? participantsData : []
  } catch (error) {
    console.error('Failed to load data:', error)
  }
}

const goToTeamSelection = () => {
  router.push({ name: 'team-selection', query: { eventId: eventId.value } })
}

const goBack = () => {
  router.back()
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.my-team-screen {
  min-height: 100vh;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
  color: #fff;
  padding: 20px;
}

.header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 32px;
}

.back-btn {
  background: transparent;
  border: 1px solid #3a3a40;
  color: #fff;
  padding: 8px 16px;
  border-radius: 8px;
  cursor: pointer;
}

.title {
  font-size: 28px;
  font-weight: 700;
}

.team-display {
  max-width: 500px;
  margin: 0 auto;
  background: #1a1a2e;
  border-radius: 20px;
  padding: 32px;
  text-align: center;
}

.team-color-bar {
  height: 12px;
  border-radius: 6px;
  margin-bottom: 24px;
}

.team-info {
  margin-bottom: 32px;
}

.team-name {
  font-size: 36px;
  font-weight: 700;
  margin-bottom: 8px;
}

.members-count {
  color: #888;
  font-size: 16px;
}

.members-list {
  margin-bottom: 32px;
  text-align: left;
}

.member-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #252530;
  border-radius: 8px;
  margin-bottom: 8px;
}

.member-name {
  font-weight: 500;
}

.you-badge {
  background: #3b82f6;
  color: #fff;
  padding: 2px 10px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 600;
}

.no-team {
  text-align: center;
  padding: 60px 20px;
}

.no-team p {
  color: #888;
  margin-bottom: 24px;
  font-size: 18px;
}

.btn-primary {
  background: #3b82f6;
  color: #fff;
  border: none;
  padding: 12px 24px;
  border-radius: 8px;
  cursor: pointer;
  font-weight: 600;
}

.btn-secondary {
  background: #2a2a30;
  color: #fff;
  border: none;
  padding: 12px 24px;
  border-radius: 8px;
  cursor: pointer;
  font-weight: 500;
}
</style>
