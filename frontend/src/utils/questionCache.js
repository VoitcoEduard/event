class QuestionCache {
    constructor() {
        this.cache = new Map()
        this.maxCacheSize = 50 // Match backend MaxQuestionsPerBatch
    }

    addBatch(batch) {
        console.log('[DEBUG] addBatch called with startIndex:', batch.startIndex, 'questions count:', batch.questions?.length)
        // Retain enough questions for the whole quiz once batch metadata is
        // known. Otherwise receiving batch N+1 evicts the question currently
        // on screen from batch N before the player can answer it.
        this.maxCacheSize = Math.max(
            this.maxCacheSize,
            (batch.totalBatches || 1) * (batch.questions?.length || 0)
        )
        batch.questions.forEach((q, idx) => {
            const globalIndex = batch.startIndex + idx
            console.log('[DEBUG] Storing question at index:', globalIndex, '(startIndex:', batch.startIndex, '+ idx:', idx, ')')
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

    clear() {
        this.cache.clear()
        console.log('[DEBUG] questionCache cleared')
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
