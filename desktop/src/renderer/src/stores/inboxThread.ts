import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { ConversationMessage } from '@/types/communication'
import type { InboxMessage } from '@/api/inbox-types'

/**
 * Mock-first overlay for inbox conversation threading.
 *
 * The backend InboxMessage is a single message, not a thread — there is no
 * ListThreadMessages RPC yet (see backend-gaps.md). To make the Posteingang
 * feel like a real conversation (Front/Missive pattern) we:
 *   1. synthesize a small deterministic seed history from the base message
 *      (`buildThreadSeed`), and
 *   2. persist user-sent replies / internal notes per conversation here, so
 *      sending a reply visibly extends the thread.
 *
 * Adapter-ready: when the backend ships a thread API, replace `buildThreadSeed`
 * + the appended entries with the fetched message list and drop this store.
 */

interface InboxThreadState {
  /** Appended messages (replies, internal notes) per conversation id. */
  appended: Record<string, ConversationMessage[]>
  getAppended: (convId: string) => ConversationMessage[]
  appendMessage: (convId: string, message: ConversationMessage) => void
  /** Cast/move a vote on a poll message (KO-8 slash command). */
  votePoll: (convId: string, messageId: string, optionId: string) => void
}

export const useInboxThread = create<InboxThreadState>()(
  persist(
    (set, get) => ({
      appended: {},
      getAppended: (convId) => get().appended[convId] ?? [],
      appendMessage: (convId, message) =>
        set((state) => ({
          appended: {
            ...state.appended,
            [convId]: [...(state.appended[convId] ?? []), message],
          },
        })),
      votePoll: (convId, messageId, optionId) =>
        set((state) => ({
          appended: {
            ...state.appended,
            [convId]: (state.appended[convId] ?? []).map((m) => {
              if (m.id !== messageId || !m.poll) return m
              const prev = m.poll.votedOptionId
              if (prev === optionId) return m
              const options = m.poll.options.map((o) => {
                let votes = o.votes
                if (o.id === optionId) votes += 1
                if (o.id === prev) votes = Math.max(0, votes - 1)
                return { ...o, votes }
              })
              return { ...m, poll: { ...m.poll, options, votedOptionId: optionId } }
            }),
          },
        })),
    }),
    { name: 'cosmi-inbox-thread' },
  ),
)

// ---------------------------------------------------------------------------
// Deterministic seed synthesis
// ---------------------------------------------------------------------------

const OPENERS = [
  'Guten Tag, ich melde mich bezüglich unserer letzten Unterhaltung.',
  'Hallo, vielen Dank für die schnelle Rückmeldung.',
  'Sehr geehrte Damen und Herren, anbei meine Anfrage.',
  'Hallo zusammen, kurze Frage zu Ihrem Angebot.',
]

function hashId(id: string): number {
  let h = 0
  for (let i = 0; i < id.length; i++) {
    h = (h * 31 + id.charCodeAt(i)) >>> 0
  }
  return h
}

function shiftMinutes(iso: string, minutes: number): string {
  const d = new Date(iso)
  d.setMinutes(d.getMinutes() + minutes)
  return d.toISOString()
}

/**
 * Build a deterministic seed thread for a base inbox message.
 * Returns 1–3 messages: an earlier inbound opener, an optional outbound
 * acknowledgment (only when the message has been read = handled), and the
 * base message itself as the latest inbound.
 */
export function buildThreadSeed(msg: InboxMessage): ConversationMessage[] {
  const h = hashId(msg.id)
  const opener = OPENERS[h % OPENERS.length]
  const senderId = msg.sender_id ?? msg.user_id

  const seed: ConversationMessage[] = [
    {
      id: `${msg.id}-seed-0`,
      conversationId: msg.id,
      direction: 'inbound',
      senderName: msg.sender_name,
      senderId,
      content: opener,
      timestamp: shiftMinutes(msg.received_at, -90),
      isRead: true,
      attachments: [],
    },
  ]

  if (msg.is_read && msg.assigned_to) {
    seed.push({
      id: `${msg.id}-seed-1`,
      conversationId: msg.id,
      direction: 'outbound',
      senderName: msg.assigned_to,
      senderId: msg.assigned_to,
      content: 'Vielen Dank für Ihre Nachricht — ich kümmere mich darum und melde mich in Kürze.',
      timestamp: shiftMinutes(msg.received_at, -45),
      isRead: true,
      attachments: [],
    })
  }

  seed.push({
    id: `${msg.id}-seed-latest`,
    conversationId: msg.id,
    direction: 'inbound',
    senderName: msg.sender_name,
    senderId,
    content: msg.preview,
    timestamp: msg.received_at,
    isRead: msg.is_read,
    attachments: [],
  })

  return seed
}
