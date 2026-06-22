/**
 * Type definitions for the Kommunikation (Unified Inbox) module.
 *
 * Design-branch types with a richer conversation model than the raw
 * api/inbox-types. These are camelCase, UI-focused, and include threading,
 * CRM context, and canned responses.
 */

// ---------------------------------------------------------------------------
// Enums / union types
// ---------------------------------------------------------------------------

export type CommunicationChannel = 'email' | 'teams' | 'whatsapp' | 'widget' | 'portal'

export type ConversationStatus = 'open' | 'pending' | 'resolved' | 'closed'

export type ConversationPriority = 'low' | 'normal' | 'high' | 'urgent'

export type MessageDirection = 'inbound' | 'outbound' | 'internal'

// ---------------------------------------------------------------------------
// Participant
// ---------------------------------------------------------------------------

export interface ConversationParticipant {
  id: string
  name: string
  email?: string
  avatarUrl?: string
  isInternal: boolean
}

// ---------------------------------------------------------------------------
// Message & attachments
// ---------------------------------------------------------------------------

export interface MessageAttachment {
  id: string
  name: string
  size: string
  type: string
  url?: string
}

export interface ConversationPollOption {
  id: string
  label: string
  votes: number
}

export interface ConversationPoll {
  question: string
  options: ConversationPollOption[]
  /** Option the current user voted for, if any. */
  votedOptionId?: string | null
}

export interface ConversationReminder {
  text: string
  dueAt: string
}

export interface ConversationMessage {
  id: string
  conversationId: string
  direction: MessageDirection
  senderName: string
  senderId: string
  content: string
  htmlContent?: string
  timestamp: string
  isRead: boolean
  attachments: MessageAttachment[]
  /** Slash-command payloads (KO-8): rendered as interactive blocks. */
  poll?: ConversationPoll
  reminder?: ConversationReminder
}

// ---------------------------------------------------------------------------
// Conversation
// ---------------------------------------------------------------------------

export interface Conversation {
  id: string
  channel: CommunicationChannel
  subject: string
  status: ConversationStatus
  priority: ConversationPriority
  assignedTo?: string
  contactId?: string
  contactName: string
  contactEmail?: string
  contactCompany?: string
  lastMessage: string
  lastMessageAt: string
  unreadCount: number
  tags: string[]
  messages: ConversationMessage[]
  participants: ConversationParticipant[]
  crmDealIds?: string[]
  crmTicketIds?: string[]
  createdAt: string
  updatedAt: string
}

// ---------------------------------------------------------------------------
// Canned responses (shared with Helpdesk)
// ---------------------------------------------------------------------------

export interface CannedResponse {
  id: string
  title: string
  content: string
  category: string
  shortcut?: string
  createdBy: string
  usageCount: number
}
