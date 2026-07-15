import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Tenant-weite Rapporte-Vorgaben ("Für alle"-Scope, siehe ModuleSettingsShell).
 * Editierbar nur durch Rapporte-Modul-Leiter/Admin (der PUT wird serverseitig
 * für andere Rollen verworfen; das Panel gatet die UI). Persönliche Prefs
 * liegen in stores/rapportePrefs.ts.
 *
 * Feldauswahl folgt dem Markt (HERO/mfr/Craftboxx): Pflicht-Unterschrift vor
 * dem Einreichen und die Firmen-Währung sind dort firmenweite Einstellungen.
 */
const MODULE_ID = 'rapporte'

export const RAPPORTE_CURRENCY_OPTIONS = ['EUR', 'CHF']

interface RapporteTenantState {
  /** Block submitting a report for approval until the customer signed. */
  requireSignature: boolean
  /** Currency used for material-cost stats and report pricing. */
  currency: string
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setRequireSignature: (v: boolean) => void
  setCurrency: (v: string) => void
  /** Hydrate from GET /settings/rapporte/tenant (once per session). */
  initFromServer: () => Promise<void>
}

/** The tenant-persisted keys, extracted as the PUT payload. */
function tenantPayload(s: RapporteTenantState): Record<string, unknown> {
  return {
    requireSignature: s.requireSignature,
    currency: s.currency,
  }
}

export const useRapporteTenantStore = create<RapporteTenantState>()(
  persist(
    (set, get) => ({
      requireSignature: false,
      currency: 'CHF',
      serverInitialized: false,
      setRequireSignature: (requireSignature) => {
        set({ requireSignature })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setCurrency: (currency) => {
        set({ currency: RAPPORTE_CURRENCY_OPTIONS.includes(currency) ? currency : 'CHF' })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'tenant')
        set((s) => ({
          requireSignature:
            typeof map.requireSignature === 'boolean' ? map.requireSignature : s.requireSignature,
          currency:
            typeof map.currency === 'string' && RAPPORTE_CURRENCY_OPTIONS.includes(map.currency)
              ? map.currency
              : s.currency,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-rapporte-tenant' },
  ),
)
