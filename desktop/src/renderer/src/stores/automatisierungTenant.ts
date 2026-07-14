/**
 * Automatisierung — tenant-weite Vorgaben ("Für alle"-Scope, siehe
 * ModuleSettingsShell). Editierbar nur durch Modul-Leiter/Admin (der PUT wird
 * serverseitig für andere Rollen verworfen; das Panel gatet die UI). Persönliche
 * Prefs (Start-Tab) liegen in stores/automatisierungPrefs.ts.
 *
 * Server sync (settings foundation, Migr. 138, GET/PUT /settings/automatisierung/tenant):
 * initFromServer() hydratet einmal pro Session (via useHydrateModuleSettings);
 * Setter schreiben durch. localStorage bleibt Optimistic-Cache.
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

const MODULE_ID = 'automatisierung'

interface AutomatisierungTenantState {
  /** How long execution-log entries are retained (days). */
  logRetentionDays: number
  /** Whether a failed automation run notifies the team by default. */
  notifyOnFailure: boolean
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setLogRetentionDays: (days: number) => void
  setNotifyOnFailure: (on: boolean) => void
  initFromServer: () => Promise<void>
}

function tenantPayload(s: AutomatisierungTenantState): Record<string, unknown> {
  return { logRetentionDays: s.logRetentionDays, notifyOnFailure: s.notifyOnFailure }
}

export const useAutomatisierungTenantStore = create<AutomatisierungTenantState>()(
  persist(
    (set, get) => ({
      logRetentionDays: 90,
      notifyOnFailure: true,
      serverInitialized: false,
      setLogRetentionDays: (logRetentionDays) => {
        set({ logRetentionDays })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setNotifyOnFailure: (notifyOnFailure) => {
        set({ notifyOnFailure })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'tenant')
        set((s) => ({
          logRetentionDays:
            typeof map.logRetentionDays === 'number' ? map.logRetentionDays : s.logRetentionDays,
          notifyOnFailure:
            typeof map.notifyOnFailure === 'boolean' ? map.notifyOnFailure : s.notifyOnFailure,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-automatisierung-tenant' },
  ),
)
