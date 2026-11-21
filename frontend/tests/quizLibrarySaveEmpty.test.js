import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const here = path.dirname(fileURLToPath(import.meta.url))
const src = (rel) => readFileSync(path.join(here, '..', rel), 'utf8')

test('all locales define library.actions.save_empty_confirm distinct from discard_confirm (Bug #17)', () => {
  for (const loc of ['en', 'ro', 'ru']) {
    const messages = JSON.parse(src(`src/locales/${loc}.json`))
    const saveEmpty = messages?.library?.actions?.save_empty_confirm
    const discard = messages?.library?.actions?.discard_confirm
    assert.ok(
      typeof saveEmpty === 'string' && saveEmpty.trim().length > 0,
      `${loc}.json must define non-empty library.actions.save_empty_confirm`
    )
    assert.notEqual(
      saveEmpty,
      discard,
      `${loc}.json save_empty_confirm must be distinct from discard_confirm`
    )
  }
})

test('QuizLibraryScreen saveTemplate prompts save_empty_confirm instead of discard_confirm (Bug #17)', () => {
  const sfc = src('src/views/QuizLibraryScreen.vue')
  const saveFnMatch = sfc.match(/const saveTemplate = async \(\) => \{[\s\S]*?\n\}/)
  assert.ok(saveFnMatch, 'saveTemplate function must exist in QuizLibraryScreen.vue')
  const saveFn = saveFnMatch[0]
  assert.match(saveFn, /library\.actions\.save_empty_confirm/)
  assert.doesNotMatch(saveFn, /library\.actions\.discard_confirm/)
})
