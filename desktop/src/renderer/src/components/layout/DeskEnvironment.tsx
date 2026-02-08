/**
 * Outermost layout wrapper implementing the desk/room metaphor.
 *
 * Room layout:
 * - Background: room wall color
 * - Left/Right: large decoration panels (wall space for shelves, plants, photos)
 * - Bottom: desk surface
 * - Center: work area "window" (the main functional UI)
 *
 * Maximize mode expands the work area to fill the entire viewport.
 */
import { useMemo, useEffect, useCallback, useSyncExternalStore } from 'react'
import { useUIStore } from '@/stores/ui'
import { DESK_THEMES, DEFAULT_DESK_THEME_ID } from '@/config/desk-themes'
import { DeskFrame } from './DeskFrame'
import { DeskDecorations } from '@/components/desk/DeskDecorations'
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
  const deskDecorations = useUIStore((s) => s.deskDecorations)
  const deskDecorationsVisible = useUIStore((s) => s.deskDecorationsVisible)
  const toggleDeskMaximized = useUIStore((s) => s.toggleDeskMaximized)
  const storeTheme = useUIStore((s) => s.theme)
  const systemIsDark = useSyncExternalStore(subscribeSystemTheme, getSystemIsDark)

  const isDark = storeTheme === 'dark' || systemIsDark
  const activeTheme = DESK_THEMES[deskThemeId] ?? DESK_THEMES[DEFAULT_DESK_THEME_ID]

  // Build inline CSS variables from active theme
  const themeStyle = useMemo(() => {
    const vars: Record<string, string> = {}
    const base = activeTheme.cssVariables
    const dark = isDark ? activeTheme.cssVariablesDark ?? {} : {}
    const merged = { ...base, ...dark }

    for (const [key, value] of Object.entries(merged)) {
      vars[`--${key}`] = value
    }
    return vars
  }, [activeTheme, isDark])

  // Work area positioning via padding
  const workAreaStyle = useMemo(() => {
    if (deskMaximized) {
      return {
        paddingTop: '8px',
        paddingRight: '8px',
        paddingBottom: '8px',
        paddingLeft: '8px',
      }
    }
    const { top, right, bottom, left } = activeTheme.frame
    return {
      paddingTop: `${top}px`,
      paddingRight: `${right}px`,
      paddingBottom: `${bottom}px`,
      paddingLeft: `${left}px`,
    }
  }, [deskMaximized, activeTheme.frame])

  // Keyboard shortcut: Ctrl+Shift+F to toggle maximize
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'F') {
        e.preventDefault()
        toggleDeskMaximized()
      }
    },
    [toggleDeskMaximized]
  )

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  return (
    <div
      className="h-screen w-screen overflow-hidden relative"
      style={{
        ...themeStyle,
        backgroundColor: 'var(--desk-wall-bg)',
      }}
    >
      {/* Room scene (wall panels, desk surface, decorations) */}
      <DeskFrame visible={!deskMaximized} theme={activeTheme}>
        <DeskDecorations
          theme={activeTheme}
          placements={deskDecorations}
          visible={deskDecorationsVisible && !deskMaximized}
        />
      </DeskFrame>

      {/* Work area (the "window" — contains all functional UI) */}
      <div
        className="relative z-10 h-full"
        style={{
          ...workAreaStyle,
          transitionProperty: 'padding',
          transitionDuration: 'var(--desk-transition-duration)',
          transitionTimingFunction: 'var(--desk-transition-easing)',
        }}
      >
        <div
          className="h-full overflow-hidden"
          style={{
            borderRadius: 'var(--desk-window-radius)',
            boxShadow: 'var(--desk-window-shadow)',
            border: '1px solid var(--desk-window-border)',
            transition: 'border-radius var(--desk-transition-duration) var(--desk-transition-easing), box-shadow var(--desk-transition-duration) var(--desk-transition-easing)',
          }}
        >
          <AppShell />
        </div>
      </div>
    </div>
  )
}
