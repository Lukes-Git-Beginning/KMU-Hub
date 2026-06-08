import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * Personal preferences for the unified Kommunikation module (Team chat +
 * customer Posteingang). Scope: personal — every user adapts their own view.
 * Tenant-wide settings (channels, routing, team inbox) live elsewhere.
 */

export type KommunikationBereich = 'team' | 'posteingang'

/** Channels that can be muted for personal notifications. */
export type MutableChannel = 'email' | 'chat' | 'notification'

interface KommunikationPrefsState {
  /** Which area opens by default when entering the module without a query param. */
  defaultBereich: KommunikationBereich
  /** Show message density compact vs. comfortable. */
  density: 'comfortable' | 'compact'
  /** Send message on Enter (vs. Ctrl/Cmd+Enter). */
  enterToSend: boolean
  /** Channels the user has muted for notifications. */
  mutedChannels: MutableChannel[]
  setDefaultBereich: (b: KommunikationBereich) => void
  setDensity: (d: 'comfortable' | 'compact') => void
  setEnterToSend: (v: boolean) => void
  toggleMutedChannel: (c: MutableChannel) => void
}

export const useKommunikationPrefs = create<KommunikationPrefsState>()(
  persist(
    (set) => ({
      defaultBereich: 'posteingang',
      density: 'comfortable',
      enterToSend: true,
      mutedChannels: [],
      setDefaultBereich: (defaultBereich) => set({ defaultBereich }),
      setDensity: (density) => set({ density }),
      setEnterToSend: (enterToSend) => set({ enterToSend }),
      toggleMutedChannel: (c) =>
        set((state) => ({
          mutedChannels: state.mutedChannels.includes(c)
            ? state.mutedChannels.filter((x) => x !== c)
            : [...state.mutedChannels, c],
        })),
    }),
    { name: 'cosmi-kommunikation-prefs' },
  ),
)
