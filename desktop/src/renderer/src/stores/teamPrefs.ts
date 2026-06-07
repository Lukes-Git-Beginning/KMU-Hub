import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * Personal team preferences (personal scope, see ModuleSettingsShell).
 * Per-user comfort; persisted locally, no tenant impact.
 */
export type TeamStartTab = 'last' | 'members' | 'requests' | 'absences' | 'lohnvorbereitung'
export type TeamView = 'grid' | 'list'

interface TeamPrefsState {
  /** Which tab opens when entering Team. */
  startTab: TeamStartTab
  /** Default member view (grid cards vs. list rows). */
  defaultView: TeamView
  setStartTab: (t: TeamStartTab) => void
  setDefaultView: (v: TeamView) => void
}

export const useTeamPrefsStore = create<TeamPrefsState>()(
  persist(
    (set) => ({
      startTab: 'last',
      defaultView: 'grid',
      setStartTab: (startTab) => set({ startTab }),
      setDefaultView: (defaultView) => set({ defaultView }),
    }),
    { name: 'cosmi-team-prefs' },
  ),
)
