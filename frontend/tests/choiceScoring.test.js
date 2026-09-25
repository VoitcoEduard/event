import { describe, it } from 'node:test'
import assert from 'node:assert/strict'
import { calculateChoiceScore, normalizeChoiceMaxPoints } from '../src/utils/choiceScoring.js'

describe('A7 choice scoring honours admin maxPoints', () => {
    it('keeps the historical 1000-point curve when maxPoints is 1000', () => {
        assert.equal(calculateChoiceScore({ maxPoints: 1000, timeLimit: 20, timeSpent: 0, streak: 0 }), 1000)
        assert.equal(calculateChoiceScore({ maxPoints: 1000, timeLimit: 20, timeSpent: 10, streak: 0 }), 625)
        assert.equal(calculateChoiceScore({ maxPoints: 1000, timeLimit: 20, timeSpent: 20, streak: 0 }), 250)
    })

    it('scales the curve to custom maxPoints', () => {
        assert.equal(calculateChoiceScore({ maxPoints: 500, timeLimit: 20, timeSpent: 0, streak: 0 }), 500)
        assert.equal(calculateChoiceScore({ maxPoints: 500, timeLimit: 20, timeSpent: 20, streak: 0 }), 125)
        // half time: 500 - 10 * (375 / 20) = 312.5 -> floor 312
        assert.equal(calculateChoiceScore({ maxPoints: 500, timeLimit: 20, timeSpent: 10, streak: 0 }), 312)
    })

    it('scales the streak bonus with maxPoints', () => {
        // 1000-point question: +100 per streak, capped at +500
        assert.equal(calculateChoiceScore({ maxPoints: 1000, timeLimit: 20, timeSpent: 0, streak: 3 }), 1300)
        assert.equal(calculateChoiceScore({ maxPoints: 1000, timeLimit: 20, timeSpent: 0, streak: 9 }), 1500)
        // 500-point question: +50 per streak, capped at +250
        assert.equal(calculateChoiceScore({ maxPoints: 500, timeLimit: 20, timeSpent: 0, streak: 3 }), 650)
    })

    it('falls back to 1000 for missing or invalid maxPoints', () => {
        assert.equal(normalizeChoiceMaxPoints(undefined), 1000)
        assert.equal(normalizeChoiceMaxPoints(0), 1000)
        assert.equal(normalizeChoiceMaxPoints('abc'), 1000)
        assert.equal(
            calculateChoiceScore({ maxPoints: undefined, timeLimit: 20, timeSpent: 0, streak: 0 }),
            1000
        )
    })
})
