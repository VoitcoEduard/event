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
        this.userId = null
        this.isHost = false
        this.isProjector = false
        this.eventId = null
        this.intentionalDisconnect = false
    }

    connect(moduleId = null, nickname = null, isHost = false, isProjector = false, eventId = null) {
        // Store connection params for reconnection
        this.moduleId = moduleId
        this.nickname = nickname
        this.isHost = isHost
        this.isProjector = isProjector
        this.eventId = eventId
        this.intentionalDisconnect = false
        this.userId = getUserId() // Get or generate userId

        // Determine WS URL.
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

        if (this.socket) {
            this.socket.close()
        }

        const socket = new WebSocket(wsUrl)
        this.socket = socket

        socket.onopen = () => {
            if (this.socket !== socket) return
            this.isConnected = true
            this.reconnectAttempts = 0
            // Configure signal queue sender
            signalQueue.setSender((signal) => {
                this.send('player:signal', signal)
            })
            signalQueue.onReconnect()

            // Send join message based on client type
            let type
            if (this.isProjector) {
                type = 'projector:join'
            } else if (this.isHost) {
                type = 'host:join'
            } else {
                type = 'player:join'
            }

            const payload = { moduleId: this.moduleId }
            if (this.eventId) {
                payload.eventId = this.eventId
            }
            if (!this.isHost && !this.isProjector) {
                payload.nickname = this.nickname
                payload.userId = this.userId
            }

            this.send(type, payload)
            // Consumers can now safely send follow-up messages (such as
            // player:reconnect) because the server has associated this socket
            // with its player/session.
            this.dispatch('open')
        }

        socket.onmessage = (event) => {
            if (this.socket !== socket) return
            const messages = event.data.split('\n').filter(msg => msg.trim())
            messages.forEach(messageStr => {
                try {
                    const message = JSON.parse(messageStr)
                    this.dispatch(message.type, message.payload)
                } catch (e) {
                    console.error('Error parsing WS message:', e, 'Raw:', messageStr)
                }
            })
        }

        socket.onclose = (event) => {
            if (this.socket !== socket) return
            this.isConnected = false
            signalQueue.onDisconnect()
			if (event.code === 1008) {
				this.intentionalDisconnect = true
				this.dispatch('auth:required')
				window.location.assign('/login')
				return
			}

            if (!this.intentionalDisconnect && this.reconnectAttempts < 5) {
                const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 5000)
                setTimeout(() => {
                    this.reconnectAttempts++
                    this.connect(this.moduleId, this.nickname, this.isHost, this.isProjector, this.eventId)
                }, delay)
            }

            this.dispatch('disconnect')
        }

        socket.onerror = (error) => {
            if (this.socket !== socket) return
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
        this.intentionalDisconnect = true
        this.reconnectAttempts = 0
        if (this.socket) {
            this.socket.close()
            this.socket = null
        }
    }

    clearHandlers() {
        this.handlers.clear()
    }
}

// Singleton instance export
export const quizWS = new QuizWebSocket()
