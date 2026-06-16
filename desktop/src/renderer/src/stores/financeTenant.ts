import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { ChartFramework } from '@/modules/finanzen/lib/skr-accounts'

/**
 * Tenant-weite Buchhaltungs-Einstellungen ("Für alle"-Scope, siehe
 * ModuleSettingsShell). Editierbar nur durch Buchhaltungs-Modul-Leiter/Admin.
 *
 * Aktuell: der Kontenrahmen (SKR03/SKR04), der die Sachkonto-Vorschläge bei der
 * Kontierung steuert. Persistiert lokal; im echten Backend kommt das später aus
 * den CompanySettings (siehe finanzen P3 / DATEV-Export).
 */
interface FinanceTenantState {
  chartFramework: ChartFramework
  setChartFramework: (f: ChartFramework) => void
}

export const useFinanceTenantStore = create<FinanceTenantState>()(
  persist(
    (set) => ({
      chartFramework: 'SKR03',
      setChartFramework: (chartFramework) => set({ chartFramework }),
    }),
    { name: 'cosmi-finance-tenant' },
  ),
)
