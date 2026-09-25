import test from 'node:test'
import assert from 'node:assert/strict'
import { calculateDistance, calculateGuessScore, calculateTotalScore } from '../src/utils/wrongPicturesScoring.js'

const target = (id, xPercent, yPercent, radius = 5) => ({
  id,
  xPercent,
  yPercent,
  radius,
  label: id
})

const config = {
  tolerancePercent: 2,
  maxPoints: 100
}

test('calculates percentage distance between two points', () => {
  assert.equal(calculateDistance(0, 0, 3, 4), 5)
})

test('awards points inside the target radius and tolerance', () => {
  assert.equal(calculateGuessScore({ xPercent: 50, yPercent: 50 }, target('one', 50, 50, 5), 2, 100), 100)
  assert.equal(calculateGuessScore({ xPercent: 57, yPercent: 50 }, target('one', 50, 50, 5), 2, 100), 100)
  assert.equal(calculateGuessScore({ xPercent: 58, yPercent: 50 }, target('one', 50, 50, 5), 2, 100), 0)
})

test('marks each submitted guess and records its score', () => {
  const result = calculateTotalScore([
    { xPercent: 50, yPercent: 50, isPlaced: true },
    { xPercent: 85, yPercent: 15, isPlaced: true }
  ], [
    target('first', 50, 50, 5),
    target('second', 90, 10, 5)
  ], config)

  assert.equal(result.totalScore, 100)
  assert.equal(result.scoredGuesses[0].isCorrect, true)
  assert.equal(result.scoredGuesses[0].targetId, 'first')
  assert.equal(result.scoredGuesses[0].scoreEarned, 100)

  assert.equal(result.scoredGuesses[1].isCorrect, false)
  assert.equal(result.scoredGuesses[1].targetId, null)
  assert.equal(result.scoredGuesses[1].scoreEarned, 0)
})

test('uses the closest accepted target when accepted areas overlap', () => {
  const result = calculateTotalScore([
    { xPercent: 20, yPercent: 50, isPlaced: true }
  ], [
    target('farther', 30, 50, 10),
    target('closer', 22, 50, 10)
  ], { tolerancePercent: 0, maxPoints: 50 })

  assert.equal(result.scoredGuesses[0].targetId, 'closer')
  assert.equal(result.totalScore, 50)
})

test('allows multiple correct guesses for the same target', () => {
  const result = calculateTotalScore([
    { xPercent: 49, yPercent: 50, isPlaced: true },
    { xPercent: 51, yPercent: 50, isPlaced: true }
  ], [target('one', 50, 50, 5)], config)

  assert.equal(result.totalScore, 200)
  assert.equal(result.scoredGuesses.filter((guess) => guess.isCorrect).length, 2)
})

test('handles no targets and empty guesses', () => {
  const noTargets = calculateTotalScore([{ xPercent: 10, yPercent: 10 }], [], config)
  assert.equal(noTargets.totalScore, 0)
  assert.equal(noTargets.scoredGuesses[0].isCorrect, false)
  assert.equal(noTargets.scoredGuesses[0].targetId, null)
  assert.equal(noTargets.scoredGuesses[0].scoreEarned, 0)

  const noGuesses = calculateTotalScore([], [target('one', 50, 50)], config)
  assert.deepEqual(noGuesses, { totalScore: 0, scoredGuesses: [] })
})
