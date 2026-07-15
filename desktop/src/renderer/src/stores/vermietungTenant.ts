import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Tenant-weite Vermietung-Vorgaben ("Für alle"-Scope, siehe ModuleSettingsShell).
 * Editierbar nur durch Vermietung-Modul-Leiter/Admin (der PUT wird serverseitig
 * für andere Rollen verworfen; das Panel gatet die UI). Persönliche
 * Ansichts-Prefs liegen in stores/vermietungViewPrefs.ts.
 *
 * Feldauswahl folgt dem Markt (Rentman/Booqable/easyJob): Standardwährung,
 * Puffer-/Vorbereitungszeit zwischen Vermietungen und Kautions-Pflicht sind
 * dort durchgängig firmenweite Einstellungen.
 */
const MODULE_ID = 'vermietung'

export const VERMIETUNG_CURRENCY_OPTIONS = ['EUR', 'CHF', 'USD']
const BUFFER_OPTIONS = [0, 1, 2, 3]

interface VermietungTenantState {
  /** Currency pre-selected for new rental objects. */
  defaultCurrency: string
  /** Preparation buffer (days) blocked between two rentals of the same object. */
  bufferDays: number
  /** Whether a deposit is required before a rental can start. */
  requireDeposit: boolean
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setDefaultCurrency: (v: string) => void
  setBufferDays: (v: number) => void
  setRequireDeposit: (v: boolean) => void
  /** Hydrate from GET /settings/vermietung/tenant (once per session). */
  initFromServer: () => Promise<void>
}

/** The tenant-persisted keys, extracted as the PUT payload. */
function tenantPayload(s: VermietungTenantState): Record<string, unknown> {
  return {
    defaultCurrency: s.defaultCurrency,
    bufferDays: s.bufferDays,
    requireDeposit: s.requireDeposit,
  }
}

export const useVermietungTenantStore = create<VermietungTenantState>()(
  persist(
    (set, get) => ({
      defaultCurrency: 'EUR',
      bufferDays: 0,
      requireDeposit: false,
      serverInitialized: false,
      setDefaultCurrency: (defaultCurrency) => {
        set({ defaultCurrency })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setBufferDays: (bufferDays) => {
        set({ bufferDays: BUFFER_OPTIONS.includes(bufferDays) ? bufferDays : 0 })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setRequireDeposit: (requireDeposit) => {
        set({ requireDeposit })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'tenant')
        set((s) => ({
          defaultCurrency:
            typeof map.defaultCurrency === 'string' &&
            VERMIETUNG_CURRENCY_OPTIONS.includes(map.defaultCurrency)
              ? map.defaultCurrency
              : s.defaultCurrency,
          bufferDays: BUFFER_OPTIONS.includes(map.bufferDays as number)
            ? (map.bufferDays as number)
            : s.bufferDays,
          requireDeposit:
            typeof map.requireDeposit === 'boolean' ? map.requireDeposit : s.requireDeposit,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-vermietung-tenant' },
  ),
)
