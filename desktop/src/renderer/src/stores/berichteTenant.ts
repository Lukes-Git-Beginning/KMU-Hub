/**
 * Berichte — tenant-weite Vorgaben ("Für alle"-Scope, siehe ModuleSettingsShell):
 * erlaubte Export-Formate + zugelassene Empfänger-Domains für geplante Berichte.
 * Editierbar nur durch Modul-Leiter/Admin (der PUT wird serverseitig für andere
 * Rollen verworfen; das Panel gatet die UI). Persönliche Prefs liegen in
 * stores/berichtePrefs.ts.
 *
 * Server sync (settings foundation, Migr. 138, GET/PUT /settings/berichte/tenant):
 * initFromServer() hydratet einmal pro Session (via useHydrateModuleSettings);
 * Setter schreiben durch. localStorage bleibt Optimistic-Cache.
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'
import type { ReportFormat } from '@/api/berichte-types'

const MODULE_ID = 'berichte'

interface BerichteTenantState {
  allowedFormats: ReportFormat[]
  scheduleDomains: string[]
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  toggleAllowedFormat: (f: ReportFormat) => void
  addScheduleDomain: (d: string) => void
  removeScheduleDomain: (d: string) => void
  initFromServer: () => Promise<void>
}

function tenantPayload(s: BerichteTenantState): Record<string, unknown> {
  return { allowedFormats: s.allowedFormats, scheduleDomains: s.scheduleDomains }
}

export const useBerichteTenantStore = create<BerichteTenantState>()(
  persist(
    (set, get) => ({
      allowedFormats: ['pdf', 'xlsx', 'csv'],
      scheduleDomains: ['zentria.tech'],
      serverInitialized: false,
      toggleAllowedFormat: (f) => {
        const cur = get().allowedFormats
        // keep at least one format enabled
        const next = cur.includes(f) ? cur.filter((x) => x !== f) : [...cur, f]
        if (next.length === 0) return
        set({ allowedFormats: next })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      addScheduleDomain: (d) => {
        const dom = d.trim().toLowerCase().replace(/^@/, '')
        if (!dom || get().scheduleDomains.includes(dom)) return
        set({ scheduleDomains: [...get().scheduleDomains, dom] })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      removeScheduleDomain: (d) => {
        set({ scheduleDomains: get().scheduleDomains.filter((x) => x !== d) })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'tenant')
        set((s) => ({
          allowedFormats:
            Array.isArray(map.allowedFormats) && map.allowedFormats.length > 0
              ? (map.allowedFormats.filter((x) => typeof x === 'string') as ReportFormat[])
              : s.allowedFormats,
          scheduleDomains: Array.isArray(map.scheduleDomains)
            ? (map.scheduleDomains.filter((x) => typeof x === 'string') as string[])
            : s.scheduleDomains,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-berichte-tenant' },
  ),
)
