/**
 * UI state store (Zustand with localStorage persistence).
 *
 * Manages sidebar layout, locale, theme preferences, and desk
 * environment settings (desk theme, maximize state, decorations).
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { DecorationPlacement } from '@/types/desk-theme'
import { DEFAULT_DESK_THEME_ID } from '@/config/desk-themes'

interface UIState {
  sidebarCollapsed: boolean
  sidebarWidth: number
  locale: string
  theme: 'light' | 'dark'

  // Desk environment
  deskMaximized: boolean
  deskThemeId: string
  deskDecorations: Record<string, DecorationPlacement>
  deskDecorationsVisible: boolean

  toggleSidebar: () => void
  setSidebarWidth: (width: number) => void
  setLocale: (locale: string) => void
  setTheme: (theme: 'light' | 'dark') => void

  // Desk actions
  toggleDeskMaximized: () => void
  setDeskMaximized: (maximized: boolean) => void
  setDeskThemeId: (themeId: string) => void
  setDeskDecoration: (slotId: string, placement: DecorationPlacement | null) => void
  toggleDeskDecorations: () => void
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      sidebarCollapsed: false,
      sidebarWidth: 256,
      locale: 'de',
      theme: 'light',

      // Desk defaults
      deskMaximized: false,
      deskThemeId: DEFAULT_DESK_THEME_ID,
      deskDecorations: {
        'left-wall-clock': {
          slotId: 'left-wall-clock',
          type: 'clock',
          variant: 'analog',
        },
      },
      deskDecorationsVisible: true,

      toggleSidebar: () =>
        set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),

      setSidebarWidth: (width: number) =>
        set({ sidebarWidth: width }),

      setLocale: (locale: string) =>
        set({ locale }),

      setTheme: (theme: 'light' | 'dark') =>
        set({ theme }),

      toggleDeskMaximized: () =>
        set((state) => ({ deskMaximized: !state.deskMaximized })),

      setDeskMaximized: (maximized: boolean) =>
        set({ deskMaximized: maximized }),

      setDeskThemeId: (themeId: string) =>
        set({ deskThemeId: themeId }),

      setDeskDecoration: (slotId: string, placement: DecorationPlacement | null) =>
        set((state) => {
          const next = { ...state.deskDecorations }
          if (placement) {
            next[slotId] = placement
          } else {
            delete next[slotId]
          }
          return { deskDecorations: next }
        }),

      toggleDeskDecorations: () =>
        set((state) => ({ deskDecorationsVisible: !state.deskDecorationsVisible })),
    }),
    {
      name: 'kmuhub-ui',
    }
  )
)
