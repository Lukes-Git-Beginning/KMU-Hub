import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { ALL_WIDGET_IDS, type WidgetId } from '@/stores/dashboard'

/**
 * Tenant-wide dashboard settings — administered via the "Für alle" section of
 * the dashboard settings panel (admins only; dashboard is not delegable as
 * Modul-Leiter). Mock-first: persisted locally until the backend persists
 * tenant settings for the dashboard (tenant_settings, module_id='dashboard')
 * — tracked in backend-handover-luke.md. The allowed-widgets list already
 * filters the widget picker for real.
 */

interface DashboardSettingsState {
  /** Widget set new employees start with (team default layout). */
  defaultWidgets: WidgetId[]
  /** Widgets users may add at all (licence/module gating). */
  allowedWidgets: WidgetId[]

  toggleDefaultWidget: (id: WidgetId) => void
  toggleAllowedWidget: (id: WidgetId) => void
}

export const useDashboardSettingsStore = create<DashboardSettingsState>()(
  persist(
    (set) => ({
      defaultWidgets: [...ALL_WIDGET_IDS],
      allowedWidgets: [...ALL_WIDGET_IDS],

      toggleDefaultWidget: (id) =>
        set((s) => ({
          defaultWidgets: s.defaultWidgets.includes(id)
            ? s.defaultWidgets.filter((w) => w !== id)
            : [...s.defaultWidgets, id],
        })),

      // Disallowing a widget also drops it from the team default — a widget
      // nobody may add must not be part of the starter layout either.
      toggleAllowedWidget: (id) =>
        set((s) => {
          const allowed = s.allowedWidgets.includes(id)
          return allowed
            ? {
                allowedWidgets: s.allowedWidgets.filter((w) => w !== id),
                defaultWidgets: s.defaultWidgets.filter((w) => w !== id),
              }
            : { allowedWidgets: [...s.allowedWidgets, id] }
        }),
    }),
    { name: 'cosmi-dashboard-settings' },
  ),
)
