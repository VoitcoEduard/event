<template>
  <div class="teams-admin-screen">
    <div class="header">
      <div class="header-content">
        <div class="header-left">
          <button class="back-btn" @click="goBack">{{ $t('teams_admin.back_btn') }}</button>
          <h1 class="page-title">{{ moduleName || $t('teams_admin.title') }}</h1>
        </div>
        <div class="header-right">
          <span class="connected-badge">{{ $t('teams_admin.connected_users', { count: connectedUsers }) }}</span>
          <button class="btn-primary" @click="showCreateModal = true">{{ $t('teams_admin.create_team') }}</button>
          <button v-if="teams.length > 0" class="btn-danger" @click="endTeams">{{ $t('teams_admin.end_teams') }}</button>
        </div>
      </div>
    </div>

    <div class="content-grid">
      <div class="card teams-card">
        <div class="card-header">
          <p class="card-title">{{ $t('teams_admin.teams_header') }}</p>
        </div>
        <div class="teams-list">
          <div v-if="teams.length === 0" class="empty-state">
            {{ $t('teams_admin.no_teams') }}
          </div>
          <div v-for="team in teams" :key="team.id" class="team-item">
            <div class="team-color" :style="{ backgroundColor: team.color }"></div>
            <div class="team-info">
              <span class="team-name">{{ team.name }}</span>
              <span class="team-members">{{ $t('teams_admin.members_count', { count: getTeamMemberCount(team.id) }) }}</span>
            </div>
            <div class="team-actions">
              <button class="action-btn" @click="editTeam(team)">✏️</button>
              <button class="action-btn delete" @click="confirmDeleteTeam(team)">🗑️</button>
            </div>
          </div>
        </div>
      </div>

      <div class="card participants-card">
        <div class="card-header">
          <p class="card-title">{{ $t('teams_admin.participants_header') }}</p>
          <button v-if="unassignedParticipants.length > 0 && teams.length > 0" class="btn-secondary" @click="autoAssign">
            {{ $t('teams_admin.auto_assign_btn') }}
          </button>
        </div>
        <div class="participants-list">
          <div v-if="participants.length === 0" class="empty-state">
            {{ $t('teams_admin.no_participants') }}
          </div>
          <template v-for="team in teams" :key="'team-' + team.id">
            <div class="team-section">
              <div class="team-section-header">
                <div class="team-color-sm" :style="{ backgroundColor: team.color }"></div>
                <span>{{ team.name }}</span>
              </div>
              <div class="team-members-list">
                <div v-for="participant in getTeamParticipants(team.id)" :key="participant.id" class="participant-item">
                  <span class="participant-name">{{ participant.displayName }}</span>
                  <button class="action-btn small" @click="unassignParticipant(participant)">↩️</button>
                </div>
              </div>
            </div>
          </template>
          <div v-if="unassignedParticipants.length > 0" class="unassigned-section">
            <div class="section-header">{{ $t('teams_admin.unassigned_section') }}</div>
            <div v-for="participant in unassignedParticipants" :key="participant.id" class="participant-item unassigned">
              <span class="participant-name">{{ participant.displayName }}</span>
              <div class="assign-dropdown">
                <select @change="assignToTeam(participant, $event.target.value)">
                  <option value="">{{ $t('teams_admin.assign_to_team') }}</option>
                  <option v-for="team in teams" :key="team.id" :value="team.id">
                    {{ team.name }}
                  </option>
                </select>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="showCreateModal || showEditModal" class="modal-overlay" @click.self="closeModals">
      <div class="modal">
        <h2 class="modal-title">{{ showEditModal ? $t('teams_admin.edit_team') : $t('teams_admin.create_team') }}</h2>
        <div class="form-group">
          <label>{{ $t('teams_admin.team_name_label') }}</label>
          <input v-model="teamForm.name" type="text" class="form-input" />
        </div>
        <div class="form-group">
          <label>{{ $t('teams_admin.team_color_label') }}</label>
          <div class="color-options">
            <div v-for="color in colorOptions" :key="color" class="color-option"
              :class="{ selected: teamForm.color === color }"
              :style="{ backgroundColor: color }"
              @click="teamForm.color = color">
            </div>
          </div>
        </div>
        <div class="modal-actions">
          <button class="btn-secondary" @click="closeModals">{{ $t('common.cancel') || 'Cancel' }}</button>
          <button class="btn-primary" @click="saveTeam">{{ showEditModal ? 'Update' : 'Create' }}</button>
        </div>
      </div>
    </div>

    <div v-if="showDeleteModal" class="modal-overlay" @click.self="showDeleteModal = false">
      <div class="modal">
        <h2 class="modal-title">{{ $t('teams_admin.delete_team') }}</h2>
        <p>{{ deleteConfirmMessage }}</p>
        <div class="modal-actions">
          <button class="btn-secondary" @click="showDeleteModal = false">{{ $t('common.cancel') || 'Cancel' }}</button>
          <button class="btn-danger" @click="deleteTeam">{{ $t('common.delete') || 'Delete' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { teamsService } from '../services/teams'
import { quizWS } from '../services/websocket'
import { getUserId, getAuthHeaders } from '../utils'

const route = useRoute()
const router = useRouter()

const teams = ref([])
const participants = ref([])
const connectedUsers = ref(0)
const showCreateModal = ref(false)
const showEditModal = ref(false)
const showDeleteModal = ref(false)
const selectedTeam = ref(null)
const deleteConfirmMessage = ref('')

const teamForm = ref({
  name: '',
  color: '#3B82F6',
})

const colorOptions = [
  '#EF4444', '#F97316', '#F59E0B', '#EAB308',
  '#84CC16', '#22C55E', '#10B981', '#14B8A6',
  '#06B6D4', '#0EA5E9', '#3B82F6', '#6366F1',
  '#8B5CF6', '#A855F7', '#D946EF', '#EC4899',
]

const eventId = computed(() => route.params.id || route.query.eventId)
const moduleId = computed(() => route.query.moduleId || '')
const moduleName = computed(() => route.query.moduleName || '')

const getTeamParticipants = (teamId) => {
  return participants.value.filter(p => p.teamId === teamId)
}

const getTeamMemberCount = (teamId) => {
  return getTeamParticipants(teamId).length
}

const unassignedParticipants = computed(() => {
  return participants.value.filter(p => !p.teamId)
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

const saveTeam = async () => {
  try {
    if (showEditModal.value && selectedTeam.value) {
      await teamsService.updateTeam(eventId.value, selectedTeam.value.id, {
        name: teamForm.value.name,
        color: teamForm.value.color,
      })
    } else {
      await teamsService.createTeam(eventId.value, {
        name: teamForm.value.name,
        color: teamForm.value.color,
      })
    }
    closeModals()
    await loadData()
  } catch (error) {
    console.error('Failed to save team:', error)
  }
}

const editTeam = (team) => {
  selectedTeam.value = team
  teamForm.value.name = team.name
  teamForm.value.color = team.color
  showEditModal.value = true
}

const confirmDeleteTeam = (team) => {
  selectedTeam.value = team
  const memberCount = getTeamMemberCount(team.id)
  if (memberCount > 0) {
    deleteConfirmMessage.value = `This team has ${memberCount} members. They will become unassigned. Continue?`
  } else {
    deleteConfirmMessage.value = 'Delete this team?'
  }
  showDeleteModal.value = true
}

const deleteTeam = async () => {
  try {
    await teamsService.deleteTeam(eventId.value, selectedTeam.value.id)
    showDeleteModal.value = false
    selectedTeam.value = null
    await loadData()
  } catch (error) {
    console.error('Failed to delete team:', error)
  }
}

const assignToTeam = async (participant, teamId) => {
  if (!teamId) return
  try {
    await teamsService.assignParticipant(eventId.value, participant.id, teamId)
    await loadData()
  } catch (error) {
    console.error('Failed to assign participant:', error)
  }
}

const unassignParticipant = async (participant) => {
  try {
    await teamsService.assignParticipant(eventId.value, participant.id, '')
    await loadData()
  } catch (error) {
    console.error('Failed to unassign participant:', error)
  }
}

const autoAssign = async () => {
  try {
    await teamsService.autoAssignParticipants(eventId.value)
    await loadData()
  } catch (error) {
    console.error('Failed to auto-assign:', error)
  }
}

const closeModals = () => {
  showCreateModal.value = false
  showEditModal.value = false
  selectedTeam.value = null
  teamForm.value = { name: '', color: '#3B82F6' }
}

const goBack = () => {
  router.back()
}

const endTeams = () => {
  if (!confirm('End teams module? This will save the final team state to the data tab.')) return
  quizWS.send('host:end_teams', {
    eventId: eventId.value,
    moduleId: moduleId.value,
    moduleName: moduleName.value
  })
}

const setupWebSocket = () => {
  const onWSOpen = () => {
    quizWS.send('teams:admin_join', { eventId: eventId.value })
  }

  quizWS.connect(null, null, false, false, eventId.value)

  if (quizWS.isConnected) {
    onWSOpen()
  } else {
    quizWS.on('open', onWSOpen)
  }

  quizWS.on('teams:admin_joined', (data) => {
    connectedUsers.value++
  })

  quizWS.on('team:created', () => loadData())
  quizWS.on('team:updated', () => loadData())
  quizWS.on('team:deleted', () => loadData())
  quizWS.on('participant:assigned', () => loadData())
  quizWS.on('team:auto_assigned', () => loadData())
  quizWS.on('participant:joined', () => loadData())
  quizWS.on('disconnect', () => {
    connectedUsers.value = Math.max(0, connectedUsers.value - 1)
  })
}

onMounted(() => {
  loadData()
  setupWebSocket()
})

onUnmounted(() => {
  quizWS.off('open', onWSOpen)
  quizWS.off('teams:admin_joined')
  quizWS.off('team:created')
  quizWS.off('team:updated')
  quizWS.off('team:deleted')
  quizWS.off('participant:assigned')
  quizWS.off('team:auto_assigned')
  quizWS.off('participant:joined')
  quizWS.off('disconnect')
})
</script>

<style scoped>
.teams-admin-screen {
  min-height: 100vh;
  background: #0f0f12;
  color: #fff;
}

.header {
  background: #1a1a1f;
  padding: 16px 24px;
  border-bottom: 1px solid #2a2a30;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  max-width: 1200px;
  margin: 0 auto;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.back-btn {
  background: transparent;
  border: 1px solid #3a3a40;
  color: #fff;
  padding: 8px 16px;
  border-radius: 8px;
  cursor: pointer;
}

.page-title {
  font-size: 24px;
  font-weight: 600;
}

.connected-badge {
  background: #22c55e;
  color: #000;
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 14px;
  font-weight: 500;
}

.btn-primary {
  background: #3b82f6;
  color: #fff;
  border: none;
  padding: 10px 20px;
  border-radius: 8px;
  cursor: pointer;
  font-weight: 500;
}

.btn-secondary {
  background: #2a2a30;
  color: #fff;
  border: none;
  padding: 10px 20px;
  border-radius: 8px;
  cursor: pointer;
}

.btn-danger {
  background: #ef4444;
  color: #fff;
  border: none;
  padding: 10px 20px;
  border-radius: 8px;
  cursor: pointer;
}

.content-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
}

.card {
  background: #1a1a1f;
  border-radius: 12px;
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #2a2a30;
}

.card-title {
  font-size: 18px;
  font-weight: 600;
}

.teams-list, .participants-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.empty-state {
  text-align: center;
  color: #666;
  padding: 40px 20px;
}

.team-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: #252530;
  border-radius: 8px;
}

.team-color {
  width: 16px;
  height: 40px;
  border-radius: 4px;
}

.team-color-sm {
  width: 12px;
  height: 12px;
  border-radius: 3px;
}

.team-info {
  flex: 1;
}

.team-name {
  font-weight: 500;
  display: block;
}

.team-members {
  font-size: 12px;
  color: #888;
}

.team-actions {
  display: flex;
  gap: 8px;
}

.action-btn {
  background: transparent;
  border: none;
  color: #888;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
}

.action-btn:hover {
  background: #3a3a40;
  color: #fff;
}

.action-btn.delete:hover {
  background: #ef4444;
}

.action-btn.small {
  padding: 2px 6px;
  font-size: 12px;
}

.participant-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  background: #252530;
  border-radius: 8px;
}

.participant-item.unassigned {
  background: #252530;
  border: 1px dashed #3a3a40;
}

.participant-name {
  font-weight: 500;
}

.assign-dropdown select {
  background: #3a3a40;
  color: #fff;
  border: none;
  padding: 6px 12px;
  border-radius: 6px;
  cursor: pointer;
}

.team-section {
  margin-bottom: 16px;
}

.team-section-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 0;
  color: #888;
  font-size: 14px;
}

.team-members-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding-left: 20px;
}

.unassigned-section {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px dashed #3a3a40;
}

.section-header {
  color: #888;
  font-size: 14px;
  margin-bottom: 8px;
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: #1a1a1f;
  border-radius: 16px;
  padding: 24px;
  width: 400px;
  max-width: 90%;
}

.modal-title {
  font-size: 20px;
  font-weight: 600;
  margin-bottom: 20px;
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  color: #888;
  font-size: 14px;
}

.form-input {
  width: 100%;
  padding: 12px;
  background: #252530;
  border: 1px solid #3a3a40;
  border-radius: 8px;
  color: #fff;
  font-size: 16px;
}

.color-options {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.color-option {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  cursor: pointer;
  border: 2px solid transparent;
}

.color-option.selected {
  border-color: #fff;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 24px;
}
</style>
