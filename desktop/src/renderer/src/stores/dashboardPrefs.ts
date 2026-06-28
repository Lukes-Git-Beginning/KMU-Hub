import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Personal dashboard preferences (personal/user scope, see ModuleSettingsShell).
 * Server-synced via /settings/dashboard-prefs/user (X-4); hydrated by the central
 * settings hydrator on app shell mount, write-through on each setter.
 * (Module id distinct from the tenant dashboard layout under /dashboard/layout.)
 */
const MODULE_ID = 'dashboard-prefs'

export type DashboardDensity = 'comfortable' | 'compact'

interface DashboardPrefsState {
  /** Show the greeting header ("Guten Morgen …") above the dashboard. */
  showGreeting: boolean
  /** Widget grid density — compact shrinks row height and gaps. */
  density: DashboardDensity
  serverInitialized: boolean
  setShowGreeting: (v: boolean) => void
  setDensity: (d: DashboardDensity) => void
  initFromServer: () => Promise<void>
}

function userPayload(s: DashboardPrefsState): Record<string, unknown> {
  return { showGreeting: s.showGreeting, density: s.density }
}

export const useDashboardPrefsStore = create<DashboardPrefsState>()(
  persist(
    (set, get) => ({
      showGreeting: true,
      density: 'comfortable',
      serverInitialized: false,
      setShowGreeting: (showGreeting) => {
        set({ showGreeting })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setDensity: (density) => {
        set({ density })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          showGreeting: typeof map.showGreeting === 'boolean' ? map.showGreeting : s.showGreeting,
          density: map.density === 'comfortable' || map.density === 'compact' ? map.density : s.density,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-dashboard-prefs' },
  ),
)
