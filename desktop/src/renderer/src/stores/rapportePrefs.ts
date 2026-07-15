import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Personal Rapporte preferences (personal/user scope, see ModuleSettingsShell).
 * Backed by the settings foundation (GET/PUT /settings/rapporte/user):
 * localStorage is the optimistic cache, `initFromServer()` hydrates once per
 * session (central hydrator) and each setter writes through.
 *
 * Default work times seed the new-report dialog — the per-user default shift
 * is the closest match to how HERO/mfr pre-fill daily reports.
 */
const MODULE_ID = 'rapporte'

export type RapporteDateFilter = 'week' | 'month' | 'all'
const DATE_FILTERS: RapporteDateFilter[] = ['week', 'month', 'all']

const TIME_RE = /^\d{2}:\d{2}$/

interface RapportePrefsState {
  /** Date filter the report list opens with. */
  defaultDateFilter: RapporteDateFilter
  /** Pre-filled work start for new reports (HH:mm). */
  defaultWorkStart: string
  /** Pre-filled work end for new reports (HH:mm). */
  defaultWorkEnd: string
  /** Pre-filled break minutes for new reports. */
  defaultBreakMinutes: number
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setDefaultDateFilter: (v: RapporteDateFilter) => void
  setDefaultWorkStart: (v: string) => void
  setDefaultWorkEnd: (v: string) => void
  setDefaultBreakMinutes: (v: number) => void
  /** Hydrate from GET /settings/rapporte/user (once per session). */
  initFromServer: () => Promise<void>
}

function userPayload(s: RapportePrefsState): Record<string, unknown> {
  return {
    defaultDateFilter: s.defaultDateFilter,
    defaultWorkStart: s.defaultWorkStart,
    defaultWorkEnd: s.defaultWorkEnd,
    defaultBreakMinutes: s.defaultBreakMinutes,
  }
}

export const useRapportePrefsStore = create<RapportePrefsState>()(
  persist(
    (set, get) => ({
      defaultDateFilter: 'all',
      defaultWorkStart: '07:00',
      defaultWorkEnd: '17:00',
      defaultBreakMinutes: 60,
      serverInitialized: false,
      setDefaultDateFilter: (defaultDateFilter) => {
        set({ defaultDateFilter })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setDefaultWorkStart: (defaultWorkStart) => {
        set({ defaultWorkStart })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setDefaultWorkEnd: (defaultWorkEnd) => {
        set({ defaultWorkEnd })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setDefaultBreakMinutes: (defaultBreakMinutes) => {
        set({ defaultBreakMinutes: Math.max(0, defaultBreakMinutes) })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          defaultDateFilter: DATE_FILTERS.includes(map.defaultDateFilter as RapporteDateFilter)
            ? (map.defaultDateFilter as RapporteDateFilter)
            : s.defaultDateFilter,
          defaultWorkStart:
            typeof map.defaultWorkStart === 'string' && TIME_RE.test(map.defaultWorkStart)
              ? map.defaultWorkStart
              : s.defaultWorkStart,
          defaultWorkEnd:
            typeof map.defaultWorkEnd === 'string' && TIME_RE.test(map.defaultWorkEnd)
              ? map.defaultWorkEnd
              : s.defaultWorkEnd,
          defaultBreakMinutes:
            typeof map.defaultBreakMinutes === 'number' && map.defaultBreakMinutes >= 0
              ? map.defaultBreakMinutes
              : s.defaultBreakMinutes,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-rapporte-prefs' },
  ),
)
