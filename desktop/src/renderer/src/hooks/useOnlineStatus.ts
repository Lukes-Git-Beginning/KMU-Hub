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
import { drain } from '@/api/offline-queue'
import { API_BASE_URL } from '@/lib/constants'

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

      // Drain queued offline mutations FIRST, then invalidate queries so the
      // refetch picks up the freshly drained server state. Running both in
      // parallel raced: invalidate fired before drain finished, so the UI
      // showed pre-drain data until the next polling tick.
      // Drain errors are non-fatal; the OfflineBanner provides manual retry.
      const { accessToken: tokenForDrain } = useAuthStore.getState()
      drain(API_BASE_URL, () => tokenForDrain)
        .catch(() => undefined)
        .finally(() => {
          queryClient.invalidateQueries().catch(() => undefined)
        })

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
