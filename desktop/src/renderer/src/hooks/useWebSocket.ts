/**
 * React hooks for WebSocket lifecycle management.
 *
 * useWebSocket: Connects/disconnects the WebSocket manager based on auth state.
 * useWSSubscription: Subscribes to a specific message type with automatic cleanup.
 */
import { useEffect, useSyncExternalStore } from 'react'
import { wsManager, type WSConnectionState, type WSMessageHandler } from '@/api/websocket'
import { useAuthStore } from '@/stores/auth'

/**
 * Manage WebSocket connection lifecycle based on authentication state.
 * Call this once in the app shell -- it auto-connects when authenticated
 * and disconnects on logout.
 */
export function useWebSocket() {
  const accessToken = useAuthStore((s) => s.accessToken)
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)

  useEffect(() => {
    if (isAuthenticated && accessToken) {
      wsManager.connect(accessToken)
    } else {
      wsManager.disconnect()
    }
  }, [isAuthenticated, accessToken])
}

/**
 * Subscribe to a specific WebSocket message type.
 * Automatically unsubscribes on unmount or when dependencies change.
 *
 * @param type - The message type to listen for (e.g., 'notification.new')
 * @param handler - Callback invoked when a matching message arrives
 */
export function useWSSubscription(type: string, handler: WSMessageHandler) {
  useEffect(() => {
    const unsubscribe = wsManager.on(type, handler)
    return unsubscribe
  }, [type, handler])
}

// Stable references for useSyncExternalStore (avoids re-subscribe on every render)
const subscribeWSState = (cb: () => void) => wsManager.subscribe(cb)
const getWSStateSnapshot = () => wsManager.getState()

/**
 * Reactively track the WebSocket connection state.
 * Returns 'disconnected' | 'connecting' | 'connected'.
 */
export function useWSConnectionState(): WSConnectionState {
  return useSyncExternalStore(subscribeWSState, getWSStateSnapshot)
}
