import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Personal Produktion preferences (personal/user scope, see ModuleSettingsShell).
 * Backed by the settings foundation (GET/PUT /settings/produktion/user):
 * localStorage is the optimistic cache, `initFromServer()` hydrates once per
 * session (central hydrator) and each setter writes through.
 */
const MODULE_ID = 'produktion'

export type ProduktionTab = 'auftraege' | 'stuecklisten' | 'qualitaet' | 'maschinen'
const TABS: ProduktionTab[] = ['auftraege', 'stuecklisten', 'qualitaet', 'maschinen']

export type ProduktionStatusFilter = 'all' | 'planned' | 'in_progress' | 'completed' | 'cancelled'
const STATUS_FILTERS: ProduktionStatusFilter[] = [
  'all', 'planned', 'in_progress', 'completed', 'cancelled',
]

interface ProduktionPrefsState {
  /** Tab the module opens with. */
  defaultTab: ProduktionTab
  /** Status filter pre-selected on the orders tab (market: saved list filters). */
  defaultStatusFilter: ProduktionStatusFilter
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setDefaultTab: (v: ProduktionTab) => void
  setDefaultStatusFilter: (v: ProduktionStatusFilter) => void
  /** Hydrate from GET /settings/produktion/user (once per session). */
  initFromServer: () => Promise<void>
}

function userPayload(s: ProduktionPrefsState): Record<string, unknown> {
  return {
    defaultTab: s.defaultTab,
    defaultStatusFilter: s.defaultStatusFilter,
  }
}

export const useProduktionPrefsStore = create<ProduktionPrefsState>()(
  persist(
    (set, get) => ({
      defaultTab: 'auftraege',
      defaultStatusFilter: 'all',
      serverInitialized: false,
      setDefaultTab: (defaultTab) => {
        set({ defaultTab })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setDefaultStatusFilter: (defaultStatusFilter) => {
        set({ defaultStatusFilter })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          defaultTab: TABS.includes(map.defaultTab as ProduktionTab)
            ? (map.defaultTab as ProduktionTab)
            : s.defaultTab,
          defaultStatusFilter: STATUS_FILTERS.includes(map.defaultStatusFilter as ProduktionStatusFilter)
            ? (map.defaultStatusFilter as ProduktionStatusFilter)
            : s.defaultStatusFilter,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-produktion-prefs' },
  ),
)
