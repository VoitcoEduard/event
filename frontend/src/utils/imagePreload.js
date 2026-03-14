// Chunked image preloader for memory efficiency
export class ImagePreloadManager {
    constructor() {
        this.preloadedImages = new Map();
        this.loadingQueue = [];
        this.currentLoading = 0;
        this.maxConcurrent = 3;
    }

    // Preload questions sequentially with background limits
    async preloadQuestions(questions, onProgress) {
        if (!questions || questions.length === 0) return;

        const total = questions.length;
        let loaded = 0;

        for (let i = 0; i < total; i++) {
            const question = questions[i];
            if (!question.imageUrl) {
                loaded++;
                continue;
            }

            // Load first question immediately and wait
            if (i === 0) {
                await this.loadImage(question.imageUrl);
                loaded++;
                if (onProgress) onProgress(loaded / total);
                continue;
            }

            // Background load remaining questions
            this.loadInBackground(question.imageUrl).then(() => {
                loaded++;
                if (onProgress) onProgress(loaded / total);
            }).catch(err => {
                console.error('Failed to preload:', question.imageUrl, err);
                loaded++; // count as processed to avoid hanging progress
                if (onProgress) onProgress(loaded / total);
            });
        }
    }

    async loadImage(url) {
        return new Promise((resolve, reject) => {
            // By using a fetch with 'no-store' or a native Image object,
            // it hits the backend chunk streamer.
            const img = new Image();
            img.onload = () => {
                this.preloadedImages.set(url, img);
                resolve(img);
            };
            img.onerror = reject;
            // Append a tiny token or just use it if relying on session cookies
            img.src = url;
        });
    }

    async loadInBackground(url) {
        if (this.preloadedImages.has(url)) {
            return this.preloadedImages.get(url);
        }

        // Wait for available slot
        while (this.currentLoading >= this.maxConcurrent) {
            // Simple poll (not perfect but decent for background)
            await new Promise(r => setTimeout(r, 100));
        }

        this.currentLoading++;
        try {
            return await this.loadImage(url);
        } finally {
            this.currentLoading--;
        }
    }

    getImage(url) {
        return this.preloadedImages.get(url);
    }
}

// Export singleton instance
export const preloader = new ImagePreloadManager();
