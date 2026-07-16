import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Tenant-weite Einkauf-Vorgaben ("Für alle"-Scope, siehe ModuleSettingsShell).
 * Editierbar nur durch Einkauf-Modul-Leiter/Admin (der PUT wird serverseitig
 * für andere Rollen verworfen; das Panel gatet die UI). Persönliche Prefs
 * liegen in stores/einkaufPrefs.ts.
 *
 * Feldauswahl folgt dem Markt (weclapp/Xentral/Precoro): Freigabegrenze
 * (Approval-Limit), Standardwährung, Standard-Zahlungsbedingung und der
 * Bestellnummern-Präfix (Nummernkreis) sind dort firmenweite Einstellungen.
 * approvalThreshold ersetzt den gleichnamigen Wert aus dem alten
 * stores/einkauf.ts-Mock-Store.
 */
const MODULE_ID = 'einkauf'

export const EINKAUF_CURRENCY_OPTIONS = ['EUR', 'CHF', 'USD']

interface EinkaufTenantState {
  /** Orders above this amount show the approval banner (0 = no approvals). */
  approvalThreshold: number
  /** Default currency pre-selected for new purchase orders. */
  currency: string
  /** Default payment terms pre-selected for new suppliers. */
  defaultPaymentTerms: string
  /** Prefix for generated PO numbers, e.g. "PO" -> PO-2026-011. */
  poNumberPrefix: string
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setApprovalThreshold: (v: number) => void
  setCurrency: (v: string) => void
  setDefaultPaymentTerms: (v: string) => void
  setPoNumberPrefix: (v: string) => void
  /** Hydrate from GET /settings/einkauf/tenant (once per session). */
  initFromServer: () => Promise<void>
}

/** The tenant-persisted keys, extracted as the PUT payload. */
function tenantPayload(s: EinkaufTenantState): Record<string, unknown> {
  return {
    approvalThreshold: s.approvalThreshold,
    currency: s.currency,
    defaultPaymentTerms: s.defaultPaymentTerms,
    poNumberPrefix: s.poNumberPrefix,
  }
}

function clampThreshold(v: number): number {
  return Math.min(1_000_000, Math.max(0, Math.round(v)))
}

function sanitizePrefix(v: string): string {
  return v.replace(/[^A-Za-z0-9-]/g, '').slice(0, 8).toUpperCase() || 'PO'
}

export const useEinkaufTenantStore = create<EinkaufTenantState>()(
  persist(
    (set, get) => ({
      approvalThreshold: 5000,
      currency: 'EUR',
      defaultPaymentTerms: '30 Tage netto',
      poNumberPrefix: 'PO',
      serverInitialized: false,
      setApprovalThreshold: (approvalThreshold) => {
        set({ approvalThreshold: clampThreshold(approvalThreshold) })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setCurrency: (currency) => {
        set({ currency: EINKAUF_CURRENCY_OPTIONS.includes(currency) ? currency : 'EUR' })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setDefaultPaymentTerms: (defaultPaymentTerms) => {
        set({ defaultPaymentTerms })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setPoNumberPrefix: (poNumberPrefix) => {
        set({ poNumberPrefix: sanitizePrefix(poNumberPrefix) })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'tenant')
        set((s) => ({
          approvalThreshold:
            typeof map.approvalThreshold === 'number' && map.approvalThreshold >= 0
              ? clampThreshold(map.approvalThreshold)
              : s.approvalThreshold,
          currency:
            typeof map.currency === 'string' && EINKAUF_CURRENCY_OPTIONS.includes(map.currency)
              ? map.currency
              : s.currency,
          defaultPaymentTerms:
            typeof map.defaultPaymentTerms === 'string' && map.defaultPaymentTerms
              ? map.defaultPaymentTerms
              : s.defaultPaymentTerms,
          poNumberPrefix:
            typeof map.poNumberPrefix === 'string' && map.poNumberPrefix
              ? sanitizePrefix(map.poNumberPrefix)
              : s.poNumberPrefix,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-einkauf-tenant' },
  ),
)
