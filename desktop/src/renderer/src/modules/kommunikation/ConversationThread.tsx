import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { MessageSquareText, Loader2, Eye } from 'lucide-react'
import { toast } from 'sonner'
import { getCollision } from '@/lib/inbox-collision'
import { useKommunikationStore } from '@/stores/kommunikation'
import { useInboxMessage, useMarkRead, useReplyToMessage } from '@/api/hooks/useInbox'
import { useInboxThread, buildThreadSeed } from '@/stores/inboxThread'
import { ConversationThreadHeader } from './ConversationThreadHeader'
import { MessageTimeline } from './MessageTimeline'
import { ReplyComposer } from './ReplyComposer'
import { PollDialog, ReminderDialog } from './SlashCommandDialogs'
import type { SlashCommand } from './SlashCommandPalette'
import type { InboxChannel } from '@/api/inbox-types'
import type { ConversationMessage, ConversationPoll, ConversationReminder } from '@/types/communication'

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
  const [showPoll, setShowPoll] = useState(false)
  const [showReminder, setShowReminder] = useState(false)

  // Mock-first thread: deterministic seed history + persisted replies/notes.
  // Replace with a backend thread API when available (backend-gaps.md).
  const threadMessages = useMemo<ConversationMessage[]>(() => {
    if (!message) return []
    return [...buildThreadSeed(message), ...(appended ?? [])]
  }, [message, appended])

  const collision = selectedId ? getCollision(selectedId) : null

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

  const senderName = message.assigned_to || t('kommunikation.thread.you')
  const senderId = message.assigned_to ?? message.user_id

  // Slash commands: /umfrage + /erinnerung are real; /giphy is a labelled stub.
  const handleSlashCommand = (cmd: SlashCommand) => {
    if (cmd.name === 'umfrage') setShowPoll(true)
    else if (cmd.name === 'erinnerung') setShowReminder(true)
    else toast.info(t('kommunikation.slash.comingSoon'))
  }

  const handleCreatePoll = (poll: ConversationPoll) => {
    appendMessage(message.id, {
      id: `${message.id}-poll-${threadMessages.length}`,
      conversationId: message.id,
      direction: 'outbound',
      senderName,
      senderId,
      content: '',
      timestamp: new Date().toISOString(),
      isRead: true,
      attachments: [],
      poll,
    })
    setShowPoll(false)
  }

  const handleCreateReminder = (reminder: ConversationReminder) => {
    appendMessage(message.id, {
      id: `${message.id}-reminder-${threadMessages.length}`,
      conversationId: message.id,
      direction: 'internal',
      senderName,
      senderId,
      content: '',
      timestamp: new Date().toISOString(),
      isRead: true,
      attachments: [],
      reminder,
    })
    setShowReminder(false)
  }

  return (
    <div className="flex flex-1 flex-col overflow-hidden">
      {/* Header: subject, channel, tags */}
      <ConversationThreadHeader message={message} />

      {/* Collision hint — a colleague is viewing this conversation (mock-first) */}
      {collision && (
        <div className="flex items-center gap-2 border-b border-warning/20 bg-warning/5 px-4 py-1.5">
          <Eye className="h-3.5 w-3.5 shrink-0 text-warning" />
          <span className="text-[11px] text-warning">
            {t('kommunikation.collision.editing', { name: collision.name })}
          </span>
        </div>
      )}

      {/* Message timeline — mock-first thread (seed history + persisted replies) */}
      <MessageTimeline messages={threadMessages} />

      {/* Reply composer */}
      <ReplyComposer
        channel={mapChannelForComposer(message.channel)}
        onSendReply={handleReply}
        onSendInternalNote={handleInternalNote}
        onSlashCommand={handleSlashCommand}
      />

      {/* Slash-command dialogs (poll / reminder) */}
      <PollDialog open={showPoll} onOpenChange={setShowPoll} onCreate={handleCreatePoll} />
      <ReminderDialog open={showReminder} onOpenChange={setShowReminder} onCreate={handleCreateReminder} nowMs={Date.now()} />
    </div>
  )
}
