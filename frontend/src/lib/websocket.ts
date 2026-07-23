type WSEventHandler = (data: any) => void

class WebSocketService {
  private ws: WebSocket | null = null
  private handlers: Map<string, WSEventHandler[]> = new Map()
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 5000
  private intentionalClose = false
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private reconnectPausedUntil = 0

  connect() {
    if (Date.now() < this.reconnectPausedUntil) {
      return
    }

    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return
    }

    if (this.ws) {
      this.intentionalClose = true
      this.ws.onclose = null
      this.ws.close()
    }

    this.intentionalClose = false
    const WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:3001'
    this.ws = new WebSocket(`${WS_URL}/ws`)

    this.ws.onopen = () => {
      console.log('WebSocket connected')
      this.reconnectAttempts = 0
    }

    this.ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data)
        const handlers = this.handlers.get(message.event) || []
        handlers.forEach((handler) => handler(message.data))
      } catch (error) {
        console.error('WebSocket message parse error:', error)
      }
    }

    this.ws.onclose = () => {
      console.log('WebSocket disconnected')
      if (this.intentionalClose) {
        this.intentionalClose = false
        return
      }
      this.attemptReconnect()
    }

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error)
    }
  }

  private attemptReconnect() {
    if (Date.now() < this.reconnectPausedUntil) return
    if (typeof navigator !== 'undefined' && !navigator.onLine) return
    if (this.reconnectTimer) return
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++
      const delay = Math.min(this.reconnectDelay * this.reconnectAttempts, 60000)
      this.reconnectTimer = setTimeout(() => {
        this.reconnectTimer = null
        console.log(`WebSocket reconnecting... attempt ${this.reconnectAttempts}`)
        this.connect()
      }, delay)
    }
  }

  pauseReconnect(ms = 60000) {
    this.reconnectPausedUntil = Date.now() + ms
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    if (this.ws && this.ws.readyState !== WebSocket.CLOSED) {
      this.intentionalClose = true
      this.ws.onclose = null
      this.ws.close()
    }
    this.ws = null
  }

  on(event: string, handler: WSEventHandler) {
    if (!this.handlers.has(event)) {
      this.handlers.set(event, [])
    }
    this.handlers.get(event)!.push(handler)
  }

  off(event: string, handler: WSEventHandler) {
    const handlers = this.handlers.get(event) || []
    this.handlers.set(event, handlers.filter((h) => h !== handler))
  }

  send(event: string, data: any) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ event, data }))
    }
  }

  joinConversation(conversationId: string) {
    this.send('join_conversation', { conversation_id: conversationId })
  }

  leaveConversation(conversationId: string) {
    this.send('leave_conversation', { conversation_id: conversationId })
  }

  sendTyping(conversationId: string, isTyping: boolean) {
    this.send('typing', { conversation_id: conversationId, is_typing: isTyping })
  }

  disconnect() {
    if (this.ws) {
      this.intentionalClose = true
      this.ws.onclose = null
      this.ws.close()
      this.ws = null
    }
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    this.reconnectAttempts = 0
    this.reconnectPausedUntil = 0
  }
}

export const wsService = new WebSocketService()
export default wsService
