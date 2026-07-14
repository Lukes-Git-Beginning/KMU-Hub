/**
 * Automatisierung — personal preferences (user scope, see ModuleSettingsShell).
 * The tenant-wide defaults (log retention, failure notifications) live in
 * stores/automatisierungTenant.ts.
 *
 * Server sync (settings foundation, Migr. 138, GET/PUT /settings/automatisierung/user):
 * initFromServer() hydrates once per session (via useHydrateModuleSettings);
 * each setter writes through. localStorage stays the optimistic cache.
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

const MODULE_ID = 'automatisierung'

export type AutomatisierungStartTab = 'automations' | 'templates' | 'log'
const START_TABS: AutomatisierungStartTab[] = ['automations', 'templates', 'log']

interface AutomatisierungPrefsState {
  startTab: AutomatisierungStartTab
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setStartTab: (tab: AutomatisierungStartTab) => void
  initFromServer: () => Promise<void>
}

function userPayload(s: AutomatisierungPrefsState): Record<string, unknown> {
  return { startTab: s.startTab }
}

export const useAutomatisierungPrefsStore = create<AutomatisierungPrefsState>()(
  persist(
    (set, get) => ({
      startTab: 'automations',
      serverInitialized: false,
      setStartTab: (startTab) => {
        set({ startTab })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          startTab: START_TABS.includes(map.startTab as AutomatisierungStartTab)
            ? (map.startTab as AutomatisierungStartTab)
            : s.startTab,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-automatisierung-prefs' },
  ),
)
