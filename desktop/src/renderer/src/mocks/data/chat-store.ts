/**
 * Stateful in-memory chat store for demo mode.
 *
 * Mirrors the pattern of `email-store.ts`: the MSW chat handlers mutate and read
 * through this module so that sending, editing, deleting, reacting, marking-read,
 * threading and channel management *persist* across requests (until reload).
 *
 * Seeds are deep-cloned from `chat-data.ts` so the immutable seeds stay intact
 * and `resetChatStore()` can restore a clean slate.
 */
import {
  mockChannels,
  mockDMs,
  mockUnread,
  mockMessagesByChannel,
  mockChannelMembers,
  mockMentions,
} from './chat-data'
import { EMPLOYEES } from '../mock-db'
import { CURRENT_USER } from './shared-ids'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface ChatChannel {
  id: string
  name: string
  is_dm: boolean
  is_private: boolean
  description?: string
  member_count: number
  my_role?: string
  created_at: string
  archived?: boolean
  // Group DM extensions (KO-2)
  is_group_dm?: boolean
  other_user_id?: string
  other_user_name?: string
  participant_ids?: string[]
  participant_names?: string[]
}

export interface ChatFile {
  id: string
  message_id?: string
  channel_id?: string
  filename: string
  mime_type: string
  file_size: number
  uploaded_by: string
  uploader_first_name?: string
  uploader_last_name?: string
  has_thumbnail?: boolean
  created_at: string
  /** demo-only: data URL or inline content for real downloads (KO-5) */
  data_url?: string
}

export interface ChatPoll {
  question: string
  options: Array<{ id: string; label: string; votes: string[] }>
  closed?: boolean
}

export interface ChatMessage {
  id: string
  content: string
  channel_id: string
  /** Real API field used by the UI (isOwn, presence, profile trigger). */
  created_by?: string
  sender_id?: string
  sender_name?: string
  sender_first_name?: string
  sender_last_name?: string
  created_at: string
  reply_count: number
  reactions?: Array<{ emoji: string; count: number }>
  edited_at: string | null
  parent_message_id?: string | null
  files?: ChatFile[]
  bookmarked?: boolean
  /** demo-only structured payloads for slash commands (KO-8) */
  poll?: ChatPoll
  reminder?: { text: string; due_at: string }
  system?: boolean
}

export interface ChatReaction {
  message_id: string
  user_id: string
  emoji: string
  created_at: string
  first_name?: string
  last_name?: string
}

interface ChatState {
  channels: ChatChannel[]
  dms: ChatChannel[]
  messagesByChannel: Record<string, ChatMessage[]>
  reactionsByMessage: Record<string, ChatReaction[]>
  members: Record<string, Array<{ user_id: string; name: string; role: string; joined_at: string }>>
  defaultMembers: Array<{ user_id: string; name: string; role: string; joined_at: string }>
  mentions: Array<Record<string, unknown>>
  unread: Record<string, number>
  bookmarks: string[]
}

/** The demo "current user" — single source of truth ([[reference_current_user_source]]). */
export const CHAT_CURRENT_USER = {
  id: CURRENT_USER.id,
  reactionId: CURRENT_USER.id,
  firstName: CURRENT_USER.firstName,
  lastName: CURRENT_USER.lastName,
  name: CURRENT_USER.name,
}

/**
 * Seed messages use the legacy `sender_id`/`sender_name` shape, but the UI reads
 * the real API fields `created_by` + `sender_first_name`/`sender_last_name`
 * (isOwn check, avatar, presence). Without this, every message shows "Unbekannt"
 * and own messages get no edit/delete actions. Normalize once at seed time.
 */
function normalizeMessage(m: ChatMessage): ChatMessage {
  const createdBy = m.created_by ?? m.sender_id
  let first = m.sender_first_name
  let last = m.sender_last_name
  if (!first && !last && m.sender_name) {
    const parts = m.sender_name.trim().split(/\s+/)
    first = parts[0]
    last = parts.slice(1).join(' ')
  }
  return { ...m, created_by: createdBy, sender_first_name: first, sender_last_name: last }
}

// ---------------------------------------------------------------------------
// Seeding
// ---------------------------------------------------------------------------

function clone<T>(v: T): T {
  return JSON.parse(JSON.stringify(v)) as T
}

let counter = 0
function nextId(prefix: string): string {
  counter += 1
  return `${prefix}-${counter}`
}

/** Expand a seed message's aggregated `reactions:[{emoji,count}]` into individual reaction rows. */
function seedReactions(messages: ChatMessage[]): Record<string, ChatReaction[]> {
  const map: Record<string, ChatReaction[]> = {}
  for (const m of messages) {
    const agg = m.reactions ?? []
    if (!agg.length) continue
    const rows: ChatReaction[] = []
    for (const r of agg) {
      for (let i = 0; i < (r.count ?? 0); i++) {
        rows.push({
          message_id: m.id,
          user_id: `seed-${m.id}-${r.emoji}-${i}`,
          emoji: r.emoji,
          created_at: m.created_at,
        })
      }
    }
    if (rows.length) map[m.id] = rows
  }
  return map
}

const SEED_REPLY_TEXTS = [
  'Klingt gut, ich kümmere mich darum.',
  'Danke für die schnelle Rückmeldung!',
  'Können wir das morgen kurz im Daily besprechen?',
  'Sehe ich genauso — passt für mich.',
  'Ist erledigt, habe es gerade aktualisiert.',
  'Guter Punkt, daran hatte ich nicht gedacht.',
  'Ich hänge die Unterlagen gleich hier an.',
  'Top, dann machen wir das so.',
]

function hashStr(s: string): number {
  let h = 0
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) >>> 0
  return h
}

function splitName(name: string): [string, string] {
  const parts = name.trim().split(/\s+/)
  return [parts[0] ?? '', parts.slice(1).join(' ')]
}

/**
 * Generate deterministic thread replies so reply_count indicators have real
 * content instead of an empty thread panel. The parent's reply_count is aligned
 * to the number actually generated (capped) for consistency.
 */
function generateThreadReplies(
  messages: ChatMessage[],
  members: ChatState['defaultMembers'],
): ChatMessage[] {
  const replies: ChatMessage[] = []
  for (const parent of messages) {
    const target = parent.reply_count ?? 0
    const pool = members.filter((m) => m.user_id !== parent.created_by)
    if (target <= 0 || pool.length === 0) continue
    const count = Math.min(target, 4)
    const base = new Date(parent.created_at).getTime()
    const h = hashStr(parent.id)
    for (let i = 0; i < count; i++) {
      const member = pool[(h + i * 7) % pool.length]
      const [first, last] = splitName(member.name)
      replies.push({
        id: `${parent.id}-r${i + 1}`,
        content: SEED_REPLY_TEXTS[(h + i) % SEED_REPLY_TEXTS.length],
        channel_id: parent.channel_id,
        created_by: member.user_id,
        sender_id: member.user_id,
        sender_name: member.name,
        sender_first_name: first,
        sender_last_name: last,
        created_at: new Date(base + (i + 1) * 7 * 60000).toISOString(),
        reply_count: 0,
        reactions: [],
        edited_at: null,
        parent_message_id: parent.id,
      })
    }
    parent.reply_count = count
  }
  return replies
}

function seed(): ChatState {
  const channels = clone(mockChannels.channels) as ChatChannel[]
  const dms = clone(mockDMs.channels) as ChatChannel[]
  const defaultMembers = clone(mockChannelMembers.members) as ChatState['defaultMembers']
  const messagesByChannel: Record<string, ChatMessage[]> = {}
  let allMessages: ChatMessage[] = []

  for (const [chId, bucket] of Object.entries(mockMessagesByChannel)) {
    const rawMsgs = (bucket as { messages?: unknown[] }).messages ?? []
    const msgs = (clone(rawMsgs) as unknown as ChatMessage[]).map(normalizeMessage)
    const threadReplies = generateThreadReplies(msgs, defaultMembers)
    messagesByChannel[chId] = [...msgs, ...threadReplies]
    allMessages = allMessages.concat(msgs, threadReplies)
  }

  const reactionsByMessage = seedReactions(allMessages)

  return {
    channels,
    dms,
    messagesByChannel,
    reactionsByMessage,
    members: {},
    defaultMembers,
    mentions: clone(mockMentions.mentions) as Array<Record<string, unknown>>,
    unread: clone(mockUnread.unread_counts) as Record<string, number>,
    bookmarks: [],
  }
}

let state: ChatState = seed()

export function resetChatStore(): void {
  counter = 0
  state = seed()
}

// ---------------------------------------------------------------------------
// Channels & DMs
// ---------------------------------------------------------------------------

export function getChannels(includeArchived = false): ChatChannel[] {
  return state.channels.filter((c) => includeArchived || !c.archived)
}

export function getDMs(): ChatChannel[] {
  return state.dms
}

export function getChannel(id: string): ChatChannel | undefined {
  return [...state.channels, ...state.dms].find((c) => c.id === id)
}

export function getMembers(channelId: string): ChatState['defaultMembers'] {
  return state.members[channelId] ?? state.defaultMembers
}

export function createChannel(input: {
  name: string
  description?: string
  is_private?: boolean
}): ChatChannel {
  const channel: ChatChannel = {
    id: nextId('ch-new'),
    name: input.name || 'neuer-kanal',
    is_dm: false,
    is_private: input.is_private ?? false,
    description: input.description ?? '',
    member_count: 1,
    my_role: 'owner',
    created_at: new Date().toISOString(),
  }
  state.channels.push(channel)
  state.messagesByChannel[channel.id] = []
  return channel
}

export function renameChannel(
  id: string,
  patch: { name?: string; description?: string; is_private?: boolean },
): ChatChannel | undefined {
  const channel = state.channels.find((c) => c.id === id)
  if (!channel) return undefined
  if (patch.name !== undefined) channel.name = patch.name
  if (patch.description !== undefined) channel.description = patch.description
  if (patch.is_private !== undefined) channel.is_private = patch.is_private
  return channel
}

export function joinChannel(id: string): ChatChannel | undefined {
  const channel = state.channels.find((c) => c.id === id)
  if (channel) {
    channel.member_count += 1
    channel.my_role = channel.my_role ?? 'member'
  }
  return channel
}

export function leaveChannel(id: string): void {
  state.channels = state.channels.filter((c) => c.id !== id)
  delete state.messagesByChannel[id]
  delete state.unread[id]
}

/** Get-or-create a 1:1 DM with another user. */
export function getOrCreateDM(otherUserId: string): { channel: ChatChannel; created: boolean } {
  const existing = state.dms.find((c) => !c.is_group_dm && c.other_user_id === otherUserId)
  if (existing) return { channel: existing, created: false }

  const employee = EMPLOYEES.find((e) => `usr-${e.id}` === otherUserId)
  const name = employee ? `${employee.firstName} ${employee.lastName}` : 'Neuer Kontakt'
  const channel: ChatChannel = {
    id: nextId('dm-new'),
    name,
    is_dm: true,
    is_private: true,
    other_user_id: otherUserId,
    other_user_name: name,
    member_count: 2,
    created_at: new Date().toISOString(),
  }
  state.dms.push(channel)
  state.messagesByChannel[channel.id] = []
  return { channel, created: true }
}

/** Create a group DM with multiple participants (KO-2). */
export function createGroupDM(participantIds: string[]): { channel: ChatChannel; created: boolean } {
  const ids = Array.from(new Set(participantIds)).filter(Boolean)
  const names = ids.map((id) => {
    const e = EMPLOYEES.find((emp) => `usr-${emp.id}` === id)
    return e ? `${e.firstName} ${e.lastName}` : 'Mitglied'
  })

  // Reuse an existing group DM with the exact same participant set
  const key = [...ids].sort().join(',')
  const existing = state.dms.find(
    (c) => c.is_group_dm && [...(c.participant_ids ?? [])].sort().join(',') === key,
  )
  if (existing) return { channel: existing, created: false }

  const label = names.length <= 3 ? names.join(', ') : `${names.slice(0, 2).join(', ')} +${names.length - 2}`
  const channel: ChatChannel = {
    id: nextId('gdm-new'),
    name: label,
    is_dm: true,
    is_group_dm: true,
    is_private: true,
    participant_ids: ids,
    participant_names: names,
    member_count: ids.length + 1,
    created_at: new Date().toISOString(),
  }
  state.dms.push(channel)
  state.messagesByChannel[channel.id] = []
  return { channel, created: true }
}

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

export interface ListMessagesResult {
  messages: ChatMessage[]
  has_more: boolean
}

/** Top-level messages for a channel (thread replies excluded), oldest→newest. */
export function listMessages(channelId: string, before?: string, limit = 50): ListMessagesResult {
  const all = (state.messagesByChannel[channelId] ?? []).filter((m) => !m.parent_message_id)
  // Demo: single page (channels are small). A cursor means "next page" → empty.
  if (before) return { messages: [], has_more: false }
  const slice = all.slice(-limit)
  return { messages: slice, has_more: all.length > limit }
}

export function getMessage(id: string): ChatMessage | undefined {
  for (const msgs of Object.values(state.messagesByChannel)) {
    const found = msgs.find((m) => m.id === id)
    if (found) return found
  }
  return undefined
}

export function appendMessage(input: {
  channelId: string
  content: string
  parentMessageId?: string | null
  files?: ChatFile[]
  extra?: Partial<ChatMessage>
}): ChatMessage {
  const msg: ChatMessage = {
    id: nextId('msg-new'),
    content: input.content ?? '',
    channel_id: input.channelId,
    created_by: CHAT_CURRENT_USER.id,
    sender_id: CHAT_CURRENT_USER.id,
    sender_name: CHAT_CURRENT_USER.name,
    sender_first_name: CHAT_CURRENT_USER.firstName,
    sender_last_name: CHAT_CURRENT_USER.lastName,
    created_at: new Date().toISOString(),
    reply_count: 0,
    reactions: [],
    edited_at: null,
    parent_message_id: input.parentMessageId ?? null,
    files: input.files,
    ...input.extra,
  }

  if (!state.messagesByChannel[input.channelId]) state.messagesByChannel[input.channelId] = []
  state.messagesByChannel[input.channelId].push(msg)

  // A reply bumps the parent's reply_count.
  if (msg.parent_message_id) {
    const parent = getMessage(msg.parent_message_id)
    if (parent) parent.reply_count = (parent.reply_count ?? 0) + 1
  }
  return msg
}

export function editMessage(id: string, content: string): ChatMessage | undefined {
  const msg = getMessage(id)
  if (!msg) return undefined
  msg.content = content
  msg.edited_at = new Date().toISOString()
  return msg
}

export function deleteMessage(id: string): void {
  for (const [chId, msgs] of Object.entries(state.messagesByChannel)) {
    const idx = msgs.findIndex((m) => m.id === id)
    if (idx >= 0) {
      const [removed] = msgs.splice(idx, 1)
      if (removed?.parent_message_id) {
        const parent = getMessage(removed.parent_message_id)
        if (parent) parent.reply_count = Math.max(0, (parent.reply_count ?? 1) - 1)
      }
      // Cascade-delete this message's own replies.
      state.messagesByChannel[chId] = msgs.filter((m) => m.parent_message_id !== id)
      delete state.reactionsByMessage[id]
      state.bookmarks = state.bookmarks.filter((b) => b !== id)
      return
    }
  }
}

// ---------------------------------------------------------------------------
// Threads (KO-3)
// ---------------------------------------------------------------------------

export function getThread(parentId: string): { parent?: ChatMessage; replies: ChatMessage[]; has_more: boolean } {
  const parent = getMessage(parentId)
  const channelId = parent?.channel_id
  const pool = channelId ? state.messagesByChannel[channelId] ?? [] : []
  const replies = pool.filter((m) => m.parent_message_id === parentId)
  return { parent, replies, has_more: false }
}

// ---------------------------------------------------------------------------
// Reactions
// ---------------------------------------------------------------------------

export function getReactions(messageId: string): ChatReaction[] {
  return state.reactionsByMessage[messageId] ?? []
}

export function toggleReaction(
  messageId: string,
  emoji: string,
  userId = CHAT_CURRENT_USER.reactionId,
): { reactions: ChatReaction[]; added: boolean } {
  const rows = state.reactionsByMessage[messageId] ?? []
  const existingIdx = rows.findIndex((r) => r.user_id === userId && r.emoji === emoji)
  let added: boolean
  if (existingIdx >= 0) {
    rows.splice(existingIdx, 1)
    added = false
  } else {
    rows.push({
      message_id: messageId,
      user_id: userId,
      emoji,
      created_at: new Date().toISOString(),
      first_name: 'Du',
      last_name: '',
    })
    added = true
  }
  state.reactionsByMessage[messageId] = rows
  syncMessageReactionAgg(messageId)
  return { reactions: rows.filter((r) => r.emoji === emoji), added }
}

/** Keep the inline aggregated `reactions` on the message in sync (used by some views). */
function syncMessageReactionAgg(messageId: string): void {
  const msg = getMessage(messageId)
  if (!msg) return
  const counts = new Map<string, number>()
  for (const r of state.reactionsByMessage[messageId] ?? []) {
    counts.set(r.emoji, (counts.get(r.emoji) ?? 0) + 1)
  }
  msg.reactions = Array.from(counts.entries()).map(([emoji, count]) => ({ emoji, count }))
}

export interface ReactionSummary {
  message_id: string
  emoji: string
  count: number
  user_ids: string[]
  current_user_reacted: boolean
}

export function getReactionSummary(messageIds: string[]): ReactionSummary[] {
  const summaries: ReactionSummary[] = []
  for (const mid of messageIds) {
    const rows = state.reactionsByMessage[mid] ?? []
    const byEmoji = new Map<string, ChatReaction[]>()
    for (const r of rows) {
      const list = byEmoji.get(r.emoji) ?? []
      list.push(r)
      byEmoji.set(r.emoji, list)
    }
    for (const [emoji, list] of byEmoji.entries()) {
      summaries.push({
        message_id: mid,
        emoji,
        count: list.length,
        user_ids: list.map((r) => r.user_id),
        current_user_reacted: list.some((r) => r.user_id === CHAT_CURRENT_USER.reactionId),
      })
    }
  }
  return summaries
}

// ---------------------------------------------------------------------------
// Unread
// ---------------------------------------------------------------------------

export function getUnreadCounts(): Record<string, number> {
  return state.unread
}

export function markChannelRead(channelId: string): void {
  delete state.unread[channelId]
}

// ---------------------------------------------------------------------------
// Bookmarks (KO-4)
// ---------------------------------------------------------------------------

export function getBookmarks(): ChatMessage[] {
  return state.bookmarks
    .map((id) => getMessage(id))
    .filter((m): m is ChatMessage => !!m)
}

export function toggleBookmark(messageId: string): boolean {
  const msg = getMessage(messageId)
  if (!msg) return false
  if (state.bookmarks.includes(messageId)) {
    state.bookmarks = state.bookmarks.filter((b) => b !== messageId)
    msg.bookmarked = false
    return false
  }
  state.bookmarks.push(messageId)
  msg.bookmarked = true
  return true
}

export function isBookmarked(messageId: string): boolean {
  return state.bookmarks.includes(messageId)
}

// ---------------------------------------------------------------------------
// Mentions & Search
// ---------------------------------------------------------------------------

export function getMentions(): { mentions: Array<Record<string, unknown>>; total: number } {
  return { mentions: state.mentions, total: state.mentions.length }
}

export interface SearchResult {
  type: string
  id: string
  channel_id: string
  channel_name: string
  score: number
  snippet: string
  created_at?: string
  first_name?: string
  last_name?: string
}

export function searchMessages(query: string, channelFilter?: string | null): SearchResult[] {
  const q = query.toLowerCase().trim()
  if (q.length < 2) return []
  const channelName = (id: string) => getChannel(id)?.name ?? ''
  const results: SearchResult[] = []
  for (const [chId, msgs] of Object.entries(state.messagesByChannel)) {
    if (channelFilter && chId !== channelFilter) continue
    for (const m of msgs) {
      const content = m.content ?? ''
      const idx = content.toLowerCase().indexOf(q)
      if (idx < 0) continue
      const snippet =
        content.slice(0, idx) +
        '<mark>' +
        content.slice(idx, idx + q.length) +
        '</mark>' +
        content.slice(idx + q.length)
      results.push({
        type: 'message',
        id: m.id,
        channel_id: chId,
        channel_name: channelName(chId),
        score: 1,
        snippet,
        created_at: m.created_at,
        first_name: m.sender_first_name,
        last_name: m.sender_last_name,
      })
    }
  }
  return results
}
