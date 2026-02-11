/**
 * Individual message bubble component.
 *
 * Displays sender avatar, name, timestamp, content, thread indicator,
 * reactions bar, and hover actions (reply, react, edit, delete for own messages).
 */
import { useState } from 'react'
import { formatDistanceToNow } from 'date-fns'
import { de } from 'date-fns/locale'
import { MessageSquare, Pencil, Trash2 } from 'lucide-react'
import { cn } from '@/lib/cn'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { ReactionPicker } from '@/components/chat/ReactionPicker'
import { ReactionBar } from '@/components/chat/ReactionBar'
import type { components } from '@/api/types'

type MessageInfo = components['schemas']['MessageInfo']

interface MessageBubbleProps {
  message: MessageInfo
  isOwn: boolean
  onOpenThread?: (messageId: string) => void
  onEdit?: (messageId: string, content: string) => void
  onDelete?: (messageId: string) => void
}

export function MessageBubble({ message, isOwn, onOpenThread, onEdit, onDelete }: MessageBubbleProps) {
  const [showActions, setShowActions] = useState(false)

  const senderName = [message.sender_first_name, message.sender_last_name]
    .filter(Boolean)
    .join(' ') || 'Unbekannt'

  const initials = [message.sender_first_name, message.sender_last_name]
    .filter(Boolean)
    .map((n) => n!.charAt(0))
    .join('')
    .toUpperCase() || '??'

  const timeAgo = message.created_at
    ? formatDistanceToNow(new Date(message.created_at), { addSuffix: true, locale: de })
    : ''

  const isEdited = !!message.edited_at
  const replyCount = message.reply_count ?? 0

  // Render message content with mention highlights
  const renderedContent = renderContent(message.content ?? '')

  if (message.is_deleted) {
    return (
      <div className="flex items-start gap-3 px-4 py-1.5 opacity-50">
        <Avatar className="h-8 w-8 mt-0.5 shrink-0">
          <AvatarFallback className="text-xs">{initials}</AvatarFallback>
        </Avatar>
        <div>
          <div className="flex items-baseline gap-2">
            <span className="text-sm font-medium text-foreground">{senderName}</span>
            <span className="text-xs text-muted-foreground">{timeAgo}</span>
          </div>
          <p className="text-sm italic text-muted-foreground">
            Diese Nachricht wurde geloescht.
          </p>
        </div>
      </div>
    )
  }

  return (
    <div
      className={cn(
        'group relative flex items-start gap-3 px-4 py-1.5 transition-colors hover:bg-accent/50',
        isOwn && 'bg-blue-50/50 dark:bg-blue-950/20'
      )}
      onMouseEnter={() => setShowActions(true)}
      onMouseLeave={() => setShowActions(false)}
    >
      <Avatar className="h-8 w-8 mt-0.5 shrink-0">
        <AvatarFallback className="text-xs">{initials}</AvatarFallback>
      </Avatar>

      <div className="min-w-0 flex-1">
        <div className="flex items-baseline gap-2">
          <span className="text-sm font-medium text-foreground">{senderName}</span>
          <span className="text-xs text-muted-foreground">{timeAgo}</span>
          {isEdited && (
            <span className="text-xs text-muted-foreground">(bearbeitet)</span>
          )}
        </div>

        <div className="text-sm text-foreground whitespace-pre-wrap break-words">
          {renderedContent}
        </div>

        {/* Thread indicator */}
        {replyCount > 0 && (
          <button
            className="mt-1 flex items-center gap-1 text-xs text-primary hover:underline"
            onClick={() => message.id && onOpenThread?.(message.id)}
          >
            <MessageSquare className="h-3 w-3" />
            {replyCount} {replyCount === 1 ? 'Antwort' : 'Antworten'}
          </button>
        )}

        {/* Reactions */}
        {message.id && <ReactionBar messageId={message.id} />}
      </div>

      {/* Hover actions */}
      {showActions && (
        <div className="absolute right-2 top-0 -translate-y-1/2 flex items-center gap-0.5 rounded-md border border-border bg-card px-1 py-0.5 shadow-sm">
          {/* Reaction picker */}
          {message.id && <ReactionPicker messageId={message.id} />}

          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={() => message.id && onOpenThread?.(message.id)}
              >
                <MessageSquare className="h-3.5 w-3.5" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Antworten</TooltipContent>
          </Tooltip>

          {isOwn && (
            <>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-6 w-6"
                    onClick={() => message.id && onEdit?.(message.id, message.content ?? '')}
                  >
                    <Pencil className="h-3.5 w-3.5" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Bearbeiten</TooltipContent>
              </Tooltip>

              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-6 w-6 text-destructive hover:text-destructive"
                    onClick={() => message.id && onDelete?.(message.id)}
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Loeschen</TooltipContent>
              </Tooltip>
            </>
          )}
        </div>
      )}
    </div>
  )
}

/** Render message content with @mention highlights. */
function renderContent(content: string): React.ReactNode {
  // Simple @username mention highlighting
  const parts = content.split(/(@\w+)/g)
  if (parts.length <= 1) return content

  return parts.map((part, i) => {
    if (part.startsWith('@')) {
      return (
        <span key={i} className="rounded-sm bg-blue-100 px-0.5 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300">
          {part}
        </span>
      )
    }
    return part
  })
}
