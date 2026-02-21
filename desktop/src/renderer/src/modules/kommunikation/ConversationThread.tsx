import { useEffect, useMemo } from 'react'
import { MessageSquareText } from 'lucide-react'
import { useKommunikationStore } from '@/stores/kommunikation'
import { ConversationThreadHeader } from './ConversationThreadHeader'
import { MessageTimeline } from './MessageTimeline'
import { ReplyComposer } from './ReplyComposer'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function ConversationThread() {
  const selectedId = useKommunikationStore((s) => s.selectedConversationId)
  const conversations = useKommunikationStore((s) => s.conversations)
  const markAsRead = useKommunikationStore((s) => s.markAsRead)
  const addMessage = useKommunikationStore((s) => s.addMessage)

  const conv = useMemo(
    () => conversations.find((c) => c.id === selectedId) ?? null,
    [conversations, selectedId],
  )

  // Mark conversation as read when selected
  useEffect(() => {
    if (conv && conv.unreadCount > 0) {
      markAsRead(conv.id)
    }
  }, [conv?.id])

  // Empty state
  if (!conv) {
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

  const handleReply = (content: string) => {
    addMessage(conv.id, {
      conversationId: conv.id,
      direction: 'outbound',
      senderName: 'Du',
      senderId: 'self',
      content,
      attachments: [],
    })
  }

  const handleInternalNote = (content: string) => {
    addMessage(conv.id, {
      conversationId: conv.id,
      direction: 'internal',
      senderName: 'Du',
      senderId: 'self',
      content,
      attachments: [],
    })
  }

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      {/* Header: subject, channel, status, tags */}
      <ConversationThreadHeader conversation={conv} />

      {/* Message timeline */}
      <MessageTimeline messages={conv.messages} />

      {/* Reply composer */}
      <ReplyComposer
        channel={conv.channel}
        onSendReply={handleReply}
        onSendInternalNote={handleInternalNote}
      />
    </div>
  )
}
