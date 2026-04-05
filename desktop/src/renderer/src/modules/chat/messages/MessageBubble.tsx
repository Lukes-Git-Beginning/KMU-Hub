/**
 * Individual message bubble component.
 *
 * Displays sender avatar, name, timestamp, content, thread indicator,
 * and hover actions (reply, edit, delete for own messages).
 */
import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { formatDistanceToNow } from 'date-fns'
import { de } from 'date-fns/locale'
import { MessageSquare, Pencil, Trash2, SmilePlus } from 'lucide-react'
import { cn } from '@/lib/cn'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { usePresenceStore } from '@/stores/presence'
import { ReactionBar, generateMockReactions, type Reaction } from './ReactionBar'
import { ReactionPicker } from './ReactionPicker'
import { FileAttachmentCard } from './FileAttachmentCard'
import type { AttachedFile } from './FileDropZone'
import type { components } from '@/api/types'

type MessageInfo = components['schemas']['MessageInfo']

const PRESENCE_COLORS: Record<string, string> = {
  online: 'bg-emerald-500',
  away: 'bg-amber-400',
  dnd: 'bg-red-500',
  offline: 'bg-gray-400',
}

interface MessageBubbleProps {
  message: MessageInfo
  isOwn: boolean
  onOpenThread?: (messageId: string) => void
  onEdit?: (messageId: string, content: string) => void
  onDelete?: (messageId: string) => void
  attachments?: AttachedFile[]
}

export function MessageBubble({ message, isOwn, onOpenThread, onEdit, onDelete, attachments }: MessageBubbleProps) {
  const { t } = useTranslation()
  const [showActions, setShowActions] = useState(false)
  const [reactions, setReactions] = useState<Reaction[]>(() => generateMockReactions(message.id ?? ''))
  const [showPicker, setShowPicker] = useState(false)
  const presenceMap = usePresenceStore((s) => s.presenceMap)

  const presence = message.created_by ? presenceMap[message.created_by] ?? 'offline' : 'offline'

  const handleToggleReaction = useCallback((emoji: string) => {
    setReactions((prev) => {
      const existing = prev.find((r) => r.emoji === emoji)
      if (existing) {
        if (existing.count <= 1) return prev.filter((r) => r.emoji !== emoji)
        return prev.map((r) => r.emoji === emoji ? { ...r, count: r.count - 1 } : r)
      }
      return [...prev, { emoji, users: ['me'], count: 1 }]
    })
  }, [])

  const handlePickEmoji = useCallback((emoji: string) => {
    setReactions((prev) => {
      const existing = prev.find((r) => r.emoji === emoji)
      if (existing) return prev.map((r) => r.emoji === emoji ? { ...r, count: r.count + 1, users: [...r.users, 'me'] } : r)
      return [...prev, { emoji, users: ['me'], count: 1 }]
    })
    setShowPicker(false)
  }, [])

  const senderName = [message.sender_first_name, message.sender_last_name]
    .filter(Boolean)
    .join(' ') || t('chat.unknown')

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
            {t('chat.messages.deleted')}
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
      <div className="relative mt-0.5 shrink-0">
        <Avatar className="h-8 w-8">
          <AvatarFallback className="text-xs">{initials}</AvatarFallback>
        </Avatar>
        <span
          className={`absolute -bottom-0.5 -right-0.5 h-2.5 w-2.5 rounded-full border-2 border-card ${PRESENCE_COLORS[presence] ?? 'bg-gray-400'}`}
        />
      </div>

      <div className="min-w-0 flex-1">
        <div className="flex items-baseline gap-2">
          <span className="text-sm font-medium text-foreground">{senderName}</span>
          <span className="text-xs text-muted-foreground">{timeAgo}</span>
          {isEdited && (
            <span className="text-xs text-muted-foreground">({t('chat.messages.edited')})</span>
          )}
        </div>

        <div className="text-sm text-foreground whitespace-pre-wrap break-words">
          {renderedContent}
        </div>

        {/* File attachments */}
        {attachments && attachments.length > 0 && (
          <div className="mt-1.5 flex flex-wrap gap-2">
            {attachments.map((file) => (
              <FileAttachmentCard key={file.id} file={file} compact />
            ))}
          </div>
        )}

        {/* Reactions */}
        <div className="relative">
          <ReactionBar
            reactions={reactions}
            currentUserId="me"
            onToggleReaction={handleToggleReaction}
            onOpenPicker={() => setShowPicker(true)}
          />
          {showPicker && (
            <ReactionPicker onSelect={handlePickEmoji} onClose={() => setShowPicker(false)} />
          )}
        </div>

        {/* Thread indicator */}
        {replyCount > 0 && (
          <button
            className="mt-1 flex items-center gap-1 text-xs text-primary hover:underline"
            onClick={() => message.id && onOpenThread?.(message.id)}
          >
            <MessageSquare className="h-3 w-3" />
            {t('chat.messages.replyCount', { count: replyCount })}
          </button>
        )}
      </div>

      {/* Hover actions */}
      {showActions && (
        <div className="absolute right-2 top-0 -translate-y-1/2 flex items-center gap-0.5 rounded-md border border-border bg-card px-1 py-0.5 shadow-sm">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={() => setShowPicker(!showPicker)}
              >
                <SmilePlus className="h-3.5 w-3.5" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{t('chat.messages.react')}</TooltipContent>
          </Tooltip>

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
            <TooltipContent>{t('chat.messages.reply')}</TooltipContent>
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
                <TooltipContent>{t('common.edit')}</TooltipContent>
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
                <TooltipContent>{t('common.delete')}</TooltipContent>
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
