import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Personal Fuhrpark preferences (personal/user scope, see ModuleSettingsShell).
 * Backed by the settings foundation (GET/PUT /settings/fuhrpark/user):
 * localStorage is the optimistic cache, `initFromServer()` hydrates once per
 * session (central hydrator) and each setter writes through.
 */
const MODULE_ID = 'fuhrpark'

export type FuhrparkTab = 'fahrzeuge' | 'wartung' | 'tankprotokoll' | 'tracking' | 'fahrtenbuch'
const TABS: FuhrparkTab[] = ['fahrzeuge', 'wartung', 'tankprotokoll', 'tracking', 'fahrtenbuch']

export type TripCategory = 'business' | 'private'

interface FuhrparkPrefsState {
  /** Tab the module opens with. */
  defaultTab: FuhrparkTab
  /** Show the seasonal tire-change reminder banner. */
  showTireReminder: boolean
  /** Pre-selected category for new trip-log entries (market: Vimcar per-user default). */
  defaultTripCategory: TripCategory
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setDefaultTab: (v: FuhrparkTab) => void
  setShowTireReminder: (v: boolean) => void
  setDefaultTripCategory: (v: TripCategory) => void
  /** Hydrate from GET /settings/fuhrpark/user (once per session). */
  initFromServer: () => Promise<void>
}

function userPayload(s: FuhrparkPrefsState): Record<string, unknown> {
  return {
    defaultTab: s.defaultTab,
    showTireReminder: s.showTireReminder,
    defaultTripCategory: s.defaultTripCategory,
  }
}

export const useFuhrparkPrefsStore = create<FuhrparkPrefsState>()(
  persist(
    (set, get) => ({
      defaultTab: 'fahrzeuge',
      showTireReminder: true,
      defaultTripCategory: 'business',
      serverInitialized: false,
      setDefaultTab: (defaultTab) => {
        set({ defaultTab })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setShowTireReminder: (showTireReminder) => {
        set({ showTireReminder })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setDefaultTripCategory: (defaultTripCategory) => {
        set({ defaultTripCategory })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          defaultTab: TABS.includes(map.defaultTab as FuhrparkTab)
            ? (map.defaultTab as FuhrparkTab)
            : s.defaultTab,
          showTireReminder:
            typeof map.showTireReminder === 'boolean' ? map.showTireReminder : s.showTireReminder,
          defaultTripCategory:
            map.defaultTripCategory === 'business' || map.defaultTripCategory === 'private'
              ? map.defaultTripCategory
              : s.defaultTripCategory,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-fuhrpark-prefs' },
  ),
)
