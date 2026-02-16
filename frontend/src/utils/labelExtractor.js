/**
 * Label Extractor Utility
 * Extracts a compact label from a question string without using LLM
 * Uses rule-based pattern matching and keyword extraction
 * Supports English, Russian, and Romanian
 */

/**
 * Common word mappings for question patterns
 */
const WORD_MAPPINGS = {
    // English
    'old': 'age',
    'yrs': 'age',
    'years': 'age',
    'dob': 'birthdate',
    'birth': 'birthdate',

    // Russian
    'лет': 'age',
    'год': 'age',
    'возраст': 'age',

    // Romanian
    'ani': 'age',
    'varsta': 'age',
    'vârsta': 'age',
    'vârstă': 'age',
}

/**
 * Words to skip (common adjectives, pronouns, etc.)
 */
const SKIP_WORDS = new Set([
    // --- ENGLISH ---
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

    // --- RUSSIAN ---
    // Pronouns and possessives
    'я', 'ты', 'он', 'она', 'оно', 'мы', 'вы', 'они', 'себя',
    'мой', 'твой', 'ваш', 'наш', 'свой', 'его', 'ее', 'её', 'их',
    // Question words
    'кто', 'что', 'где', 'когда', 'куда', 'откуда', 'почему', 'зачем', 'как', 'какой', 'какая', 'какие', 'сколько', 'чей',
    // Common verbs (forms of be, have, do, etc - extremely simplified)
    'быть', 'есть', 'был', 'была', 'были', 'будет', 'будут',
    'иметь', 'имеет', 'делать', 'сделать',
    'живешь', 'живете', 'проживаете',
    // Prepositions and particles
    'в', 'на', 'с', 'со', 'из', 'от', 'у', 'к', 'по', 'за', 'о', 'об', 'для', 'до', 'при', 'под', 'над', 'без',
    'и', 'а', 'но', 'или', 'ли', 'бы', 'же', 'то', 'не', 'ни',
    // Other common words
    'пожалуйста', 'прошу', 'скажите', 'укажите', 'введите', 'выберите', 'ответьте', 'напишите',
    'тип', 'вид', 'любимый', 'лучший', 'худший', 'первый', 'последний', 'следующий', 'предыдущий',
    'текущий', 'новый', 'старый',

    // --- ROMANIAN ---
    // Pronouns and possessives
    'eu', 'tu', 'el', 'ea', 'noi', 'voi', 'ei', 'ele',
    'meu', 'ta', 'tau', 'tău', 'sa', 'lui', 'ei', 'nostru', 'vostru', 'lor',
    'dumneavoastra', 'dumneavoastră', 'tine', 'mine',
    // Question words
    'cine', 'ce', 'care', 'unde', 'cand', 'când', 'cum', 'cat', 'cât', 'de ce',
    // Common verbs
    'este', 'sunt', 'esti', 'ești', 'era', 'fost', 'fi',
    'avea', 'ai', 'am', 'au', 'aveti', 'aveți',
    'face', 'faceti', 'faceți',
    'locuiesti', 'locuiți', 'locuiti',
    // Prepositions and articles
    'la', 'în', 'in', 'pe', 'cu', 'de', 'din', 'pentru', 'despre', 'spre', 'la', 'sub', 'peste',
    'un', 'o', 'ul', 'a', 'al', 'ai', 'ale',
    'si', 'și', 'sau', 'dar', 'nici',
    // Other common words
    'te', 'rog', 'va', 'vă', 'rugam', 'rugăm',
    'spune', 'introdu', 'introduceți', 'introduceti', 'scrie', 'alege', 'selecteaza', 'selectează', 'raspunde', 'răspunde',
    'tip', 'fel', 'gen', 'favorit', 'preferat', 'primul', 'ultimul', 'urmatorul', 'următorul',
    'nou', 'vechi'
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
    text = text.replace(/[?؟;!]+$/, '')

    // Split into words using Unicode-aware regex
    const words = text.split(/\s+/)
        .map(w => w.replace(/[^\p{L}\p{N}-]/gu, '')) // Remove punctuation, keep letters (Unicode) and numbers
        .filter(w => w.length > 0)

    if (words.length === 0) {
        // Fallback: use the original question, cleaned
        return question.replace(/[^\p{L}\p{N}]/gu, '').toLowerCase().slice(0, 20) || 'field'
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
