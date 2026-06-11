import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * Personal vertraege (Verträge) preferences — per-user comfort settings
 * (personal scope, see ModuleSettingsShell). Persisted locally; no tenant
 * impact.
 */
export type VertraegeTab = 'aktiv' | 'auslaufend' | 'archiv' | 'vorlagen'
export type VertraegeDensity = 'comfortable' | 'compact'

interface VertraegePrefsState {
  /** Tab pre-selected when opening the module. */
  defaultTab: VertraegeTab
  /** Row density of the contract table. */
  density: VertraegeDensity
  /**
   * Personal reminder pre-selection for new contracts (days before end).
   * null = no personal override, the tenant default applies.
   */
  defaultReminderDays: number[] | null
  setDefaultTab: (t: VertraegeTab) => void
  setDensity: (d: VertraegeDensity) => void
  setDefaultReminderDays: (days: number[] | null) => void
}

export const useVertraegePrefsStore = create<VertraegePrefsState>()(
  persist(
    (set) => ({
      defaultTab: 'aktiv',
      density: 'comfortable',
      defaultReminderDays: null,
      setDefaultTab: (defaultTab) => set({ defaultTab }),
      setDensity: (density) => set({ density }),
      setDefaultReminderDays: (defaultReminderDays) => set({ defaultReminderDays }),
    }),
    { name: 'cosmi-vertraege-prefs' },
  ),
)
