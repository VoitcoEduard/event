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
// New scoring: Each correct guess (within tolerance of ANY target) scores maxPoints
// No need to match to specific targets - just count correct guesses
// This allows user to guess any number of markers without knowing target count
export function calculateTotalScore(guesses, targets, config) {
    let totalScore = 0;
    const scoredGuesses = [];

    if (!guesses || guesses.length === 0) {
        return { totalScore: 0, scoredGuesses: [] };
    }

    if (!targets || targets.length === 0) {
        return {
            totalScore: 0,
            scoredGuesses: guesses.map((guess) => ({
                ...guess,
                targetId: null,
                distance: Infinity,
                isCorrect: false,
                scoreEarned: 0
            }))
        };
    }

    const maxPointsPerGuess = config.maxPoints;

    for (const guess of guesses) {
        let isCorrect = false;
        let matchedTargetId = null;
        let bestDistance = Infinity;
        let bestMatchDistance = Infinity;

        // Check if this guess is within tolerance of ANY target
        for (const target of targets) {
            const distance = calculateDistance(
                guess.xPercent,
                guess.yPercent,
                target.xPercent,
                target.yPercent
            );

            if (distance < bestDistance) {
                bestDistance = distance;
            }

            if (distance <= target.radius + config.tolerancePercent && distance < bestMatchDistance) {
                isCorrect = true;
                matchedTargetId = target.id;
                bestMatchDistance = distance;
            }
        }

        if (isCorrect) {
            // Each correct guess scores maxPoints
            totalScore += maxPointsPerGuess;

            scoredGuesses.push({
                ...guess,
                targetId: matchedTargetId,
                distance: bestDistance,
                isCorrect: true,
                scoreEarned: maxPointsPerGuess
            });
        } else {
            // Incorrect guess - no match within tolerance
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
