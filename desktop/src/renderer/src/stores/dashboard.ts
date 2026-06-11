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
 *
 * Version history:
 *   v1 → v2: Added per-scope (personal/team) layout/activeWidgets.
 *             Migration: old flat activeWidgets/layouts migrate to personal scope.
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
  'kpi-revenue',
  'kpi-tasks',
  'kpi-deals',
  'calendar-upcoming',
  'team-status',
  'mini-chart',
  'my-tasks',
  'my-calendar',
  'time-clock',
  'team-chat',
  'absences',
  'birthdays',
  'cross-module-overview',
  'team-worktime',
  'open-tickets',
] as const

export type WidgetId = (typeof ALL_WIDGET_IDS)[number]

export type DashboardScope = 'personal' | 'team'

export type SyncStatus = 'synced' | 'syncing' | 'offline' | 'idle'

/** Default 12-column grid layout for the PERSONAL scope. */
export function getDefaultPersonalLayout(): Layout[] {
  return [
    // Row 1: personal (visible for ALL employees)
    { i: 'my-tasks', x: 0, y: 0, w: 4, h: 4, minW: 3, minH: 3 },
    { i: 'my-calendar', x: 4, y: 0, w: 4, h: 4, minW: 3, minH: 3 },
    { i: 'time-clock', x: 8, y: 0, w: 4, h: 3, minW: 3, minH: 2 },
    // Row 2: team overview
    { i: 'team-chat', x: 0, y: 4, w: 4, h: 4, minW: 3, minH: 3 },
    { i: 'absences', x: 4, y: 4, w: 4, h: 3, minW: 3, minH: 2 },
    { i: 'birthdays', x: 8, y: 3, w: 4, h: 3, minW: 3, minH: 2 },
    // Row 3: KPI overview (manager/admin)
    { i: 'kpi-revenue', x: 0, y: 8, w: 4, h: 3, minW: 3, minH: 2 },
    { i: 'kpi-tasks', x: 4, y: 7, w: 4, h: 3, minW: 3, minH: 2 },
    { i: 'kpi-deals', x: 8, y: 6, w: 4, h: 3, minW: 3, minH: 2 },
    // Row 4: data + charts
    { i: 'calendar-upcoming', x: 0, y: 11, w: 4, h: 4, minW: 3, minH: 3 },
    { i: 'mini-chart', x: 4, y: 10, w: 4, h: 3, minW: 4, minH: 2 },
    { i: 'team-status', x: 8, y: 9, w: 4, h: 4, minW: 3, minH: 3 },
    // Row 5: CRM data
    { i: 'recent-contacts', x: 0, y: 15, w: 4, h: 3, minW: 2, minH: 2 },
    { i: 'deal-pipeline', x: 4, y: 13, w: 4, h: 4, minW: 4, minH: 3 },
    { i: 'unread-messages', x: 8, y: 13, w: 4, h: 3, minW: 2, minH: 2 },
    // Row 6: misc
    { i: 'activity-feed', x: 0, y: 18, w: 4, h: 4, minW: 3, minH: 3 },
    { i: 'quick-actions', x: 4, y: 17, w: 4, h: 2, minW: 2, minH: 2 },
    { i: 'notification-summary', x: 8, y: 16, w: 4, h: 3, minW: 2, minH: 2 },
    // Row 7: cross-module overview (always available)
    { i: 'cross-module-overview', x: 0, y: 22, w: 6, h: 4, minW: 4, minH: 3 },
  ]
}

/** Default active widget IDs for the PERSONAL scope. */
function getDefaultPersonalWidgets(): string[] {
  return getDefaultPersonalLayout().map((l) => l.i)
}

/** Default widget IDs for the TEAM scope. */
export const DEFAULT_TEAM_WIDGET_IDS: string[] = [
  'team-status',
  'absences',
  'birthdays',
  'time-clock',
  'team-worktime',
  'open-tickets',
]

/** Default 12-column grid layout for the TEAM scope. */
export function getDefaultTeamLayout(): Layout[] {
  return [
    { i: 'team-status',   x: 0, y: 0, w: 4, h: 4, minW: 3, minH: 3 },
    { i: 'absences',      x: 4, y: 0, w: 4, h: 3, minW: 3, minH: 2 },
    { i: 'birthdays',     x: 8, y: 0, w: 4, h: 3, minW: 3, minH: 2 },
    { i: 'time-clock',    x: 0, y: 4, w: 4, h: 3, minW: 3, minH: 2 },
    { i: 'team-worktime', x: 4, y: 3, w: 4, h: 4, minW: 3, minH: 3 },
    { i: 'open-tickets',  x: 8, y: 3, w: 4, h: 4, minW: 3, minH: 3 },
  ]
}

/** Debounce timer for server sync. */
let syncTimer: ReturnType<typeof setTimeout> | null = null

interface DashboardState {
  /** Current scope: personal or team. */
  scope: DashboardScope

  /** Current grid layout for the PERSONAL scope. */
  personalLayouts: Layout[]
  /** IDs of widgets currently displayed on the PERSONAL dashboard. */
  personalActiveWidgets: string[]

  /** Current grid layout for the TEAM scope (mock-first, localStorage only). */
  teamLayouts: Layout[]
  /** IDs of widgets currently displayed on the TEAM dashboard. */
  teamActiveWidgets: string[]

  /** Whether the user is in edit mode (drag/resize enabled). */
  isEditing: boolean
  /** Server sync status indicator. */
  serverSyncStatus: SyncStatus
  /** Whether the store has been initialized from server. */
  serverInitialized: boolean

  /** Switch between personal and team scope. */
  setScope: (scope: DashboardScope) => void
  /** Update layout from react-grid-layout onLayoutChange (scoped). */
  updateLayout: (layout: Layout[], scope: DashboardScope) => void
  /** Add a widget to the dashboard at an optional position (scoped). */
  addWidget: (widgetId: string, scope: DashboardScope, position?: { x: number; y: number }) => void
  /** Remove a widget from the dashboard (scoped). */
  removeWidget: (widgetId: string, scope: DashboardScope) => void
  /** Toggle edit mode on/off. */
  toggleEditing: () => void
  /** Reset to default layout and widget set (personal scope only; calls server DELETE). */
  resetToDefaults: () => void
  /** Ensure defaults are loaded (call on first mount). */
  ensureDefaults: () => void
  /** Initialize from server (fetch layout, merge with local). */
  initFromServer: () => Promise<void>
  /** Set sync status. */
  setSyncStatus: (status: SyncStatus) => void
}

/**
 * Debounced save to server (PERSONAL scope only).
 * Team scope is mock-first / localStorage only.
 */
function debouncedServerSync(layouts: Layout[], activeWidgets: string[], setSyncStatus: (s: SyncStatus) => void) {
  if (syncTimer) clearTimeout(syncTimer)

  syncTimer = setTimeout(async () => {
    setSyncStatus('syncing')
    try {
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
      scope: 'personal' as DashboardScope,

      personalLayouts: [],
      personalActiveWidgets: [],

      teamLayouts: [],
      teamActiveWidgets: [],

      isEditing: false,
      serverSyncStatus: 'idle' as SyncStatus,
      serverInitialized: false,

      setScope: (scope: DashboardScope) => {
        set({ scope, isEditing: false })
      },

      updateLayout: (layout: Layout[], scope: DashboardScope) => {
        const { personalActiveWidgets, setSyncStatus } = get()
        if (scope === 'team') {
          set({ teamLayouts: layout })
          // Team layout is localStorage-only (mock-first, no server PUT)
        } else {
          set({ personalLayouts: layout })
          debouncedServerSync(layout, personalActiveWidgets, setSyncStatus)
        }
      },

      addWidget: (widgetId: string, scope: DashboardScope, position?: { x: number; y: number }) => {
        const { personalActiveWidgets, personalLayouts, teamActiveWidgets, teamLayouts, setSyncStatus } = get()

        if (scope === 'team') {
          if (teamActiveWidgets.includes(widgetId)) return
          const defaultItem = getDefaultTeamLayout().find((l) => l.i === widgetId)
          const newItem: Layout = {
            i: widgetId,
            x: position?.x ?? 0,
            y: position?.y ?? Infinity,
            w: defaultItem?.w ?? 4,
            h: defaultItem?.h ?? 3,
            minW: defaultItem?.minW ?? 2,
            minH: defaultItem?.minH ?? 2,
          }
          set({
            teamActiveWidgets: [...teamActiveWidgets, widgetId],
            teamLayouts: [...teamLayouts, newItem],
          })
        } else {
          if (personalActiveWidgets.includes(widgetId)) return
          const defaultItem = getDefaultPersonalLayout().find((l) => l.i === widgetId)
          const newItem: Layout = {
            i: widgetId,
            x: position?.x ?? 0,
            y: position?.y ?? Infinity,
            w: defaultItem?.w ?? 4,
            h: defaultItem?.h ?? 3,
            minW: defaultItem?.minW ?? 2,
            minH: defaultItem?.minH ?? 2,
          }
          const newActiveWidgets = [...personalActiveWidgets, widgetId]
          const newLayouts = [...personalLayouts, newItem]
          set({
            personalActiveWidgets: newActiveWidgets,
            personalLayouts: newLayouts,
          })
          debouncedServerSync(newLayouts, newActiveWidgets, setSyncStatus)
        }
      },

      removeWidget: (widgetId: string, scope: DashboardScope) => {
        const { personalActiveWidgets, personalLayouts, teamActiveWidgets, teamLayouts, setSyncStatus } = get()

        if (scope === 'team') {
          set({
            teamActiveWidgets: teamActiveWidgets.filter((id) => id !== widgetId),
            teamLayouts: teamLayouts.filter((l) => l.i !== widgetId),
          })
        } else {
          const newActiveWidgets = personalActiveWidgets.filter((id) => id !== widgetId)
          const newLayouts = personalLayouts.filter((l) => l.i !== widgetId)
          set({
            personalActiveWidgets: newActiveWidgets,
            personalLayouts: newLayouts,
          })
          debouncedServerSync(newLayouts, newActiveWidgets, setSyncStatus)
        }
      },

      toggleEditing: () => {
        set((state) => ({ isEditing: !state.isEditing }))
      },

      resetToDefaults: () => {
        set({
          personalLayouts: getDefaultPersonalLayout(),
          personalActiveWidgets: getDefaultPersonalWidgets(),
          isEditing: false,
          serverSyncStatus: 'syncing',
        })

        ;(async () => {
          try {
            const { apiClient } = await import('@/api/client')
            await apiClient.DELETE('/api/v1/dashboard/layout')

            const { data } = await apiClient.GET('/api/v1/dashboard/layout')
            if (data) {
              const serverLayout = (data.layout ?? []) as unknown as Layout[]
              const serverWidgets = (data.active_widgets ?? []) as string[]
              if (serverLayout.length > 0) {
                set({
                  personalLayouts: serverLayout.map((l) => ({
                    ...l,
                    minW: getDefaultPersonalLayout().find((d) => d.i === l.i)?.minW ?? 2,
                    minH: getDefaultPersonalLayout().find((d) => d.i === l.i)?.minH ?? 2,
                  })),
                  personalActiveWidgets: serverWidgets,
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
        const { personalActiveWidgets, teamActiveWidgets } = get()
        const updates: Partial<Pick<DashboardState, 'personalLayouts' | 'personalActiveWidgets' | 'teamLayouts' | 'teamActiveWidgets'>> = {}
        if (personalActiveWidgets.length === 0) {
          updates.personalLayouts = getDefaultPersonalLayout()
          updates.personalActiveWidgets = getDefaultPersonalWidgets()
        }
        if (teamActiveWidgets.length === 0) {
          updates.teamLayouts = getDefaultTeamLayout()
          updates.teamActiveWidgets = [...DEFAULT_TEAM_WIDGET_IDS]
        }
        if (Object.keys(updates).length > 0) {
          set(updates)
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
            setSyncStatus('offline')
            set({ serverInitialized: true })
            return
          }

          const serverLayout = (data.layout ?? []) as unknown as Layout[]
          const serverWidgets = (data.active_widgets ?? []) as string[]

          if (serverLayout.length > 0) {
            set({
              personalLayouts: serverLayout.map((l) => ({
                ...l,
                minW: getDefaultPersonalLayout().find((d) => d.i === l.i)?.minW ?? 2,
                minH: getDefaultPersonalLayout().find((d) => d.i === l.i)?.minH ?? 2,
              })),
              personalActiveWidgets: serverWidgets,
              serverSyncStatus: 'synced',
              serverInitialized: true,
            })
          } else {
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
      name: 'cosmi-dashboard',
      version: 2,
      /**
       * Persist migration: v1 → v2
       * v1 stored flat { layouts, activeWidgets }.
       * v2 stores per-scope { personalLayouts, personalActiveWidgets, teamLayouts, teamActiveWidgets, scope }.
       * Migration: move old flat fields into personal scope; preserve widget customizations.
       */
      migrate(persistedState: unknown, version: number) {
        if (version < 2) {
          const old = persistedState as {
            layouts?: Layout[]
            activeWidgets?: string[]
          }
          return {
            scope: 'personal' as DashboardScope,
            personalLayouts: old.layouts ?? [],
            personalActiveWidgets: old.activeWidgets ?? [],
            teamLayouts: [] as Layout[],
            teamActiveWidgets: [] as string[],
          }
        }
        return persistedState
      },
      partialize: (state) => ({
        scope: state.scope,
        personalLayouts: state.personalLayouts,
        personalActiveWidgets: state.personalActiveWidgets,
        teamLayouts: state.teamLayouts,
        teamActiveWidgets: state.teamActiveWidgets,
      }),
    }
  )
)
