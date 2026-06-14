import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * Personal zeiterfassung preferences (personal scope, see ModuleSettingsShell).
 * Persisted locally; no tenant impact. Wired for real into the module tab.
 */
export type ZeiterfassungDefaultView = 'today' | 'week' | 'analytics'

interface ZeiterfassungPrefsState {
  /** View pre-selected when opening the module. */
  defaultView: ZeiterfassungDefaultView
  /** Personal daily target in hours (drives the today progress bar). */
  dailyTargetHours: number
  /** Remind me to clock out at end of day. */
  clockOutReminder: boolean
  setDefaultView: (v: ZeiterfassungDefaultView) => void
  setDailyTargetHours: (h: number) => void
  setClockOutReminder: (b: boolean) => void
}

export const useZeiterfassungPrefsStore = create<ZeiterfassungPrefsState>()(
  persist(
    (set) => ({
      defaultView: 'today',
      dailyTargetHours: 8,
      clockOutReminder: true,
      setDefaultView: (defaultView) => set({ defaultView }),
      setDailyTargetHours: (dailyTargetHours) => set({ dailyTargetHours }),
      setClockOutReminder: (clockOutReminder) => set({ clockOutReminder }),
    }),
    { name: 'cosmi-zeiterfassung-prefs' },
  ),
)
