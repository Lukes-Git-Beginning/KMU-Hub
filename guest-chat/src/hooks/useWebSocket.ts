import { useCallback, useEffect, useRef, useState } from 'react'
import type { WSMessage } from '../api/types'

export type ConnectionState = 'connecting' | 'connected' | 'disconnected'

export function useWebSocket(
  token: string,
  channelId: string,
  onMessage: (msg: WSMessage) => void,
) {
  const [connectionState, setConnectionState] = useState<ConnectionState>('disconnected')
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectAttemptRef = useRef(0)
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const heartbeatTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const onMessageRef = useRef(onMessage)
  onMessageRef.current = onMessage

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) return

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/api/v1/ws?guest_token=${encodeURIComponent(token)}`

    setConnectionState('connecting')
    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onopen = () => {
      setConnectionState('connected')
      reconnectAttemptRef.current = 0

      // Subscribe to channel
      ws.send(JSON.stringify({
        type: 'channel.subscribe',
        payload: { channel_id: channelId },
      }))

      // Start heartbeat
      heartbeatTimerRef.current = setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'ping' }))
        }
      }, 30_000)
    }

    ws.onmessage = (event) => {
      try {
        const msg: WSMessage = JSON.parse(event.data)
        onMessageRef.current(msg)
      } catch {
        // ignore malformed messages
      }
    }

    ws.onclose = () => {
      setConnectionState('disconnected')
      if (heartbeatTimerRef.current) {
        clearInterval(heartbeatTimerRef.current)
      }

      // Reconnect with exponential backoff + jitter
      const attempt = reconnectAttemptRef.current++
      const baseDelay = Math.min(1000 * Math.pow(2, attempt), 30_000)
      const jitter = Math.random() * 1000
      reconnectTimerRef.current = setTimeout(connect, baseDelay + jitter)
    }

    ws.onerror = () => {
      ws.close()
    }
  }, [token, channelId])

  useEffect(() => {
    if (!token || !channelId) return

    connect()

    return () => {
      if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current)
      if (heartbeatTimerRef.current) clearInterval(heartbeatTimerRef.current)
      reconnectAttemptRef.current = 0
      if (wsRef.current) {
        wsRef.current.onclose = null
        wsRef.current.close()
        wsRef.current = null
      }
    }
  }, [connect])

  const send = useCallback((msg: WSMessage) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(msg))
    }
  }, [])

  return { connectionState, send }
}
