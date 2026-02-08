/**
 * Desk room scene rendered around the work area.
 *
 * Layout:
 * - Left panel: wall decoration zone
 * - Right panel: wall decoration zone
 * - Bottom strip: desk surface
 * - Background: room wall color
 *
 * Uses pointer-events: none to avoid interfering with the
 * functional UI. Interactive decorations re-enable pointer events.
 */
import { cn } from '@/lib/cn'
import type { DeskTheme } from '@/types/desk-theme'

interface DeskFrameProps {
  visible: boolean
  theme: DeskTheme
  children?: React.ReactNode
}

export function DeskFrame({ visible, theme, children }: DeskFrameProps) {
  const { top, right, bottom, left } = theme.frame

  return (
    <div
      className={cn(
        'absolute inset-0 pointer-events-none',
        visible
          ? 'opacity-100 visible'
          : 'opacity-0 invisible'
      )}
      style={{
        transitionProperty: 'opacity, visibility',
        transitionDuration: 'var(--desk-transition-duration)',
        transitionTimingFunction: 'var(--desk-transition-easing)',
      }}
    >
      {/* Left wall / decoration panel */}
      {left > 0 && (
        <div
          className="absolute top-0 left-0 h-full"
          style={{
            width: `${left}px`,
            backgroundColor: 'var(--desk-deco-bg)',
          }}
        />
      )}

      {/* Right wall / decoration panel */}
      {right > 0 && (
        <div
          className="absolute top-0 right-0 h-full"
          style={{
            width: `${right}px`,
            backgroundColor: 'var(--desk-deco-bg)',
          }}
        />
      )}

      {/* Desk surface (bottom strip) */}
      {bottom > 0 && (
        <div
          className="absolute bottom-0 left-0 right-0"
          style={{
            height: `${bottom}px`,
            backgroundColor: 'var(--desk-surface-bg)',
            backgroundImage: 'var(--desk-surface-texture)',
            borderTop: '2px solid var(--desk-surface-border)',
          }}
        />
      )}

      {/* Top gap (thin) */}
      {top > 0 && (
        <div
          className="absolute top-0 left-0 right-0"
          style={{
            height: `${top}px`,
            backgroundColor: 'var(--desk-wall-bg)',
          }}
        />
      )}

      {/* Decoration layer */}
      {children}
    </div>
  )
}
