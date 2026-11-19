import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import { parse } from '@vue/compiler-sfc'

const here = path.dirname(fileURLToPath(import.meta.url))
const src = (rel) => readFileSync(path.join(here, '..', rel), 'utf8')

test('all locales define mobile library labels', () => {
  for (const loc of ['en', 'ru', 'ro']) {
    const messages = JSON.parse(src(`src/locales/${loc}.json`))
    for (const [label, value] of [
      ['library.back_btn', messages?.library?.back_btn],
      ['library.language_label', messages?.library?.language_label],
      ['library.actions.export_btn', messages?.library?.actions?.export_btn]
    ]) {
      assert.ok(
        typeof value === 'string' && value.trim().length > 0,
        `${loc}.json must define a non-empty ${label}`
      )
    }
  }
})

test('QuizLibraryScreen.vue parses cleanly and supports accessible vertical mobile layout', () => {
  const source = src('src/views/QuizLibraryScreen.vue')
  const { descriptor, errors } = parse(source)
  assert.deepEqual(errors, [])

  const template = descriptor.template?.content || ''
  assert.match(template, /<header\b/)
  assert.match(template, /<main\b/)
  assert.match(template, /<h1\b/)
  assert.match(template, /role="tablist"/)
  assert.match(template, /role="tab"/)
  assert.match(template, /role="tabpanel"/)
  assert.match(template, /aria-selected/)

  const styles = descriptor.styles.map(s => s.content).join('\n')
  assert.match(styles, /container:\s*library\s*\/\s*inline-size/)
  assert.match(styles, /100dvh/)
  assert.match(styles, /pointer:\s*coarse/)
  assert.match(styles, /minmax\(min\(100%,/)
  assert.match(styles, /safe-area-inset-/)
})

test('admin-theme.css registers scroll fade properties and themes library mobile states', () => {
  const themeCss = src('src/assets/admin-theme.css')
  assert.match(themeCss, /@property\s+--fade-start/)
  assert.match(themeCss, /@property\s+--fade-end/)
  assert.match(themeCss, /\.screensaver-card/)
  assert.match(themeCss, /\.track-card/)
  assert.match(themeCss, /\.library-screen\s+\.tab-btn\[aria-selected="true"\]/)
})
