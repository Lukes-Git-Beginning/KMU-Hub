/**
 * Presence tracking store (Zustand with partial persistence).
 *
 * Maintains a map of user presence statuses and the current user's
 * manual status override. Only myStatus is persisted to localStorage
 * so the user's DND/away preference survives app restart.
 *
 * Set<string> serialized as array for JSON compatibility (same
 * pattern as calendar store visibleCalendarIds).
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { PresenceLevel } from '@/api/video-types'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface PresenceState {
  // User presence map (userId -> status)
  presenceMap: Record<string, PresenceLevel>
  subscribedUsers: Set<string>
  myStatus: PresenceLevel

  // Actions
  updatePresence: (userId: string, status: PresenceLevel) => void
  updateBulkPresence: (updates: Record<string, PresenceLevel>) => void
  setMyStatus: (status: PresenceLevel) => void
  subscribeToUser: (userId: string) => void
  unsubscribeFromUser: (userId: string) => void
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const usePresenceStore = create<PresenceState>()(
  persist(
    (set) => ({
      // Defaults
      presenceMap: {},
      subscribedUsers: new Set<string>(),
      myStatus: 'online' as PresenceLevel,

      updatePresence: (userId: string, status: PresenceLevel) =>
        set((state) => ({
          presenceMap: { ...state.presenceMap, [userId]: status },
        })),

      updateBulkPresence: (updates: Record<string, PresenceLevel>) =>
        set((state) => ({
          presenceMap: { ...state.presenceMap, ...updates },
        })),

      setMyStatus: (status: PresenceLevel) =>
        set({ myStatus: status }),

      subscribeToUser: (userId: string) =>
        set((state) => {
          const next = new Set(state.subscribedUsers)
          next.add(userId)
          return { subscribedUsers: next }
        }),

      unsubscribeFromUser: (userId: string) =>
        set((state) => {
          const next = new Set(state.subscribedUsers)
          next.delete(userId)
          return { subscribedUsers: next }
        }),
    }),
    {
      name: 'cosmi-presence',
      // Only persist myStatus (manual override); presenceMap is transient
      partialize: (state) => ({
        myStatus: state.myStatus,
      }),
    },
  ),
)
