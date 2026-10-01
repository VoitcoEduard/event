import { readonly, ref } from 'vue'

export const ADMIN_THEME_KEY = 'adminTheme'
export const ADMIN_THEMES = ['light', 'dark']

export const normalizeAdminTheme = value => ADMIN_THEMES.includes(value) ? value : 'light'

export const readAdminTheme = storage => {
  try {
    return normalizeAdminTheme(storage?.getItem(ADMIN_THEME_KEY))
  } catch {
    return 'light'
  }
}

const theme = ref(typeof window === 'undefined' ? 'light' : readAdminTheme(window.localStorage))
const adminAppearanceActive = ref(false)
let storageListenerInstalled = false

const applyDocumentTheme = () => {
  if (typeof document === 'undefined') return

  if (adminAppearanceActive.value) {
    document.documentElement.dataset.adminAppearance = 'true'
    document.documentElement.dataset.adminTheme = theme.value
    document.documentElement.style.colorScheme = theme.value
  } else {
    delete document.documentElement.dataset.adminAppearance
    delete document.documentElement.dataset.adminTheme
    document.documentElement.style.removeProperty('color-scheme')
  }
}

const persistTheme = value => {
  try {
    window.localStorage.setItem(ADMIN_THEME_KEY, value)
  } catch {
    // The theme still works for this tab when storage is unavailable.
  }
}

export const setAdminTheme = value => {
  theme.value = normalizeAdminTheme(value)
  persistTheme(theme.value)
  applyDocumentTheme()
}

export const toggleAdminTheme = () => {
  setAdminTheme(theme.value === 'light' ? 'dark' : 'light')
}

export const setAdminAppearanceActive = active => {
  adminAppearanceActive.value = Boolean(active)
  applyDocumentTheme()
}

export const installAdminTheme = router => {
  if (!storageListenerInstalled && typeof window !== 'undefined') {
    window.addEventListener('storage', event => {
      if (event.key !== ADMIN_THEME_KEY) return
      theme.value = normalizeAdminTheme(event.newValue)
      applyDocumentTheme()
    })
    storageListenerInstalled = true
  }

  router.afterEach(to => setAdminAppearanceActive(to.meta.adminAppearance === true))
}

export const useAdminTheme = () => ({
  theme: readonly(theme),
  adminAppearanceActive: readonly(adminAppearanceActive),
  setTheme: setAdminTheme,
  toggleTheme: toggleAdminTheme
})
