import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * Personal dashboard preferences — per-user comfort settings (personal scope,
 * see ModuleSettingsShell). Persisted locally; no tenant impact.
 */
export type DashboardDensity = 'comfortable' | 'compact'

interface DashboardPrefsState {
  /** Show the greeting header ("Guten Morgen …") above the dashboard. */
  showGreeting: boolean
  /** Widget grid density — compact shrinks row height and gaps. */
  density: DashboardDensity
  setShowGreeting: (v: boolean) => void
  setDensity: (d: DashboardDensity) => void
}

export const useDashboardPrefsStore = create<DashboardPrefsState>()(
  persist(
    (set) => ({
      showGreeting: true,
      density: 'comfortable',
      setShowGreeting: (showGreeting) => set({ showGreeting }),
      setDensity: (density) => set({ density }),
    }),
    { name: 'cosmi-dashboard-prefs' },
  ),
)
