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

// Active template-library surfaces must tell the user when loading failed;
// silently showing an empty list looks like "no templates saved".
for (const rel of ['src/views/AdminModuleEditorScreen.vue', 'src/components/AddModuleModal.vue']) {
  test(`${rel} surfaces template loading failures`, () => {
    const sfc = parsedSfc(rel)
    assert.ok(sfc.scriptSetup, `${rel} must use <script setup>`)
    const script = sfc.scriptSetup.content

    // A failed or non-OK /templates request must flip the error flag… (the
    // editor screen uses library/templateError, the modal templatesLoadError).
    const errorFlag = rel.endsWith('AdminModuleEditorScreen.vue') ? 'templateError' : 'templatesLoadError'
    assert.match(script, new RegExp(`${errorFlag}\\.value = false`))
    assert.match(script, /throw new Error\(`HTTP \$\{(?:response|res)\.status\}`\)/)
    assert.match(script, new RegExp(`catch[^\\{]*\\{[\\s\\S]*?${errorFlag}\\.value = true`))

    // …and the surface must render it instead of the empty-list message.
    const template = sfc.template.content
    assert.match(template, new RegExp(`v-else-if="${errorFlag}"`))
    if (rel.endsWith('AddModuleModal.vue')) {
      assert.match(template, /add_module\.errors\.templates_load_failed/)
    } else {
      assert.match(template, /quiz_config\.template_load_failed/)
    }
  })
}

test('all locales define the template-load-failed messages', () => {
  for (const loc of ['en', 'ru', 'ro']) {
    const messages = JSON.parse(src(`src/locales/${loc}.json`))
    for (const key of ['quiz_config.template_load_failed', 'add_module.errors.templates_load_failed']) {
      const value = key.split('.').reduce((acc, part) => acc?.[part], messages)
      assert.ok(
        typeof value === 'string' && value.trim().length > 0,
        `${loc}.json must define a non-empty ${key} message`
      )
    }
  }
})
