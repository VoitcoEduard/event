export function teamAwardTracker() {
  const seen = new Set()
  let initialized = false
  return awards => {
    const newAwards = initialized ? awards.filter(award => !seen.has(award.id)) : []
    for (const award of awards) seen.add(award.id)
    initialized = true
    return newAwards
  }
}
export function activityTotals(activity) {
  const teams = new Map(), people = new Map()
  for (const award of activity?.awards || []) {
    teams.set(award.teamId, (teams.get(award.teamId) || 0) + award.points)
    for (const id of award.recipients || []) people.set(id, (people.get(id) || 0) + award.points)
  }
  return { teams, people }
}
export const defaultTeamsConfig = () => ({ sourceModuleId: '', method: 'random', teamCount: 0, targetSize: 0, capacity: 0, presets: [1, 5, 10] })
