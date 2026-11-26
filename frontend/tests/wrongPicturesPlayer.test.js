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

test('MapCoordinatePicker stays in sync with parent markers (no stale pins)', () => {
  const script = scriptSetupOf('src/components/MapCoordinatePicker.vue')

  assert.match(script, /watch\(\(\)\s*=>\s*props\.initialMarkers/)
  assert.match(script, /localMarkers\.value\s*=\s*\[\.\.\.\(?Array\.isArray\(newVal\)/)
})

test('player timer auto-submits placed wrong-pictures pins on expiry', () => {
  const script = scriptSetupOf('src/views/QuizPlayerScreen.vue')

  assert.match(script, /const autoSubmitOnTimeout = \(\) => \{/)
  assert.match(script, /timeRemaining\.value = Math\.max\(0, Math\.ceil\(\(deadline - Date\.now\(\)\) \/ 1000\)\)\s+if \(timeRemaining\.value <= 0\) \{\s+clearInterval\(timerInterval\)\s+autoSubmitOnTimeout\(\)/)

  assert.match(script, /question\?\.type === 'wrong_pictures'/)
  assert.match(script, /!hasSubmitted\.value/)
  assert.match(script, /wpGuesses\.value\.some\(\(guess\) => guess\.isPlaced\)/)
  assert.match(script, /submitWrongPictures\(\)/)

  assert.match(script, /transition\(States\.QUESTION_ENDED\)/)
})
