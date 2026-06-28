import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Personal zeiterfassung preferences (personal/user scope, see ModuleSettingsShell).
 * Server-synced via /settings/zeiterfassung/user (X-4); hydrated by the central
 * settings hydrator on app shell mount, write-through on each setter.
 */
const MODULE_ID = 'zeiterfassung'

export type ZeiterfassungDefaultView = 'today' | 'week' | 'analytics'

interface ZeiterfassungPrefsState {
  /** View pre-selected when opening the module. */
  defaultView: ZeiterfassungDefaultView
  /** Remind me to clock out at end of day. */
  clockOutReminder: boolean
  serverInitialized: boolean
  setDefaultView: (v: ZeiterfassungDefaultView) => void
  setClockOutReminder: (b: boolean) => void
  initFromServer: () => Promise<void>
}

function userPayload(s: ZeiterfassungPrefsState): Record<string, unknown> {
  return { defaultView: s.defaultView, clockOutReminder: s.clockOutReminder }
}

export const useZeiterfassungPrefsStore = create<ZeiterfassungPrefsState>()(
  persist(
    (set, get) => ({
      defaultView: 'today',
      clockOutReminder: true,
      serverInitialized: false,
      setDefaultView: (defaultView) => {
        set({ defaultView })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setClockOutReminder: (clockOutReminder) => {
        set({ clockOutReminder })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          defaultView:
            map.defaultView === 'today' || map.defaultView === 'week' || map.defaultView === 'analytics'
              ? map.defaultView
              : s.defaultView,
          clockOutReminder: typeof map.clockOutReminder === 'boolean' ? map.clockOutReminder : s.clockOutReminder,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-zeiterfassung-prefs' },
  ),
)
