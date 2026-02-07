/**
 * Renders decorative items placed in desk frame slots.
 *
 * Each decoration is positioned absolutely within the frame based
 * on its slot definition. Items have pointer-events: auto so they
 * remain interactive even though the frame layer is pointer-events: none.
 *
 * Phase 1: Only the 'clock' decoration type is implemented.
 * Future phases add plants, photos, and custom items.
 */
import type { DeskTheme, DecorationPlacement } from '@/types/desk-theme'
import { DeskClock } from './decorations/DeskClock'

interface DeskDecorationsProps {
  theme: DeskTheme
  placements: Record<string, DecorationPlacement>
  visible: boolean
}

function DecorationRenderer({ placement }: { placement: DecorationPlacement }) {
  switch (placement.type) {
    case 'clock':
      return <DeskClock size={18} />
    default:
      return null
  }
}

export function DeskDecorations({ theme, placements, visible }: DeskDecorationsProps) {
  if (!visible) return null

  return (
    <>
      {theme.decorationSlots.map((slot) => {
        const placement = placements[slot.id]
        if (!placement) return null

        return (
          <div
            key={slot.id}
            className="absolute pointer-events-auto"
            style={{
              left: slot.position.x,
              top: slot.position.y,
              transform: 'translate(-50%, -50%)',
              maxWidth: `${slot.maxSize.width}px`,
              maxHeight: `${slot.maxSize.height}px`,
            }}
          >
            <DecorationRenderer placement={placement} />
          </div>
        )
      })}
    </>
  )
}
