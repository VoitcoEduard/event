export function streakAfterTimeout({ hasSubmitted, streak }) {
    if (hasSubmitted) return streak
    return 0
}

export function streakAfterCatchup({ missedCount, streak }) {
    if (missedCount > 0) return 0
    return streak
}
