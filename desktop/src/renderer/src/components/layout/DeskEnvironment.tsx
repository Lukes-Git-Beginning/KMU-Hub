/**
 * Outermost layout wrapper implementing the desk/room metaphor.
 *
 * 5-Layer workspace:
 *   L1 – Room Scene (painted room + window view)
 *   L2 – Furniture Overlays (desk, shelves)
 *   L3 – Decorations (on mount points)
 *   L4 – Mount Points (data only)
 *   L5 – UI Skin (CSS variable overrides on work area)
 *
 * The functional UI "floats" inside the room's window.
 * Minimal theme skips the room entirely.
 */
import { useMemo, useEffect, useCallback, useSyncExternalStore } from 'react'
import { useUIStore } from '@/stores/ui'
import { DESK_THEMES, DEFAULT_DESK_THEME_ID } from '@/config/desk-themes'
import { DeskFrame } from './DeskFrame'
import { BackgroundPattern } from './BackgroundPattern'
import { AppShell } from './AppShell'

// Detect system dark mode preference reactively
const darkQuery = window.matchMedia('(prefers-color-scheme: dark)')
function subscribeSystemTheme(cb: () => void) {
  darkQuery.addEventListener('change', cb)
  return () => darkQuery.removeEventListener('change', cb)
}
function getSystemIsDark() {
  return darkQuery.matches
}

export function DeskEnvironment() {
  const deskMaximized = useUIStore((s) => s.deskMaximized)
  const deskThemeId = useUIStore((s) => s.deskThemeId)
  const toggleDeskMaximized = useUIStore((s) => s.toggleDeskMaximized)
  const storeTheme = useUIStore((s) => s.theme)
  const uiLook = useUIStore((s) => s.uiLook)
  const colorTheme = useUIStore((s) => s.colorTheme)
  const accentIntensity = useUIStore((s) => s.accentIntensity)
  const systemIsDark = useSyncExternalStore(subscribeSystemTheme, getSystemIsDark)

  const isDark = storeTheme === 'auto' ? systemIsDark : storeTheme === 'dark'
  const activeTheme = DESK_THEMES[deskThemeId] ?? DESK_THEMES[DEFAULT_DESK_THEME_ID]

  // Sync .dark and .ui-glass classes on <html>
  useEffect(() => {
    document.documentElement.classList.toggle('dark', isDark)
  }, [isDark])

  useEffect(() => {
    document.documentElement.classList.toggle('ui-glass', uiLook === 'glass')
    document.documentElement.classList.toggle('ui-crystal', uiLook === 'crystal')
  }, [uiLook])

  // Sync color theme class on <html> (graphit = default, no class needed)
  useEffect(() => {
    document.documentElement.classList.remove(
      'theme-sand', 'theme-ozean', 'theme-lavendel',
      'theme-wald', 'theme-rose', 'theme-mitternacht', 'theme-terrakotta'
    )
    if (colorTheme !== 'graphit') {
      document.documentElement.classList.add(`theme-${colorTheme}`)
    }
  }, [colorTheme])

  // Sync accent intensity class on <html>
  useEffect(() => {
    document.documentElement.classList.toggle('accent-vivid', accentIntensity === 'vivid')
  }, [accentIntensity])

  // Build room-level CSS variables from active theme
  const roomStyle = useMemo(() => {
    const vars: Record<string, string> = {}
    const base = activeTheme.roomVariables
    const dark = isDark ? activeTheme.roomVariablesDark ?? {} : {}
    const merged = { ...base, ...dark }
    for (const [key, value] of Object.entries(merged)) {
      vars[`--${key}`] = value
    }
    return vars
  }, [activeTheme, isDark])

  // Build UI skin CSS variables (Layer 5)
  const uiSkinStyle = useMemo(() => {
    const vars: Record<string, string> = {}
    const base = activeTheme.uiSkin.variables
    const dark = isDark ? activeTheme.uiSkin.variablesDark ?? {} : {}
    const merged = { ...base, ...dark }
    for (const [key, value] of Object.entries(merged)) {
      vars[`--${key}`] = value
    }
    return vars
  }, [activeTheme, isDark])

  // Work area positioning — percentage-based window within room
  const workAreaStyle = useMemo<React.CSSProperties>(() => {
    if (deskMaximized || activeTheme.isMinimal) {
      return {
        position: 'absolute' as const,
        top: '0',
        right: '0',
        bottom: '0',
        left: '0',
        padding: activeTheme.isMinimal ? '0' : '8px',
      }
    }
    const { top, right, bottom, left, innerPadding } = activeTheme.window
    return {
      position: 'absolute' as const,
      top,
      right,
      bottom,
      left,
      padding: innerPadding,
    }
  }, [deskMaximized, activeTheme])

  // Keyboard shortcut: Ctrl+Shift+F to toggle maximize
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (activeTheme.isMinimal) return
      if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'F') {
        e.preventDefault()
        toggleDeskMaximized()
      }
    },
    [toggleDeskMaximized, activeTheme.isMinimal]
  )

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  const showFrame = !activeTheme.isMinimal

  return (
    <div
      className="h-screen w-screen overflow-hidden relative"
      style={{
        ...roomStyle,
        backgroundColor: 'var(--desk-room-bg)',
      }}
    >
      {/* Background pattern layer (visible through glass/crystal) */}
      <BackgroundPattern />

      {/* Room scene (L1 room bg, L2 furniture) */}
      {showFrame && (
        <DeskFrame
          visible={!deskMaximized || uiLook !== 'solid'}
          theme={activeTheme}
          isDark={isDark}
          zoomToWindow={deskMaximized && uiLook !== 'solid'}
        />
      )}

      {/* Work area — the "window" containing all functional UI */}
      <div
        className="z-10"
        style={{
          ...workAreaStyle,
          transitionProperty: 'top, right, bottom, left, padding',
          transitionDuration: 'var(--desk-transition-duration)',
          transitionTimingFunction: 'var(--desk-transition-easing)',
        }}
      >
        <div
          className={`h-full overflow-hidden ${activeTheme.uiSkin.className ?? ''}`}
          style={{
            ...uiSkinStyle,
            borderRadius: activeTheme.isMinimal ? '0' : 'var(--desk-window-radius)',
            boxShadow: activeTheme.isMinimal ? 'none' : 'var(--desk-window-shadow)',
            border: activeTheme.isMinimal ? 'none' : '1px solid var(--desk-window-border)',
            transition: `border-radius var(--desk-transition-duration) var(--desk-transition-easing),
                         box-shadow var(--desk-transition-duration) var(--desk-transition-easing)`,
          }}
        >
          <AppShell />
        </div>
      </div>
    </div>
  )
}
