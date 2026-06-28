import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { loadModuleSettings, saveModuleSettings } from '@/api/settings-persist'

/**
 * Wiki personal preferences (personal/user scope) — applied for real in the module:
 *  - defaultView: sidebar shows the category tree ('tree') or a flat list ('flat')
 *  - readingWidth: article/editor content width ('normal' constrained, 'wide' full)
 *  - sidebarDefaultOpen: whether category nodes start expanded
 * Server-synced via /settings/wiki/user (X-4); hydrated by the central settings
 * hydrator on app shell mount, write-through on each setter.
 */
const MODULE_ID = 'wiki'

export type WikiView = 'tree' | 'flat'
export type WikiReadingWidth = 'normal' | 'wide'

interface WikiPrefsState {
  defaultView: WikiView
  readingWidth: WikiReadingWidth
  sidebarDefaultOpen: boolean
  serverInitialized: boolean
  setDefaultView: (v: WikiView) => void
  setReadingWidth: (w: WikiReadingWidth) => void
  setSidebarDefaultOpen: (open: boolean) => void
  initFromServer: () => Promise<void>
}

function userPayload(s: WikiPrefsState): Record<string, unknown> {
  return {
    defaultView: s.defaultView,
    readingWidth: s.readingWidth,
    sidebarDefaultOpen: s.sidebarDefaultOpen,
  }
}

export const useWikiPrefsStore = create<WikiPrefsState>()(
  persist(
    (set, get) => ({
      defaultView: 'tree',
      readingWidth: 'normal',
      sidebarDefaultOpen: true,
      serverInitialized: false,
      setDefaultView: (defaultView) => {
        set({ defaultView })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setReadingWidth: (readingWidth) => {
        set({ readingWidth })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      setSidebarDefaultOpen: (sidebarDefaultOpen) => {
        set({ sidebarDefaultOpen })
        void saveModuleSettings(MODULE_ID, 'user', userPayload(get()))
      },
      initFromServer: async () => {
        if (get().serverInitialized) return
        const map = await loadModuleSettings(MODULE_ID, 'user')
        set((s) => ({
          defaultView: map.defaultView === 'tree' || map.defaultView === 'flat' ? map.defaultView : s.defaultView,
          readingWidth:
            map.readingWidth === 'normal' || map.readingWidth === 'wide' ? map.readingWidth : s.readingWidth,
          sidebarDefaultOpen:
            typeof map.sidebarDefaultOpen === 'boolean' ? map.sidebarDefaultOpen : s.sidebarDefaultOpen,
          serverInitialized: true,
        }))
      },
    }),
    { name: 'cosmi-wiki-prefs' },
  ),
)
