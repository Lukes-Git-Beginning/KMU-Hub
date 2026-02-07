/**
 * Outermost layout wrapper implementing the desk metaphor.
 *
 * Manages the relationship between the desk frame (decorative border)
 * and the work area (functional UI). Handles theme CSS variable application,
 * maximize/restore transitions, and keyboard shortcuts.
 *
 * When maximized: work area fills viewport, frame fades out.
 * When normal: work area is inset by frame dimensions, frame is visible.
 */
import { useMemo, useEffect, useCallback } from 'react'
import { useUIStore } from '@/stores/ui'
import { DESK_THEMES, DEFAULT_DESK_THEME_ID } from '@/config/desk-themes'
import { DeskFrame } from './DeskFrame'
import { DeskDecorations } from '@/components/desk/DeskDecorations'
import { AppShell } from './AppShell'

export function DeskEnvironment() {
  const deskMaximized = useUIStore((s) => s.deskMaximized)
  const deskThemeId = useUIStore((s) => s.deskThemeId)
  const deskDecorations = useUIStore((s) => s.deskDecorations)
  const deskDecorationsVisible = useUIStore((s) => s.deskDecorationsVisible)
  const toggleDeskMaximized = useUIStore((s) => s.toggleDeskMaximized)
  const theme = useUIStore((s) => s.theme)

  const activeTheme = DESK_THEMES[deskThemeId] ?? DESK_THEMES[DEFAULT_DESK_THEME_ID]

  // Build inline CSS variables from active theme
  const themeStyle = useMemo(() => {
    const vars: Record<string, string> = {}
    const base = activeTheme.cssVariables
    const dark = theme === 'dark' ? activeTheme.cssVariablesDark ?? {} : {}
    const merged = { ...base, ...dark }

    for (const [key, value] of Object.entries(merged)) {
      vars[`--${key}`] = value
    }
    return vars
  }, [activeTheme, theme])

  // Work area inset controlled by frame dimensions
  const workAreaStyle = useMemo(() => {
    if (deskMaximized) {
      return { padding: 0 }
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
      style={themeStyle}
    >
      {/* Desk background (full viewport, behind everything) */}
      <div
        className="absolute inset-0 desk-frame-edge"
        style={{ boxShadow: 'var(--desk-frame-shadow)' }}
      />

      {/* Desk frame edges (decorative, hidden when maximized) */}
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
        <AppShell />
      </div>
    </div>
  )
}
