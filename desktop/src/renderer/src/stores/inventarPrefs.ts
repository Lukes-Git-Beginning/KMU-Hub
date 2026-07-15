import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Personal Inventar preferences — per-user view defaults for the Inventar
 * module (personal/user scope, see ModuleSettingsShell). Backed by the
 * settings foundation (GET/PUT /settings/inventar/user): localStorage is the
 * optimistic cache, `initFromServer()` hydrates once per session (central
 * hydrator) and each setter writes through.
 */
const MODULE_ID = 'inventar'

export type InventarTab = 'artikel' | 'lagerorte' | 'bewegungen' | 'inventur'
const TABS: InventarTab[] = ['artikel', 'lagerorte', 'bewegungen', 'inventur']

export type InventarDensity = 'comfortable' | 'compact'
const DENSITIES: InventarDensity[] = ['comfortable', 'compact']

interface InventarPrefsState {
  /** Tab the module opens on. */
  defaultTab: InventarTab
  /** Table row density (comfortable = default padding, compact = tighter rows). */
  density: InventarDensity
  /** Show low-stock warnings (reorder icons, critical banner, header counters). */
  showMinStockWarnings: boolean
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setDefaultTab: (v: InventarTab) => void
  setDensity: (v: InventarDensity) => void
  setShowMinStockWarnings: (v: boolean) => void
  /** Hydrate from GET /settings/inventar/user (once per session). */
  initFromServer: () => Promise<void>
}

function userPayload(s: InventarPrefsState): Record<string, unknown> {
  return {
    defaultTab: s.defaultTab,
    density: s.density,
    showMinStockWarnings: s.showMinStockWarnings,
  }
}

export const useInventarPrefsStore = create<InventarPrefsState>()(
  persist(
    (set, get) => ({
      defaultTab: 'artikel',
      density: 'comfortable',
      showMinStockWarnings: true,
      serverInitialized: false,
      setDefaultTab: (defaultTab) => {
        set({ defaultTab })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setDensity: (density) => {
        set({ density })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setShowMinStockWarnings: (showMinStockWarnings) => {
        set({ showMinStockWarnings })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          defaultTab: TABS.includes(map.defaultTab as InventarTab)
            ? (map.defaultTab as InventarTab)
            : s.defaultTab,
          density: DENSITIES.includes(map.density as InventarDensity)
            ? (map.density as InventarDensity)
            : s.density,
          showMinStockWarnings:
            typeof map.showMinStockWarnings === 'boolean'
              ? map.showMinStockWarnings
              : s.showMinStockWarnings,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-inventar-prefs' },
  ),
)
