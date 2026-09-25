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
// One-to-one matching: each target can be claimed by a single guess (the
// nearest qualifying unused one), so stacking pins on the same spot cannot
// multiply the score. A question is worth config.maxPoints in total, split
// across its targets as integer shares (leftover points go to the earliest
// claims, so a fully solved question pays exactly maxPoints).
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

    const targetCount = targets.length;
    const basePoints = Math.floor(config.maxPoints / targetCount);
    let remainder = config.maxPoints - basePoints * targetCount;
    const usedTargetIds = new Set();

    for (const guess of guesses) {
        let bestDistance = Infinity;
        let matchedTarget = null;
        let matchedDistance = Infinity;

        // Check if this guess is within tolerance of any unclaimed target
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

            if (usedTargetIds.has(target.id)) continue;

            if (distance <= target.radius + config.tolerancePercent && distance < matchedDistance) {
                matchedTarget = target;
                matchedDistance = distance;
            }
        }

        if (matchedTarget) {
            // Claim the target so later pins on the same spot score nothing
            usedTargetIds.add(matchedTarget.id);
            let points = basePoints;
            if (remainder > 0) {
                points += 1;
                remainder -= 1;
            }
            const scoreEarned = calculateGuessScore(guess, matchedTarget, config.tolerancePercent, points, aspect);
            totalScore += scoreEarned;

            scoredGuesses.push({
                ...guess,
                targetId: matchedTarget.id,
                distance: bestDistance,
                isCorrect: true,
                scoreEarned
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
