import { useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { MessageSquareText, Loader2 } from 'lucide-react'
import { useKommunikationStore } from '@/stores/kommunikation'
import { useInboxMessage, useMarkRead, useReplyToMessage } from '@/api/hooks/useInbox'
import { useInboxThread, buildThreadSeed } from '@/stores/inboxThread'
import { ConversationThreadHeader } from './ConversationThreadHeader'
import { MessageTimeline } from './MessageTimeline'
import { ReplyComposer } from './ReplyComposer'
import type { InboxChannel } from '@/api/inbox-types'
import type { ConversationMessage } from '@/types/communication'

// ---------------------------------------------------------------------------
// Channel mapping helper
// ---------------------------------------------------------------------------

// Map InboxChannel to a display channel for ReplyComposer
// ReplyComposer expects CommunicationChannel ('email'|'teams'|'whatsapp'|'widget'|'portal')
// We map 'chat' -> 'teams', 'notification' -> 'email' as best-fit defaults
function mapChannelForComposer(ch: InboxChannel): 'email' | 'teams' | 'whatsapp' | 'widget' | 'portal' {
  if (ch === 'email') return 'email'
  if (ch === 'chat') return 'teams'
  return 'email'
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function ConversationThread() {
  const { t } = useTranslation()
  const selectedId = useKommunikationStore((s) => s.selectedConversationId)

  const { data: message, isLoading } = useInboxMessage(selectedId ?? '')
  const markRead = useMarkRead()
  const replyMutation = useReplyToMessage()
  const appended = useInboxThread((s) => (selectedId ? s.appended[selectedId] : undefined))
  const appendMessage = useInboxThread((s) => s.appendMessage)

  // Mock-first thread: deterministic seed history + persisted replies/notes.
  // Replace with a backend thread API when available (backend-gaps.md).
  const threadMessages = useMemo<ConversationMessage[]>(() => {
    if (!message) return []
    return [...buildThreadSeed(message), ...(appended ?? [])]
  }, [message, appended])

  // Mark message as read when selected
  useEffect(() => {
    if (message && !message.is_read) {
      markRead.mutate(message.id)
    }
  }, [message?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  // Empty state — no selection
  if (!selectedId) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <div className="text-center space-y-2">
          <MessageSquareText className="h-10 w-10 mx-auto text-muted-foreground/40" />
          <p className="text-sm text-muted-foreground">
            {t('kommunikation.thread.selectConversation')}
          </p>
          <p className="text-xs text-muted-foreground/60">
            {t('kommunikation.thread.keyboardHint')}
          </p>
        </div>
      </div>
    )
  }

  // Loading state
  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  // Message not found
  if (!message) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <p className="text-sm text-muted-foreground">{t('kommunikation.thread.messageNotFound')}</p>
      </div>
    )
  }

  const handleReply = (content: string) => {
    replyMutation.mutate({ id: message.id, body: content })
    // Mock-first: extend the local thread so the sent reply is visible.
    appendMessage(message.id, {
      id: `${message.id}-reply-${threadMessages.length}`,
      conversationId: message.id,
      direction: 'outbound',
      senderName: message.assigned_to || t('kommunikation.thread.you'),
      senderId: message.assigned_to ?? message.user_id,
      content,
      timestamp: new Date().toISOString(),
      isRead: true,
      attachments: [],
    })
  }

  const handleInternalNote = (content: string) => {
    if (!content.trim()) return
    // Mock-first: internal notes live in the local thread overlay (Phase 5
    // wires presence/collision). Backend internal-note API: backend-gaps.md.
    appendMessage(message.id, {
      id: `${message.id}-note-${threadMessages.length}`,
      conversationId: message.id,
      direction: 'internal',
      senderName: message.assigned_to || t('kommunikation.thread.you'),
      senderId: message.assigned_to ?? message.user_id,
      content,
      timestamp: new Date().toISOString(),
      isRead: true,
      attachments: [],
    })
  }

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      {/* Header: subject, channel, tags */}
      <ConversationThreadHeader message={message} />

      {/* Message timeline — mock-first thread (seed history + persisted replies) */}
      <MessageTimeline messages={threadMessages} />

      {/* Reply composer */}
      <ReplyComposer
        channel={mapChannelForComposer(message.channel)}
        onSendReply={handleReply}
        onSendInternalNote={handleInternalNote}
      />
    </div>
  )
}
