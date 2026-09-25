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

test('scales Y distance by image aspect ratio (radius is width-percent)', () => {
  // Wide image: aspect 2, so 10% of height == 5% of width in pixels.
  // Target radius 5 + tolerance 2 => hit boundary at 7 width-percent units.
  assert.equal(calculateDistance(50, 50, 50, 64, 2), 7)
  assert.equal(calculateGuessScore({ xPercent: 50, yPercent: 64 }, target('wide', 50, 50, 5), 2, 100, 2), 100)
  assert.equal(calculateGuessScore({ xPercent: 50, yPercent: 65 }, target('wide', 50, 50, 5), 2, 100, 2), 0)

  // X distance is unaffected by aspect ratio.
  assert.equal(calculateDistance(50, 50, 57, 50, 2), 7)
  assert.equal(calculateGuessScore({ xPercent: 58, yPercent: 50 }, target('wide', 50, 50, 5), 2, 100, 2), 0)

  // Tall image: aspect 0.5, so 2.5% of height == 5% of width in pixels.
  assert.equal(calculateDistance(50, 50, 50, 52.5, 0.5), 5)
  assert.equal(calculateGuessScore({ xPercent: 50, yPercent: 52.5 }, target('tall', 50, 50, 5), 0, 100, 0.5), 100)
  assert.equal(calculateGuessScore({ xPercent: 50, yPercent: 56 }, target('tall', 50, 50, 5), 0, 100, 0.5), 0)
})

test('aspect correction matches pixel distance on a 1600x900 image', () => {
  const aspect = 1600 / 900
  // 45px vertical offset on a 900px-tall image = 5 height-percent.
  // In width-percent units that is 45 / 16 = 2.8125.
  const d = calculateDistance(50, 50, 50, 55, aspect)
  assert.ok(Math.abs(d - 2.8125) < 1e-10)
})

test('calculateTotalScore accepts aspect ratio explicitly and via config', () => {
  const targets = [target('one', 50, 50, 5)]
  const guesses = [{ xPercent: 50, yPercent: 64, isPlaced: true }]

  const explicit = calculateTotalScore(guesses, targets, config, 2)
  assert.equal(explicit.scoredGuesses[0].isCorrect, true)
  assert.equal(explicit.totalScore, 100)

  const viaConfig = calculateTotalScore(guesses, targets, { ...config, aspectRatio: 2 })
  assert.equal(viaConfig.scoredGuesses[0].isCorrect, true)

  // Same guess on a square image is out of range (distance 14 > 7).
  const square = calculateTotalScore(guesses, targets, config, 1)
  assert.equal(square.scoredGuesses[0].isCorrect, false)
  assert.equal(square.totalScore, 0)
})

test('invalid aspect ratios fall back to square behavior', () => {
  for (const aspect of [undefined, null, NaN, 0, -2, Infinity]) {
    assert.equal(calculateDistance(0, 0, 3, 4, aspect), 5)
    assert.equal(
      calculateTotalScore(
        [{ xPercent: 50, yPercent: 50, isPlaced: true }],
        [target('one', 50, 50, 5)],
        config,
        aspect
      ).totalScore,
      100
    )
  }
})
