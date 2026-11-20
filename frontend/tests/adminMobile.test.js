import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync, readdirSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import { parse } from '@vue/compiler-sfc'

const here = path.dirname(fileURLToPath(import.meta.url))
const src = (rel) => readFileSync(path.join(here, '..', rel), 'utf8')

const ADMIN_VIEWS = [
  'LoginScreen.vue',
  'PendingAccessScreen.vue',
  'AdminUsersScreen.vue',
  'AdminScreen.vue',
  'AdminEventDetailsScreen.vue',
  'DataLibraryScreen.vue',
  'TeamsAdminScreen.vue',
  'ScreenManagerScreen.vue',
  'QuizHostScreen.vue',
  'SimpleQuizHostScreen.vue',
  'SimpleQuizJudgeScreen.vue',
  'QuizLibraryScreen.vue'
]

const ADMIN_COMPONENTS = [
  'Toast.vue',
  'FormEditor.vue',
  'FormModuleConfig.vue',
  'QuizEditor.vue',
  'QuizQuestionEditor.vue',
  'QuestionTypeSelector.vue',
  'ImageOptionEditor.vue',
  'ImageUploader.vue',
  'CoordinatePicker.vue',
  'QuizModuleConfig.vue',
  'WrongPicturesModuleConfig.vue',
  'SimpleQuizModuleConfig.vue',
  'TeamsModuleConfig.vue',
  'TeamsConfigFields.vue',
  'TeamsActivityAdmin.vue'
]

test('all adapted admin views and components parse cleanly with @vue/compiler-sfc', () => {
  for (const view of ADMIN_VIEWS) {
    const { errors } = parse(src(`src/views/${view}`))
    assert.deepEqual(errors, [], `src/views/${view} should parse without SFC errors`)
  }
  for (const component of ADMIN_COMPONENTS) {
    const { errors } = parse(src(`src/components/${component}`))
    assert.deepEqual(errors, [], `src/components/${component} should parse without SFC errors`)
  }
})

test('admin-theme.css layers theme control below modals, sets scroll-padding/overscroll, and avoids overflow-x: hidden on .data-page', () => {
  const themeCss = src('src/assets/admin-theme.css')

  const themeControlMatch = themeCss.match(/\.admin-theme-control\s*\{[^}]*z-index:\s*(\d+)\s*;[^}]*\}/s)
  assert.ok(themeControlMatch, '.admin-theme-control must define a numeric z-index')
  assert.ok(
    Number(themeControlMatch[1]) < 1000,
    `.admin-theme-control z-index (${themeControlMatch[1]}) must be below 1000 so modals render above it`
  )

  assert.match(themeCss, /scroll-padding-bottom\s*:/)
  assert.match(themeCss, /dialog:modal[\s\S]*?overscroll-behavior:\s*contain/)
  assert.match(themeCss, /dialog::backdrop\s*\{[^}]*overscroll-behavior:\s*contain/s)
  assert.doesNotMatch(themeCss, /\.data-page[^{]*\{[^}]*overflow-x:\s*hidden/s)
})

test('Toast.vue anchors .toast-container at the top on mobile viewports', () => {
  const { descriptor, errors } = parse(src('src/components/Toast.vue'))
  assert.deepEqual(errors, [])

  const styles = descriptor.styles.map((s) => s.content).join('\n')
  const mobileBlock = styles.match(/@media\s*\(max-width:\s*640px\)\s*\{([\s\S]*?)\n\}/)
  assert.ok(mobileBlock, 'Toast.vue must define a @media (max-width: 640px) block')
  assert.match(mobileBlock[1], /\.toast-container\s*\{[^}]*top:\s*[^;]+;[^}]*bottom:\s*auto\s*;/s)
})

test('DataLibraryScreen.vue uses container-driven stacked cards and locales define clean admin mobile strings', () => {
  const { descriptor, errors } = parse(src('src/views/DataLibraryScreen.vue'))
  assert.deepEqual(errors, [])

  const template = descriptor.template?.content || ''
  assert.match(template, /\bdata-label=/)

  const styles = descriptor.styles.map((s) => s.content).join('\n')
  assert.match(styles, /container:\s*data-table\s*\/\s*inline-size/)
  assert.match(styles, /@container\s+data-table\s*\(/)
  assert.match(styles, /thead\s*\{\s*display:\s*none\s*;?\s*\}/)
  assert.match(styles, /attr\(data-label\)/)

  for (const loc of ['en', 'ru', 'ro']) {
    const messages = JSON.parse(src(`src/locales/${loc}.json`))
    const backToEvent = messages?.data_library?.back_to_event
    assert.ok(
      typeof backToEvent === 'string' && backToEvent.trim().length > 0,
      `${loc}.json must define non-empty data_library.back_to_event`
    )
    assert.ok(
      !backToEvent.trimStart().startsWith('←'),
      `${loc}.json data_library.back_to_event must not have a leading ←`
    )

    for (const [label, value] of [
      ['auth.noUsers', messages?.auth?.noUsers],
      ['screen_manager.editor.mobile_slides', messages?.screen_manager?.editor?.mobile_slides],
      ['screen_manager.editor.mobile_properties', messages?.screen_manager?.editor?.mobile_properties]
    ]) {
      assert.ok(
        typeof value === 'string' && value.trim().length > 0,
        `${loc}.json must define a non-empty ${label}`
      )
    }
  }
})

test('ScreenManagerScreen.vue uses Pointer Events, touch-action: none, keyboard arrow nudging, aria-live, and @container screen-manager', () => {
  const { descriptor, errors } = parse(src('src/views/ScreenManagerScreen.vue'))
  assert.deepEqual(errors, [])

  const template = descriptor.template?.content || ''
  assert.match(template, /@pointerdown/)
  assert.match(template, /@pointermove/)
  assert.match(template, /@pointerup/)
  assert.match(template, /aria-live="polite"/)

  const script = descriptor.scriptSetup?.content || descriptor.script?.content || ''
  assert.match(script, /setPointerCapture/)
  assert.match(script, /ArrowUp/)
  assert.match(script, /ArrowDown/)
  assert.match(script, /ArrowLeft/)
  assert.match(script, /ArrowRight/)

  const styles = descriptor.styles.map((s) => s.content).join('\n')
  assert.match(styles, /touch-action:\s*none/)
  assert.match(styles, /@container\s+screen-manager\s*\(/)
})

test('QuestionTypeSelector.vue uses semantic <button> elements for .type-card', () => {
  const { descriptor, errors } = parse(src('src/components/QuestionTypeSelector.vue'))
  assert.deepEqual(errors, [])

  const template = descriptor.template?.content || ''
  assert.match(template, /<button\b[^>]*class="[^"]*\btype-card\b/)
  assert.doesNotMatch(template, /<div\b[^>]*class="[^"]*\btype-card\b/)
})

test('QuizHostScreen.vue uses clamp() for questionFontSize and getAnswerFontSize', () => {
  const { descriptor, errors } = parse(src('src/views/QuizHostScreen.vue'))
  assert.deepEqual(errors, [])

  const script = descriptor.scriptSetup?.content || descriptor.script?.content || ''
  const questionFn = script.match(/const\s+questionFontSize\s*=\s*computed\([\s\S]*?\n\}\)/)
  assert.ok(questionFn, 'QuizHostScreen.vue must define questionFontSize computed property')
  assert.match(questionFn[0], /return\s+'clamp\(/)
  assert.doesNotMatch(questionFn[0], /return\s+`[^`]*\$\{[^}]+\}vw`/)

  const answerFn = script.match(/const\s+getAnswerFontSize\s*=\s*\([^)]*\)\s*=>\s*\{[\s\S]*?\n\}/)
  assert.ok(answerFn, 'QuizHostScreen.vue must define getAnswerFontSize helper')
  assert.match(answerFn[0], /return\s+'clamp\(/)
  assert.doesNotMatch(answerFn[0], /return\s+`[^`]*\$\{[^}]+\}vw`/)
})

test('FormEditor.vue wraps .type-selector and QuizQuestionEditor.vue stacks .editor-main-row and supports 45s timeLimit', () => {
  const formEditor = parse(src('src/components/FormEditor.vue'))
  assert.deepEqual(formEditor.errors, [])
  const formStyles = formEditor.descriptor.styles.map((s) => s.content).join('\n')
  assert.match(formStyles, /\.type-selector\s*\{[^}]*flex-wrap:\s*wrap/s)

  const questionEditor = parse(src('src/components/QuizQuestionEditor.vue'))
  assert.deepEqual(questionEditor.errors, [])
  const questionStyles = questionEditor.descriptor.styles.map((s) => s.content).join('\n')
  assert.match(questionStyles, /\.editor-main-row\s*\{[^}]*flex-direction:\s*column/s)

  const questionScript = questionEditor.descriptor.scriptSetup?.content || questionEditor.descriptor.script?.content || ''
  assert.match(questionScript, /BASE_TIME_LIMITS\s*=\s*\[[^\]]*\b45\b[^\]]*\]/)
  assert.match(questionScript, /wrong_pictures/)
})

test('no SFC in views/ or components/ places a descendant selector outside :global(html[data-admin-appearance="true"])', () => {
  for (const dir of ['src/views', 'src/components']) {
    const absDir = path.join(here, '..', dir)
    for (const entry of readdirSync(absDir)) {
      if (!entry.endsWith('.vue')) continue
      const content = src(`${dir}/${entry}`)
      assert.doesNotMatch(
        content,
        /:global\(\s*html\[data-admin-appearance=["']true["']\]\s*\)(?!\s*[,{])/,
        `${dir}/${entry} must wrap the full descendant selector inside :global(html[data-admin-appearance="true"] ...)`
      )
    }
  }
})

test('ScreenManagerScreen.vue uses library.back_btn alongside arrowLeft icon and teams-activity.css themes .teams-notice', () => {
  const { descriptor, errors } = parse(src('src/views/ScreenManagerScreen.vue'))
  assert.deepEqual(errors, [])

  const template = descriptor.template?.content || ''
  assert.match(template, /<AppIcon\s+name="arrowLeft"/)
  assert.match(template, /\$t\('library\.back_btn'\)/)
  assert.doesNotMatch(template, /\$t\('admin\.details\.back_btn'\)/)

  const teamsCss = src('src/styles/teams-activity.css')
  assert.match(
    teamsCss,
    /html\[data-admin-appearance=["']true["']\][^{]*\.teams-notice\s*\{/,
    'teams-activity.css must theme .teams-notice under html[data-admin-appearance="true"]'
  )
})

test('admin-theme.css and SimpleQuiz host/judge screens preserve contrast, urgency cues, form theming, and screen isolation', () => {
  const themeCss = src('src/assets/admin-theme.css')
  assert.match(
    themeCss,
    /\.quiz-host-screen\s+\.question-card\.has-image\s+\.question-text\s*\{[^}]*color:\s*#ffffff\s*!important/s
  )
  assert.match(
    themeCss,
    /\.quiz-host-screen\s+\.timer\.timer-low\s*\{[^}]*border-color:\s*var\(--admin-danger\)\s*!important;[^}]*color:\s*var\(--admin-danger\)\s*!important/s
  )
  assert.match(
    themeCss,
    /\.quiz-host-screen\s+\.answer-counter-widget\s*\{[^}]*background:\s*var\(--admin-surface\)\s*!important/s
  )
  assert.match(
    themeCss,
    /\.quiz-host-screen\s+\.counter-label\s*\{[^}]*color:\s*var\(--admin-text-muted\)\s*!important/s
  )
  assert.match(
    themeCss,
    /\.quiz-host-screen\s*,\s*\.judge-screen\s*\)\s*:is\(input,\s*textarea,\s*select\)/s
  )

  const simpleHost = src('src/views/SimpleQuizHostScreen.vue')
  assert.match(simpleHost, /class="quiz-host-screen simple-quiz-host-screen"/)
  assert.doesNotMatch(simpleHost, /:global\(\s*html\[data-admin-appearance=["']true["']\]\s+\.quiz-host-screen\b/)
  assert.match(simpleHost, /:global\(html\[data-admin-appearance="true"\] \.simple-quiz-host-screen \.host-error\)/)
  assert.match(simpleHost, /:global\(html\[data-admin-appearance="true"\] \.simple-quiz-host-screen \.judge-url\)/)

  const simpleJudge = src('src/views/SimpleQuizJudgeScreen.vue')
  assert.match(
    simpleJudge,
    /:global\(html\[data-admin-appearance="true"\] \.judge-screen \.q-badge\)\s*\{[^}]*background:\s*var\(--admin-primary-solid\)\s*!important;[^}]*color:\s*#fff\s*!important/s
  )
})
