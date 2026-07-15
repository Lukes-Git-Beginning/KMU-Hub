import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Personal Schichten preferences (personal/user scope, see ModuleSettingsShell).
 * Backed by the settings foundation (GET/PUT /settings/schichten/user):
 * localStorage is the optimistic cache, `initFromServer()` hydrates once per
 * session (central hydrator) and each setter writes through.
 *
 * Market pattern (Planday/Shiftbase): the per-user defaults are the view the
 * planner opens with and which decorations the grid shows.
 */
const MODULE_ID = 'schichten'

export type SchichtenTab = 'wochenplan' | 'vorlagen' | 'anfragen' | 'verfügbarkeit'
const TABS: SchichtenTab[] = ['wochenplan', 'vorlagen', 'anfragen', 'verfügbarkeit']

interface SchichtenPrefsState {
  /** Tab the module opens with. */
  defaultTab: SchichtenTab
  /** Show surcharge badges (Nacht/WE/Feiertag) on grid cells. */
  showSurcharges: boolean
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setDefaultTab: (v: SchichtenTab) => void
  setShowSurcharges: (v: boolean) => void
  /** Hydrate from GET /settings/schichten/user (once per session). */
  initFromServer: () => Promise<void>
}

function userPayload(s: SchichtenPrefsState): Record<string, unknown> {
  return {
    defaultTab: s.defaultTab,
    showSurcharges: s.showSurcharges,
  }
}

export const useSchichtenPrefsStore = create<SchichtenPrefsState>()(
  persist(
    (set, get) => ({
      defaultTab: 'wochenplan',
      showSurcharges: true,
      serverInitialized: false,
      setDefaultTab: (defaultTab) => {
        set({ defaultTab })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setShowSurcharges: (showSurcharges) => {
        set({ showSurcharges })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          defaultTab: TABS.includes(map.defaultTab as SchichtenTab)
            ? (map.defaultTab as SchichtenTab)
            : s.defaultTab,
          showSurcharges:
            typeof map.showSurcharges === 'boolean' ? map.showSurcharges : s.showSurcharges,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-schichten-prefs' },
  ),
)
