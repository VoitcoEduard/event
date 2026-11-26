
const WORD_MAPPINGS = {
    'old': 'age',
    'yrs': 'age',
    'years': 'age',
    'dob': 'birthdate',
    'birth': 'birthdate',

    'лет': 'age',
    'год': 'age',
    'возраст': 'age',

    'ani': 'age',
    'varsta': 'age',
    'vârsta': 'age',
    'vârstă': 'age',
}

const SKIP_WORDS = new Set([
    'your', 'my', 'our', 'their', 'his', 'her', 'its', 'you', 'i', 'me', 'we', 'they', 'us',
    'what', 'which', 'who', 'whom', 'whose', 'how', 'why', 'when', 'where', 'whats', 'whos',
    'is', 'are', 'was', 'were', 'be', 'been', 'being', 'do', 'does', 'did', 'have', 'has', 'had',
    'live', 'living', 'lives',
    'will', 'would', 'could', 'should', 'may', 'might', 'must', 'shall', 'can', 'need',
    'to', 'of', 'in', 'for', 'on', 'with', 'at', 'by', 'from', 'as', 'into', 'through',
    'during', 'before', 'after', 'above', 'below', 'between', 'under', 'over', 'the', 'a', 'an',
    'please', 'kind', 'sort', 'type', 'specific', 'most', 'least', 'best', 'worst',
    'new', 'young', 'first', 'last', 'next', 'previous', 'current',
    'favorite', 'favourite', 'preferred', 'prefer', 'like', 'select', 'choose', 'pick',
    'enter', 'provide', 'tell', 'give', 'state', 'answer', 'respond',

    'я', 'ты', 'он', 'она', 'оно', 'мы', 'вы', 'они', 'себя',
    'мой', 'твой', 'ваш', 'наш', 'свой', 'его', 'ее', 'её', 'их',
    'кто', 'что', 'где', 'когда', 'куда', 'откуда', 'почему', 'зачем', 'как', 'какой', 'какая', 'какие', 'сколько', 'чей',
    'быть', 'есть', 'был', 'была', 'были', 'будет', 'будут',
    'иметь', 'имеет', 'делать', 'сделать',
    'живешь', 'живете', 'проживаете',
    'в', 'на', 'с', 'со', 'из', 'от', 'у', 'к', 'по', 'за', 'о', 'об', 'для', 'до', 'при', 'под', 'над', 'без',
    'и', 'а', 'но', 'или', 'ли', 'бы', 'же', 'то', 'не', 'ни',
    'пожалуйста', 'прошу', 'скажите', 'укажите', 'введите', 'выберите', 'ответьте', 'напишите',
    'тип', 'вид', 'любимый', 'лучший', 'худший', 'первый', 'последний', 'следующий', 'предыдущий',
    'текущий', 'новый', 'старый',

    'eu', 'tu', 'el', 'ea', 'noi', 'voi', 'ei', 'ele',
    'meu', 'ta', 'tau', 'tău', 'sa', 'lui', 'ei', 'nostru', 'vostru', 'lor',
    'dumneavoastra', 'dumneavoastră', 'tine', 'mine',
    'cine', 'ce', 'care', 'unde', 'cand', 'când', 'cum', 'cat', 'cât', 'de ce',
    'este', 'sunt', 'esti', 'ești', 'era', 'fost', 'fi',
    'avea', 'ai', 'am', 'au', 'aveti', 'aveți',
    'face', 'faceti', 'faceți',
    'locuiesti', 'locuiți', 'locuiti',
    'la', 'în', 'in', 'pe', 'cu', 'de', 'din', 'pentru', 'despre', 'spre', 'la', 'sub', 'peste',
    'un', 'o', 'ul', 'a', 'al', 'ai', 'ale',
    'si', 'și', 'sau', 'dar', 'nici',
    'te', 'rog', 'va', 'vă', 'rugam', 'rugăm',
    'spune', 'introdu', 'introduceți', 'introduceti', 'scrie', 'alege', 'selecteaza', 'selectează', 'raspunde', 'răspunde',
    'tip', 'fel', 'gen', 'favorit', 'preferat', 'primul', 'ultimul', 'urmatorul', 'următorul',
    'nou', 'vechi'
])

export function extractLabel(question) {
    if (!question || typeof question !== 'string') {
        return 'field'
    }

    let text = question.trim()

    text = text.replace(/[?؟;!]+$/, '')

    const words = text.split(/\s+/)
        .map(w => w.replace(/[^\p{L}\p{N}-]/gu, ''))
        .filter(w => w.length > 0)

    if (words.length === 0) {
        return question.replace(/[^\p{L}\p{N}]/gu, '').toLowerCase().slice(0, 20) || 'field'
    }

    const meaningfulWords = []

    for (let i = 0; i < words.length; i++) {
        const word = words[i].toLowerCase()

        if (!SKIP_WORDS.has(word)) {
            const mappedWord = WORD_MAPPINGS[word] || word
            meaningfulWords.push(mappedWord)
        }
    }

    if (meaningfulWords.length === 0) {
        return question.replace(/[^\p{L}\p{N}]/gu, '').toLowerCase().slice(0, 20) || 'field'
    }

    const resultWords = meaningfulWords.slice(-2)

    return resultWords.join('_')
}
