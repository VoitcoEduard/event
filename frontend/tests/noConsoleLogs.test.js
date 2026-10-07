import { describe, it } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const srcRoot = path.join(path.dirname(fileURLToPath(import.meta.url)), '..', 'src')

// Debug console.log statements must not ship in production code. Several of
// these ran on every WebSocket message (with JSON.stringify of the payload),
// adding per-message work and log spam. console.error and console.warn stay
// allowed: they mark error paths that are worth keeping.
const listSourceFiles = (dir) => {
    const entries = readdirSync(dir)
    const files = []
    for (const entry of entries) {
        const full = path.join(dir, entry)
        if (statSync(full).isDirectory()) {
            files.push(...listSourceFiles(full))
        } else if (/\.(js|vue)$/.test(entry)) {
            files.push(full)
        }
    }
    return files
}

const offenders = []
for (const file of listSourceFiles(srcRoot)) {
    const lines = readFileSync(file, 'utf8').split('\n')
    lines.forEach((line, i) => {
        // Ignore commented-out code so stale log lines don't fail the build.
        const code = line.replace(/\/\/.*$/, '')
        if (/console\.log\s*\(/.test(code)) {
            offenders.push(`${path.relative(srcRoot, file)}:${i + 1}`)
        }
    })
}

describe('production code contains no console.log debug statements', () => {
    it(`src/**/*.{js,vue} has zero console.log calls (found: ${offenders.length})`, () => {
        assert.deepEqual(offenders, [])
    })
})
