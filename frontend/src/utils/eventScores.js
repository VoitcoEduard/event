export const isGame = module => ['quiz', 'wrong_pictures'].includes(module.type)
export const manualTarget = module => module?.data?.manualScoreTarget || 'both'
export const allowsManual = (module, target) => ['both', target].includes(manualTarget(module))
export const manualID = (eventId, moduleId, target, id) => `manual:${JSON.stringify([eventId, moduleId, target, id])}`

export function scoreRows(event, participants, teams, scores, target, moduleId = 'unified') {
  const rows = new Map((target === 'players' ? participants : teams).map(item => [item.id, {
    id: item.id, name: item.displayName || item.name || item.id, registered: true, game: {}, manual: {}, total: 0
  }]))
  const modules = new Set((event.modules || []).filter(m => moduleId === 'unified' || m.id === moduleId).map(m => m.id))
  if (target === 'players') for (const game of scores.games || []) {
    if (!modules.has(game.moduleId)) continue
    for (const player of game.players || []) {
      if (!rows.has(player.id)) rows.set(player.id, { id: player.id, name: player.nickname || player.id, registered: false, game: {}, manual: {}, total: 0 })
      rows.get(player.id).game[game.moduleId] = player.score
      rows.get(player.id).total += player.score
    }
  }
  for (const entry of scores.entries || []) {
    if (!modules.has(entry.moduleId)) continue
    const row = rows.get(target === 'players' ? entry.participantId : entry.teamId)
    if (!row) continue
    row.total += entry.points
    if (entry.id === manualID(event.id, entry.moduleId, target, row.id)) row.manual[entry.moduleId] = entry.points
  }
  return [...rows.values()].sort((a, b) => a.name.localeCompare(b.name))
}

export function parsePoints(value, integer = false) {
  if (String(value).trim() === '') throw new Error('Enter points; use 0 to clear a contribution.')
  const points = Number(value)
  if (!Number.isFinite(points) || (integer && !Number.isSafeInteger(points))) throw new Error(integer ? 'Game scores must be whole numbers.' : 'Enter a valid number of points.')
  return points
}
