import { API_BASE_URL, getUserId } from '../utils'
import { signalQueue } from '../utils/signalQueue'

class QuizWebSocket {
    constructor() {
        this.socket = null
        this.handlers = new Map()
        this.isConnected = false
        this.reconnectAttempts = 0
        this.moduleId = null
        this.nickname = null
        this.userId = null // Store userId
        this.isHost = false
        this.isBigScreen = false
        this.eventId = null
        this.intentionalDisconnect = false
    }

    connect(moduleId, nickname = '', isHost = false, isBigScreen = false, eventId = null) {
        // Mark as intentional if reconnecting while already connected
        if (this.socket) {
            this.intentionalDisconnect = true
            this.socket.close()
        }

        this.moduleId = moduleId
        this.nickname = nickname
        this.userId = getUserId()
        this.isHost = isHost
        this.isBigScreen = isBigScreen
        this.eventId = eventId
        this.intentionalDisconnect = false

        // Determine WS URL.
        // If API_BASE_URL is empty (same-origin build), derive from current page location.
        let protocol, host
        if (API_BASE_URL) {
            const apiUrl = new URL(API_BASE_URL)
            protocol = apiUrl.protocol === 'https:' ? 'wss:' : 'ws:'
            host = apiUrl.host
        } else {
            protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
            host = window.location.host
        }
        const wsUrl = `${protocol}//${host}/ws`

        console.log('Connecting to WebSocket:', wsUrl)

        this.socket = new WebSocket(wsUrl)

        this.socket.onopen = () => {
            console.log('WebSocket Connected')
            this.isConnected = true
            this.reconnectAttempts = 0

            // Configure signal queue sender
            signalQueue.setSender((signal) => {
                this.send('player:signal', signal)
            })
            signalQueue.onReconnect()

            // Send join message based on client type
            let type
            if (isBigScreen) {
                type = 'bigscreen:join'
            } else if (isHost) {
                type = 'host:join'
            } else {
                type = 'player:join'
            }

            const payload = { moduleId }
            if (eventId) {
                payload.eventId = eventId
            }
            if (!isHost && !isBigScreen) {
                payload.nickname = nickname
                payload.userId = this.userId
            }

            this.send(type, payload)
        }

        this.socket.onmessage = (event) => {
            // Split by newline to handle multiple messages concatenated in one frame
            const messages = event.data.split('\n').filter(msg => msg.trim())
            messages.forEach(messageStr => {
                try {
                    const message = JSON.parse(messageStr)
                    // Dispatch logic is already defined in the original file
                    this.dispatch(message.type, message.payload)
                } catch (e) {
                    console.error('Error parsing WS message:', e, 'Raw:', messageStr)
                }
            })
        }

        this.socket.onclose = () => {
            console.log('WebSocket Disconnected')
            this.isConnected = false
            signalQueue.onDisconnect()

            // Only reconnect if not an intentional disconnect
            if (!this.intentionalDisconnect && this.reconnectAttempts < 5) {
                const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 5000)
                console.log(`Reconnecting in ${delay}ms... (Attempt ${this.reconnectAttempts + 1})`)
                setTimeout(() => {
                    this.reconnectAttempts++
                    this.connect(this.moduleId, this.nickname, this.isHost, this.isBigScreen, this.eventId)
                }, delay)
            }

            this.dispatch('disconnect')
        }

        this.socket.onerror = (error) => {
            console.error('WebSocket Error:', error)
        }
    }

    send(type, payload) {
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
            this.socket.send(JSON.stringify({ type, payload }))
        } else {
            console.warn('WebSocket not connected, cannot send:', type)
        }
    }

    on(type, handler) {
        if (!this.handlers.has(type)) {
            this.handlers.set(type, [])
        }
        this.handlers.get(type).push(handler)
    }

    off(type, handler) {
        if (!this.handlers.has(type)) return
        const handlers = this.handlers.get(type)
        if (!handlers) return
        const index = handlers.indexOf(handler)
        if (index !== -1) {
            handlers.splice(index, 1)
        }
    }

    dispatch(type, payload) {
        if (this.handlers.has(type)) {
            this.handlers.get(type).forEach(handler => handler(payload))
        }
    }

    disconnect() {
        console.log('[DEBUG] disconnect: Called, intentionalDisconnect=true')
        this.intentionalDisconnect = true
        this.reconnectAttempts = 0
        if (this.socket) {
            this.socket.close()
            this.socket = null
        }
        console.log('[DEBUG] disconnect: Socket closed and nulled')
    }

    clearHandlers() {
        this.handlers.clear()
    }
}

// Singleton instance export
export const quizWS = new QuizWebSocket()
