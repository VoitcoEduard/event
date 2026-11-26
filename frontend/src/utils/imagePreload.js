import { getImageUrl } from '../services/media'

export class ImagePreloadManager {
    constructor() {
        this.preloadedImages = new Map();
        this.currentLoading = 0;
        this.maxConcurrent = 3;
    }

    extractImageUrls(question) {
        if (!question) return [];
        const urls = [];

        if (typeof question === 'string') {
            urls.push(question);
            return urls;
        }

        if (question.imageUrl) {
            urls.push(question.imageUrl);
        }
        if (question.thumbUrl) {
            urls.push(question.thumbUrl);
        }
        if (question.imageId) {
            const url = getImageUrl(question.imageId);
            if (url) urls.push(url);
        }
        if (Array.isArray(question.imageOptions)) {
            question.imageOptions.forEach(opt => {
                if (opt?.imageId) {
                    const url = getImageUrl(opt.imageId);
                    if (url) urls.push(url);
                } else if (opt?.imageUrl) {
                    urls.push(opt.imageUrl);
                }
            });
        }
        return urls;
    }

    async preloadQuestions(questions, onProgress) {
        if (!questions || questions.length === 0) return;

        const allUrls = [];
        questions.forEach(q => {
            const urls = this.extractImageUrls(q);
            urls.forEach(u => {
                if (u && !allUrls.includes(u)) {
                    allUrls.push(u);
                }
            });
        });

        if (allUrls.length === 0) return;

        const total = allUrls.length;
        let loaded = 0;

        for (let i = 0; i < total; i++) {
            const url = allUrls[i];

            if (i === 0) {
                await this.loadImage(url).catch(() => {});
                loaded++;
                if (onProgress) onProgress(loaded / total);
                continue;
            }

            this.loadInBackground(url).then(() => {
                loaded++;
                if (onProgress) onProgress(loaded / total);
            }).catch(err => {
                console.warn('Failed to preload image:', url, err);
                loaded++;
                if (onProgress) onProgress(loaded / total);
            });
        }
    }

    async preloadQuestion(question) {
        const urls = this.extractImageUrls(question);
        return Promise.all(urls.map(url => this.loadImage(url).catch(() => null)));
    }

    async loadImage(url) {
        if (!url) return null;
        if (this.preloadedImages.has(url)) {
            return this.preloadedImages.get(url);
        }
        return new Promise((resolve, reject) => {
            const img = new Image();
            img.onload = () => {
                this.preloadedImages.set(url, img);
                resolve(img);
            };
            img.onerror = (err) => {
                reject(err);
            };
            img.src = url;
        });
    }

    async loadInBackground(url) {
        if (!url) return null;
        if (this.preloadedImages.has(url)) {
            return this.preloadedImages.get(url);
        }

        while (this.currentLoading >= this.maxConcurrent) {
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

    clear() {
        this.preloadedImages.clear();
    }
}

export const preloader = new ImagePreloadManager();

