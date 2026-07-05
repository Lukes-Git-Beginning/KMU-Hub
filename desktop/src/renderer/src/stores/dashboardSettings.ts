import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { ALL_WIDGET_IDS, type WidgetId } from '@/stores/dashboard'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Tenant-wide dashboard settings — administered via the "Für alle" section of
 * the dashboard settings panel (admins only; the PUT is dropped server-side for
 * other roles). The allowed-widgets list also filters the widget picker for real.
 *
 * Server sync (settings foundation, Migr. 138, GET/PUT /settings/dashboard/tenant):
 *   - `initFromServer()` hydrates once per session (via useHydrateModuleSettings),
 *     else local defaults. localStorage stays the optimistic cache.
 */
const MODULE_ID = 'dashboard'

/** Keep only valid, known widget ids from a server-loaded list. */
function sanitizeWidgetIds(value: unknown, fallback: WidgetId[]): WidgetId[] {
  if (!Array.isArray(value)) return fallback
  const valid = value.filter((id): id is WidgetId => ALL_WIDGET_IDS.includes(id as WidgetId))
  return valid.length > 0 ? valid : fallback
}

interface DashboardSettingsState {
  /** Widget set new employees start with (team default layout). */
  defaultWidgets: WidgetId[]
  /** Widgets users may add at all (licence/module gating). */
  allowedWidgets: WidgetId[]
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean

  toggleDefaultWidget: (id: WidgetId) => void
  toggleAllowedWidget: (id: WidgetId) => void
  /** Hydrate from GET /settings/dashboard/tenant (once per session). */
  initFromServer: () => Promise<void>
}

/** The tenant-persisted keys, extracted as the PUT payload. */
function tenantPayload(s: DashboardSettingsState): Record<string, unknown> {
  return {
    defaultWidgets: s.defaultWidgets,
    allowedWidgets: s.allowedWidgets,
  }
}

export const useDashboardSettingsStore = create<DashboardSettingsState>()(
  persist(
    (set, get) => ({
      defaultWidgets: [...ALL_WIDGET_IDS],
      allowedWidgets: [...ALL_WIDGET_IDS],
      serverInitialized: false,

      toggleDefaultWidget: (id) => {
        set((s) => ({
          defaultWidgets: s.defaultWidgets.includes(id)
            ? s.defaultWidgets.filter((w) => w !== id)
            : [...s.defaultWidgets, id],
        }))
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },

      // Disallowing a widget also drops it from the team default — a widget
      // nobody may add must not be part of the starter layout either.
      toggleAllowedWidget: (id) => {
        set((s) => {
          const allowed = s.allowedWidgets.includes(id)
          return allowed
            ? {
                allowedWidgets: s.allowedWidgets.filter((w) => w !== id),
                defaultWidgets: s.defaultWidgets.filter((w) => w !== id),
              }
            : { allowedWidgets: [...s.allowedWidgets, id] }
        })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },

      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'tenant')
        set((s) => ({
          defaultWidgets: sanitizeWidgetIds(map.defaultWidgets, s.defaultWidgets),
          allowedWidgets: sanitizeWidgetIds(map.allowedWidgets, s.allowedWidgets),
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-dashboard-settings' },
  ),
)
