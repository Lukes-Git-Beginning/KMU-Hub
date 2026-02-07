/**
 * Decorative desk frame rendered around the work area.
 *
 * Consists of four absolutely-positioned strips (top, right, bottom, left)
 * that create the visual "desk surface" border. Uses pointer-events: none
 * to avoid interfering with the functional UI. Interactive decorations
 * re-enable pointer events on their own elements.
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
      {/* Top edge */}
      <div
        className="absolute top-0 left-0 right-0 desk-frame-edge"
        style={{ height: `${top}px` }}
      />

      {/* Bottom edge */}
      <div
        className="absolute bottom-0 left-0 right-0 desk-frame-edge"
        style={{ height: `${bottom}px` }}
      />

      {/* Left edge */}
      {left > 0 && (
        <div
          className="absolute left-0 desk-frame-edge"
          style={{
            top: `${top}px`,
            bottom: `${bottom}px`,
            width: `${left}px`,
          }}
        />
      )}

      {/* Right edge */}
      <div
        className="absolute right-0 desk-frame-edge"
        style={{
          top: `${top}px`,
          bottom: `${bottom}px`,
          width: `${right}px`,
        }}
      />

      {/* Decoration layer */}
      {children}
    </div>
  )
}
