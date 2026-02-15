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
import { DECO_ASSETS } from '@/config/desk-asset-urls'

/** Build default decorations for a given theme */
function buildDefaultDecorations(themeId: string): Record<string, DecorationPlacement> {
  const placements: Record<string, DecorationPlacement> = {
    'left-wall-clock': { slotId: 'left-wall-clock', type: 'clock', variant: 'analog' },
    'left-wall-calendar': { slotId: 'left-wall-calendar', type: 'calendar', variant: 'tearoff' },
  }

  // Add a few image decorations from the theme's available decos
  const themeDecos = DECO_ASSETS.filter((d) => d.themes.includes(themeId))
  const imageSlots = ['left-wall-deco', 'right-wall-deco1', 'desk-surface-right']
  themeDecos.slice(0, imageSlots.length).forEach((deco, i) => {
    placements[imageSlots[i]] = {
      slotId: imageSlots[i],
      type: 'image',
      variant: deco.id,
      imageUrl: deco.image,
      animationClass: deco.animation ?? undefined,
    }
  })

  return placements
}

interface UIState {
  sidebarCollapsed: boolean
  sidebarWidth: number
  sidebarMobileOpen: boolean
  locale: string
  theme: 'light' | 'dark' | 'auto'
  uiLook: 'solid' | 'glass' | 'crystal'

  // Desk environment
  deskMaximized: boolean
  deskThemeId: string
  deskDecorations: Record<string, DecorationPlacement>
  deskDecorationsVisible: boolean

  // Onboarding
  onboardingCompleted: boolean

  toggleSidebar: () => void
  setSidebarWidth: (width: number) => void
  setSidebarMobileOpen: (open: boolean) => void
  toggleSidebarMobile: () => void
  setLocale: (locale: string) => void
  setTheme: (theme: 'light' | 'dark' | 'auto') => void
  setUILook: (look: 'solid' | 'glass' | 'crystal') => void

  // Onboarding
  setOnboardingCompleted: (completed: boolean) => void

  // Desk actions
  toggleDeskMaximized: () => void
  setDeskMaximized: (maximized: boolean) => void
  setDeskThemeId: (themeId: string) => void
  setDeskThemeAndDecorations: (themeId: string) => void
  setDeskDecoration: (slotId: string, placement: DecorationPlacement | null) => void
  toggleDeskDecorations: () => void
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      sidebarCollapsed: false,
      sidebarWidth: 256,
      sidebarMobileOpen: false,
      locale: 'de',
      theme: 'light',
      uiLook: 'solid',

      // Desk defaults
      deskMaximized: false,
      deskThemeId: DEFAULT_DESK_THEME_ID,
      deskDecorations: buildDefaultDecorations(DEFAULT_DESK_THEME_ID),
      deskDecorationsVisible: true,
      onboardingCompleted: false,

      setOnboardingCompleted: (completed: boolean) =>
        set({ onboardingCompleted: completed }),

      toggleSidebar: () =>
        set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),

      setSidebarWidth: (width: number) =>
        set({ sidebarWidth: width }),

      setSidebarMobileOpen: (open: boolean) =>
        set({ sidebarMobileOpen: open }),

      toggleSidebarMobile: () =>
        set((state) => ({ sidebarMobileOpen: !state.sidebarMobileOpen })),

      setLocale: (locale: string) =>
        set({ locale }),

      setTheme: (theme: 'light' | 'dark' | 'auto') =>
        set({ theme }),

      setUILook: (look: 'solid' | 'glass') =>
        set({ uiLook: look }),

      toggleDeskMaximized: () =>
        set((state) => ({ deskMaximized: !state.deskMaximized })),

      setDeskMaximized: (maximized: boolean) =>
        set({ deskMaximized: maximized }),

      setDeskThemeId: (themeId: string) =>
        set({ deskThemeId: themeId }),

      /** Atomically switch theme and reset decorations to theme defaults */
      setDeskThemeAndDecorations: (themeId: string) =>
        set({
          deskThemeId: themeId,
          deskDecorations: buildDefaultDecorations(themeId),
        }),

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
