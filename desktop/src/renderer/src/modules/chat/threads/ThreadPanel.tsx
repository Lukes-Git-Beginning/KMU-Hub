/**
 * Thread panel for viewing and replying to message threads.
 *
 * Slides in from the right side of the chat layout (350px wide).
 * Shows the parent message, thread replies, and a reply input.
 */
import { useTranslation } from 'react-i18next'
import { X, Loader2 } from 'lucide-react'
import { useThreadReplies, useThreadWebSocket } from '@/api/hooks/useMessages'
import { useAuthStore } from '@/stores/auth'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { MessageBubble } from '../messages/MessageBubble'
import { MessageInput } from '../messages/MessageInput'

interface ThreadPanelProps {
  messageId: string
  channelId: string
  onClose: () => void
}

export function ThreadPanel({ messageId, channelId, onClose }: ThreadPanelProps) {
  const { t } = useTranslation()
  const currentUserId = useAuthStore((s) => s.user?.id)
  const { data, isLoading } = useThreadReplies(messageId)

  // Subscribe to real-time thread updates
  useThreadWebSocket(messageId)

  const parentMessage = data?.parent
  const replies = data?.replies ?? []

  return (
    <div className="flex h-full flex-col border-l border-border bg-card">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <div>
          <h3 className="text-sm font-semibold text-foreground">{t('chat.thread.title')}</h3>
          {replies.length > 0 && (
            <p className="text-xs text-muted-foreground">
              {t('chat.thread.replyCount', { count: replies.length })}
            </p>
          )}
        </div>
        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      {isLoading ? (
        <div className="flex flex-1 items-center justify-center">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : (
        <>
          {/* Parent message */}
          {parentMessage && (
            <div className="border-b border-border">
              <MessageBubble
                message={parentMessage}
                isOwn={parentMessage.created_by === currentUserId}
              />
            </div>
          )}

          {/* Replies separator */}
          {replies.length > 0 && (
            <div className="flex items-center gap-3 px-4 py-2">
              <div className="flex-1 border-t border-border" />
              <span className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                {t('chat.thread.replyCount', { count: replies.length })}
              </span>
              <div className="flex-1 border-t border-border" />
            </div>
          )}

          {/* Thread replies */}
          <ScrollArea className="flex-1">
            {replies.length === 0 ? (
              <div className="flex items-center justify-center py-8">
                <p className="text-sm text-muted-foreground">
                  {t('chat.thread.empty')}
                </p>
              </div>
            ) : (
              <div className="py-2">
                {replies.map((reply) => (
                  <MessageBubble
                    key={reply.id}
                    message={reply}
                    isOwn={reply.created_by === currentUserId}
                  />
                ))}
              </div>
            )}
          </ScrollArea>

          {/* Reply input */}
          <MessageInput
            channelId={channelId}
            parentMessageId={messageId}
            placeholder={t('chat.thread.replyPlaceholder')}
          />
        </>
      )}
    </div>
  )
}
