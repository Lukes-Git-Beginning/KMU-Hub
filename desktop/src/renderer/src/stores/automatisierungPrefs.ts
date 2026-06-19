/**
 * Automatisierung preferences (persisted, mock-first).
 *
 * personal: which tab opens by default (wired into AutomatisierungPage).
 * tenant: execution-log retention + failure-notification default — demo-stateful,
 * gated to a Modul-Leiter / admin by ModuleSettingsShell.
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type AutomatisierungStartTab = 'automations' | 'templates' | 'log'

interface AutomatisierungPrefsState {
  // personal
  startTab: AutomatisierungStartTab
  setStartTab: (tab: AutomatisierungStartTab) => void
  // tenant
  logRetentionDays: number
  setLogRetentionDays: (days: number) => void
  notifyOnFailure: boolean
  setNotifyOnFailure: (on: boolean) => void
}

export const useAutomatisierungPrefsStore = create<AutomatisierungPrefsState>()(
  persist(
    (set) => ({
      startTab: 'automations',
      setStartTab: (startTab) => set({ startTab }),
      logRetentionDays: 90,
      setLogRetentionDays: (logRetentionDays) => set({ logRetentionDays }),
      notifyOnFailure: true,
      setNotifyOnFailure: (notifyOnFailure) => set({ notifyOnFailure }),
    }),
    { name: 'cosmi-automatisierung-prefs' },
  ),
)
