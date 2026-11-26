export function normalizeAspectRatio(value) {
    const aspect = Number(value);
    if (!Number.isFinite(aspect) || aspect <= 0) return 1;
    return aspect;
}

export function calculateDistance(x1, y1, x2, y2, aspectRatio = 1) {
    const aspect = normalizeAspectRatio(aspectRatio);
    const dx = x2 - x1;
    const dy = (y2 - y1) / aspect;
    return Math.sqrt(Math.pow(dx, 2) + Math.pow(dy, 2));
}

export function calculateGuessScore(guess, target, tolerancePercent, maxPoints, aspectRatio = 1) {
    const distance = calculateDistance(
        guess.xPercent,
        guess.yPercent,
        target.xPercent,
        target.yPercent,
        aspectRatio
    );

    if (distance <= target.radius + tolerancePercent) {
        return Math.round(maxPoints);
    }

    return 0;
}

export function calculateTotalScore(guesses, targets, config, aspectRatio) {
    let totalScore = 0;
    const scoredGuesses = [];

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
