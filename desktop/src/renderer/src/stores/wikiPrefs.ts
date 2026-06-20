import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * Wiki personal preferences — applied for real in the module:
 *  - defaultView: sidebar shows the category tree ('tree') or a flat list ('flat')
 *  - readingWidth: article/editor content width ('normal' constrained, 'wide' full)
 *  - sidebarDefaultOpen: whether category nodes start expanded
 */
export type WikiView = 'tree' | 'flat'
export type WikiReadingWidth = 'normal' | 'wide'

interface WikiPrefsState {
  defaultView: WikiView
  readingWidth: WikiReadingWidth
  sidebarDefaultOpen: boolean

  setDefaultView: (v: WikiView) => void
  setReadingWidth: (w: WikiReadingWidth) => void
  setSidebarDefaultOpen: (open: boolean) => void
}

export const useWikiPrefsStore = create<WikiPrefsState>()(
  persist(
    (set) => ({
      defaultView: 'tree',
      readingWidth: 'normal',
      sidebarDefaultOpen: true,
      setDefaultView: (defaultView) => set({ defaultView }),
      setReadingWidth: (readingWidth) => set({ readingWidth }),
      setSidebarDefaultOpen: (sidebarDefaultOpen) => set({ sidebarDefaultOpen }),
    }),
    { name: 'cosmi-wiki-prefs' },
  ),
)
