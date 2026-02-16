/**
 * Label Extractor Utility
 * Extracts a compact label from a question string without using LLM
 * Uses rule-based pattern matching and keyword extraction
 */

/**
 * Common word mappings for question patterns
 */
const WORD_MAPPINGS = {
    'old': 'age',
    'yrs': 'age',
    'years': 'age',
}

/**
 * Words to skip (common adjectives, pronouns, etc.)
 */
const SKIP_WORDS = new Set([
    // Pronouns and possessives
    'your', 'my', 'our', 'their', 'his', 'her', 'its', 'you', 'i', 'me', 'we', 'they', 'us',
    // Question words (including common contractions)
    'what', 'which', 'who', 'whom', 'whose', 'how', 'why', 'when', 'where', 'whats', 'whos',
    // Common verbs
    'is', 'are', 'was', 'were', 'be', 'been', 'being', 'do', 'does', 'did', 'have', 'has', 'had',
    'live', 'living', 'lives',
    // Modal verbs
    'will', 'would', 'could', 'should', 'may', 'might', 'must', 'shall', 'can', 'need',
    // Prepositions and articles
    'to', 'of', 'in', 'for', 'on', 'with', 'at', 'by', 'from', 'as', 'into', 'through',
    'during', 'before', 'after', 'above', 'below', 'between', 'under', 'over', 'the', 'a', 'an',
    // Other common words
    'please', 'kind', 'sort', 'type', 'specific', 'most', 'least', 'best', 'worst',
    'new', 'young', 'first', 'last', 'next', 'previous', 'current',
    'favorite', 'favourite', 'preferred', 'prefer', 'like', 'select', 'choose', 'pick',
    'enter', 'provide', 'tell', 'give', 'state', 'answer', 'respond',
])

/**
 * Extracts a compact label from a question string
 * @param {string} question - The full question text
 * @returns {string} A compact label (e.g., "color", "email_address")
 */
export function extractLabel(question) {
    if (!question || typeof question !== 'string') {
        return 'field'
    }

    let text = question.trim()

    // Remove question mark at the end
    text = text.replace(/\?$/, '')

    // Split into words
    const words = text.split(/\s+/)
        .map(w => w.replace(/[^a-zA-Z0-9-]/g, '')) // Remove punctuation
        .filter(w => w.length > 0)

    if (words.length === 0) {
        // Fallback: use the original question, cleaned
        return question.replace(/[^a-zA-Z0-9]/g, '').toLowerCase().slice(0, 20) || 'field'
    }

    // Strategy: Find the last 1-2 words that are NOT in the skip list
    // This works because most questions end with the key noun
    let labelWords = []

    // Go from the end and collect non-skip words
    for (let i = words.length - 1; i >= 0; i--) {
        const word = words[i].toLowerCase()

        // Skip if in skip list
        if (SKIP_WORDS.has(word)) {
            continue
        }

        // Apply word mapping if exists (e.g., 'old' -> 'age')
        const mappedWord = WORD_MAPPINGS[word] || word
        labelWords.push(mappedWord)

        // Stop if we have 2 good words
        if (labelWords.length >= 2) {
            break
        }

        // Also stop if we've gone too far back (more than 2 words from end)
        if (words.length - i > 3) {
            break
        }
    }

    // If we have no words, use the last word of the question
    if (labelWords.length === 0) {
        return words[words.length - 1].toLowerCase()
    }

    // Reverse to get back to normal order and join with underscore
    return labelWords.reverse().join('_')
}
