/**
 * Berichte (Reports/BI) — personal preferences (user scope, see
 * ModuleSettingsShell): default export format, period and colour palette for new
 * reports (applied in the ReportBuilder + schedule dialog). The tenant-wide
 * defaults (allowed formats, recipient domains) live in stores/berichteTenant.ts.
 *
 * Server sync (settings foundation, Migr. 138, GET/PUT /settings/berichte/user):
 * initFromServer() hydrates once per session (via useHydrateModuleSettings);
 * each setter writes through. localStorage stays the optimistic cache.
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'
import type { ReportFormat } from '@/api/berichte-types'

const MODULE_ID = 'berichte'

export type BerichtePeriod = 'last30' | 'thisMonth' | 'thisQuarter' | 'thisYear'
const PERIODS: BerichtePeriod[] = ['last30', 'thisMonth', 'thisQuarter', 'thisYear']

interface BerichtePrefsState {
  defaultFormat: ReportFormat
  defaultPeriod: BerichtePeriod
  /** Default colour palette for new builder reports (palette id). */
  defaultPalette: string
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setDefaultFormat: (f: ReportFormat) => void
  setDefaultPeriod: (p: BerichtePeriod) => void
  setDefaultPalette: (p: string) => void
  initFromServer: () => Promise<void>
}

function userPayload(s: BerichtePrefsState): Record<string, unknown> {
  return {
    defaultFormat: s.defaultFormat,
    defaultPeriod: s.defaultPeriod,
    defaultPalette: s.defaultPalette,
  }
}

export const useBerichtePrefsStore = create<BerichtePrefsState>()(
  persist(
    (set, get) => ({
      defaultFormat: 'pdf',
      defaultPeriod: 'thisMonth',
      defaultPalette: 'default',
      serverInitialized: false,
      setDefaultFormat: (defaultFormat) => {
        set({ defaultFormat })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setDefaultPeriod: (defaultPeriod) => {
        set({ defaultPeriod })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setDefaultPalette: (defaultPalette) => {
        set({ defaultPalette })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          defaultFormat:
            typeof map.defaultFormat === 'string'
              ? (map.defaultFormat as ReportFormat)
              : s.defaultFormat,
          defaultPeriod: PERIODS.includes(map.defaultPeriod as BerichtePeriod)
            ? (map.defaultPeriod as BerichtePeriod)
            : s.defaultPeriod,
          defaultPalette:
            typeof map.defaultPalette === 'string' ? map.defaultPalette : s.defaultPalette,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-berichte-prefs' },
  ),
)
