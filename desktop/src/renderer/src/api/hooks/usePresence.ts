/**
 * TanStack Query hooks for user presence tracking.
 *
 * Presence queries use a short staleTime (10s) since presence is
 * near-real-time data. WebSocket events (presence.updated) should
 * also invalidate these queries for immediate updates.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getPresence,
  getBulkPresence,
  getPresenceConfig,
  setPresenceStatus,
  updatePresenceConfig,
} from '../video-client'
import type { PresenceLevel } from '../video-types'

/** Short stale time for presence data (10 seconds). */
const PRESENCE_STALE_TIME = 10_000

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

/** Get a single user's presence status. */
export function usePresence(userId: string) {
  return useQuery({
    queryKey: ['presence', userId],
    queryFn: () => getPresence(userId),
    enabled: !!userId,
    staleTime: PRESENCE_STALE_TIME,
  })
}

/** Get presence status for multiple users at once. */
export function useBulkPresence(userIds: string[]) {
  return useQuery({
    queryKey: ['presence', 'bulk', userIds],
    queryFn: () => getBulkPresence(userIds),
    enabled: userIds.length > 0,
    staleTime: PRESENCE_STALE_TIME,
  })
}

/** Get presence configuration (away timeout). */
export function usePresenceConfig() {
  return useQuery({
    queryKey: ['presence', 'config'],
    queryFn: getPresenceConfig,
  })
}

// ---------------------------------------------------------------------------
// Mutations
// ---------------------------------------------------------------------------

/** Set the current user's presence status (manual override). */
export function useSetPresenceStatus() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (status: PresenceLevel) => setPresenceStatus(status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['presence'] })
    },
  })
}

/** Update presence configuration (admin only). */
export function useUpdatePresenceConfig() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (awayTimeoutSeconds: number) =>
      updatePresenceConfig(awayTimeoutSeconds),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['presence', 'config'] })
    },
  })
}
