import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import { parse } from '@vue/compiler-sfc'

const here = path.dirname(fileURLToPath(import.meta.url))
const src = (rel) => readFileSync(path.join(here, '..', rel), 'utf8')

const rel = 'src/views/QuizPlayerScreen.vue'
const raw = src(rel)
const { descriptor, errors } = parse(raw)
assert.deepEqual(errors, [])
assert.ok(descriptor.scriptSetup, `${rel} must use <script setup>`)
const script = descriptor.scriptSetup.content
const template = descriptor.template.content

test('submitAnswer is idempotent: re-checks hasSubmitted before scoring', () => {
  // A pending confirmSelection timeout must not be able to score twice:
  // submitAnswer itself guards on the flag, not only its callers.
  assert.match(
    script,
    /const submitAnswer = \(answerData\) => \{\s*(\/\/[^\n]*\n\s*)*if \(hasSubmitted\.value\) return/
  )
  // Flag is still flipped before any scoring happens.
  const body = script.slice(script.indexOf('const submitAnswer'))
  const guardIdx = body.indexOf('if (hasSubmitted.value) return')
  const flagIdx = body.indexOf('hasSubmitted.value = true')
  const scoreIdx = body.indexOf('validateAndScore(answerData)')
  assert.ok(guardIdx > -1 && guardIdx < flagIdx && flagIdx < scoreIdx)
})

test('confirmSelection blocks re-entry during the 300ms delay', () => {
  // isSubmitting acts as an in-flight guard so a second click cannot queue
  // another timeout while the first is pending.
  assert.match(
    script,
    /const confirmSelection = \(\) => \{\s*if \(hasSubmitted\.value \|\| isSubmitting\.value\) return/
  )
  // The timeout callback re-checks hasSubmitted (flag may have flipped via
  // another submit path while the timeout was pending).
  assert.match(script, /if \(hasSubmitted\.value\) \{\s*isSubmitting\.value = false\s*return/)
})

test('pending confirmSelection timeout is tracked and cleaned up', () => {
  assert.match(script, /let confirmSelectionTimeout = null/)
  assert.match(script, /confirmSelectionTimeout = setTimeout\(/)
  // Cleared when a new question resets the player state…
  assert.match(script, /isSubmitting\.value = false\s*clearTimeout\(confirmSelectionTimeout\)\s*confirmSelectionTimeout = null/)
  // …and on unmount.
  assert.match(
    script,
    /onUnmounted\(\(\) => \{[^}]*clearTimeout\(confirmSelectionTimeout\)/
  )
})

test('carousel submit button is disabled while submitting', () => {
  assert.match(
    template,
    /:disabled="!hasPreselection \|\| hasSubmitted \|\| isSubmitting"/
  )
})
