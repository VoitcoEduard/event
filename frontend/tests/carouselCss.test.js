import { describe, it } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const componentPath = path.join(
    path.dirname(fileURLToPath(import.meta.url)),
    '../src/components/HorizontalCarousel.vue'
)

// Extract top-level rule selectors from the component's scoped style block.
const extractSelectors = () => {
    const source = readFileSync(componentPath, 'utf8')
    const styleMatch = source.match(/<style[^>]*>([\s\S]*?)<\/style>/)
    assert.ok(styleMatch, 'HorizontalCarousel.vue should contain a style block')
    const css = styleMatch[1].replace(/\/\*[\s\S]*?\*\//g, '')
    const selectors = []
    for (const rule of css.matchAll(/(^|})\s*([^{}@]+?)\s*\{/g)) {
        selectors.push(rule[2].trim())
    }
    return selectors
}

describe('HorizontalCarousel CSS has no duplicate selector declarations', () => {
    it('declares .carousel-track exactly once', () => {
        const count = extractSelectors().filter(s => s === '.carousel-track').length
        assert.equal(count, 1)
    })

    it('declares .carousel-item exactly once', () => {
        const count = extractSelectors().filter(s => s === '.carousel-item').length
        assert.equal(count, 1)
    })

    it('keeps the effective track padding and item sizing from the winning rules', () => {
        const source = readFileSync(componentPath, 'utf8')
        const style = source.match(/<style[^>]*>([\s\S]*?)<\/style>/)[1].replace(/\/\*[\s\S]*?\*\//g, '')
        const trackRule = style.match(/\.carousel-track\s*\{([^}]*)\}/)[1]
        const itemRule = style.match(/\.carousel-item\s*\{([^}]*)\}/)[1]

        assert.match(trackRule, /padding:\s*0\s*;/)
        assert.doesNotMatch(trackRule, /padding:\s*0\s+50%/)

        assert.match(itemRule, /flex:\s*0\s+0\s+100%\s*;/)
        assert.match(itemRule, /width:\s*100%\s*;/)
        assert.doesNotMatch(itemRule, /flex:\s*0\s+0\s+80%/)
    })
})
