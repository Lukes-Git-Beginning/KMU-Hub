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
  /** Remind me to clock out at end of day. */
  clockOutReminder: boolean
  setDefaultView: (v: ZeiterfassungDefaultView) => void
  setClockOutReminder: (b: boolean) => void
}

export const useZeiterfassungPrefsStore = create<ZeiterfassungPrefsState>()(
  persist(
    (set) => ({
      defaultView: 'today',
      clockOutReminder: true,
      setDefaultView: (defaultView) => set({ defaultView }),
      setClockOutReminder: (clockOutReminder) => set({ clockOutReminder }),
    }),
    { name: 'cosmi-zeiterfassung-prefs' },
  ),
)
