/**
 * Dashboard layout store (Zustand with localStorage + server sync).
 *
 * Manages widget grid layouts, active widget set, and edit mode.
 * Layouts persist locally via localStorage (offline cache) and sync
 * to the server as the source of truth.
 *
 * Flow:
 * 1. On init: load from localStorage immediately (fast startup)
 * 2. Fetch from server and merge (prefer server if newer)
 * 3. On layout changes: update local state, debounce PUT to server
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { Layout } from 'react-grid-layout'

/** All widget IDs that ship with the app. */
export const ALL_WIDGET_IDS = [
  'recent-contacts',
  'deal-pipeline',
  'unread-messages',
  'activity-feed',
  'quick-actions',
  'notification-summary',
  'notification-feed',
] as const

export type WidgetId = (typeof ALL_WIDGET_IDS)[number]

export type SyncStatus = 'synced' | 'syncing' | 'offline' | 'idle'

/** Default 12-column grid layout: 2 rows of 3 widgets. */
function getDefaultLayout(): Layout[] {
  return [
    { i: 'recent-contacts', x: 0, y: 0, w: 4, h: 3, minW: 2, minH: 2 },
    { i: 'deal-pipeline', x: 4, y: 0, w: 4, h: 4, minW: 4, minH: 3 },
    { i: 'unread-messages', x: 8, y: 0, w: 4, h: 3, minW: 2, minH: 2 },
    { i: 'activity-feed', x: 0, y: 3, w: 4, h: 4, minW: 3, minH: 3 },
    { i: 'quick-actions', x: 4, y: 4, w: 4, h: 2, minW: 2, minH: 2 },
    { i: 'notification-summary', x: 8, y: 3, w: 4, h: 3, minW: 2, minH: 2 },
  ]
}

/** Debounce timer for server sync. */
let syncTimer: ReturnType<typeof setTimeout> | null = null

interface DashboardState {
  /** Current grid layout keyed by widget ID. */
  layouts: Layout[]
  /** IDs of widgets currently displayed on the dashboard. */
  activeWidgets: string[]
  /** Whether the user is in edit mode (drag/resize enabled). */
  isEditing: boolean
  /** Server sync status indicator. */
  serverSyncStatus: SyncStatus
  /** Whether the store has been initialized from server. */
  serverInitialized: boolean

  /** Update layout from react-grid-layout onLayoutChange. */
  updateLayout: (layout: Layout[]) => void
  /** Add a widget to the dashboard at an optional position. */
  addWidget: (widgetId: string, position?: { x: number; y: number }) => void
  /** Remove a widget from the dashboard. */
  removeWidget: (widgetId: string) => void
  /** Toggle edit mode on/off. */
  toggleEditing: () => void
  /** Reset to default layout and widget set (calls server DELETE). */
  resetToDefaults: () => void
  /** Ensure defaults are loaded (call on first mount). */
  ensureDefaults: () => void
  /** Initialize from server (fetch layout, merge with local). */
  initFromServer: () => Promise<void>
  /** Set sync status. */
  setSyncStatus: (status: SyncStatus) => void
}

/**
 * Debounced save to server. Waits 2 seconds after last layout change
 * before sending PUT to avoid hammering the server during drag operations.
 */
function debouncedServerSync(layouts: Layout[], activeWidgets: string[], setSyncStatus: (s: SyncStatus) => void) {
  if (syncTimer) clearTimeout(syncTimer)

  syncTimer = setTimeout(async () => {
    setSyncStatus('syncing')
    try {
      // Lazy import to avoid circular dependency
      const { apiClient } = await import('@/api/client')
      const { error } = await apiClient.PUT('/api/v1/dashboard/layout', {
        body: {
          layout: layouts.map((l) => ({
            i: l.i,
            x: l.x,
            y: l.y,
            w: l.w,
            h: l.h,
          })),
          active_widgets: activeWidgets,
        },
      })
      if (error) {
        setSyncStatus('offline')
      } else {
        setSyncStatus('synced')
      }
    } catch {
      setSyncStatus('offline')
    }
  }, 2000)
}

export const useDashboardStore = create<DashboardState>()(
  persist(
    (set, get) => ({
      layouts: [],
      activeWidgets: [],
      isEditing: false,
      serverSyncStatus: 'idle' as SyncStatus,
      serverInitialized: false,

      updateLayout: (layout: Layout[]) => {
        set({ layouts: layout })
        // Trigger debounced server sync
        const { activeWidgets, setSyncStatus } = get()
        debouncedServerSync(layout, activeWidgets, setSyncStatus)
      },

      addWidget: (widgetId: string, position?: { x: number; y: number }) => {
        const { activeWidgets, layouts, setSyncStatus } = get()
        if (activeWidgets.includes(widgetId)) return

        const defaultItem = getDefaultLayout().find((l) => l.i === widgetId)
        const newItem: Layout = {
          i: widgetId,
          x: position?.x ?? 0,
          y: position?.y ?? Infinity,
          w: defaultItem?.w ?? 4,
          h: defaultItem?.h ?? 3,
          minW: defaultItem?.minW ?? 2,
          minH: defaultItem?.minH ?? 2,
        }

        const newActiveWidgets = [...activeWidgets, widgetId]
        const newLayouts = [...layouts, newItem]
        set({
          activeWidgets: newActiveWidgets,
          layouts: newLayouts,
        })
        debouncedServerSync(newLayouts, newActiveWidgets, setSyncStatus)
      },

      removeWidget: (widgetId: string) => {
        const { activeWidgets, layouts, setSyncStatus } = get()
        const newActiveWidgets = activeWidgets.filter((id) => id !== widgetId)
        const newLayouts = layouts.filter((l) => l.i !== widgetId)
        set({
          activeWidgets: newActiveWidgets,
          layouts: newLayouts,
        })
        debouncedServerSync(newLayouts, newActiveWidgets, setSyncStatus)
      },

      toggleEditing: () => {
        set((state) => ({ isEditing: !state.isEditing }))
      },

      resetToDefaults: () => {
        set({
          layouts: getDefaultLayout(),
          activeWidgets: [...ALL_WIDGET_IDS],
          isEditing: false,
          serverSyncStatus: 'syncing',
        })

        // Call server DELETE then refetch
        ;(async () => {
          try {
            const { apiClient } = await import('@/api/client')
            await apiClient.DELETE('/api/v1/dashboard/layout')

            // Refetch the role default from server
            const { data } = await apiClient.GET('/api/v1/dashboard/layout')
            if (data) {
              const serverLayout = (data.layout ?? []) as unknown as Layout[]
              const serverWidgets = (data.active_widgets ?? []) as string[]
              if (serverLayout.length > 0) {
                set({
                  layouts: serverLayout.map((l) => ({
                    ...l,
                    minW: getDefaultLayout().find((d) => d.i === l.i)?.minW ?? 2,
                    minH: getDefaultLayout().find((d) => d.i === l.i)?.minH ?? 2,
                  })),
                  activeWidgets: serverWidgets,
                  serverSyncStatus: 'synced',
                })
                return
              }
            }
            set({ serverSyncStatus: 'synced' })
          } catch {
            set({ serverSyncStatus: 'offline' })
          }
        })()
      },

      ensureDefaults: () => {
        const { activeWidgets } = get()
        if (activeWidgets.length === 0) {
          set({
            layouts: getDefaultLayout(),
            activeWidgets: [...ALL_WIDGET_IDS],
          })
        }
      },

      initFromServer: async () => {
        const { serverInitialized, setSyncStatus } = get()
        if (serverInitialized) return

        setSyncStatus('syncing')
        try {
          const { apiClient } = await import('@/api/client')
          const { data, error } = await apiClient.GET('/api/v1/dashboard/layout')

          if (error || !data) {
            // Server unreachable -- keep localStorage state
            setSyncStatus('offline')
            set({ serverInitialized: true })
            return
          }

          const serverLayout = (data.layout ?? []) as unknown as Layout[]
          const serverWidgets = (data.active_widgets ?? []) as string[]

          if (serverLayout.length > 0) {
            // Merge server layout with minW/minH from defaults
            set({
              layouts: serverLayout.map((l) => ({
                ...l,
                minW: getDefaultLayout().find((d) => d.i === l.i)?.minW ?? 2,
                minH: getDefaultLayout().find((d) => d.i === l.i)?.minH ?? 2,
              })),
              activeWidgets: serverWidgets,
              serverSyncStatus: 'synced',
              serverInitialized: true,
            })
          } else {
            // No server layout -- use local defaults
            set({ serverSyncStatus: 'synced', serverInitialized: true })
          }
        } catch {
          setSyncStatus('offline')
          set({ serverInitialized: true })
        }
      },

      setSyncStatus: (status: SyncStatus) => {
        set({ serverSyncStatus: status })
      },
    }),
    {
      name: 'kmuhub-dashboard',
      partialize: (state) => ({
        layouts: state.layouts,
        activeWidgets: state.activeWidgets,
      }),
    }
  )
)
