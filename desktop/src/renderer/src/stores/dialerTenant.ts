import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Tenant-weite Dialer-Vorgaben ("Für alle"-Scope, siehe ModuleSettingsShell).
 * Editierbar nur durch Dialer-Modul-Leiter/Admin (der PUT wird serverseitig für
 * andere Rollen verworfen; das Panel gatet die UI). Persönliche Komfort-Prefs
 * (Nachbearbeitungszeit, Auto-Advance) liegen in stores/dialerPrefs.ts.
 *
 * Server sync (settings foundation, Migr. 138, GET/PUT /settings/dialer/tenant):
 *   - `initFromServer()` hydratet einmal pro Session (via useHydrateModuleSettings),
 *     sonst lokale Defaults.
 *   - Setter schreiben durch (tenant-Scope). localStorage bleibt Optimistic-Cache.
 */
const MODULE_ID = 'dialer'

const CONCURRENT_OPTIONS = [1, 2, 3]

interface DialerTenantState {
  /** Max simultaneous live calls allowed per agent. */
  maxConcurrentCalls: number
  /** Whether the recording-consent toggle defaults to on for new calls. */
  recordingConsentDefault: boolean
  /** Outcome pre-selected in wrap-up (null = none). */
  defaultOutcomeId: string | null
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setMaxConcurrentCalls: (n: number) => void
  setRecordingConsentDefault: (v: boolean) => void
  setDefaultOutcomeId: (id: string | null) => void
  /** Hydrate from GET /settings/dialer/tenant (once per session). */
  initFromServer: () => Promise<void>
}

/** The tenant-persisted keys, extracted as the PUT payload. */
function tenantPayload(s: DialerTenantState): Record<string, unknown> {
  return {
    maxConcurrentCalls: s.maxConcurrentCalls,
    recordingConsentDefault: s.recordingConsentDefault,
    defaultOutcomeId: s.defaultOutcomeId,
  }
}

export const useDialerTenantStore = create<DialerTenantState>()(
  persist(
    (set, get) => ({
      maxConcurrentCalls: 1,
      recordingConsentDefault: true,
      defaultOutcomeId: null,
      serverInitialized: false,
      setMaxConcurrentCalls: (maxConcurrentCalls) => {
        set({ maxConcurrentCalls })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setRecordingConsentDefault: (recordingConsentDefault) => {
        set({ recordingConsentDefault })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setDefaultOutcomeId: (defaultOutcomeId) => {
        set({ defaultOutcomeId })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'tenant')
        set((s) => ({
          maxConcurrentCalls: CONCURRENT_OPTIONS.includes(map.maxConcurrentCalls as number)
            ? (map.maxConcurrentCalls as number)
            : s.maxConcurrentCalls,
          recordingConsentDefault:
            typeof map.recordingConsentDefault === 'boolean'
              ? map.recordingConsentDefault
              : s.recordingConsentDefault,
          defaultOutcomeId:
            typeof map.defaultOutcomeId === 'string' ? map.defaultOutcomeId : s.defaultOutcomeId,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-dialer-tenant' },
  ),
)
