// A9: a skipped question breaks the streak. Answering wrong already resets
// it inside validateAndScore, but timing out (or missing questions while
// disconnected) never did, handing a stale streak bonus to the next answer.
export function streakAfterTimeout({ hasSubmitted, streak }) {
    if (hasSubmitted) return streak
    return 0
}

export function streakAfterCatchup({ missedCount, streak }) {
    if (missedCount > 0) return 0
    return streak
}
