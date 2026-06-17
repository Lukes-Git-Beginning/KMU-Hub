import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * Personal helpdesk preferences — per-user comfort settings (personal scope,
 * see ModuleSettingsShell). Persisted locally; no tenant impact.
 */
export type HelpdeskStartTab = 'tickets' | 'wissensdatenbank' | 'statistik'
export type HelpdeskStatusDefault = 'all' | 'open' | 'in_progress' | 'waiting' | 'resolved' | 'closed'

interface HelpdeskPrefsState {
  /** Tab the helpdesk opens on. */
  startTab: HelpdeskStartTab
  /** Pre-selected status filter in the ticket list. */
  defaultStatusFilter: HelpdeskStatusDefault
  setStartTab: (t: HelpdeskStartTab) => void
  setDefaultStatusFilter: (s: HelpdeskStatusDefault) => void
}

export const useHelpdeskPrefsStore = create<HelpdeskPrefsState>()(
  persist(
    (set) => ({
      startTab: 'tickets',
      defaultStatusFilter: 'all',
      setStartTab: (startTab) => set({ startTab }),
      setDefaultStatusFilter: (defaultStatusFilter) => set({ defaultStatusFilter }),
    }),
    { name: 'cosmi-helpdesk-prefs' },
  ),
)
