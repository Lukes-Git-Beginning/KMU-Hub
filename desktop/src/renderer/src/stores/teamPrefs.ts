import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Personal team preferences (personal/user scope, see ModuleSettingsShell).
 * Server-synced via /settings/team/user (X-4); hydrated by the central settings
 * hydrator on app shell mount, write-through on each setter.
 */
const MODULE_ID = 'team'

export type TeamStartTab = 'last' | 'members' | 'requests' | 'absences' | 'lohnvorbereitung'
export type TeamView = 'grid' | 'list'

interface TeamPrefsState {
  /** Which tab opens when entering Team. */
  startTab: TeamStartTab
  /** Default member view (grid cards vs. list rows). */
  defaultView: TeamView
  serverInitialized: boolean
  setStartTab: (t: TeamStartTab) => void
  setDefaultView: (v: TeamView) => void
  initFromServer: () => Promise<void>
}

function userPayload(s: TeamPrefsState): Record<string, unknown> {
  return { startTab: s.startTab, defaultView: s.defaultView }
}

export const useTeamPrefsStore = create<TeamPrefsState>()(
  persist(
    (set, get) => ({
      startTab: 'last',
      defaultView: 'grid',
      serverInitialized: false,
      setStartTab: (startTab) => {
        set({ startTab })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setDefaultView: (defaultView) => {
        set({ defaultView })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          startTab: typeof map.startTab === 'string' ? (map.startTab as TeamStartTab) : s.startTab,
          defaultView: map.defaultView === 'grid' || map.defaultView === 'list' ? map.defaultView : s.defaultView,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-team-prefs' },
  ),
)
