/**
 * Renders decorative items placed at mount points within the desk frame.
 *
 * Each decoration is positioned absolutely within the room container based
 * on its mount point definition (percentage-based x/y). Items have
 * pointer-events: auto so they remain interactive even though the frame
 * layer is pointer-events: none.
 *
 * Supports: clock, calendar, and image-based decorations with CSS animations.
 */
import type { DeskTheme, DecorationPlacement } from '@/types/desk-theme'
import { DeskClock } from './decorations/DeskClock'
import { DeskCalendar } from './decorations/DeskCalendar'

interface DeskDecorationsProps {
  theme: DeskTheme
  placements: Record<string, DecorationPlacement>
  visible: boolean
}

function DecorationRenderer({ placement, maxSize }: { placement: DecorationPlacement; maxSize: { width: number; height: number } }) {
  switch (placement.type) {
    case 'clock':
      return <DeskClock size={64} />
    case 'calendar':
      return <DeskCalendar size={52} />
    case 'image':
      if (!placement.imageUrl) return null
      return (
        <img
          src={placement.imageUrl}
          alt={placement.variant}
          className={placement.animationClass ?? undefined}
          draggable={false}
          loading="lazy"
          decoding="async"
          style={{
            maxWidth: `${maxSize.width}px`,
            maxHeight: `${maxSize.height}px`,
            objectFit: 'contain',
            userSelect: 'none',
          }}
        />
      )
    default:
      return null
  }
}

export function DeskDecorations({ theme, placements, visible }: DeskDecorationsProps) {
  if (!visible) return null

  return (
    <>
      {theme.mountPoints.map((mp) => {
        const placement = placements[mp.id]
        if (!placement) return null

        return (
          <div
            key={mp.id}
            className="absolute pointer-events-auto"
            style={{
              left: mp.position.x,
              top: mp.position.y,
              transform: 'translate(-50%, -50%)',
              maxWidth: `${mp.maxSize.width}px`,
              maxHeight: `${mp.maxSize.height}px`,
            }}
          >
            <DecorationRenderer placement={placement} maxSize={mp.maxSize} />
          </div>
        )
      })}
    </>
  )
}
