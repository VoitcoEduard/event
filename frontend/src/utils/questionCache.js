class QuestionCache {
    constructor() {
        this.cache = new Map()
        this.maxCacheSize = 10 // Keep 10 questions max
    }

    addBatch(batch) {
        batch.questions.forEach((q, idx) => {
            const globalIndex = batch.startIndex + idx
            this.cache.set(globalIndex, q)
        })
        this.evictOld()
    }

    get(index) {
        return this.cache.get(index)
    }

    has(index) {
        return this.cache.has(index)
    }

    needsBatch(currentIndex) {
        // Request next batch when we're 2 questions from end of cache
        const maxCached = Math.max(...Array.from(this.cache.keys()), -1)
        return currentIndex >= maxCached - 2
    }

    evictOld() {
        // LRU eviction if cache too large
        if (this.cache.size > this.maxCacheSize) {
            const sorted = [...this.cache.entries()].sort((a, b) => a[0] - b[0])
            const toRemove = sorted.slice(0, sorted.length - this.maxCacheSize)
            toRemove.forEach(([key]) => this.cache.delete(key))
        }
    }
}

export const questionCache = new QuestionCache()