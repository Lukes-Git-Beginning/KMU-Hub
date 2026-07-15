import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Personal Vermietung view preferences (personal/user scope, see
 * ModuleSettingsShell). Backed by the settings foundation
 * (GET/PUT /settings/vermietung/user): localStorage is the optimistic cache,
 * `initFromServer()` hydrates once per session (central hydrator) and each
 * setter writes through.
 *
 * NOTE: stores/vermietungPrefs.ts is a different store — it holds per-object/
 * per-rental mock companion data (weekly rate, serial number, pickup/return),
 * not user settings.
 */
const MODULE_ID = 'vermietung'

export type VermietungTab = 'objekte' | 'reservierungen' | 'kalender'
const TABS: VermietungTab[] = ['objekte', 'reservierungen', 'kalender']

interface VermietungViewPrefsState {
  /** Tab the module opens on. */
  defaultTab: VermietungTab
  /** Show the KPI row (availability / utilization) above the tabs. */
  showKpis: boolean
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setDefaultTab: (v: VermietungTab) => void
  setShowKpis: (v: boolean) => void
  /** Hydrate from GET /settings/vermietung/user (once per session). */
  initFromServer: () => Promise<void>
}

function userPayload(s: VermietungViewPrefsState): Record<string, unknown> {
  return {
    defaultTab: s.defaultTab,
    showKpis: s.showKpis,
  }
}

export const useVermietungViewPrefsStore = create<VermietungViewPrefsState>()(
  persist(
    (set, get) => ({
      defaultTab: 'objekte',
      showKpis: true,
      serverInitialized: false,
      setDefaultTab: (defaultTab) => {
        set({ defaultTab })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setShowKpis: (showKpis) => {
        set({ showKpis })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          defaultTab: TABS.includes(map.defaultTab as VermietungTab)
            ? (map.defaultTab as VermietungTab)
            : s.defaultTab,
          showKpis: typeof map.showKpis === 'boolean' ? map.showKpis : s.showKpis,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-vermietung-view-prefs' },
  ),
)
