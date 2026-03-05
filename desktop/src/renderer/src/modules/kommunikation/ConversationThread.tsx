import { useEffect } from 'react'
import { MessageSquareText, Loader2 } from 'lucide-react'
import { useKommunikationStore } from '@/stores/kommunikation'
import { useInboxMessage, useMarkRead, useReplyToMessage } from '@/api/hooks/useInbox'
import { ConversationThreadHeader } from './ConversationThreadHeader'
import { MessageTimeline } from './MessageTimeline'
import { ReplyComposer } from './ReplyComposer'
import type { InboxChannel } from '@/api/inbox-types'

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
  const selectedId = useKommunikationStore((s) => s.selectedConversationId)

  const { data: message, isLoading } = useInboxMessage(selectedId ?? '')
  const markRead = useMarkRead()
  const replyMutation = useReplyToMessage()

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
            Waehle eine Konversation aus
          </p>
          <p className="text-xs text-muted-foreground/60">
            j/k zum Navigieren · Escape zum Abwaehlen
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
        <p className="text-sm text-muted-foreground">Nachricht nicht gefunden</p>
      </div>
    )
  }

  const handleReply = (content: string) => {
    replyMutation.mutate({ id: message.id, body: content })
  }

  const handleInternalNote = (_content: string) => {
    // TODO: No internal note API exists yet — currently a no-op
  }

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      {/* Header: subject, channel, tags */}
      <ConversationThreadHeader message={message} />

      {/* Message timeline */}
      {/* TODO: InboxMessage does not have a messages[] thread — showing single message preview as content.
         A thread/messages API is needed for full conversation threading. */}
      <MessageTimeline messages={[
        {
          id: message.id,
          conversationId: message.id,
          direction: 'inbound' as const,
          senderName: message.sender_name,
          senderId: message.sender_id ?? message.user_id,
          content: message.preview,
          timestamp: message.received_at,
          isRead: message.is_read,
          attachments: [],
        },
      ]} />

      {/* Reply composer */}
      <ReplyComposer
        channel={mapChannelForComposer(message.channel)}
        onSendReply={handleReply}
        onSendInternalNote={handleInternalNote}
      />
    </div>
  )
}
