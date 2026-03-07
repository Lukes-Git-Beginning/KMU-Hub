/**
 * Track browser/OS online/offline status with side effects.
 *
 * Returns { isOnline, wasOffline } that update reactively when the network
 * connection changes.
 *
 * Side effects on transitions:
 * - Online -> Offline: disconnect WebSocket gracefully, log warning
 * - Offline -> Online: reconnect WebSocket, invalidate all queries (refetch stale data)
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { wsManager } from '@/api/websocket'
import { useAuthStore } from '@/stores/auth'

export function useOnlineStatus() {
  const [isOnline, setIsOnline] = useState(navigator.onLine)
  const [wasOffline, setWasOffline] = useState(false)
  const prevOnlineRef = useRef(navigator.onLine)
  const queryClient = useQueryClient()

  const handleOnline = useCallback(() => {
    setIsOnline(true)

    // Transition: offline -> online
    if (!prevOnlineRef.current) {
      setWasOffline(true)

      // Reconnect WebSocket
      const { accessToken, isAuthenticated } = useAuthStore.getState()
      if (isAuthenticated && accessToken) {
        wsManager.connect(accessToken)
      }

      // Invalidate all queries to trigger refetch of stale data
      queryClient.invalidateQueries()

      // Clear "was offline" indicator after 3 seconds
      setTimeout(() => setWasOffline(false), 3000)
    }

    prevOnlineRef.current = true
  }, [queryClient])

  const handleOffline = useCallback(() => {
    setIsOnline(false)

    // Transition: online -> offline
    if (prevOnlineRef.current) {
      // Disconnect WebSocket gracefully to avoid reconnect loops
      wsManager.disconnect()
    }

    prevOnlineRef.current = false
  }, [])

   
  useEffect(() => {
    window.addEventListener('online', handleOnline)
    window.addEventListener('offline', handleOffline)

    return () => {
      window.removeEventListener('online', handleOnline)
      window.removeEventListener('offline', handleOffline)
    }
  }, [handleOnline, handleOffline])

  return { isOnline, wasOffline }
}
