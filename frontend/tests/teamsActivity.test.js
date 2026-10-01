import test from 'node:test'
import assert from 'node:assert/strict'
import { teamAwardTracker, activityTotals } from '../src/utils/teamsActivity.js'
import { scoreRows } from '../src/utils/eventScores.js'

test('award tracker ignores history and duplicate deliveries after reconnect', () => {
  const track = teamAwardTracker()
  assert.deepEqual(track([{ id: 'old', points: 20 }]), [])
  assert.deepEqual(track([{ id: 'old' }, { id: 'new' }]).map(a => a.id), ['new'])
  assert.deepEqual(track([{ id: 'old' }, { id: 'new' }]), [])
})
test('team and individual totals agree after a move and undo', () => {
  const totals = activityTotals({ awards: [
    { teamId: 'blue', points: 20, recipients: ['anna', 'bob'] },
    { teamId: 'red', points: 15, recipients: ['anna', 'cara'] },
    { teamId: 'blue', points: -20, recipients: ['anna', 'bob'] }
  ] })
  assert.equal(totals.people.get('anna'), 15)
  assert.equal(totals.people.get('bob'), 0)
  assert.equal(totals.teams.get('blue'), 0)
  assert.equal(totals.teams.get('red'), 15)
})
test('score sheet includes activity participants, teams and a separate award breakdown', () => {
  const event = { id: 'event', modules: [{ id: 'quiz', type: 'quiz' }, { id: 'relay', type: 'teams' }, { id: 'build', type: 'teams' }] }
  const scores = { games: [{ moduleId: 'quiz', players: [{ id: 'anna', score: 7 }] }], participants: [{ id: 'anna', displayName: 'Anna' }], teams: [{ id: 'blue', name: 'Blue', moduleId: 'relay' }, { id: 'red', name: 'Red', moduleId: 'build' }], entries: [
    { id: 'teams:a:win', moduleId: 'relay', teamId: 'blue', points: 20 },
    { id: 'teams:a:win:anna', moduleId: 'relay', participantId: 'anna', points: 20 },
    { id: 'teams:b:win', moduleId: 'build', teamId: 'red', points: 15 },
    { id: 'teams:b:win:anna', moduleId: 'build', participantId: 'anna', points: 15 }
  ] }
  const person = scoreRows(event, [], [], scores, 'players')[0]
  assert.equal(person.name, 'Anna'); assert.equal(person.total, 42)
  assert.equal(person.awards.relay, 20); assert.equal(person.awards.build, 15)
  assert.equal(person.manual.relay, undefined)
  assert.equal(scoreRows(event, [], [], scores, 'teams', 'relay')[0].total, 20)
  assert.equal(scoreRows(event, [], [], scores, 'teams', 'relay').length, 1)
})
