import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Personal CRM preferences — per-user comfort settings that adapt the Kontakte
 * module to one's own workflow (personal/user scope, see ModuleSettingsShell).
 *
 * Server sync (settings foundation, Migr. 138, GET/PUT /settings/crm/user):
 *   - `initFromServer()` once per session hydrates from the backend (call on the
 *     CRM settings panel / Kontakte page mount); falls back to local defaults.
 *   - Each setter writes through to the user-scope endpoint. localStorage stays
 *     as the optimistic cache so the UI is instant and survives offline.
 */
const MODULE_ID = 'crm'

export type ContactView = 'list' | 'grid'
export type Density = 'comfortable' | 'compact'

interface CrmPrefsState {
  /** Default view when opening the contacts list. */
  defaultContactView: ContactView
  /** Row/card density. */
  density: Density
  /** Show contact avatars in lists. */
  showAvatars: boolean
  /** Whether the store has hydrated from the backend at least once this session. */
  serverInitialized: boolean
  setDefaultContactView: (v: ContactView) => void
  setDensity: (d: Density) => void
  setShowAvatars: (b: boolean) => void
  /** Hydrate from GET /settings/crm/user (once per session). */
  initFromServer: () => Promise<void>
}

/** The user-persisted keys, extracted as the PUT payload. */
function userPayload(s: CrmPrefsState): Record<string, unknown> {
  return {
    defaultContactView: s.defaultContactView,
    density: s.density,
    showAvatars: s.showAvatars,
  }
}

export const useCrmPrefsStore = create<CrmPrefsState>()(
  persist(
    (set, get) => ({
      defaultContactView: 'list',
      density: 'comfortable',
      showAvatars: true,
      serverInitialized: false,
      setDefaultContactView: (defaultContactView) => {
        set({ defaultContactView })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setDensity: (density) => {
        set({ density })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setShowAvatars: (showAvatars) => {
        set({ showAvatars })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          defaultContactView:
            map.defaultContactView === 'list' || map.defaultContactView === 'grid'
              ? map.defaultContactView
              : s.defaultContactView,
          density:
            map.density === 'comfortable' || map.density === 'compact'
              ? map.density
              : s.density,
          showAvatars: typeof map.showAvatars === 'boolean' ? map.showAvatars : s.showAvatars,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-crm-prefs' },
  ),
)
