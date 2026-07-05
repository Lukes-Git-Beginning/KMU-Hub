import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Tenant-wide zeiterfassung settings (tenant scope — only Modul-Leiter/admin;
 * the PUT is dropped server-side for other roles, the panel gates the UI).
 *
 * Server sync (settings foundation, Migr. 138, GET/PUT /settings/zeiterfassung/tenant):
 *   - `initFromServer()` hydrates once per session (via useHydrateModuleSettings),
 *     else local defaults. localStorage stays the optimistic cache.
 */
const MODULE_ID = 'zeiterfassung'

export type HolidayRegion = 'DE' | 'AT' | 'CH'
export type RoundingMode = 'none' | '5' | '15'

export const HOLIDAY_REGIONS: HolidayRegion[] = ['DE', 'AT', 'CH']
export const ROUNDING_MODES: RoundingMode[] = ['none', '5', '15']

interface ZeiterfassungSettingsState {
  /**
   * ArbZG: after this many hours a break becomes mandatory.
   * (Weekly target is NOT here — it is per-employee contract data set in the
   * Team/HR module, not a tenant-wide setting.)
   */
  autoBreakAfterHours: number
  /** Auto-deducted break minutes once the threshold is crossed. */
  autoBreakMinutes: number
  /** Rounding of clock entries (minutes). */
  rounding: RoundingMode
  /** Public-holiday region for working-day calculations. */
  holidayRegion: HolidayRegion
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setAutoBreakAfterHours: (h: number) => void
  setAutoBreakMinutes: (m: number) => void
  setRounding: (r: RoundingMode) => void
  setHolidayRegion: (r: HolidayRegion) => void
  /** Hydrate from GET /settings/zeiterfassung/tenant (once per session). */
  initFromServer: () => Promise<void>
}

/** The tenant-persisted keys, extracted as the PUT payload. */
function tenantPayload(s: ZeiterfassungSettingsState): Record<string, unknown> {
  return {
    autoBreakAfterHours: s.autoBreakAfterHours,
    autoBreakMinutes: s.autoBreakMinutes,
    rounding: s.rounding,
    holidayRegion: s.holidayRegion,
  }
}

export const useZeiterfassungSettingsStore = create<ZeiterfassungSettingsState>()(
  persist(
    (set, get) => ({
      autoBreakAfterHours: 6,
      autoBreakMinutes: 30,
      rounding: 'none',
      holidayRegion: 'DE',
      serverInitialized: false,
      setAutoBreakAfterHours: (autoBreakAfterHours) => {
        set({ autoBreakAfterHours })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setAutoBreakMinutes: (autoBreakMinutes) => {
        set({ autoBreakMinutes })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setRounding: (rounding) => {
        set({ rounding })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setHolidayRegion: (holidayRegion) => {
        set({ holidayRegion })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'tenant')
        set((s) => ({
          autoBreakAfterHours:
            typeof map.autoBreakAfterHours === 'number'
              ? map.autoBreakAfterHours
              : s.autoBreakAfterHours,
          autoBreakMinutes:
            typeof map.autoBreakMinutes === 'number'
              ? map.autoBreakMinutes
              : s.autoBreakMinutes,
          rounding: ROUNDING_MODES.includes(map.rounding as RoundingMode)
            ? (map.rounding as RoundingMode)
            : s.rounding,
          holidayRegion: HOLIDAY_REGIONS.includes(map.holidayRegion as HolidayRegion)
            ? (map.holidayRegion as HolidayRegion)
            : s.holidayRegion,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-zeiterfassung-settings' },
  ),
)
