import { X, PanelRightClose, Loader2 } from 'lucide-react'
import { useKommunikationStore } from '@/stores/kommunikation'
import { useInboxMessage } from '@/api/hooks/useInbox'
import { ContactCard } from './ContactCard'
import { OpenDeals } from './OpenDeals'
import { OpenTickets } from './OpenTickets'
import { ActivityTimeline } from './ActivityTimeline'
import type { Conversation } from '@/types/communication'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function ContextPanel() {
  const selectedId = useKommunikationStore((s) => s.selectedConversationId)
  const detailPaneOpen = useKommunikationStore((s) => s.detailPaneOpen)
  const toggleDetailPane = useKommunikationStore((s) => s.toggleDetailPane)

  const { data: message, isLoading } = useInboxMessage(selectedId ?? '')

  // No message selected -> no panel
  if (!selectedId) return null

  // Panel collapsed -> show toggle button
  if (!detailPaneOpen) {
    return (
      <div className="flex shrink-0 flex-col items-center border-l border-border bg-card/30 px-1.5 py-2">
        <button
          onClick={toggleDetailPane}
          className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
          title="Kontext-Panel oeffnen"
        >
          <PanelRightClose className="h-4 w-4" />
        </button>
      </div>
    )
  }

  // Loading state
  if (isLoading) {
    return (
      <div className="flex w-72 shrink-0 flex-col border-l border-border bg-card/30 items-center justify-center">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!message) return null

  // Adapt InboxMessage to Conversation shape for ContactCard (which expects Conversation)
  // This is a temporary adapter until ContactCard is updated to accept InboxMessage
  const conversationAdapter: Conversation = {
    id: message.id,
    channel: message.channel === 'email' ? 'email' : 'teams',
    subject: message.subject,
    status: 'open',
    priority: 'normal',
    assignedTo: message.assigned_to,
    contactId: message.crm_contact_id,
    contactName: message.sender_name,
    contactEmail: message.sender_email,
    lastMessage: message.preview,
    lastMessageAt: message.received_at,
    unreadCount: message.is_read ? 0 : 1,
    tags: message.tags,
    messages: [],
    participants: [],
    crmDealIds: [],
    crmTicketIds: [],
    createdAt: message.created_at,
    updatedAt: message.updated_at,
  }

  return (
    <div className="flex w-72 shrink-0 flex-col border-l border-border bg-card/30 overflow-y-auto">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2.5 border-b border-border">
        <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
          Kontext
        </h3>
        <button
          onClick={toggleDetailPane}
          className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
          title="Panel schliessen"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      </div>

      {/* Contact card */}
      <ContactCard conversation={conversationAdapter} />

      {/* Open deals */}
      {/* TODO: InboxMessage has no crmDealIds — would need a CRM lookup by crm_contact_id */}
      <OpenDeals dealIds={[]} />

      {/* Open tickets */}
      {/* TODO: InboxMessage has no crmTicketIds — would need a CRM lookup by crm_contact_id */}
      <OpenTickets ticketIds={[]} />

      {/* Activity timeline */}
      <ActivityTimeline contactName={message.sender_name} />

      {/* Participants */}
      {/* TODO: InboxMessage has no participants array — thread/participants API needed */}
      <div className="p-3">
        <h4 className="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
          Teilnehmer
        </h4>
        <div className="space-y-1.5">
          <div className="flex items-center gap-2 text-xs">
            <div className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-secondary text-[9px] font-medium text-muted-foreground">
              {message.sender_name
                .split(' ')
                .map((n) => n[0])
                .join('')
                .toUpperCase()
                .slice(0, 2)}
            </div>
            <span className="text-foreground/90 truncate">{message.sender_name}</span>
          </div>
        </div>
      </div>
    </div>
  )
}
