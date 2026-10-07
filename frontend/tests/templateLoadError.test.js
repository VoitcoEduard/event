import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import { parse } from '@vue/compiler-sfc'

const here = path.dirname(fileURLToPath(import.meta.url))
const src = (rel) => readFileSync(path.join(here, '..', rel), 'utf8')

const parsedSfc = (rel) => {
  const { descriptor, errors } = parse(src(rel))
  assert.deepEqual(errors, [])
  return descriptor
}

// Both template-library modals must tell the user when loading failed;
// silently showing an empty list looks like "no templates saved".
for (const rel of ['src/components/FormModuleConfig.vue', 'src/components/QuizModuleConfig.vue']) {
  test(`${rel} surfaces template loading failures`, () => {
    const sfc = parsedSfc(rel)
    assert.ok(sfc.scriptSetup, `${rel} must use <script setup>`)
    const script = sfc.scriptSetup.content

    // A failed or non-OK /templates request must flip the error flag…
    assert.match(script, /templateError\.value = false/)
    assert.match(script, /throw new Error\(`HTTP \$\{(?:response|res)\.status\}`\)/)
    assert.match(script, /catch[^\{]*\{[\s\S]*?templateError\.value = true/)

    // …and the modal must render it instead of the empty-list message.
    const template = sfc.template.content
    assert.match(template, /v-else-if="templateError"/)
    assert.match(template, /quiz_config\.template_load_failed/)
  })
}

test('all locales define quiz_config.template_load_failed', () => {
  for (const loc of ['en', 'ru', 'ro']) {
    const messages = JSON.parse(src(`src/locales/${loc}.json`))
    const value = messages?.quiz_config?.template_load_failed
    assert.ok(
      typeof value === 'string' && value.trim().length > 0,
      `${loc}.json must define a non-empty quiz_config.template_load_failed message`
    )
  }
})
