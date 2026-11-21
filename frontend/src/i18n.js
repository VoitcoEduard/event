import { createI18n } from 'vue-i18n'
import en from './locales/en.json'
import ru from './locales/ru.json'
import ro from './locales/ro.json'
import { safeLocalGet } from './utils/storage'

const messages = {
  en,
  ru,
  ro
}

// Get language from localStorage or browser settings
const browserLang = typeof navigator !== 'undefined' && navigator.language ? navigator.language.split('-')[0] : 'en'
const userLang = safeLocalGet('userLanguage') || browserLang
const locale = messages[userLang] ? userLang : 'en'

const i18n = createI18n({
  legacy: false, // Use Composition API
  locale: locale,
  fallbackLocale: 'en',
  messages
})

export default i18n
