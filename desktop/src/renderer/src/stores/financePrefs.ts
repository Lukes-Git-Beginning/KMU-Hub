import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { FinanceTabKey } from '@/stores/finance'

/**
 * Personal Buchhaltung preferences (personal scope, see ModuleSettingsShell).
 * Per-user comfort; persisted locally, no tenant impact.
 */

/** 'last' = remember the last visited tab; otherwise force a fixed start tab. */
export type FinanceStartTab = 'last' | FinanceTabKey

interface FinancePrefsState {
  /** Which tab opens when entering Buchhaltung. */
  startTab: FinanceStartTab
  setStartTab: (t: FinanceStartTab) => void
}

export const useFinancePrefsStore = create<FinancePrefsState>()(
  persist(
    (set) => ({
      startTab: 'last',
      setStartTab: (startTab) => set({ startTab }),
    }),
    { name: 'cosmi-finance-prefs' },
  ),
)
