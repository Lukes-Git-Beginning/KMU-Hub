import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Personal Dialer preferences — per-user comfort settings for the call workspace
 * (personal/user scope, see ModuleSettingsShell). The tenant-wide defaults
 * (max concurrent, recording-consent, default outcome) live in stores/dialerTenant.ts.
 *
 * Server sync (settings foundation, Migr. 138, GET/PUT /settings/dialer/user):
 *   - `initFromServer()` hydrates once per session (via useHydrateModuleSettings);
 *     falls back to local defaults.
 *   - Each setter writes through to the user-scope endpoint. localStorage stays as
 *     the optimistic cache so the UI is instant and survives offline.
 */
const MODULE_ID = 'dialer'

const WRAP_UP_OPTIONS = [15, 30, 45, 60]

interface DialerPrefsState {
  /** Seconds the wrap-up timer counts before auto-advance kicks in. */
  defaultWrapUpSeconds: number
  /** After completing wrap-up, load the next queued contact automatically. */
  autoAdvance: boolean
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setDefaultWrapUpSeconds: (s: number) => void
  setAutoAdvance: (v: boolean) => void
  /** Hydrate from GET /settings/dialer/user (once per session). */
  initFromServer: () => Promise<void>
}

/** The user-persisted keys, extracted as the PUT payload. */
function userPayload(s: DialerPrefsState): Record<string, unknown> {
  return {
    defaultWrapUpSeconds: s.defaultWrapUpSeconds,
    autoAdvance: s.autoAdvance,
  }
}

export const useDialerPrefsStore = create<DialerPrefsState>()(
  persist(
    (set, get) => ({
      defaultWrapUpSeconds: 30,
      autoAdvance: false,
      serverInitialized: false,
      setDefaultWrapUpSeconds: (defaultWrapUpSeconds) => {
        set({ defaultWrapUpSeconds })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setAutoAdvance: (autoAdvance) => {
        set({ autoAdvance })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          defaultWrapUpSeconds: WRAP_UP_OPTIONS.includes(map.defaultWrapUpSeconds as number)
            ? (map.defaultWrapUpSeconds as number)
            : s.defaultWrapUpSeconds,
          autoAdvance: typeof map.autoAdvance === 'boolean' ? map.autoAdvance : s.autoAdvance,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-dialer-prefs' },
  ),
)
