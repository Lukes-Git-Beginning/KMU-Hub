/**
 * WebSocket manager for real-time communication.
 *
 * Handles connection lifecycle, message routing, and automatic reconnection
 * with exponential backoff. Used by the auth store and useWebSocket hook.
 */
import { WS_RECONNECT_DELAYS, MAX_RECONNECT_ATTEMPTS, API_BASE_URL } from '@/lib/constants'

export type WSConnectionState = 'disconnected' | 'connecting' | 'connected'

export type WSMessageHandler = (data: Record<string, unknown>) => void

export class WebSocketManager {
  private ws: WebSocket | null = null
  private handlers = new Map<string, Set<WSMessageHandler>>()
  private stateListeners = new Set<() => void>()
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private reconnectAttempt = 0
  private baseUrl: string
  private currentToken: string | null = null
  private state: WSConnectionState = 'disconnected'

  constructor(baseUrl: string) {
    // Convert http(s) to ws(s)
    this.baseUrl = baseUrl.replace(/^http/, 'ws')
  }

  /** Connect to the WebSocket server with the given access token. */
  connect(accessToken: string): void {
    // If already connected or connecting with same token, skip
    if (this.ws && this.currentToken === accessToken && this.state !== 'disconnected') {
      return
    }

    // Clean up any existing connection
    this.cleanupSocket()
    this.clearReconnectTimer()

    this.currentToken = accessToken
    this.setState('connecting')
    this.reconnectAttempt = 0

    this.createConnection()
  }

  /** Disconnect and stop reconnection attempts. */
  disconnect(): void {
    this.currentToken = null
    this.setState('disconnected')
    this.reconnectAttempt = 0
    this.clearReconnectTimer()
    this.cleanupSocket()
  }

  /**
   * Register a handler for a specific message type.
   * Returns an unsubscribe function.
   */
  on(type: string, handler: WSMessageHandler): () => void {
    let typeHandlers = this.handlers.get(type)
    if (!typeHandlers) {
      typeHandlers = new Set()
      this.handlers.set(type, typeHandlers)
    }
    typeHandlers.add(handler)

    return () => {
      typeHandlers!.delete(handler)
      if (typeHandlers!.size === 0) {
        this.handlers.delete(type)
      }
    }
  }

  /** Send a message over the WebSocket connection. */
  send(message: Record<string, unknown>): void {
    if (this.ws && this.state === 'connected') {
      this.ws.send(JSON.stringify(message))
    }
  }

  /** Get the current connection state. */
  getState(): WSConnectionState {
    return this.state
  }

  /**
   * Subscribe to connection state changes.
   * Returns an unsubscribe function. Compatible with useSyncExternalStore.
   */
  subscribe(listener: () => void): () => void {
    this.stateListeners.add(listener)
    return () => {
      this.stateListeners.delete(listener)
    }
  }

  private setState(newState: WSConnectionState): void {
    if (this.state !== newState) {
      this.state = newState
      for (const listener of this.stateListeners) {
        listener()
      }
    }
  }

  private createConnection(): void {
    if (!this.currentToken) return

    const wsUrl = `${this.baseUrl}/api/v1/ws`

    try {
      // Pass token via Sec-WebSocket-Protocol header instead of query parameter
      // to prevent token leakage in access logs and caches
      this.ws = new WebSocket(wsUrl, ['access_token', this.currentToken])
    } catch {
      this.scheduleReconnect()
      return
    }

    this.ws.onopen = () => {
      this.setState('connected')
      this.reconnectAttempt = 0
    }

    this.ws.onmessage = (event: MessageEvent) => {
      try {
        const data = JSON.parse(event.data as string) as Record<string, unknown>
        const type = data.type as string | undefined
        if (type) {
          const typeHandlers = this.handlers.get(type)
          if (typeHandlers) {
            for (const handler of typeHandlers) {
              handler(data)
            }
          }
        }
      } catch {
        // Ignore malformed messages
      }
    }

    this.ws.onclose = () => {
      this.setState('disconnected')
      this.ws = null
      // Only reconnect if we haven't been explicitly disconnected
      if (this.currentToken) {
        this.scheduleReconnect()
      }
    }

    this.ws.onerror = () => {
      // onerror is always followed by onclose, so reconnect logic runs there
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempt >= MAX_RECONNECT_ATTEMPTS) {
      this.setState('disconnected')
      return
    }

    const delayIndex = Math.min(this.reconnectAttempt, WS_RECONNECT_DELAYS.length - 1)
    const delay = WS_RECONNECT_DELAYS[delayIndex]

    this.reconnectTimer = setTimeout(() => {
      this.reconnectAttempt++
      this.setState('connecting')
      this.createConnection()
    }, delay)
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
  }

  private cleanupSocket(): void {
    if (this.ws) {
      // Remove handlers before closing to avoid triggering reconnect
      this.ws.onopen = null
      this.ws.onmessage = null
      this.ws.onclose = null
      this.ws.onerror = null
      if (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING) {
        this.ws.close()
      }
      this.ws = null
    }
  }
}

/** Singleton WebSocket manager instance.
 *  Takes the resolved API base rather than re-reading the env var: in the web
 *  build the address comes from the origin, and a second copy of this logic
 *  would leave the socket pointing at localhost. */
export const wsManager = new WebSocketManager(API_BASE_URL)
