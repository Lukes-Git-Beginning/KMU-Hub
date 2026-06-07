import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * Personal preferences for the unified Kommunikation module (Team chat +
 * customer Posteingang). Scope: personal — every user adapts their own view.
 * Tenant-wide settings (channels, routing, team inbox) live elsewhere.
 */

export type KommunikationBereich = 'team' | 'posteingang'

interface KommunikationPrefsState {
  /** Which area opens by default when entering the module without a query param. */
  defaultBereich: KommunikationBereich
  /** Show message density compact vs. comfortable. */
  density: 'comfortable' | 'compact'
  /** Send message on Enter (vs. Ctrl/Cmd+Enter). */
  enterToSend: boolean
  setDefaultBereich: (b: KommunikationBereich) => void
  setDensity: (d: 'comfortable' | 'compact') => void
  setEnterToSend: (v: boolean) => void
}

export const useKommunikationPrefs = create<KommunikationPrefsState>()(
  persist(
    (set) => ({
      defaultBereich: 'posteingang',
      density: 'comfortable',
      enterToSend: true,
      setDefaultBereich: (defaultBereich) => set({ defaultBereich }),
      setDensity: (density) => set({ density }),
      setEnterToSend: (enterToSend) => set({ enterToSend }),
    }),
    { name: 'cosmi-kommunikation-prefs' },
  ),
)
