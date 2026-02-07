/**
 * Dashboard layout store (Zustand with localStorage persistence).
 *
 * Manages widget grid layouts, active widget set, and edit mode.
 * Layouts persist across navigations and app restarts via localStorage.
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
] as const

export type WidgetId = (typeof ALL_WIDGET_IDS)[number]

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

interface DashboardState {
  /** Current grid layout keyed by widget ID. */
  layouts: Layout[]
  /** IDs of widgets currently displayed on the dashboard. */
  activeWidgets: string[]
  /** Whether the user is in edit mode (drag/resize enabled). */
  isEditing: boolean

  /** Update layout from react-grid-layout onLayoutChange. */
  updateLayout: (layout: Layout[]) => void
  /** Add a widget to the dashboard at an optional position. */
  addWidget: (widgetId: string, position?: { x: number; y: number }) => void
  /** Remove a widget from the dashboard. */
  removeWidget: (widgetId: string) => void
  /** Toggle edit mode on/off. */
  toggleEditing: () => void
  /** Reset to default layout and widget set. */
  resetToDefaults: () => void
  /** Ensure defaults are loaded (call on first mount). */
  ensureDefaults: () => void
}

export const useDashboardStore = create<DashboardState>()(
  persist(
    (set, get) => ({
      layouts: [],
      activeWidgets: [],
      isEditing: false,

      updateLayout: (layout: Layout[]) => {
        set({ layouts: layout })
      },

      addWidget: (widgetId: string, position?: { x: number; y: number }) => {
        const { activeWidgets, layouts } = get()
        if (activeWidgets.includes(widgetId)) return

        // Look up default size from the default layout
        const defaultItem = getDefaultLayout().find((l) => l.i === widgetId)
        const newItem: Layout = {
          i: widgetId,
          x: position?.x ?? 0,
          y: position?.y ?? Infinity, // Infinity puts it at bottom
          w: defaultItem?.w ?? 4,
          h: defaultItem?.h ?? 3,
          minW: defaultItem?.minW ?? 2,
          minH: defaultItem?.minH ?? 2,
        }

        set({
          activeWidgets: [...activeWidgets, widgetId],
          layouts: [...layouts, newItem],
        })
      },

      removeWidget: (widgetId: string) => {
        const { activeWidgets, layouts } = get()
        set({
          activeWidgets: activeWidgets.filter((id) => id !== widgetId),
          layouts: layouts.filter((l) => l.i !== widgetId),
        })
      },

      toggleEditing: () => {
        set((state) => ({ isEditing: !state.isEditing }))
      },

      resetToDefaults: () => {
        set({
          layouts: getDefaultLayout(),
          activeWidgets: [...ALL_WIDGET_IDS],
          isEditing: false,
        })
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
    }),
    {
      name: 'kmuhub-dashboard',
    }
  )
)
