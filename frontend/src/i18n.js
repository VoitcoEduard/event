import { createI18n } from 'vue-i18n'
import en from './locales/en.json'
import ru from './locales/ru.json'
import ro from './locales/ro.json'

const messages = {
  en,
  ru,
  ro
}

// Get language from localStorage or browser settings
const userLang = localStorage.getItem('userLanguage') || navigator.language.split('-')[0]
const locale = messages[userLang] ? userLang : 'en'

const i18n = createI18n({
  legacy: false, // Use Composition API
  locale: locale,
  fallbackLocale: 'en',
  messages
})

export default i18n
