/**
 * Reaction bar displayed below a chat message.
 *
 * Shows emoji pills with count and highlighted state for the current
 * user. Clicking a pill toggles the reaction. A "+" button at the
 * end opens the ReactionPicker for adding new reactions.
 *
 * Returns null when there are no reactions (no empty-state rendering).
 */
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/cn'
import { useReactions, useToggleReaction } from '@/api/hooks/useReactions'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { ReactionPicker } from './ReactionPicker'

interface ReactionBarProps {
  messageId: string
}

export function ReactionBar({ messageId }: ReactionBarProps) {
  const { t } = useTranslation()
  const { data: reactions } = useReactions(messageId)
  const toggleReaction = useToggleReaction()

  if (!reactions || reactions.length === 0) {
    return null
  }

  const handleToggle = (emoji: string) => {
    toggleReaction.mutate({ message_id: messageId, emoji })
  }

  return (
    <div className="mt-1 flex flex-wrap items-center gap-1">
      {reactions.map((reaction) => (
        <Tooltip key={reaction.emoji}>
          <TooltipTrigger asChild>
            <button
              className={cn(
                'inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs transition-all hover:scale-105',
                reaction.current_user_reacted
                  ? 'border-info bg-info-light text-info-foreground'
                  : 'border-border bg-muted text-muted-foreground hover:bg-accent',
              )}
              onClick={() => handleToggle(reaction.emoji)}
            >
              <span className="text-sm">{reaction.emoji}</span>
              <span className="font-medium">{reaction.count}</span>
            </button>
          </TooltipTrigger>
          <TooltipContent side="top" className="text-xs">
            {reaction.user_ids.length <= 5
              ? reaction.user_ids.join(', ')
              : t('chat.reactionBar.andMore', {
                  names: reaction.user_ids.slice(0, 5).join(', '),
                  count: reaction.user_ids.length - 5,
                })}
          </TooltipContent>
        </Tooltip>
      ))}

      {/* Add reaction button */}
      <ReactionPicker messageId={messageId} />
    </div>
  )
}
