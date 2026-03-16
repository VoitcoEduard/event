// Calculate distance between two percentage-based coordinates
export function calculateDistance(x1, y1, x2, y2) {
    return Math.sqrt(Math.pow(x2 - x1, 2) + Math.pow(y2 - y1, 2));
}

// Calculate score based on distance from target
export function calculateGuessScore(guess, target, tolerancePercent, maxPoints) {
    const distance = calculateDistance(
        guess.xPercent,
        guess.yPercent,
        target.xPercent,
        target.yPercent
    );

    // All-or-nothing scoring: if within radius (including tolerance), full points. Otherwise 0.
    if (distance <= target.radius + tolerancePercent) {
        return Math.round(maxPoints);
    }

    return 0;
}

// Calculate total score for all guesses
export function calculateTotalScore(guesses, targets, config) {
    let totalScore = 0;
    const scoredGuesses = [];

    if (!targets || targets.length === 0 || !guesses || guesses.length === 0) {
        return { totalScore: 0, scoredGuesses: guesses || [] };
    }

    // Match each guess to nearest unmatched target
    const usedTargets = new Set();
    const maxPointsPerTarget = config.maxPoints / targets.length;

    for (const guess of guesses) {
        let bestMatch = null;
        let bestDistance = Infinity;

        for (const target of targets) {
            if (usedTargets.has(target.id)) continue;

            const distance = calculateDistance(
                guess.xPercent,
                guess.yPercent,
                target.xPercent,
                target.yPercent
            );

            if (distance < bestDistance) {
                bestDistance = distance;
                bestMatch = target;
            }
        }

        if (bestMatch && bestDistance <= bestMatch.radius + config.tolerancePercent) {
            usedTargets.add(bestMatch.id);
            const score = calculateGuessScore(
                guess,
                bestMatch,
                config.tolerancePercent,
                maxPointsPerTarget
            );
            totalScore += score;

            scoredGuesses.push({
                ...guess,
                targetId: bestMatch.id,
                distance: bestDistance,
                isCorrect: score > 0,
                scoreEarned: score
            });
        } else {
            // no match within tolerance
            scoredGuesses.push({
                ...guess,
                targetId: null,
                distance: bestDistance,
                isCorrect: false,
                scoreEarned: 0
            });
        }
    }

    return { totalScore, scoredGuesses };
}
