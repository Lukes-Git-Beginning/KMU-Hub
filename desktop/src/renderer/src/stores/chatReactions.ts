/**
 * Chat reactions store.
 *
 * The real source of truth is now the API (useReactions / useToggleReaction
 * via TanStack Query, wired to /api/v1/messages). The demo-seed helpers and
 * in-memory toggle map below are kept for backward-compatibility with
 * MessageBubble.tsx, which reads from this store while the component migration
 * (swapping store reads for useReactions) is pending.
 *
 * TODO (cleanup sprint): migrate MessageBubble to useReactions, then remove
 * seedReactions, applyToggle, byMessage and CURRENT_REACTION_USER from here.
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
  /** Optimistic hint for components using the real API. */
  lastToggled: Record<string, string>
  setLastToggled: (messageId: string, emoji: string) => void
}

export const useChatReactionsStore = create<ChatReactionsState>((set) => ({
  byMessage: {},
  toggle: (messageId, emoji) =>
    set((s) => {
      const base = s.byMessage[messageId] ?? seedReactions(messageId)
      return { byMessage: { ...s.byMessage, [messageId]: applyToggle(base, emoji) } }
    }),
  lastToggled: {},
  setLastToggled: (messageId, emoji) =>
    set((s) => ({ lastToggled: { ...s.lastToggled, [messageId]: emoji } })),
}))

export const CURRENT_REACTION_USER = CURRENT_USER
