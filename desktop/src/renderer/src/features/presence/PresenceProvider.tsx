/**
 * Presence lifecycle provider.
 *
 * Wraps the app and manages:
 * 1. Heartbeat every 30 seconds via WebSocket
 * 2. Listening for 'presence.update' WebSocket events
 * 3. Page visibility detection (auto-away when tab hidden)
 * 4. Initial bulk presence fetch for known contacts
 *
 * Renders no UI of its own -- purely a side-effect provider.
 */
import { useEffect, useRef, useCallback, useMemo } from 'react'
import { wsManager } from '@/api/websocket'
import { useWSSubscription } from '@/hooks/useWebSocket'
import { usePresenceStore } from '@/stores/presence'
import { useBulkPresence } from '@/api/hooks/usePresence'
import { useAuthStore } from '@/stores/auth'
import type { PresenceLevel } from '@/api/video-types'

/** Heartbeat interval in milliseconds (30 seconds). */
const HEARTBEAT_INTERVAL = 30_000

/** Delay before marking user as away when tab is hidden (2 minutes). */
const AWAY_DELAY = 120_000

interface PresenceProviderProps {
  children: React.ReactNode
}

export function PresenceProvider({ children }: PresenceProviderProps) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const myStatus = usePresenceStore((s) => s.myStatus)
  const updatePresence = usePresenceStore((s) => s.updatePresence)
  const updateBulkPresence = usePresenceStore((s) => s.updateBulkPresence)
  const setMyStatus = usePresenceStore((s) => s.setMyStatus)
  const subscribedUsers = usePresenceStore((s) => s.subscribedUsers)

  const awayTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const previousStatusRef = useRef<PresenceLevel>(myStatus)

  // -----------------------------------------------------------------------
  // 1. Heartbeat every 30 seconds
  // -----------------------------------------------------------------------
  useEffect(() => {
    if (!isAuthenticated) return

    // Send initial heartbeat immediately
    wsManager.send({ type: 'presence.heartbeat' })

    const interval = setInterval(() => {
      wsManager.send({ type: 'presence.heartbeat' })
    }, HEARTBEAT_INTERVAL)

    return () => clearInterval(interval)
  }, [isAuthenticated])

  // -----------------------------------------------------------------------
  // 2. Listen for presence.update WebSocket events
  // -----------------------------------------------------------------------
  const handlePresenceUpdate = useCallback(
    (data: Record<string, unknown>) => {
      const userId = data.user_id as string | undefined
      const status = data.status as PresenceLevel | undefined
      if (userId && status) {
        updatePresence(userId, status)
      }
    },
    [updatePresence],
  )

  useWSSubscription('presence.update', handlePresenceUpdate)

  // -----------------------------------------------------------------------
  // 3. Page visibility detection
  // -----------------------------------------------------------------------
  useEffect(() => {
    if (!isAuthenticated) return

    const handleVisibilityChange = () => {
      if (document.hidden) {
        // Start away timer -- don't switch immediately
        awayTimerRef.current = setTimeout(() => {
          // Only auto-away if user hasn't manually set DND
          if (myStatus !== 'dnd') {
            previousStatusRef.current = myStatus
            setMyStatus('away')
            wsManager.send({
              type: 'presence.set_status',
              status: 'away',
            })
          }
        }, AWAY_DELAY)
      } else {
        // Cancel pending away timer
        if (awayTimerRef.current) {
          clearTimeout(awayTimerRef.current)
          awayTimerRef.current = null
        }

        // Restore previous status if we auto-switched to away
        if (myStatus === 'away' && previousStatusRef.current !== 'away') {
          const restoreStatus = previousStatusRef.current
          setMyStatus(restoreStatus)
          wsManager.send({
            type: 'presence.set_status',
            status: restoreStatus,
          })
        } else if (myStatus !== 'dnd') {
          wsManager.send({
            type: 'presence.set_status',
            status: 'online',
          })
        }
      }
    }

    document.addEventListener('visibilitychange', handleVisibilityChange)
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange)
      if (awayTimerRef.current) {
        clearTimeout(awayTimerRef.current)
      }
    }
  }, [isAuthenticated, myStatus, setMyStatus])

  // -----------------------------------------------------------------------
  // 4. Initial bulk fetch for subscribed users
  // -----------------------------------------------------------------------
  const userIds = useMemo(() => Array.from(subscribedUsers), [subscribedUsers])
  const { data: bulkData } = useBulkPresence(userIds)

  useEffect(() => {
    if (bulkData) {
      const mapped: Record<string, PresenceLevel> = {}
      for (const [uid, presence] of Object.entries(bulkData)) {
        mapped[uid] = (presence as { status: PresenceLevel }).status
      }
      if (Object.keys(mapped).length > 0) {
        updateBulkPresence(mapped)
      }
    }
  }, [bulkData, updateBulkPresence])

  // -----------------------------------------------------------------------
  // 5. Subscribe to presence for known users via WebSocket
  // -----------------------------------------------------------------------
  useEffect(() => {
    if (!isAuthenticated || userIds.length === 0) return

    wsManager.send({
      type: 'presence.subscribe',
      user_ids: userIds,
    })
  }, [isAuthenticated, userIds])

  return <>{children}</>
}
