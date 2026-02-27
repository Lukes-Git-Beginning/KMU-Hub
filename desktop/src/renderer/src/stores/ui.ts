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

export type ColorTheme = 'graphit' | 'sand' | 'ozean' | 'lavendel' | 'wald' | 'rose' | 'mitternacht' | 'terrakotta'
export type NavLayout = 'sidebar' | 'dock' | 'topnav' | 'classic'
export type AccentIntensity = 'subtle' | 'vivid'
export type WindowStyle = 'full' | 'bubble'

interface UIState {
  sidebarCollapsed: boolean
  sidebarWidth: number
  sidebarMobileOpen: boolean
  locale: string
  theme: 'light' | 'dark' | 'auto'
  uiLook: 'solid' | 'glass' | 'crystal'

  // Design system (Wave 14+)
  colorTheme: ColorTheme
  navLayout: NavLayout
  accentIntensity: AccentIntensity

  // Window style
  windowStyle: WindowStyle

  // Sidebar pinned modules
  pinnedModules: string[]

  // Desk environment
  deskMaximized: boolean
  deskThemeId: string
  deskDecorations: Record<string, DecorationPlacement>
  deskDecorationsVisible: boolean

  // Background pattern
  backgroundPattern: string

  // Desk background (gradient/image preset for glass mode)
  deskBackground: string | null

  // Header widgets (up to 3 mini-widget slots)
  headerWidgets: string[]

  // Onboarding
  onboardingCompleted: boolean

  setBackgroundPattern: (pattern: string) => void
  setDeskBackground: (bg: string | null) => void

  setHeaderWidgets: (widgets: string[]) => void
  toggleHeaderWidget: (widgetId: string) => void

  toggleSidebar: () => void
  setSidebarWidth: (width: number) => void
  setSidebarMobileOpen: (open: boolean) => void
  toggleSidebarMobile: () => void
  setLocale: (locale: string) => void
  setTheme: (theme: 'light' | 'dark' | 'auto') => void
  setUILook: (look: 'solid' | 'glass' | 'crystal') => void
  setColorTheme: (theme: ColorTheme) => void
  setNavLayout: (layout: NavLayout) => void
  setAccentIntensity: (intensity: AccentIntensity) => void
  setWindowStyle: (style: WindowStyle) => void

  // Sidebar pinned modules
  setPinnedModules: (modules: string[]) => void
  togglePinModule: (moduleId: string) => void

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

      // Design system (Wave 14+)
      colorTheme: 'graphit',
      navLayout: 'sidebar',
      accentIntensity: 'subtle',
      windowStyle: 'full' as WindowStyle,

      // Default pinned modules (most-used for typical KMU)
      pinnedModules: [
        'dashboard', 'projects', 'tasks', 'chat', 'contacts',
        'team', 'calendar', 'mail', 'finance',
      ],

      // Background pattern default
      backgroundPattern: 'none',
      deskBackground: null,

      // Header widget defaults
      headerWidgets: ['next-meeting', 'weather', 'pomodoro'],

      // Desk defaults
      deskMaximized: false,
      deskThemeId: DEFAULT_DESK_THEME_ID,
      deskDecorations: buildDefaultDecorations(DEFAULT_DESK_THEME_ID),
      deskDecorationsVisible: true,
      onboardingCompleted: false,

      setPinnedModules: (pinnedModules) =>
        set({ pinnedModules }),

      togglePinModule: (moduleId) =>
        set((state) => {
          const has = state.pinnedModules.includes(moduleId)
          return {
            pinnedModules: has
              ? state.pinnedModules.filter((id) => id !== moduleId)
              : [...state.pinnedModules, moduleId],
          }
        }),

      setBackgroundPattern: (backgroundPattern) =>
        set({ backgroundPattern }),

      setDeskBackground: (deskBackground) =>
        set({ deskBackground }),

      setHeaderWidgets: (headerWidgets) =>
        set({ headerWidgets }),

      toggleHeaderWidget: (widgetId) =>
        set((state) => {
          const has = state.headerWidgets.includes(widgetId)
          if (has) {
            return { headerWidgets: state.headerWidgets.filter((id) => id !== widgetId) }
          }
          if (state.headerWidgets.length >= 3) return state // max 3 slots
          return { headerWidgets: [...state.headerWidgets, widgetId] }
        }),

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

      setUILook: (look: 'solid' | 'glass' | 'crystal') =>
        set({ uiLook: look }),

      setColorTheme: (colorTheme) =>
        set({ colorTheme }),

      setNavLayout: (navLayout) =>
        set({ navLayout }),

      setAccentIntensity: (accentIntensity) =>
        set({ accentIntensity }),

      setWindowStyle: (windowStyle) =>
        set({ windowStyle }),

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
