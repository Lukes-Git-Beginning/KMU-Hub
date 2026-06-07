import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * Personal CRM preferences — per-user comfort settings that adapt the Kontakte
 * module to one's own workflow (personal scope, see ModuleSettingsShell).
 * Persisted locally; no tenant impact.
 */
export type ContactView = 'list' | 'grid'
export type Density = 'comfortable' | 'compact'

interface CrmPrefsState {
  /** Default view when opening the contacts list. */
  defaultContactView: ContactView
  /** Row/card density. */
  density: Density
  /** Show contact avatars in lists. */
  showAvatars: boolean
  setDefaultContactView: (v: ContactView) => void
  setDensity: (d: Density) => void
  setShowAvatars: (b: boolean) => void
}

export const useCrmPrefsStore = create<CrmPrefsState>()(
  persist(
    (set) => ({
      defaultContactView: 'list',
      density: 'comfortable',
      showAvatars: true,
      setDefaultContactView: (defaultContactView) => set({ defaultContactView }),
      setDensity: (density) => set({ density }),
      setShowAvatars: (showAvatars) => set({ showAvatars }),
    }),
    { name: 'cosmi-crm-prefs' },
  ),
)
