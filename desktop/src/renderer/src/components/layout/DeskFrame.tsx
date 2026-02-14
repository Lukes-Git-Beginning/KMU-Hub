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
  children?: React.ReactNode
}

export function DeskFrame({ visible, theme, isDark, children }: DeskFrameProps) {
  // Minimal theme renders nothing
  if (theme.isMinimal) return null

  const roomImg = theme.roomScene
    ? (isDark ? theme.roomScene.dark : theme.roomScene.light)
    : null

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
          ...(roomImg
            ? {
                backgroundImage: `url(${roomImg})`,
                backgroundSize: '100% 100%',
              }
            : {}),
        }}
      />

      {/* Layer 2: Furniture Overlays */}
      {theme.furniture.map((f) => {
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

      {/* Layer 3: Decorations (mount-point positioned children) */}
      {children}
    </div>
  )
}
