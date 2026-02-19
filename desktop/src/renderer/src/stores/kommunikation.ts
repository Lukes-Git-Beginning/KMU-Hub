/**
 * Kommunikation (Unified Inbox) UI state store.
 *
 * All server data is managed by TanStack Query hooks in api/hooks/useInbox.ts.
 * This store only holds ephemeral UI state: active view, selected message,
 * sidebar collapsed state.
 *
 * Persists sidebarCollapsed and activeView to localStorage.
 * Does NOT persist selectedMessageId (ephemeral).
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// ---------------------------------------------------------------------------
// Store interface
// ---------------------------------------------------------------------------

interface KommunikationState {
  // View selection
  activeView: string // 'all' | 'unread' | 'starred' | 'assigned' | InboxChannel | team_inbox_id
  selectedMessageId: string | null

  // UI state
  sidebarCollapsed: boolean
  detailPaneOpen: boolean

  // Actions
  setActiveView: (view: string) => void
  setSelectedMessage: (id: string | null) => void
  toggleSidebar: () => void
  toggleDetailPane: () => void
}

export const useKommunikationStore = create<KommunikationState>()(
  persist(
    (set) => ({
      activeView: 'all',
      selectedMessageId: null,
      sidebarCollapsed: false,
      detailPaneOpen: true,

      setActiveView: (view) =>
        set({ activeView: view, selectedMessageId: null }),
      setSelectedMessage: (id) =>
        set({ selectedMessageId: id, detailPaneOpen: id !== null }),
      toggleSidebar: () =>
        set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),
      toggleDetailPane: () =>
        set((state) => ({ detailPaneOpen: !state.detailPaneOpen })),
    }),
    {
      name: 'kmuhub-kommunikation-ui',
      partialize: (state) => ({
        activeView: state.activeView,
        sidebarCollapsed: state.sidebarCollapsed,
      }),
    },
  ),
)
