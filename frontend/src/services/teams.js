import { API_BASE_URL, getAuthHeaders } from '../utils'

const getTeams = async (eventId) => {
  const response = await fetch(`${API_BASE_URL}/events/${eventId}/teams`, {
    headers: getAuthHeaders(),
  })
  if (!response.ok) throw new Error('Failed to fetch teams')
  return response.json()
}

const createTeam = async (eventId, teamData) => {
  const response = await fetch(`${API_BASE_URL}/events/${eventId}/teams`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeaders(),
    },
    body: JSON.stringify(teamData),
  })
  if (!response.ok) throw new Error('Failed to create team')
  return response.json()
}

const updateTeam = async (eventId, teamId, teamData) => {
  const response = await fetch(`${API_BASE_URL}/events/${eventId}/teams/${teamId}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeaders(),
    },
    body: JSON.stringify(teamData),
  })
  if (!response.ok) throw new Error('Failed to update team')
  return response.json()
}

const deleteTeam = async (eventId, teamId) => {
  const response = await fetch(`${API_BASE_URL}/events/${eventId}/teams/${teamId}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  })
  if (!response.ok) throw new Error('Failed to delete team')
  return response.json()
}

const assignParticipant = async (eventId, participantId, teamId) => {
  const response = await fetch(`${API_BASE_URL}/events/${eventId}/teams/assign`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeaders(),
    },
    body: JSON.stringify({ participantId, teamId }),
  })
  if (!response.ok) throw new Error('Failed to assign participant')
  return response.json()
}

const autoAssignParticipants = async (eventId) => {
  const response = await fetch(`${API_BASE_URL}/events/${eventId}/teams/auto-assign`, {
    method: 'POST',
    headers: getAuthHeaders(),
  })
  if (!response.ok) throw new Error('Failed to auto-assign participants')
  return response.json()
}

export const teamsService = {
  getTeams,
  createTeam,
  updateTeam,
  deleteTeam,
  assignParticipant,
  autoAssignParticipants,
}