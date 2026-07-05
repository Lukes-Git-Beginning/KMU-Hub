import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'
import type { ChartFramework } from '@/modules/finanzen/lib/skr-accounts'

/**
 * Tenant-weite Buchhaltungs-Einstellungen ("Für alle"-Scope, siehe
 * ModuleSettingsShell). Editierbar nur durch Buchhaltungs-Modul-Leiter/Admin
 * (der PUT wird serverseitig für andere Rollen verworfen; das Panel gatet die UI).
 *
 * Aktuell: der Kontenrahmen (SKR03/SKR04), der die Sachkonto-Vorschläge bei der
 * Kontierung steuert.
 *
 * Server sync (settings foundation, Migr. 138, GET/PUT /settings/finance/tenant):
 *   - `initFromServer()` hydratet einmal pro Session aus dem Backend (via
 *     useHydrateModuleSettings), sonst lokale Defaults.
 *   - Setter schreiben durch (tenant-Scope). localStorage bleibt Optimistic-Cache.
 */
const MODULE_ID = 'finance'

const CHART_FRAMEWORKS: ChartFramework[] = ['SKR03', 'SKR04']

interface FinanceTenantState {
  chartFramework: ChartFramework
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setChartFramework: (f: ChartFramework) => void
  /** Hydrate from GET /settings/finance/tenant (once per session). */
  initFromServer: () => Promise<void>
}

/** The tenant-persisted keys, extracted as the PUT payload. */
function tenantPayload(s: FinanceTenantState): Record<string, unknown> {
  return { chartFramework: s.chartFramework }
}

export const useFinanceTenantStore = create<FinanceTenantState>()(
  persist(
    (set, get) => ({
      chartFramework: 'SKR03',
      serverInitialized: false,
      setChartFramework: (chartFramework) => {
        set({ chartFramework })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'tenant')
        set((s) => ({
          chartFramework: CHART_FRAMEWORKS.includes(map.chartFramework as ChartFramework)
            ? (map.chartFramework as ChartFramework)
            : s.chartFramework,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-finance-tenant' },
  ),
)
