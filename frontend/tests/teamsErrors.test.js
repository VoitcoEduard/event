import test from 'node:test'
import assert from 'node:assert/strict'
import { localizeTeamsError } from '../src/utils/teamsErrors.js'

test('localizes known Teams API errors', () => {
  const translate = key => ({
    'teams_module.errors.assign_participant_before_start': 'localized message'
  })[key]

  assert.equal(
    localizeTeamsError('Assign at least one participant before starting', translate),
    'localized message'
  )
})

test('keeps unknown Teams API errors intact', () => {
  assert.equal(localizeTeamsError('Unexpected error', () => ''), 'Unexpected error')
})
