import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { FinanceTabKey } from '@/stores/finance'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Personal Buchhaltung preferences (personal/user scope, see ModuleSettingsShell).
 * Server-synced via /settings/finance/user (X-4); hydrated by the central
 * settings hydrator on app shell mount, write-through on each setter.
 */
const MODULE_ID = 'finance'

/** 'last' = remember the last visited tab; otherwise force a fixed start tab. */
export type FinanceStartTab = 'last' | FinanceTabKey

interface FinancePrefsState {
  /** Which tab opens when entering Buchhaltung. */
  startTab: FinanceStartTab
  serverInitialized: boolean
  setStartTab: (t: FinanceStartTab) => void
  initFromServer: () => Promise<void>
}

export const useFinancePrefsStore = create<FinancePrefsState>()(
  persist(
    (set, get) => ({
      startTab: 'last',
      serverInitialized: false,
      setStartTab: (startTab) => {
        set({ startTab })
        void saveModuleSettings(MODULE_ID, 'user', { startTab })
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          startTab: typeof map.startTab === 'string' ? (map.startTab as FinanceStartTab) : s.startTab,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-finance-prefs' },
  ),
)
