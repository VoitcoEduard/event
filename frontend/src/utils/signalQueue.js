class SignalQueue {
    constructor() {
        this.queue = []
        this.isConnected = true
    }

    enqueue(signal) {
        if (this.isConnected) {
            this.flushSingle(signal)
        } else {
            this.queue.push(signal)
        }
    }

    flush() {
        if (!this.isConnected) {
            return
        }

        while (this.queue.length > 0) {
            const signal = this.queue.shift()
            this.flushSingle(signal)
        }
    }

    setSender(senderFunc) {
        this.sender = senderFunc
    }

    flushSingle(signal) {
        // Check if signal is still valid (within grace period)
        const now = Date.now()
        const signalTime = signal.timestamp || Date.now()

        // If signal is older than 10 seconds, discard it
        if (now - signalTime > 10000) {
            console.warn('Discarding expired signal:', signal)
            return
        }

        // Send signal to server
        if (this.sender) {
            console.log('Sending signal:', signal)
            this.sender(signal)
        } else {
            console.warn('No sender configured for signalQueue')
        }
    }

    onReconnect() {
        this.isConnected = true
        this.flush()
    }

    onDisconnect() {
        this.isConnected = false
    }

    clear() {
        this.queue = []
    }
}

export const signalQueue = new SignalQueue()