import test from 'node:test'
import assert from 'node:assert/strict'
import { scoreRows, manualID, allowsManual, isGame, parsePoints } from '../src/utils/eventScores.js'
const event = { id: 'e', modules: [{ id: 'q', type: 'quiz' }, { id: 'w', type: 'wrong_pictures' }] }
const participants = [{ id: 'p', displayName: 'Player' }, { id: 'new', displayName: 'New player' }]
const teams = [{ id: 't', name: 'Team' }]
const scores = {
 games: [{ moduleId: 'q', players: [{ id: 'p', score: 20 }] }, { moduleId: 'w', players: [{ id: 'p', score: 7 }] }],
 entries: [
  { id: manualID('e', 'q', 'players', 'p'), participantId: 'p', moduleId: 'q', points: 3.5 },
  { id: manualID('e', 'q', 'teams', 't'), teamId: 't', moduleId: 'q', points: 100 }
 ]
}
test('totals include latest game results, Wrong Pictures and player manual points without team duplication', () => {
 const rows = scoreRows(event, participants, teams, scores, 'players')
 assert.equal(rows.find(r => r.id === 'p').total, 30.5)
 assert.equal(rows.find(r => r.id === 'new').total, 0)
 assert.equal(rows.find(r => r.id === 'new').registered, true)
 assert.equal(scoreRows(event, participants, teams, scores, 'teams')[0].total, 100)
 assert.equal(scoreRows(event, participants, teams, scores, 'players', 'w').find(r => r.id === 'p').total, 7)
})
test('manual save replaces rather than accumulates and reload recalculates totals', () => {
 const updated = structuredClone(scores)
 updated.entries[0].points = 5
 updated.games[0].players[0].score = 40
 assert.equal(scoreRows(event, participants, teams, updated, 'players').find(r => r.id === 'p').total, 52)
})
test('manual target is configurable independently of game type', () => {
 assert.equal(isGame(event.modules[1]), true)
 assert.equal(allowsManual({ data: { manualScoreTarget: 'teams' } }, 'players'), false)
 assert.equal(allowsManual({ data: { manualScoreTarget: 'teams' } }, 'teams'), true)
 assert.equal(allowsManual({}, 'players'), true)
})
test('blank and nonfinite points cannot silently become zero; game scores require integers', () => {
 for (const value of ['', ' ', 'NaN', 'Infinity', 'abc']) assert.throws(() => parsePoints(value))
 assert.throws(() => parsePoints('2.5', true))
 assert.equal(parsePoints('0'), 0)
 assert.equal(parsePoints('-1.5'), -1.5)
})
