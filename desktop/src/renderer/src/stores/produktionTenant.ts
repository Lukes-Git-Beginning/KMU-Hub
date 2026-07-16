import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Tenant-weite Produktions-Vorgaben ("Für alle"-Scope, siehe ModuleSettingsShell).
 * Editierbar nur durch Produktions-Modul-Leiter/Admin (der PUT wird serverseitig
 * für andere Rollen verworfen; das Panel gatet die UI). Persönliche Prefs
 * liegen in stores/produktionPrefs.ts.
 *
 * Feldauswahl folgt dem Markt (Katana/MRPeasy/Xentral/weclapp): Nummernkreis
 * für Produktionsaufträge, Standard-Priorität und -Laufzeit für neue Aufträge,
 * Pflicht-QS vor Abschluss (MRPeasy-Inspektion / Xentral-Funktionstest-Pflicht)
 * und die Ausschuss-Warnschwelle sind dort firmenweite Einstellungen.
 */
const MODULE_ID = 'produktion'

interface ProduktionTenantState {
  /** Prefix for generated PA numbers, e.g. "PA" -> PA-2026-0716-1234. */
  orderNumberPrefix: string
  /** Default priority (1 highest … 5 lowest) pre-selected for new orders. */
  defaultPriority: number
  /** Default lead time in days — seeds the due date of the new-order dialog. */
  defaultLeadDays: number
  /** Completion requires at least one passed quality check on the order. */
  requireQcBeforeComplete: boolean
  /** Scrap rate (%) above which the detail shows the warning banner. */
  scrapWarnThreshold: number
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setOrderNumberPrefix: (v: string) => void
  setDefaultPriority: (v: number) => void
  setDefaultLeadDays: (v: number) => void
  setRequireQcBeforeComplete: (v: boolean) => void
  setScrapWarnThreshold: (v: number) => void
  /** Hydrate from GET /settings/produktion/tenant (once per session). */
  initFromServer: () => Promise<void>
}

/** The tenant-persisted keys, extracted as the PUT payload. */
function tenantPayload(s: ProduktionTenantState): Record<string, unknown> {
  return {
    orderNumberPrefix: s.orderNumberPrefix,
    defaultPriority: s.defaultPriority,
    defaultLeadDays: s.defaultLeadDays,
    requireQcBeforeComplete: s.requireQcBeforeComplete,
    scrapWarnThreshold: s.scrapWarnThreshold,
  }
}

function sanitizePrefix(v: string): string {
  return v.replace(/[^A-Za-z0-9-]/g, '').slice(0, 8).toUpperCase() || 'PA'
}

function clampPriority(v: number): number {
  return Math.min(5, Math.max(1, Math.round(v)))
}

function clampLeadDays(v: number): number {
  return Math.min(365, Math.max(1, Math.round(v)))
}

function clampScrapThreshold(v: number): number {
  return Math.min(100, Math.max(0, Math.round(v)))
}

export const useProduktionTenantStore = create<ProduktionTenantState>()(
  persist(
    (set, get) => ({
      orderNumberPrefix: 'PA',
      defaultPriority: 3,
      defaultLeadDays: 14,
      requireQcBeforeComplete: false,
      scrapWarnThreshold: 5,
      serverInitialized: false,
      setOrderNumberPrefix: (orderNumberPrefix) => {
        set({ orderNumberPrefix: sanitizePrefix(orderNumberPrefix) })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setDefaultPriority: (defaultPriority) => {
        set({ defaultPriority: clampPriority(defaultPriority) })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setDefaultLeadDays: (defaultLeadDays) => {
        set({ defaultLeadDays: clampLeadDays(defaultLeadDays) })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setRequireQcBeforeComplete: (requireQcBeforeComplete) => {
        set({ requireQcBeforeComplete })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setScrapWarnThreshold: (scrapWarnThreshold) => {
        set({ scrapWarnThreshold: clampScrapThreshold(scrapWarnThreshold) })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'tenant')
        set((s) => ({
          orderNumberPrefix:
            typeof map.orderNumberPrefix === 'string' && map.orderNumberPrefix
              ? sanitizePrefix(map.orderNumberPrefix)
              : s.orderNumberPrefix,
          defaultPriority:
            typeof map.defaultPriority === 'number'
              ? clampPriority(map.defaultPriority)
              : s.defaultPriority,
          defaultLeadDays:
            typeof map.defaultLeadDays === 'number'
              ? clampLeadDays(map.defaultLeadDays)
              : s.defaultLeadDays,
          requireQcBeforeComplete:
            typeof map.requireQcBeforeComplete === 'boolean'
              ? map.requireQcBeforeComplete
              : s.requireQcBeforeComplete,
          scrapWarnThreshold:
            typeof map.scrapWarnThreshold === 'number'
              ? clampScrapThreshold(map.scrapWarnThreshold)
              : s.scrapWarnThreshold,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-produktion-tenant' },
  ),
)
