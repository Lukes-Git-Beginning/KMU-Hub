/**
 * In-session demo backing for chat message reactions.
 *
 * The chat service does not expose reaction endpoints yet (no gateway
 * route, no chat-service impl — see .planning/backend-gaps.md), and
 * MessageInfo carries no reactions field. This Zustand store stands in
 * as the "demo backend": it seeds deterministic reactions per message on
 * first access and persists toggles for the session, so reactions survive
 * list virtualization (which remounts message bubbles).
 *
 * Wiring-ready: when the backend exposes ToggleReaction/ListReactions,
 * swap the store reads/writes in MessageBubble for the real hooks.
 */
import { create } from 'zustand'
import type { Reaction } from '@/modules/chat/messages/ReactionBar'

const CURRENT_USER = 'me'

/** Deterministic seed so a message shows the same starter reactions each load. */
export function seedReactions(messageId: string): Reaction[] {
  let hash = 0
  for (let i = 0; i < messageId.length; i++) hash = (hash + messageId.charCodeAt(i)) % 997
  // Roughly half the messages get reactions, for a lively-but-not-noisy demo.
  if (hash % 2 !== 0) return []
  const pool: Reaction[] = [
    { emoji: '\u{1F44D}', users: ['u1', 'u2'], count: 2 },
    { emoji: '\u{2764}\u{FE0F}', users: ['u1'], count: 1 },
    { emoji: '\u{1F604}', users: ['u2', 'u3'], count: 2 },
    { emoji: '\u{1F389}', users: ['u1', 'u2', 'u3'], count: 3 },
    { emoji: '\u{1F440}', users: ['u3'], count: 1 },
  ]
  return pool.slice(0, (hash % 3) + 1).map((r) => ({ ...r, users: [...r.users] }))
}

function applyToggle(list: Reaction[], emoji: string): Reaction[] {
  const existing = list.find((r) => r.emoji === emoji)
  if (existing) {
    const hasMine = existing.users.includes(CURRENT_USER)
    if (hasMine) {
      const users = existing.users.filter((u) => u !== CURRENT_USER)
      if (users.length === 0) return list.filter((r) => r.emoji !== emoji)
      return list.map((r) => (r.emoji === emoji ? { ...r, users, count: users.length } : r))
    }
    const users = [...existing.users, CURRENT_USER]
    return list.map((r) => (r.emoji === emoji ? { ...r, users, count: users.length } : r))
  }
  return [...list, { emoji, users: [CURRENT_USER], count: 1 }]
}

interface ChatReactionsState {
  byMessage: Record<string, Reaction[]>
  toggle: (messageId: string, emoji: string) => void
}

export const useChatReactionsStore = create<ChatReactionsState>((set) => ({
  byMessage: {},
  toggle: (messageId, emoji) =>
    set((s) => {
      const base = s.byMessage[messageId] ?? seedReactions(messageId)
      return { byMessage: { ...s.byMessage, [messageId]: applyToggle(base, emoji) } }
    }),
}))

export const CURRENT_REACTION_USER = CURRENT_USER
