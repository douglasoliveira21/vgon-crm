type WSEventHandler = (data: any) => void

class WebSocketService {
  private ws: WebSocket | null = null
  private handlers: Map<string, WSEventHandler[]> = new Map()
  private reconnectAttempts = 0
  private maxReconnectAttempts = 50
  private reconnectDelay = 2000

  connect(token: string) {
    const WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:3001'
    this.ws = new WebSocket(`${WS_URL}/ws?token=${token}`)

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
      this.attemptReconnect(token)
    }

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error)
    }
  }

  private attemptReconnect(token: string) {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++
      setTimeout(() => {
        console.log(`WebSocket reconnecting... attempt ${this.reconnectAttempts}`)
        this.connect(token)
      }, this.reconnectDelay * this.reconnectAttempts)
    }
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
      this.ws.close()
      this.ws = null
    }
  }
}

export const wsService = new WebSocketService()
export default wsService
