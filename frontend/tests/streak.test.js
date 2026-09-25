import { describe, it } from 'node:test'
import assert from 'node:assert/strict'
import { streakAfterTimeout, streakAfterCatchup } from '../src/utils/streak.js'

describe('A9 skipped questions reset the streak', () => {
    it('timeout without an answer resets the streak', () => {
        assert.equal(streakAfterTimeout({ hasSubmitted: false, streak: 4 }), 0)
    })

    it('timeout after submitting keeps the streak from scoring', () => {
        assert.equal(streakAfterTimeout({ hasSubmitted: true, streak: 4 }), 4)
    })

    it('missed questions on reconnect reset the streak', () => {
        assert.equal(streakAfterCatchup({ missedCount: 2, streak: 5 }), 0)
        assert.equal(streakAfterCatchup({ missedCount: 0, streak: 5 }), 5)
    })
})
