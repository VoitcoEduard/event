import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import { parse } from '@vue/compiler-sfc'

const here = path.dirname(fileURLToPath(import.meta.url))
const src = (rel) => readFileSync(path.join(here, '..', rel), 'utf8')

const scriptSetupOf = (rel) => {
  const { descriptor, errors } = parse(src(rel))
  assert.deepEqual(errors, [])
  assert.ok(descriptor.scriptSetup, `${rel} must use <script setup>`)
  return descriptor.scriptSetup.content
}

test('host screen releases preloaded images on unmount', () => {
  const script = scriptSetupOf('src/views/QuizHostScreen.vue')

  // The screen uses the preloader singleton; without clear() on unmount the
  // cache accumulates Image references across every hosted quiz for the
  // whole SPA session. ProjectorView already clears on exit; mirror it here.
  assert.match(script, /import \{ preloader \} from '[^']*imagePreload'/)
  assert.match(script, /onUnmounted\(\(\) => \{[\s\S]*?\bpreloader\.clear\(\)[\s\S]*?\}\)/)
})

test('both preloader consumers clear the cache when their screen ends', () => {
  const hostScript = scriptSetupOf('src/views/QuizHostScreen.vue')
  const projectorScript = scriptSetupOf('src/views/ProjectorView.vue')

  // ProjectorView clears via resetModuleState(); QuizHostScreen clears in
  // onUnmounted(). Together every consumer of the singleton releases its
  // cached images.
  assert.match(projectorScript, /\bpreloader\.clear\(\)/)
  assert.match(hostScript, /\bpreloader\.clear\(\)/)
})
