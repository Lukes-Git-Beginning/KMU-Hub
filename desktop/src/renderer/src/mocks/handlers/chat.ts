import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import * as store from '../data/chat-store'

const API = API_BASE_URL

export const chatHandlers = [
  // List channels
  http.get(`${API}/api/v1/channels`, ({ request }) => {
    const url = new URL(request.url)
    const includeArchived = url.searchParams.get('include_archived') === 'true'
    return HttpResponse.json({ channels: store.getChannels(includeArchived) })
  }),

  // List DMs
  http.get(`${API}/api/v1/channels/dm`, () => {
    return HttpResponse.json({ channels: store.getDMs() })
  }),

  // Unread counts
  http.get(`${API}/api/v1/channels/unread`, () => {
    return HttpResponse.json({ unread_counts: store.getUnreadCounts() })
  }),

  // Channel detail
  http.get(`${API}/api/v1/channels/:id`, ({ params }) => {
    const channel = store.getChannel(String(params.id))
    if (!channel) {
      return HttpResponse.json({ error: 'not found' }, { status: 404 })
    }
    return HttpResponse.json({ channel })
  }),

  // Channel members
  http.get(`${API}/api/v1/channels/:id/members`, ({ params }) => {
    return HttpResponse.json({ members: store.getMembers(String(params.id)) })
  }),

  // Create channel
  http.post(`${API}/api/v1/channels`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const channel = store.createChannel({
      name: String(body.name ?? ''),
      description: body.description ? String(body.description) : undefined,
      is_private: Boolean(body.is_private),
    })
    return HttpResponse.json({ channel }, { status: 201 })
  }),

  // Rename / update channel
  http.patch(`${API}/api/v1/channels/:id`, async ({ params, request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const channel = store.renameChannel(String(params.id), {
      name: body.name !== undefined ? String(body.name) : undefined,
      description: body.description !== undefined ? String(body.description) : undefined,
      is_private: body.is_private !== undefined ? Boolean(body.is_private) : undefined,
    })
    if (!channel) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    return HttpResponse.json({ channel })
  }),

  // Join channel
  http.post(`${API}/api/v1/channels/:id/join`, ({ params }) => {
    const channel = store.joinChannel(String(params.id))
    return HttpResponse.json({ success: true, channel })
  }),

  // Leave channel
  http.post(`${API}/api/v1/channels/:id/leave`, ({ params }) => {
    store.leaveChannel(String(params.id))
    return HttpResponse.json({ success: true })
  }),

  // Create DM (get-or-create). A `participant_ids` array creates a group DM (KO-2).
  http.post(`${API}/api/v1/channels/dm`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const participantIds = Array.isArray(body.participant_ids)
      ? (body.participant_ids as string[])
      : null

    if (participantIds && participantIds.length > 1) {
      const { channel, created } = store.createGroupDM(participantIds)
      return HttpResponse.json({ channel }, { status: created ? 201 : 200 })
    }

    const otherUserId = String(body.other_user_id ?? body.user_id ?? participantIds?.[0] ?? '')
    const { channel, created } = store.getOrCreateDM(otherUserId)
    return HttpResponse.json({ channel }, { status: created ? 201 : 200 })
  }),

  // Mark channel as read
  http.post(`${API}/api/v1/channels/:id/read`, ({ params }) => {
    store.markChannelRead(String(params.id))
    return HttpResponse.json({ success: true })
  }),

  // List messages for channel
  http.get(`${API}/api/v1/channels/:id/messages`, ({ params, request }) => {
    const url = new URL(request.url)
    const before = url.searchParams.get('before') ?? undefined
    const limit = Number(url.searchParams.get('limit') ?? 50)
    return HttpResponse.json(store.listMessages(String(params.id), before, limit))
  }),

  // Send message
  http.post(`${API}/api/v1/channels/:id/messages`, async ({ params, request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const message = store.appendMessage({
      channelId: String(params.id),
      content: String(body.content ?? ''),
      parentMessageId: body.parent_message_id ? String(body.parent_message_id) : null,
    })
    return HttpResponse.json({ message }, { status: 201 })
  }),

  // Edit message
  http.put(`${API}/api/v1/messages/:id`, async ({ params, request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const message = store.editMessage(String(params.id), String(body.content ?? ''))
    if (!message) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    return HttpResponse.json({ message })
  }),

  // Delete message
  http.delete(`${API}/api/v1/messages/:id`, ({ params }) => {
    store.deleteMessage(String(params.id))
    return HttpResponse.json({ success: true })
  }),

  // Toggle bookmark (KO-4)
  http.post(`${API}/api/v1/messages/:id/bookmark`, ({ params }) => {
    const bookmarked = store.toggleBookmark(String(params.id))
    return HttpResponse.json({ bookmarked })
  }),

  // List bookmarked messages (KO-4)
  http.get(`${API}/api/v1/messages/bookmarks`, () => {
    return HttpResponse.json({ messages: store.getBookmarks() })
  }),

  // User mentions across all channels
  http.get(`${API}/api/v1/messages/mentions`, () => {
    return HttpResponse.json(store.getMentions())
  }),

  // File upload (multipart) — echoes a FileInfo so demo mode works
  http.post(`${API}/api/v1/files/upload`, async ({ request }) => {
    const form = await request.formData().catch(() => null)
    const file = form?.get('file')
    const channelId = form?.get('channel_id')
    const messageId = form?.get('message_id')
    const name = file instanceof File ? file.name : 'datei'
    const size = file instanceof File ? file.size : 0
    const mime = file instanceof File ? file.type : 'application/octet-stream'
    return HttpResponse.json(
      {
        file: {
          id: `file-up-${Date.now()}`,
          message_id: messageId ?? undefined,
          channel_id: channelId ?? undefined,
          filename: name,
          mime_type: mime,
          file_size: size,
          uploaded_by: store.CHAT_CURRENT_USER.id,
          uploader_first_name: store.CHAT_CURRENT_USER.firstName,
          uploader_last_name: store.CHAT_CURRENT_USER.lastName,
          has_thumbnail: false,
          created_at: new Date().toISOString(),
        },
      },
      { status: 201 },
    )
  }),

  // Thread replies
  http.get(`${API}/api/v1/messages/:id/thread`, ({ params }) => {
    const { parent, replies, has_more } = store.getThread(String(params.id))
    return HttpResponse.json({ parent, messages: replies, replies, has_more })
  }),

  // -------------------------------------------------------------------------
  // Reactions
  // -------------------------------------------------------------------------

  // Toggle reaction
  http.post(`${API}/api/v1/messages/:id/reactions`, async ({ params, request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const emoji = (body.emoji as string) ?? '👍'
    const { reactions, added } = store.toggleReaction(String(params.id), emoji)
    return HttpResponse.json({ reactions, added })
  }),

  // List reactions
  http.get(`${API}/api/v1/messages/:id/reactions`, ({ params }) => {
    return HttpResponse.json({ reactions: store.getReactions(String(params.id)) })
  }),

  // Reaction summary batch
  http.post(`${API}/api/v1/messages/reactions/summary`, async ({ request }) => {
    const body = (await request.json()) as { message_ids?: string[] }
    const summaries = store.getReactionSummary(body.message_ids ?? [])
    return HttpResponse.json({ summaries })
  }),

  // Full-text search across channel messages
  http.get(`${API}/api/v1/chat/search`, ({ request }) => {
    const url = new URL(request.url)
    const q = url.searchParams.get('q') ?? ''
    const channelFilter = url.searchParams.get('channel_id')
    const results = store.searchMessages(q, channelFilter)
    return HttpResponse.json({ results, total: results.length })
  }),
]
