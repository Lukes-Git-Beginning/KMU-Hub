import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * Wiki tenant settings (mock-first — no backend field yet, see
 * backend-handover-luke.md). Persisted locally so the demo is stateful:
 *  - shareDefault: default access level proposed in the share dialog
 *  - publicModeEnabled: whether public share links are allowed at all
 * Category RBAC stays a backend concern and is only surfaced as a hint.
 */
export type WikiShareDefault = 'internal' | 'public'

interface WikiSettingsState {
  shareDefault: WikiShareDefault
  publicModeEnabled: boolean

  setShareDefault: (s: WikiShareDefault) => void
  setPublicModeEnabled: (b: boolean) => void
}

export const useWikiSettingsStore = create<WikiSettingsState>()(
  persist(
    (set) => ({
      shareDefault: 'internal',
      publicModeEnabled: false,
      setShareDefault: (shareDefault) => set({ shareDefault }),
      setPublicModeEnabled: (publicModeEnabled) => set({ publicModeEnabled }),
    }),
    { name: 'cosmi-wiki-settings' },
  ),
)
