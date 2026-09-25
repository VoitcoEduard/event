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

  // localMarkers must be re-synced whenever the parent replaces initialMarkers
  // (new-question reset, scored guesses after submit).
  assert.match(script, /watch\(\(\)\s*=>\s*props\.initialMarkers/)
  assert.match(script, /localMarkers\.value\s*=\s*\[\.\.\.\(?Array\.isArray\(newVal\)/)
})
