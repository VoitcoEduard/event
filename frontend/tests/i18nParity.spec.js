import { describe, expect, it, vi, afterEach } from 'vitest'
import { readFileSync, readdirSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { mount, flushPromises } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { parse } from '@vue/compiler-sfc'
import i18n from '../src/i18n.js'
import en from '../src/locales/en.json'
import ru from '../src/locales/ru.json'
import ro from '../src/locales/ro.json'
import AdminThemeToggle from '../src/components/AdminThemeToggle.vue'
import LoginScreen from '../src/views/LoginScreen.vue'
import PendingAccessScreen from '../src/views/PendingAccessScreen.vue'
import SimpleQuizJudgeScreen from '../src/views/SimpleQuizJudgeScreen.vue'

const here = path.dirname(fileURLToPath(import.meta.url))
const srcDir = path.resolve(here, '../src')

function walkSourceFiles(dir) {
  const results = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      results.push(...walkSourceFiles(fullPath))
    } else if (/\.(vue|js)$/.test(entry.name)) {
      results.push(fullPath)
    }
  }
  return results
}

function flattenLocale(obj, prefix = '', out = new Map()) {
  for (const [key, value] of Object.entries(obj)) {
    const fullKey = prefix ? `${prefix}.${key}` : key
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      flattenLocale(value, fullKey, out)
    } else {
      out.set(fullKey, value)
    }
  }
  return out
}

function extractStaticTranslationKeys(sourceText) {
  const keys = []
  const pattern = /(?:^|[^a-zA-Z0-9_$.])(?:\$t|t)\(\s*(['"])([a-zA-Z0-9_.-]+)\1\s*[,)]/g
  let match
  while ((match = pattern.exec(sourceText)) !== null) {
    keys.push(match[2])
  }
  return keys
}

const wrappers = []
function track(wrapper) {
  wrappers.push(wrapper)
  return wrapper
}

afterEach(() => {
  while (wrappers.length) {
    wrappers.pop().unmount()
  }
  vi.unstubAllGlobals()
})

describe('i18n parity & coverage (Phase 6)', () => {
  const enMap = flattenLocale(en)
  const ruMap = flattenLocale(ru)
  const roMap = flattenLocale(ro)

  it('maintains identical non-empty key sets across en.json, ru.json, and ro.json', () => {
    const missingInRu = []
    const missingInRo = []
    const extraInRu = []
    const extraInRo = []
    const emptyValues = []

    for (const [key, val] of enMap.entries()) {
      if (typeof val !== 'string' || val.trim().length === 0) emptyValues.push(`en:${key}`)
      if (!ruMap.has(key)) missingInRu.push(key)
      if (!roMap.has(key)) missingInRo.push(key)
    }
    for (const [key, val] of ruMap.entries()) {
      if (typeof val !== 'string' || val.trim().length === 0) emptyValues.push(`ru:${key}`)
      if (!enMap.has(key)) extraInRu.push(key)
    }
    for (const [key, val] of roMap.entries()) {
      if (typeof val !== 'string' || val.trim().length === 0) emptyValues.push(`ro:${key}`)
      if (!enMap.has(key)) extraInRo.push(key)
    }

    expect(missingInRu, `Missing in ru.json: ${missingInRu.join(', ')}`).toEqual([])
    expect(missingInRo, `Missing in ro.json: ${missingInRo.join(', ')}`).toEqual([])
    expect(extraInRu, `Extra in ru.json: ${extraInRu.join(', ')}`).toEqual([])
    expect(extraInRo, `Extra in ro.json: ${extraInRo.join(', ')}`).toEqual([])
    expect(emptyValues, `Empty locale strings: ${emptyValues.join(', ')}`).toEqual([])
  })

  it('defines every static t(...) and $t(...) key used in frontend/src/ in en.json, ru.json, and ro.json', () => {
    const files = walkSourceFiles(srcDir)
    expect(files.length).toBeGreaterThan(20)

    const usedKeys = new Map()
    for (const filePath of files) {
      const content = readFileSync(filePath, 'utf8')
      const relPath = path.relative(srcDir, filePath)
      for (const key of extractStaticTranslationKeys(content)) {
        if (!usedKeys.has(key)) usedKeys.set(key, [])
        usedKeys.get(key).push(relPath)
      }
    }

    expect(usedKeys.size).toBeGreaterThan(200)

    const missing = []
    for (const [key, locations] of usedKeys.entries()) {
      const missingLocales = []
      if (!enMap.has(key)) missingLocales.push('en')
      if (!ruMap.has(key)) missingLocales.push('ru')
      if (!roMap.has(key)) missingLocales.push('ro')
      if (missingLocales.length > 0) {
        missing.push(`${key} (missing in ${missingLocales.join(', ')}; used in ${[...new Set(locations)].join(', ')})`)
      }
    }

    expect(missing, `Missing translation keys:\n${missing.join('\n')}`).toEqual([])
  })

  it('defines all dynamic i18n key families across en.json, ru.json, and ro.json', () => {
    const dynamicKeys = [
      'appearance.light',
      'appearance.dark',
      ...['quiz', 'simple_quiz', 'wrong_pictures', 'form', 'teams'].flatMap((type) => [
        `add_module.types.${type}.title`,
        `add_module.types.${type}.desc`,
        `add_module.placeholders.${type}`,
        `event_details.module_types.${type}`,
      ]),
      ...['text', 'number', 'date', 'radio', 'checkbox', 'dropdown'].map((type) => `form_editor.types.${type}`),
      ...['classic', 'multiple_choice', 'single_select', 'multi_select', 'true_false', 'range', 'number', 'correct_picture', 'wrong_pictures', 'simple_text'].map(
        (type) => `quiz_config.question_editor.types.${type}`,
      ),
      ...['quizzes', 'forms', 'wrong_pictures', 'screensavers', 'music'].map((tab) => `library.tabs.${tab}`),
    ]

    const validationSrc = readFileSync(path.join(srcDir, 'utils/quizValidation.js'), 'utf8')
    const keyMatches = [...validationSrc.matchAll(/'(quiz_config\.validation\.[a-z_]+)'/g)].map((m) => m[1])
    for (const k of keyMatches) dynamicKeys.push(k)

    const missing = []
    for (const key of dynamicKeys) {
      for (const [loc, map] of [['en', enMap], ['ru', ruMap], ['ro', roMap]]) {
        if (!map.has(key)) missing.push(`${loc}:${key}`)
      }
    }

    expect(missing).toEqual([])
  })
})

describe('Non-shell admin-styled screens inline AdminThemeToggle & mobile touch targets', () => {
  it('renders <AdminThemeToggle variant="inline" /> in LoginScreen, PendingAccessScreen, and SimpleQuizJudgeScreen', async () => {
    i18n.global.locale.value = 'en'
    vi.stubGlobal('fetch', vi.fn(async () => ({ ok: true, json: async () => ({ mode: 'dev' }) })))

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/login', name: 'login', component: LoginScreen },
        { path: '/pending', name: 'pending', component: PendingAccessScreen },
        { path: '/simple-quiz/:eventId/:moduleId/judge', name: 'judge', component: SimpleQuizJudgeScreen },
        { path: '/admin', component: { template: '<div />' } },
      ],
    })

    await router.push('/login')
    await router.isReady()
    const loginWrapper = track(mount(LoginScreen, { global: { plugins: [i18n, router] } }))
    await flushPromises()
    const loginToggle = loginWrapper.findComponent(AdminThemeToggle)
    expect(loginToggle.exists()).toBe(true)
    expect(loginToggle.props('variant')).toBe('inline')
    expect(loginWrapper.find('.login-utility-strip .admin-theme-control--inline').exists()).toBe(true)

    await router.push('/pending')
    const pendingWrapper = track(mount(PendingAccessScreen, { global: { plugins: [i18n, router] } }))
    await flushPromises()
    const pendingToggle = pendingWrapper.findComponent(AdminThemeToggle)
    expect(pendingToggle.exists()).toBe(true)
    expect(pendingToggle.props('variant')).toBe('inline')
    expect(pendingWrapper.find('.access-utility-strip .admin-theme-control--inline').exists()).toBe(true)

    await router.push('/simple-quiz/evt-1/mod-1/judge')
    const judgeWrapper = track(mount(SimpleQuizJudgeScreen, { global: { plugins: [i18n, router] } }))
    await flushPromises()
    const judgeToggle = judgeWrapper.findComponent(AdminThemeToggle)
    expect(judgeToggle.exists()).toBe(true)
    expect(judgeToggle.props('variant')).toBe('inline')
    expect(judgeWrapper.find('.judge-utility-strip .admin-theme-control--inline').exists()).toBe(true)
  })

  it('defines >= 44px touch targets in mobile/coarse-pointer styles across all admin shell views', () => {
    const filesToCheck = [
      'components/AdminLayout.vue',
      'views/AdminEventDetailsScreen.vue',
      'views/DataLibraryScreen.vue',
      'views/ScreenManagerScreen.vue',
      'views/QuizLibraryScreen.vue',
      'views/AdminModuleEditorScreen.vue',
      'views/AdminTemplateEditorScreen.vue',
    ]

    for (const rel of filesToCheck) {
      const content = readFileSync(path.join(srcDir, rel), 'utf8')
      const { descriptor, errors } = parse(content)
      expect(errors, `${rel} should parse cleanly`).toEqual([])

      const styles = descriptor.styles.map((s) => s.content).join('\n')
      expect(styles, `${rel} must specify 44px touch targets`).toMatch(/min-(?:height|block-size):\s*44px/)
      expect(styles, `${rel} must include a <= 768px or pointer: coarse rule`).toMatch(/@media\s*\([^)]*(?:max-width:\s*768px|pointer:\s*coarse)|@container\s+\w+\s*\(inline-size\s*<=\s*768px/)
    }
  })
})
