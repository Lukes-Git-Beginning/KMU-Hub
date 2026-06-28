import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Personal helpdesk preferences (personal/user scope, see ModuleSettingsShell).
 * Server-synced via /settings/helpdesk/user (X-4); hydrated by the central
 * settings hydrator on app shell mount, write-through on each setter.
 */
const MODULE_ID = 'helpdesk'

export type HelpdeskStartTab = 'tickets' | 'wissensdatenbank' | 'statistik'
export type HelpdeskStatusDefault = 'all' | 'open' | 'in_progress' | 'waiting' | 'resolved' | 'closed'

interface HelpdeskPrefsState {
  /** Tab the helpdesk opens on. */
  startTab: HelpdeskStartTab
  /** Pre-selected status filter in the ticket list. */
  defaultStatusFilter: HelpdeskStatusDefault
  serverInitialized: boolean
  setStartTab: (t: HelpdeskStartTab) => void
  setDefaultStatusFilter: (s: HelpdeskStatusDefault) => void
  initFromServer: () => Promise<void>
}

function userPayload(s: HelpdeskPrefsState): Record<string, unknown> {
  return { startTab: s.startTab, defaultStatusFilter: s.defaultStatusFilter }
}

export const useHelpdeskPrefsStore = create<HelpdeskPrefsState>()(
  persist(
    (set, get) => ({
      startTab: 'tickets',
      defaultStatusFilter: 'all',
      serverInitialized: false,
      setStartTab: (startTab) => {
        set({ startTab })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setDefaultStatusFilter: (defaultStatusFilter) => {
        set({ defaultStatusFilter })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          startTab: typeof map.startTab === 'string' ? (map.startTab as HelpdeskStartTab) : s.startTab,
          defaultStatusFilter:
            typeof map.defaultStatusFilter === 'string'
              ? (map.defaultStatusFilter as HelpdeskStatusDefault)
              : s.defaultStatusFilter,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-helpdesk-prefs' },
  ),
)
