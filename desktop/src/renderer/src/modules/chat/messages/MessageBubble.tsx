/**
 * Individual message bubble component.
 *
 * Displays sender avatar, name, timestamp, content, thread indicator,
 * and hover actions (reply, edit, delete for own messages).
 *
 * Reactions are driven by a pre-fetched ReactionSummary (batch-loaded in
 * MessageList via useReactionSummary) to avoid per-bubble GET requests.
 */
import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { formatDistanceToNow } from 'date-fns'
import { de } from 'date-fns/locale'
import { MessageSquare, Pencil, Trash2, SmilePlus } from 'lucide-react'
import { cn } from '@/lib/cn'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { UserProfileTrigger } from '@/components/user/UserProfileCard'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { usePresenceStore } from '@/stores/presence'
import { useToggleReaction } from '@/api/hooks/useReactions'
import { ReactionBar } from './ReactionBar'
import type { Reaction } from './ReactionBar'
import { ReactionPicker } from './ReactionPicker'
import { FileAttachmentCard } from './FileAttachmentCard'
import type { AttachedFile } from './FileDropZone'
import type { components } from '@/api/types'
import type { ReactionSummary } from '@/api/video-types'

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
  /** Pre-fetched reaction summaries for this message (batch-loaded by parent). */
  reactionSummaries?: ReactionSummary[]
  /** Current authenticated user ID (passed down from parent to avoid per-bubble auth lookups). */
  currentUserId?: string
  onOpenThread?: (messageId: string) => void
  onEdit?: (messageId: string, content: string) => void
  onDelete?: (messageId: string) => void
  attachments?: AttachedFile[]
}

export function MessageBubble({ message, isOwn, reactionSummaries, currentUserId, onOpenThread, onEdit, onDelete, attachments }: MessageBubbleProps) {
  const { t } = useTranslation()
  const messageId = message.id ?? ''
  const [showActions, setShowActions] = useState(false)
  const [showPicker, setShowPicker] = useState(false)
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState('')
  const presenceMap = usePresenceStore((s) => s.presenceMap)
  const toggleReactionMutation = useToggleReaction()

  // Convert ReactionSummary[] (batch-fetched by parent) to Reaction[] for ReactionBar.
  // Falls back to empty array when summaries are not yet available.
  const reactions: Reaction[] = (reactionSummaries ?? []).map((s) => ({
    emoji: s.emoji,
    count: s.count,
    users: s.user_ids,
  }))

  const startEdit = useCallback(() => {
    setDraft(message.content ?? '')
    setEditing(true)
  }, [message.content])

  const submitEdit = useCallback(() => {
    if (message.id && draft.trim() && draft.trim() !== (message.content ?? '')) {
      onEdit?.(message.id, draft)
    }
    setEditing(false)
  }, [draft, message.id, message.content, onEdit])

  const presence = message.created_by ? presenceMap[message.created_by] ?? 'offline' : 'offline'

  const handleToggleReaction = useCallback((emoji: string) => {
    if (messageId) {
      toggleReactionMutation.mutate({ message_id: messageId, emoji })
    }
  }, [toggleReactionMutation, messageId])

  const handlePickEmoji = useCallback((emoji: string) => {
    if (messageId) {
      toggleReactionMutation.mutate({ message_id: messageId, emoji })
    }
    setShowPicker(false)
  }, [toggleReactionMutation, messageId])

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

  // Map server-side FileInfo to the attachment card shape.
  const fileAttachments: AttachedFile[] = (message.files ?? []).map((f) => ({
    id: f.id ?? '',
    name: f.filename ?? '',
    size: f.file_size ?? 0,
    type: f.mime_type ?? '',
    isImage: (f.mime_type ?? '').startsWith('image/'),
  }))

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
        {message.created_by && !isOwn ? (
          <UserProfileTrigger userId={message.created_by} name={senderName}>
            <button type="button" className="rounded-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50">
              <Avatar className="h-8 w-8">
                <AvatarFallback className="text-xs">{initials}</AvatarFallback>
              </Avatar>
            </button>
          </UserProfileTrigger>
        ) : (
          <Avatar className="h-8 w-8">
            <AvatarFallback className="text-xs">{initials}</AvatarFallback>
          </Avatar>
        )}
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

        {editing ? (
          <div className="mt-1">
            <textarea
              autoFocus
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  submitEdit()
                } else if (e.key === 'Escape') {
                  setEditing(false)
                }
              }}
              rows={Math.min(6, Math.max(1, draft.split('\n').length))}
              className="w-full resize-none rounded-md border border-border bg-input-background px-2 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
            <div className="mt-1 flex items-center gap-2">
              <button onClick={submitEdit} className="rounded px-2 py-0.5 text-xs font-medium text-primary hover:underline">
                {t('common.save')}
              </button>
              <button onClick={() => setEditing(false)} className="rounded px-2 py-0.5 text-xs text-muted-foreground hover:text-foreground">
                {t('common.cancel')}
              </button>
              <span className="text-[10px] text-muted-foreground">{t('chat.messages.editHint')}</span>
            </div>
          </div>
        ) : (
          <div className="text-sm text-foreground whitespace-pre-wrap break-words">
            {renderedContent}
          </div>
        )}

        {/* File attachments (server FileInfo + optional local previews) */}
        {(fileAttachments.length > 0 || (attachments && attachments.length > 0)) && (
          <div className="mt-1.5 flex flex-wrap gap-2">
            {fileAttachments.map((file) => (
              <FileAttachmentCard key={file.id} file={file} compact />
            ))}
            {attachments?.map((file) => (
              <FileAttachmentCard key={file.id} file={file} compact />
            ))}
          </div>
        )}

        {/* Reactions */}
        <div className="relative">
          <ReactionBar
            reactions={reactions}
            currentUserId={currentUserId}
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
                    onClick={startEdit}
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
