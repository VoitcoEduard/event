// Radius, tolerance and coordinates share one convention:
// xPercent/yPercent are 0-100 relative to image width/height, while radius
// and tolerancePercent are expressed in width-percentage points (matching the
// admin hit-area circle, which sizes itself from the image width).
// On non-square images 1% of width != 1% of height in pixels, so the Y delta
// is converted to width-percentage units via the image aspect ratio (W/H)
// before measuring Euclidean distance. For square images (aspect 1) this
// reduces to the plain percentage distance.
export function normalizeAspectRatio(value) {
    const aspect = Number(value);
    if (!Number.isFinite(aspect) || aspect <= 0) return 1;
    return aspect;
}

// Calculate aspect-corrected distance in width-percentage units
export function calculateDistance(x1, y1, x2, y2, aspectRatio = 1) {
    const aspect = normalizeAspectRatio(aspectRatio);
    const dx = x2 - x1;
    const dy = (y2 - y1) / aspect;
    return Math.sqrt(Math.pow(dx, 2) + Math.pow(dy, 2));
}

// Calculate score based on distance from target
export function calculateGuessScore(guess, target, tolerancePercent, maxPoints, aspectRatio = 1) {
    const distance = calculateDistance(
        guess.xPercent,
        guess.yPercent,
        target.xPercent,
        target.yPercent,
        aspectRatio
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
export function calculateTotalScore(guesses, targets, config, aspectRatio) {
    let totalScore = 0;
    const scoredGuesses = [];

    // Aspect may be passed explicitly or carried on the config object.
    const aspect = normalizeAspectRatio(aspectRatio ?? config?.aspectRatio ?? 1);

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
                target.yPercent,
                aspect
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
