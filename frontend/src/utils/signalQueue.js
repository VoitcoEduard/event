class SignalQueue {
    constructor() {
        this.queue = []
        this.isConnected = true
    }

    enqueue(signal) {
		this.queue.push(signal)
        if (this.isConnected && this.sender) {
            this.flushSingle(signal)
        }
    }

    flush() {
        if (!this.isConnected || !this.sender) {
            return
        }

        for (const signal of this.queue) {
            this.flushSingle(signal)
        }
    }

    setSender(senderFunc) {
        this.sender = senderFunc
    }

    flushSingle(signal) {
        if (this.sender) {
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

	settle(submissionId) {
		this.queue = this.queue.filter(signal => signal.submissionId !== submissionId)
	}

    clear() {
        this.queue = []
    }
}

export const signalQueue = new SignalQueue()
