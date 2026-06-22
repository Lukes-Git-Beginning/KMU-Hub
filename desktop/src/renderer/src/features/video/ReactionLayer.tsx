/**
 * ReactionLayer — Emoji-reactions for in-call use (Wave 2).
 *
 * Two parts:
 *  1. ReactionLayer: overlay that renders ephemeral floating reaction "floats"
 *     received via LiveKit useDataChannel. GPU-only animation (transform + opacity).
 *  2. ReactionPicker: compact picker button + popover with predefined emoji set.
 *     Emojis are allowed here as feature *content* (not UI-chrome labels).
 *
 * DataChannel topic: 'cosmi-reaction'
 * Payload: JSON { type: 'reaction', emoji: string }
 *
 * Must be rendered inside a LiveKitRoom.
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useDataChannel } from '@livekit/components-react'
import { cn } from '@/lib'

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const REACTION_TOPIC = 'cosmi-reaction'
const REACTION_TTL_MS = 2_400 // how long a float lives before being removed from state
const MAX_FLOATS = 20 // cap concurrent floats to avoid DOM thrash

// Predefined reaction set (emoji are feature content, not UI-chrome)
const REACTION_EMOJIS = ['👍', '❤️', '😄', '🎉', '👏', '😮', '🙌', '🔥']

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface ReactionPayload {
  type: 'reaction'
  emoji: string
}

interface FloatInstance {
  id: string
  emoji: string
  // Random horizontal drift for variety (-30 to +30 px)
  rx: number
  // Random horizontal start offset within the overlay (10–90 %)
  left: number
}

// ---------------------------------------------------------------------------
// useReactionChannel — wraps useDataChannel for send + receive
// ---------------------------------------------------------------------------

export function useReactionChannel() {
  const [floats, setFloats] = useState<FloatInstance[]>([])

  const onMessage = useCallback((msg: { payload: Uint8Array }) => {
    try {
      const decoded = new TextDecoder().decode(msg.payload)
      const parsed = JSON.parse(decoded) as ReactionPayload
      if (parsed.type !== 'reaction' || !parsed.emoji) return

      setFloats((prev) => {
        const next = [
          ...prev,
          {
            id: `${Date.now()}-${Math.random()}`,
            emoji: parsed.emoji,
            rx: Math.round((Math.random() - 0.5) * 60),
            left: Math.round(10 + Math.random() * 80),
          },
        ]
        // Cap to MAX_FLOATS (drop oldest)
        return next.length > MAX_FLOATS ? next.slice(next.length - MAX_FLOATS) : next
      })
    } catch {
      // Ignore malformed payloads
    }
  }, [])

  const { send } = useDataChannel(REACTION_TOPIC, onMessage)

  const sendReaction = useCallback(
    (emoji: string) => {
      const payload: ReactionPayload = { type: 'reaction', emoji }
      const encoded = new TextEncoder().encode(JSON.stringify(payload))
      send(encoded, { reliable: false })

      // Also show own reaction locally (DataChannel doesn't echo back to sender)
      setFloats((prev) => {
        const next = [
          ...prev,
          {
            id: `local-${Date.now()}-${Math.random()}`,
            emoji,
            rx: Math.round((Math.random() - 0.5) * 60),
            left: Math.round(10 + Math.random() * 80),
          },
        ]
        return next.length > MAX_FLOATS ? next.slice(next.length - MAX_FLOATS) : next
      })
    },
    [send],
  )

  // Auto-remove floats after TTL
  useEffect(() => {
    if (floats.length === 0) return
    const oldest = floats[0]
    const timer = setTimeout(() => {
      setFloats((prev) => prev.filter((f) => f.id !== oldest.id))
    }, REACTION_TTL_MS)
    return () => clearTimeout(timer)
  }, [floats])

  return { floats, sendReaction }
}

// ---------------------------------------------------------------------------
// ReactionLayer — renders the floating reactions (position: absolute overlay)
// ---------------------------------------------------------------------------

interface ReactionLayerProps {
  floats: FloatInstance[]
}

export function ReactionLayer({ floats }: ReactionLayerProps) {
  return (
    <div
      className="pointer-events-none absolute inset-0 z-20 overflow-hidden"
      aria-hidden="true"
    >
      {floats.map((f) => (
        <span
          key={f.id}
          className="animate-reaction-float absolute bottom-20 select-none text-2xl"
          style={{
            left: `${f.left}%`,
            '--rx': `${f.rx}px`,
          } as React.CSSProperties}
        >
          {f.emoji}
        </span>
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// ReactionPicker — button + popover for triggering reactions
// ---------------------------------------------------------------------------

interface ReactionPickerProps {
  onSend: (emoji: string) => void
  className?: string
}

export function ReactionPicker({ onSend, className }: ReactionPickerProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  // Close on outside click
  useEffect(() => {
    if (!open) return
    function handleClick(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [open])

  const handleEmoji = useCallback(
    (emoji: string) => {
      onSend(emoji)
      setOpen(false)
    },
    [onSend],
  )

  return (
    <div ref={containerRef} className={cn('relative', className)}>
      {/* Reaction popover */}
      {open && (
        <div className="absolute bottom-full left-1/2 z-40 mb-2 -translate-x-1/2 rounded-2xl border border-zinc-700 bg-zinc-900 p-2 shadow-xl animate-scale-in">
          <div className="flex gap-1">
            {REACTION_EMOJIS.map((emoji) => (
              <button
                key={emoji}
                onClick={() => handleEmoji(emoji)}
                className="flex h-9 w-9 items-center justify-center rounded-xl text-lg transition-colors hover:bg-zinc-700 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-zinc-400"
                aria-label={emoji}
              >
                {emoji}
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Trigger button (custom SVG heart-burst icon — no emoji in UI chrome) */}
      <button
        onClick={() => setOpen((v) => !v)}
        title={t('features.video.reactions.open')}
        aria-label={t('features.video.reactions.open')}
        className={cn(
          'flex h-11 w-11 items-center justify-center rounded-full transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-zinc-400 focus-visible:ring-offset-2 focus-visible:ring-offset-zinc-900',
          open
            ? 'bg-primary text-primary-foreground'
            : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700 hover:text-white',
        )}
      >
        {/* Smiley face SVG — custom, no emoji */}
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
          <circle cx="12" cy="12" r="10" />
          <path d="M8 13s1.5 2 4 2 4-2 4-2" />
          <line x1="9" y1="9" x2="9.01" y2="9" />
          <line x1="15" y1="9" x2="15.01" y2="9" />
        </svg>
      </button>
    </div>
  )
}
