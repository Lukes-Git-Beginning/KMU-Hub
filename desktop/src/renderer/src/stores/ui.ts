/**
 * UI state store (Zustand with localStorage persistence).
 *
 * Manages sidebar layout, locale, and theme preferences.
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface UIState {
  sidebarCollapsed: boolean
  sidebarWidth: number
  locale: string
  theme: 'light' | 'dark'

  toggleSidebar: () => void
  setSidebarWidth: (width: number) => void
  setLocale: (locale: string) => void
  setTheme: (theme: 'light' | 'dark') => void
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      sidebarCollapsed: false,
      sidebarWidth: 256,
      locale: 'de',
      theme: 'light',

      toggleSidebar: () =>
        set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),

      setSidebarWidth: (width: number) =>
        set({ sidebarWidth: width }),

      setLocale: (locale: string) =>
        set({ locale }),

      setTheme: (theme: 'light' | 'dark') =>
        set({ theme }),
    }),
    {
      name: 'kmuhub-ui',
    }
  )
)
