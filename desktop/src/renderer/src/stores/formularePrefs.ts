/**
 * Formulare — personal preferences (user scope, see ModuleSettingsShell): default
 * tab, preferred export format and the forms-tab grid/list toggle. The tenant-wide
 * DSGVO defaults (consent text, privacy link, submission notification, retention)
 * live in stores/formulareTenant.ts.
 *
 * Server sync (settings foundation, Migr. 138, GET/PUT /settings/formulare/user):
 * initFromServer() hydrates once per session (via useHydrateModuleSettings);
 * each setter writes through. localStorage stays the optimistic cache.
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'
import type { ExportFormat } from '@/api/formulare-types'

const MODULE_ID = 'formulare'

export type FormulareTab = 'formulare' | 'eingänge' | 'vorlagen'
export type FormulareView = 'grid' | 'list'
const TABS: FormulareTab[] = ['formulare', 'eingänge', 'vorlagen']
const VIEWS: FormulareView[] = ['grid', 'list']

interface FormularePrefsState {
  defaultTab: FormulareTab
  defaultExportFormat: ExportFormat
  /** FT-5 — persists the FT-4 forms-tab grid/list toggle. */
  defaultFormView: FormulareView
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setDefaultTab: (t: FormulareTab) => void
  setDefaultExportFormat: (f: ExportFormat) => void
  setDefaultFormView: (v: FormulareView) => void
  initFromServer: () => Promise<void>
}

function userPayload(s: FormularePrefsState): Record<string, unknown> {
  return {
    defaultTab: s.defaultTab,
    defaultExportFormat: s.defaultExportFormat,
    defaultFormView: s.defaultFormView,
  }
}

export const useFormularePrefsStore = create<FormularePrefsState>()(
  persist(
    (set, get) => ({
      defaultTab: 'formulare',
      defaultExportFormat: 'csv',
      defaultFormView: 'grid',
      serverInitialized: false,
      setDefaultTab: (defaultTab) => {
        set({ defaultTab })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setDefaultExportFormat: (defaultExportFormat) => {
        set({ defaultExportFormat })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setDefaultFormView: (defaultFormView) => {
        set({ defaultFormView })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          defaultTab: TABS.includes(map.defaultTab as FormulareTab)
            ? (map.defaultTab as FormulareTab)
            : s.defaultTab,
          defaultExportFormat:
            typeof map.defaultExportFormat === 'string'
              ? (map.defaultExportFormat as ExportFormat)
              : s.defaultExportFormat,
          defaultFormView: VIEWS.includes(map.defaultFormView as FormulareView)
            ? (map.defaultFormView as FormulareView)
            : s.defaultFormView,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-formulare-prefs' },
  ),
)
