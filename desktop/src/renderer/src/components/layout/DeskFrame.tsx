/**
 * Desk room scene — 3-layer compositor.
 *
 * Layer 1 (z-0):  Room scene — full-bleed background image or gradient
 * Layer 2 (z-10): Furniture overlays — absolutely positioned transparent PNGs
 * Layer 3 (z-20): Children (decorations on mount points)
 *
 * Uses pointer-events: none to avoid interfering with the
 * functional UI. Interactive decorations re-enable pointer events.
 */
import { cn } from '@/lib/cn'
import type { DeskTheme } from '@/types/desk-theme'

interface DeskFrameProps {
  visible: boolean
  theme: DeskTheme
  isDark: boolean
  /** When true, zoom into the window area only (for glass + maximized) */
  zoomToWindow?: boolean
  children?: React.ReactNode
}

export function DeskFrame({ visible, theme, isDark, zoomToWindow, children }: DeskFrameProps) {
  // Minimal theme renders nothing
  if (theme.isMinimal) return null

  const roomImg = theme.roomScene
    ? (isDark ? theme.roomScene.dark : theme.roomScene.light)
    : null

  // Compute zoom into the window area of the room scene
  const bgStyle = (() => {
    if (!roomImg) return {}
    if (!zoomToWindow) {
      return { backgroundImage: `url(${roomImg})`, backgroundSize: '100% 100%' }
    }
    // Parse window percentages
    const top = parseFloat(theme.window.top) || 0
    const right = parseFloat(theme.window.right) || 0
    const bottom = parseFloat(theme.window.bottom) || 0
    const left = parseFloat(theme.window.left) || 0
    const winW = 100 - left - right
    const winH = 100 - top - bottom
    if (winW <= 0 || winH <= 0) {
      return { backgroundImage: `url(${roomImg})`, backgroundSize: '100% 100%' }
    }
    const scaleX = (100 / winW) * 100
    const scaleY = (100 / winH) * 100
    const posX = left / (left + right) * 100
    const posY = top / (top + bottom) * 100
    return {
      backgroundImage: `url(${roomImg})`,
      backgroundSize: `${scaleX.toFixed(2)}% ${scaleY.toFixed(2)}%`,
      backgroundPosition: `${posX.toFixed(2)}% ${posY.toFixed(2)}%`,
    }
  })()

  return (
    <div
      className={cn(
        'absolute inset-0 pointer-events-none',
        visible ? 'opacity-100 visible' : 'opacity-0 invisible'
      )}
      style={{
        transitionProperty: 'opacity, visibility',
        transitionDuration: 'var(--desk-transition-duration)',
        transitionTimingFunction: 'var(--desk-transition-easing)',
      }}
    >
      {/* Layer 1: Room Scene */}
      <div
        className="absolute inset-0"
        style={{
          backgroundColor: 'var(--desk-room-bg)',
          ...bgStyle,
          transitionProperty: 'background-size, background-position',
          transitionDuration: 'var(--desk-transition-duration)',
          transitionTimingFunction: 'var(--desk-transition-easing)',
        }}
      />

      {/* Layer 2: Furniture Overlays — hidden when zoomed to window */}
      {!zoomToWindow && theme.furniture.map((f) => {
        const src = isDark ? f.image.dark : f.image.light
        return (
          <img
            key={f.id}
            src={src}
            className="absolute pointer-events-none"
            style={{ ...f.position, zIndex: f.zIndex }}
            draggable={false}
            alt=""
          />
        )
      })}

      {/* Layer 3: Decorations — hidden when zoomed to window */}
      {!zoomToWindow && children}
    </div>
  )
}
