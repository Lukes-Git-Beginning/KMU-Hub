import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import {
  mockChannels,
  mockDMs,
  mockUnread,
  mockMessagesByChannel,
  mockChannelMembers,
  mockMentions,
} from '../data/chat-data'
import { EMPLOYEES } from '../mock-db'

const API = API_BASE_URL

export const chatHandlers = [
  // List channels
  http.get(`${API}/api/v1/channels`, () => {
    return HttpResponse.json(mockChannels)
  }),

  // List DMs
  http.get(`${API}/api/v1/channels/dm`, () => {
    return HttpResponse.json(mockDMs)
  }),

  // Unread counts
  http.get(`${API}/api/v1/channels/unread`, () => {
    return HttpResponse.json(mockUnread)
  }),

  // Channel detail
  http.get(`${API}/api/v1/channels/:id`, ({ params }) => {
    const { id } = params
    const all = [...mockChannels.channels, ...mockDMs.channels]
    const channel = all.find((c) => c.id === id)
    if (!channel) {
      return HttpResponse.json({ error: 'not found' }, { status: 404 })
    }
    return HttpResponse.json({ channel })
  }),

  // Channel members
  http.get(`${API}/api/v1/channels/:id/members`, () => {
    return HttpResponse.json(mockChannelMembers)
  }),

  // Create channel
  http.post(`${API}/api/v1/channels`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json(
      {
        channel: {
          id: `ch-new-${Date.now()}`,
          name: body.name ?? 'new-channel',
          is_dm: false,
          is_private: body.is_private ?? false,
          description: body.description ?? '',
          member_count: 1,
          created_at: new Date().toISOString(),
        },
      },
      { status: 201 },
    )
  }),

  // Join channel
  http.post(`${API}/api/v1/channels/:id/join`, () => {
    return HttpResponse.json({ success: true })
  }),

  // Leave channel
  http.post(`${API}/api/v1/channels/:id/leave`, () => {
    return HttpResponse.json({ success: true })
  }),

  // Create DM (get-or-create: reuses an existing DM for the same user and
  // registers new DMs in mockDMs so detail/list lookups resolve afterwards)
  http.post(`${API}/api/v1/channels/dm`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const otherUserId = String(body.other_user_id ?? body.user_id ?? '')

    const existing = mockDMs.channels.find((c) => c.other_user_id === otherUserId)
    if (existing) {
      return HttpResponse.json({ channel: existing }, { status: 200 })
    }

    const employee = EMPLOYEES.find((e) => `usr-${e.id}` === otherUserId)
    const otherUserName = employee ? `${employee.firstName} ${employee.lastName}` : 'Neuer Kontakt'
    const channel = {
      id: `dm-new-${Date.now()}`,
      name: otherUserName,
      is_dm: true,
      is_private: true,
      other_user_id: otherUserId,
      other_user_name: otherUserName,
      member_count: 2,
      created_at: new Date().toISOString(),
    }
    mockDMs.channels.push(channel as (typeof mockDMs.channels)[number])
    return HttpResponse.json({ channel }, { status: 201 })
  }),

  // Mark channel as read
  http.post(`${API}/api/v1/channels/:id/read`, () => {
    return HttpResponse.json({ success: true })
  }),

  // List messages for channel
  http.get(`${API}/api/v1/channels/:id/messages`, ({ params }) => {
    const { id } = params
    const data = mockMessagesByChannel[id as string]
    if (!data) {
      return HttpResponse.json({ messages: [], has_more: false })
    }
    return HttpResponse.json(data)
  }),

  // Send message
  http.post(`${API}/api/v1/channels/:id/messages`, async ({ params, request }) => {
    const { id } = params
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json(
      {
        message: {
          id: `msg-new-${Date.now()}`,
          content: body.content ?? '',
          channel_id: id,
          sender_id: 'usr-e1',
          sender_name: 'Stefan Vogel',
          created_at: new Date().toISOString(),
          reply_count: 0,
          reactions: [],
          edited_at: null,
        },
      },
      { status: 201 },
    )
  }),

  // Edit message
  http.put(`${API}/api/v1/messages/:id`, async ({ params, request }) => {
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json({
      message: {
        id: params.id,
        content: body.content ?? '',
        edited_at: new Date().toISOString(),
      },
    })
  }),

  // Delete message
  http.delete(`${API}/api/v1/messages/:id`, () => {
    return HttpResponse.json({ success: true })
  }),

  // User mentions across all channels
  http.get(`${API}/api/v1/messages/mentions`, () => {
    return HttpResponse.json(mockMentions)
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
          uploaded_by: 'usr-e1',
          uploader_first_name: 'Stefan',
          uploader_last_name: 'Vogel',
          has_thumbnail: false,
          created_at: new Date().toISOString(),
        },
      },
      { status: 201 },
    )
  }),

  // Thread replies
  http.get(`${API}/api/v1/messages/:id/thread`, () => {
    return HttpResponse.json({ messages: [], has_more: false })
  }),

  // -------------------------------------------------------------------------
  // Reactions (Phase 8 — /api/v1/messages)
  // -------------------------------------------------------------------------

  // Toggle reaction (POST /{id}/reactions)
  http.post(`${API}/api/v1/messages/:id/reactions`, async ({ params, request }) => {
    const { id } = params
    const body = (await request.json()) as Record<string, unknown>
    const emoji = (body.emoji as string) ?? '👍'
    // Return a minimal ToggleReactionResponse mirroring the proto shape
    return HttpResponse.json({
      reactions: [
        {
          message_id: id,
          user_id: 'usr-demo',
          emoji,
          created_at: new Date().toISOString(),
          first_name: 'Demo',
          last_name: 'User',
        },
      ],
      added: true,
    })
  }),

  // List reactions (GET /{id}/reactions)
  http.get(`${API}/api/v1/messages/:id/reactions`, ({ params }) => {
    const { id } = params
    // Deterministic seed: same logic as old chatReactions store
    let hash = 0
    const idStr = String(id)
    for (let i = 0; i < idStr.length; i++) hash = (hash + idStr.charCodeAt(i)) % 997
    const reactions =
      hash % 2 === 0
        ? [
            { message_id: id, user_id: 'usr-u1', emoji: '👍', created_at: new Date().toISOString() },
            { message_id: id, user_id: 'usr-u2', emoji: '👍', created_at: new Date().toISOString() },
          ]
        : []
    return HttpResponse.json({ reactions })
  }),

  // Reaction summary batch (POST /reactions/summary)
  http.post(`${API}/api/v1/messages/reactions/summary`, async ({ request }) => {
    const body = (await request.json()) as { message_ids?: string[] }
    const messageIds = body.message_ids ?? []
    const summaries = messageIds.flatMap((mid) => {
      let hash = 0
      for (let i = 0; i < mid.length; i++) hash = (hash + mid.charCodeAt(i)) % 997
      if (hash % 2 !== 0) return []
      return [
        {
          message_id: mid,
          emoji: '👍',
          count: 2,
          user_ids: ['usr-u1', 'usr-u2'],
          current_user_reacted: false,
        },
      ]
    })
    return HttpResponse.json({ summaries })
  }),

  // Full-text search across channel messages (demo: scans mock messages)
  http.get(`${API}/api/v1/chat/search`, ({ request }) => {
    const url = new URL(request.url)
    const q = (url.searchParams.get('q') ?? '').toLowerCase().trim()
    const channelFilter = url.searchParams.get('channel_id')
    const channels = mockChannels.channels ?? []
    const channelName = (id: string) => channels.find((c) => c.id === id)?.name ?? ''

    const results: Record<string, unknown>[] = []
    if (q.length >= 2) {
      for (const [chId, bucket] of Object.entries(mockMessagesByChannel)) {
        if (channelFilter && chId !== channelFilter) continue
        const msgs = (bucket as { messages?: Array<Record<string, string>> }).messages ?? []
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
    }
    return HttpResponse.json({ results, total: results.length })
  }),
]
