import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Personal Einkauf preferences (personal/user scope, see ModuleSettingsShell).
 * Backed by the settings foundation (GET/PUT /settings/einkauf/user):
 * localStorage is the optimistic cache, `initFromServer()` hydrates once per
 * session (central hydrator) and each setter writes through.
 */
const MODULE_ID = 'einkauf'

export type EinkaufTab = 'bestellungen' | 'lieferanten' | 'katalog' | 'rahmenverträge'
const TABS: EinkaufTab[] = ['bestellungen', 'lieferanten', 'katalog', 'rahmenverträge']

export type EinkaufStatusFilter =
  | 'all'
  | 'draft'
  | 'submitted'
  | 'sent'
  | 'partially_received'
  | 'received'
  | 'cancelled'
const STATUS_FILTERS: EinkaufStatusFilter[] = [
  'all', 'draft', 'submitted', 'sent', 'partially_received', 'received', 'cancelled',
]

interface EinkaufPrefsState {
  /** Tab the module opens with. */
  defaultTab: EinkaufTab
  /** Status filter pre-selected on the orders tab (market: saved list filters). */
  defaultStatusFilter: EinkaufStatusFilter
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setDefaultTab: (v: EinkaufTab) => void
  setDefaultStatusFilter: (v: EinkaufStatusFilter) => void
  /** Hydrate from GET /settings/einkauf/user (once per session). */
  initFromServer: () => Promise<void>
}

function userPayload(s: EinkaufPrefsState): Record<string, unknown> {
  return {
    defaultTab: s.defaultTab,
    defaultStatusFilter: s.defaultStatusFilter,
  }
}

export const useEinkaufPrefsStore = create<EinkaufPrefsState>()(
  persist(
    (set, get) => ({
      defaultTab: 'bestellungen',
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
          defaultTab: TABS.includes(map.defaultTab as EinkaufTab)
            ? (map.defaultTab as EinkaufTab)
            : s.defaultTab,
          defaultStatusFilter: STATUS_FILTERS.includes(map.defaultStatusFilter as EinkaufStatusFilter)
            ? (map.defaultStatusFilter as EinkaufStatusFilter)
            : s.defaultStatusFilter,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-einkauf-prefs' },
  ),
)
