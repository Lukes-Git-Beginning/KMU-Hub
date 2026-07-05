import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Personal vertraege (Verträge) preferences — per-user comfort settings
 * (personal/user scope, see ModuleSettingsShell).
 *
 * Server sync (settings foundation, Migr. 138, GET/PUT /settings/vertraege/user):
 *   - `initFromServer()` once per session hydrates from the backend (fired by
 *     useHydrateModuleSettings on app-shell mount); falls back to local defaults.
 *   - Each setter writes through to the user-scope endpoint. localStorage stays
 *     as the optimistic cache so the UI is instant and survives offline.
 */
const MODULE_ID = 'vertraege'

export type VertraegeTab = 'aktiv' | 'auslaufend' | 'archiv' | 'vorlagen'
export type VertraegeDensity = 'comfortable' | 'compact'

interface VertraegePrefsState {
  /** Tab pre-selected when opening the module. */
  defaultTab: VertraegeTab
  /** Row density of the contract table. */
  density: VertraegeDensity
  /**
   * Personal reminder pre-selection for new contracts (days before end).
   * null = no personal override, the tenant default applies.
   */
  defaultReminderDays: number[] | null
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setDefaultTab: (t: VertraegeTab) => void
  setDensity: (d: VertraegeDensity) => void
  setDefaultReminderDays: (days: number[] | null) => void
  /** Hydrate from GET /settings/vertraege/user (once per session). */
  initFromServer: () => Promise<void>
}

const VERTRAEGE_TABS: VertraegeTab[] = ['aktiv', 'auslaufend', 'archiv', 'vorlagen']

/** The user-persisted keys, extracted as the PUT payload. */
function userPayload(s: VertraegePrefsState): Record<string, unknown> {
  return {
    defaultTab: s.defaultTab,
    density: s.density,
    defaultReminderDays: s.defaultReminderDays,
  }
}

export const useVertraegePrefsStore = create<VertraegePrefsState>()(
  persist(
    (set, get) => ({
      defaultTab: 'aktiv',
      density: 'comfortable',
      defaultReminderDays: null,
      serverInitialized: false,
      setDefaultTab: (defaultTab) => {
        set({ defaultTab })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setDensity: (density) => {
        set({ density })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setDefaultReminderDays: (defaultReminderDays) => {
        set({ defaultReminderDays })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          defaultTab: VERTRAEGE_TABS.includes(map.defaultTab as VertraegeTab)
            ? (map.defaultTab as VertraegeTab)
            : s.defaultTab,
          density:
            map.density === 'comfortable' || map.density === 'compact'
              ? map.density
              : s.density,
          defaultReminderDays:
            (Array.isArray(map.defaultReminderDays) &&
              map.defaultReminderDays.every((d) => typeof d === 'number')) ||
            map.defaultReminderDays === null
              ? (map.defaultReminderDays as number[] | null)
              : s.defaultReminderDays,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-vertraege-prefs' },
  ),
)
