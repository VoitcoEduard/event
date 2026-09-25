// Time-decayed scoring for single-answer choice questions
// (classic, true_false, multi_select, correct_picture).
//
// The admin-configured maxPoints is the score for an instant correct answer.
// It decays linearly to 25% of maxPoints when the timer runs out, mirroring
// the historical 1000 -> 250 curve. The streak bonus scales the same way:
// +10% of maxPoints per streak step, capped at +50%.
export const DEFAULT_CHOICE_MAX_POINTS = 1000

export function normalizeChoiceMaxPoints(value) {
    const maxPoints = Number(value)
    if (!Number.isFinite(maxPoints) || maxPoints <= 0) return DEFAULT_CHOICE_MAX_POINTS
    return Math.floor(maxPoints)
}

export function calculateChoiceScore({ maxPoints, timeLimit, timeSpent, streak }) {
    const max = normalizeChoiceMaxPoints(maxPoints)
    const limit = Number(timeLimit) > 0 ? Number(timeLimit) : 20
    const spent = Math.max(0, Number(timeSpent) || 0)
    const streakSteps = Math.max(0, Number(streak) || 0)

    const floorScore = Math.floor(max * 0.25)
    const baseScore = Math.max(floorScore, max - spent * ((max * 0.75) / limit))
    const streakBonus = Math.min(streakSteps * Math.floor(max / 10), Math.floor(max / 2))
    return Math.floor(baseScore + streakBonus)
}
