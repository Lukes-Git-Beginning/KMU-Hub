import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Tenant-weite Schichten-Vorgaben ("Für alle"-Scope, siehe ModuleSettingsShell).
 * Editierbar nur durch Schichten-Modul-Leiter/Admin (der PUT wird serverseitig
 * für andere Rollen verworfen; das Panel gatet die UI). Persönliche Prefs
 * liegen in stores/schichtenPrefs.ts.
 *
 * Feldauswahl folgt dem Markt (Planday/Deputy/Shiftbase): Schichttausch ist
 * firmenweit aktivierbar, die Wochenstunden-Grenze speist die ArbZG-Prüfung
 * und die Standard-Pause die Vorlagen.
 */
const MODULE_ID = 'schichten'

interface SchichtenTenantState {
  /** Allow shift-swap requests (gates the swap action + form). */
  swapEnabled: boolean
  /** Weekly hours ceiling used by the ArbZG check (§3: legal max 48h). */
  maxWeeklyHours: number
  /** Default break minutes for new shift templates. */
  defaultBreakMinutes: number
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setSwapEnabled: (v: boolean) => void
  setMaxWeeklyHours: (v: number) => void
  setDefaultBreakMinutes: (v: number) => void
  /** Hydrate from GET /settings/schichten/tenant (once per session). */
  initFromServer: () => Promise<void>
}

/** The tenant-persisted keys, extracted as the PUT payload. */
function tenantPayload(s: SchichtenTenantState): Record<string, unknown> {
  return {
    swapEnabled: s.swapEnabled,
    maxWeeklyHours: s.maxWeeklyHours,
    defaultBreakMinutes: s.defaultBreakMinutes,
  }
}

/** ArbZG §3: never allow configuring beyond the legal 48h ceiling. */
function clampWeeklyHours(v: number): number {
  return Math.min(48, Math.max(20, Math.round(v)))
}

export const useSchichtenTenantStore = create<SchichtenTenantState>()(
  persist(
    (set, get) => ({
      swapEnabled: true,
      maxWeeklyHours: 48,
      defaultBreakMinutes: 30,
      serverInitialized: false,
      setSwapEnabled: (swapEnabled) => {
        set({ swapEnabled })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setMaxWeeklyHours: (maxWeeklyHours) => {
        set({ maxWeeklyHours: clampWeeklyHours(maxWeeklyHours) })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setDefaultBreakMinutes: (defaultBreakMinutes) => {
        set({ defaultBreakMinutes: Math.max(0, Math.round(defaultBreakMinutes)) })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'tenant')
        set((s) => ({
          swapEnabled: typeof map.swapEnabled === 'boolean' ? map.swapEnabled : s.swapEnabled,
          maxWeeklyHours:
            typeof map.maxWeeklyHours === 'number' && map.maxWeeklyHours > 0
              ? clampWeeklyHours(map.maxWeeklyHours)
              : s.maxWeeklyHours,
          defaultBreakMinutes:
            typeof map.defaultBreakMinutes === 'number' && map.defaultBreakMinutes >= 0
              ? Math.round(map.defaultBreakMinutes)
              : s.defaultBreakMinutes,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-schichten-tenant' },
  ),
)
