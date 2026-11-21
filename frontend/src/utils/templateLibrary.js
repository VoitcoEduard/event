import { sanitizeAdminReturnTo } from '../router/index.js'
import { newUUID } from './index.js'

/** Canonical library tabs in display order. */
export const LIBRARY_TABS = [
  { id: 'quizzes', moduleType: 'quiz', labelKey: 'library.tabs.quizzes', icon: 'quiz', kind: 'template' },
  { id: 'forms', moduleType: 'form', labelKey: 'library.tabs.forms', icon: 'form', kind: 'template' },
  { id: 'wrong-pictures', moduleType: 'wrong_pictures', labelKey: 'library.tabs.wrong_pictures', icon: 'search', kind: 'template' },
  { id: 'screensavers', moduleType: 'screensaver', labelKey: 'library.tabs.screensavers', icon: 'image', kind: 'media' },
  { id: 'music', moduleType: 'music', labelKey: 'library.tabs.music', icon: 'music', kind: 'media' }
]

const TAB_ALIASES = {
  quizzes: 'quizzes',
  quiz: 'quizzes',
  forms: 'forms',
  form: 'forms',
  'wrong-pictures': 'wrong-pictures',
  wrong_pictures: 'wrong-pictures',
  screensavers: 'screensavers',
  screensaver: 'screensavers',
  music: 'music'
}

/**
 * Resolve a route param or moduleType string into a canonical library tab id.
 *
 * @param {string | string[] | undefined} rawTab
 * @param {string} [fallback='quizzes']
 * @returns {string}
 */
export function resolveLibraryTab(rawTab, fallback = 'quizzes') {
  const value = Array.isArray(rawTab) ? rawTab[0] : rawTab
  const key = String(value || '').trim().toLowerCase()
  return TAB_ALIASES[key] || fallback
}

/**
 * Return the tab configuration object for a route tab or moduleType.
 *
 * @param {string | string[] | undefined} rawTab
 * @returns {typeof LIBRARY_TABS[number]}
 */
export function getLibraryTabConfig(rawTab) {
  const id = resolveLibraryTab(rawTab)
  return LIBRARY_TABS.find(tab => tab.id === id) || LIBRARY_TABS[0]
}

/**
 * Validate a `?returnTo=` query value as an internal `/admin/...` path.
 * Uses `sanitizeAdminReturnTo` when auth route metadata is present, and
 * falls back to safe path validation for minimal unit-test routers.
 *
 * @param {string | string[] | undefined} rawReturnTo
 * @param {import('vue-router').Router | null | undefined} targetRouter
 * @returns {string | null}
 */
export function resolveSafeReturnTo(rawReturnTo, targetRouter) {
  const raw = Array.isArray(rawReturnTo) ? rawReturnTo[0] : rawReturnTo
  if (typeof raw !== 'string' || !targetRouter) return null
  const trimmed = raw.trim()
  if (
    !trimmed ||
    !trimmed.startsWith('/') ||
    trimmed.startsWith('//') ||
    trimmed.includes('\\') ||
    /[\u0000-\u001f\u007f]/.test(trimmed)
  ) {
    return null
  }

  const hasAuthMeta = targetRouter.getRoutes?.().some(
    record => Boolean(record.meta?.requiresAuth || record.meta?.requiresOwner)
  )
  if (hasAuthMeta) {
    return sanitizeAdminReturnTo(trimmed, targetRouter)
  }

  try {
    const parsed = new URL(trimmed, 'http://localhost')
    if (parsed.origin !== 'http://localhost') return null
    if (parsed.pathname !== '/admin' && !parsed.pathname.startsWith('/admin/')) return null
    const candidate = `${parsed.pathname}${parsed.search}${parsed.hash}`
    const resolved = targetRouter.resolve(candidate)
    return resolved?.matched?.length ? resolved.fullPath : null
  } catch {
    return null
  }
}

/**
 * Sanitize and normalize a single wrong_pictures round object.
 *
 * @param {object} [rawRound]
 * @returns {object}
 */
export function normalizeWrongPictureRound(rawRound = {}) {
  const cfg = rawRound?.wrongPictureConfig || {}
  const maxPoints = cfg.maxPoints !== undefined
    ? cfg.maxPoints
    : (rawRound?.scoringConfig?.maxPoints !== undefined ? rawRound.scoringConfig.maxPoints : 50)

  return {
    ...rawRound,
    id: rawRound?.id || newUUID(),
    type: 'wrong_pictures',
    text: rawRound?.text || '',
    timeLimit: rawRound?.timeLimit !== undefined ? rawRound.timeLimit : 60,
    explainTime: rawRound?.explainTime !== undefined ? rawRound.explainTime : 5,
    imageId: rawRound?.imageId || null,
    imageAlt: rawRound?.imageAlt || '',
    wrongPictureConfig: {
      targets: Array.isArray(cfg.targets) ? cfg.targets.map(target => ({ ...target })) : [],
      maxPoints,
      tolerancePercent: cfg.tolerancePercent !== undefined ? cfg.tolerancePercent : 5,
      showTargetCount: Boolean(cfg.showTargetCount)
    }
  }
}

/**
 * Sanitize an array of quiz questions or wrong_pictures rounds when loading
 * or importing templates.
 *
 * @param {string} moduleType
 * @param {Array<object>} [rawQuestions]
 * @returns {Array<object>}
 */
export function sanitizeTemplateQuestions(moduleType, rawQuestions = []) {
  const list = Array.isArray(rawQuestions) ? JSON.parse(JSON.stringify(rawQuestions)) : []

  if (moduleType === 'wrong_pictures') {
    return list.map(item => normalizeWrongPictureRound(item))
  }

  return list.map((q) => {
    const item = {
      ...q,
      id: q?.id || newUUID(),
      type: q?.type || 'classic'
    }

    if (item.type === 'wrong_pictures') {
      return normalizeWrongPictureRound(item)
    }

    if (item.type === 'range') {
      if (!item.rangeConfig) {
        item.rangeConfig = { minValue: 0, maxValue: 100, step: 1, unit: '', decimalPlaces: 0 }
      }
      if (!item.scoringConfig) {
        item.scoringConfig = { mode: 'symmetric', maxPoints: item.points ?? 100, tolerance: 10, partialPercent: 0 }
      }
    } else if (!item.scoringConfig && item.points !== undefined) {
      item.scoringConfig = { mode: 'symmetric', maxPoints: item.points, tolerance: 0, partialPercent: 0 }
    }

    if (item.type === 'true_false') {
      if (!Array.isArray(item.answers) || item.answers.length < 2 || !String(item.answers[0] || '').trim() || !String(item.answers[1] || '').trim()) {
        item.answers = ['True', 'False', '', '']
      }
      if (item.correctIndex > 1) item.correctIndex = 0
    }

    if (item.type === 'correct_picture' && !Array.isArray(item.imageOptions)) {
      item.imageOptions = [
        { id: newUUID(), imageId: '', label: '' },
        { id: newUUID(), imageId: '', label: '' }
      ]
    }

    return item
  })
}
