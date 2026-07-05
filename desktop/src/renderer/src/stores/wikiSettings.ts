import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Wiki tenant settings (tenant scope — only Modul-Leiter/admin; the PUT is
 * dropped server-side for other roles, the panel gates the UI):
 *  - shareDefault: default access level proposed in the share dialog
 *  - publicModeEnabled: whether public share links are allowed at all
 * Category RBAC stays a backend concern and is only surfaced as a hint.
 *
 * Server sync (settings foundation, Migr. 138, GET/PUT /settings/wiki/tenant):
 *   - `initFromServer()` hydrates once per session (via useHydrateModuleSettings),
 *     else local defaults. localStorage stays the optimistic cache.
 */
const MODULE_ID = 'wiki'

export type WikiShareDefault = 'internal' | 'public'

interface WikiSettingsState {
  shareDefault: WikiShareDefault
  publicModeEnabled: boolean
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean

  setShareDefault: (s: WikiShareDefault) => void
  setPublicModeEnabled: (b: boolean) => void
  /** Hydrate from GET /settings/wiki/tenant (once per session). */
  initFromServer: () => Promise<void>
}

/** The tenant-persisted keys, extracted as the PUT payload. */
function tenantPayload(s: WikiSettingsState): Record<string, unknown> {
  return {
    shareDefault: s.shareDefault,
    publicModeEnabled: s.publicModeEnabled,
  }
}

export const useWikiSettingsStore = create<WikiSettingsState>()(
  persist(
    (set, get) => ({
      shareDefault: 'internal',
      publicModeEnabled: false,
      serverInitialized: false,
      setShareDefault: (shareDefault) => {
        set({ shareDefault })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      setPublicModeEnabled: (publicModeEnabled) => {
        set({ publicModeEnabled })
        void saveModuleSettings(MODULE_ID, 'tenant', tenantPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'tenant')
        set((s) => ({
          shareDefault:
            map.shareDefault === 'internal' || map.shareDefault === 'public'
              ? map.shareDefault
              : s.shareDefault,
          publicModeEnabled:
            typeof map.publicModeEnabled === 'boolean'
              ? map.publicModeEnabled
              : s.publicModeEnabled,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-wiki-settings' },
  ),
)
