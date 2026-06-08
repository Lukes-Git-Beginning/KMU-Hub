/**
 * Emoji reaction bar displayed below message content.
 *
 * Shows reaction pills (emoji + count). Clicking toggles the current
 * user's reaction. A small "+" button opens the ReactionPicker.
 */
import { cn } from '@/lib/cn'
import { Plus } from 'lucide-react'

export interface Reaction {
  emoji: string
  users: string[]
  count: number
}

interface ReactionBarProps {
  reactions: Reaction[]
  currentUserId?: string
  onToggleReaction: (emoji: string) => void
  onOpenPicker: () => void
}

export function ReactionBar({ reactions, currentUserId, onToggleReaction, onOpenPicker }: ReactionBarProps) {
  if (reactions.length === 0) return null

  return (
    <div className="mt-1 flex flex-wrap items-center gap-1">
      {reactions.map((r) => {
        const hasReacted = currentUserId ? r.users.includes(currentUserId) : false
        return (
          <button
            key={r.emoji}
            onClick={() => onToggleReaction(r.emoji)}
            className={cn(
              'flex items-center gap-1 rounded-full border px-1.5 py-0.5 text-xs transition-colors',
              hasReacted
                ? 'border-primary/50 bg-primary/10 text-primary'
                : 'border-border bg-secondary/50 text-muted-foreground hover:bg-secondary'
            )}
          >
            <span className="text-sm leading-none">{r.emoji}</span>
            <span className="tabular-nums">{r.count}</span>
          </button>
        )
      })}
      <button
        onClick={onOpenPicker}
        className="flex h-6 w-6 items-center justify-center rounded-full border border-dashed border-border text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
      >
        <Plus className="h-3 w-3" />
      </button>
    </div>
  )
}
