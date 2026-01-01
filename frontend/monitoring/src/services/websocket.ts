/**
 * WebSocket service for real-time analysis updates
 */

export interface WSMessage {
  type: string
  instrument_id?: string
  data?: any
  error?: string
  timestamp: number
}

export type MessageHandler = (message: WSMessage) => void
export type ConnectionHandler = () => void
export type ErrorHandler = (error: Event | string) => void

export class WebSocketService {
  private ws: WebSocket | null = null
  private url: string
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 1000
  private reconnectTimer: number | null = null
  private pingInterval: number | null = null
  private isManualClose = false

  private messageHandlers: Set<MessageHandler> = new Set()
  private connectHandlers: Set<ConnectionHandler> = new Set()
  private disconnectHandlers: Set<ConnectionHandler> = new Set()
  private errorHandlers: Set<ErrorHandler> = new Set()

  private subscriptions: Set<string> = new Set()

  constructor(url: string) {
    this.url = url
  }

  /**
   * Connect to WebSocket server
   */
  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      console.warn('WebSocket already connected')
      return
    }

    this.isManualClose = false

    try {
      this.ws = new WebSocket(this.url)

      this.ws.onopen = () => {
        console.log('WebSocket connected')
        this.reconnectAttempts = 0
        this.reconnectDelay = 1000
        
        // Resubscribe to instruments after reconnection
        this.subscriptions.forEach(instId => {
          this.sendMessage({
            type: 'subscribe',
            instrument_id: instId,
            timestamp: Date.now()
          })
        })

        // Start ping interval
        this.startPing()

        // Notify connect handlers
        this.connectHandlers.forEach(handler => handler())
      }

      this.ws.onmessage = (event) => {
        try {
          const message: WSMessage = JSON.parse(event.data)
          
          // Handle ping/pong
          if (message.type === 'ping') {
            this.sendMessage({ type: 'pong', timestamp: Date.now() })
            return
          }

          // Notify message handlers
          this.messageHandlers.forEach(handler => handler(message))
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error)
          this.errorHandlers.forEach(handler => 
            handler(`Failed to parse message: ${error}`)
          )
        }
      }

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error)
        this.errorHandlers.forEach(handler => handler(error))
      }

      this.ws.onclose = () => {
        console.log('WebSocket disconnected')
        this.stopPing()
        this.disconnectHandlers.forEach(handler => handler())

        // Attempt reconnection if not manually closed
        if (!this.isManualClose) {
          this.scheduleReconnect()
        }
      }
    } catch (error) {
      console.error('Failed to create WebSocket:', error)
      this.errorHandlers.forEach(handler => 
        handler(`Failed to connect: ${error}`)
      )
      this.scheduleReconnect()
    }
  }

  /**
   * Disconnect from WebSocket server
   */
  disconnect(): void {
    this.isManualClose = true
    this.stopPing()
    
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }

    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  /**
   * Subscribe to instrument updates
   */
  subscribe(instrumentId: string): void {
    this.subscriptions.add(instrumentId)
    
    if (this.isConnected()) {
      this.sendMessage({
        type: 'subscribe',
        instrument_id: instrumentId,
        timestamp: Date.now()
      })
    }
  }

  /**
   * Unsubscribe from instrument updates
   */
  unsubscribe(instrumentId: string): void {
    this.subscriptions.delete(instrumentId)
    
    if (this.isConnected()) {
      this.sendMessage({
        type: 'unsubscribe',
        instrument_id: instrumentId,
        timestamp: Date.now()
      })
    }
  }

  /**
   * Check if WebSocket is connected
   */
  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }

  /**
   * Register message handler
   */
  onMessage(handler: MessageHandler): () => void {
    this.messageHandlers.add(handler)
    return () => this.messageHandlers.delete(handler)
  }

  /**
   * Register connect handler
   */
  onConnect(handler: ConnectionHandler): () => void {
    this.connectHandlers.add(handler)
    return () => this.connectHandlers.delete(handler)
  }

  /**
   * Register disconnect handler
   */
  onDisconnect(handler: ConnectionHandler): () => void {
    this.disconnectHandlers.add(handler)
    return () => this.disconnectHandlers.delete(handler)
  }

  /**
   * Register error handler
   */
  onError(handler: ErrorHandler): () => void {
    this.errorHandlers.add(handler)
    return () => this.errorHandlers.delete(handler)
  }

  private sendMessage(message: Partial<WSMessage>): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message))
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached')
      this.errorHandlers.forEach(handler => 
        handler('Max reconnection attempts reached')
      )
      return
    }

    this.reconnectAttempts++
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1)
    
    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`)
    
    this.reconnectTimer = window.setTimeout(() => {
      this.connect()
    }, delay)
  }

  private startPing(): void {
    this.stopPing()
    this.pingInterval = window.setInterval(() => {
      if (this.isConnected()) {
        this.sendMessage({ type: 'pong', timestamp: Date.now() })
      }
    }, 30000) // Send pong every 30 seconds
  }

  private stopPing(): void {
    if (this.pingInterval) {
      clearInterval(this.pingInterval)
      this.pingInterval = null
    }
  }
}

/**
 * Create WebSocket service instance for analysis updates
 */
export function createAnalysisWebSocket(): WebSocketService {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = import.meta.env.VITE_WS_URL || window.location.host.replace(':5173', ':8080')
  const url = `${protocol}//${host}/ws/analysis`
  
  return new WebSocketService(url)
}
