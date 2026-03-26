import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import {
  mockEmailAccount,
  mockEmailFolders,
  mockEmailMessages,
  mockEmailMessagesById,
  mockSignatures,
} from '../data/emails'

const API = API_BASE_URL

export const emailHandlers = [
  // Get email account(s)
  http.get(`${API}/api/v1/email/accounts/`, () => {
    return HttpResponse.json(mockEmailAccount)
  }),

  // Get folders for account
  http.get(`${API}/api/v1/email/folders/`, () => {
    return HttpResponse.json(mockEmailFolders)
  }),

  // List messages for a folder
  http.get(`${API}/api/v1/email/messages/`, ({ request }) => {
    const url = new URL(request.url)
    const folderId = url.searchParams.get('folder_id')
    if (folderId && mockEmailMessages[folderId]) {
      return HttpResponse.json(mockEmailMessages[folderId])
    }
    // Default to inbox
    const inbox = Object.values(mockEmailMessages)[0]
    return HttpResponse.json(inbox ?? { messages: [], total: 0 })
  }),

  // Get single message
  http.get(`${API}/api/v1/email/messages/:id`, ({ params }) => {
    const msg = mockEmailMessagesById[params.id as string]
    if (!msg) {
      return HttpResponse.json({ error: 'not found' }, { status: 404 })
    }
    return HttpResponse.json({ message: msg })
  }),

  // Mark message as read
  http.post(`${API}/api/v1/email/messages/:id/read`, () => {
    return HttpResponse.json({ success: true })
  }),

  // Toggle star
  http.post(`${API}/api/v1/email/messages/:id/star`, () => {
    return HttpResponse.json({ success: true })
  }),

  // Send email
  http.post(`${API}/api/v1/email/send/`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json(
      {
        message: {
          id: `em-sent-${Date.now()}`,
          subject: body.subject ?? '',
          from: { name: 'Stefan Mueller', email: 'stefan.mueller@techvision.de' },
          to: body.to ?? [],
          date: new Date().toISOString(),
          folder_id: 'ef-sent',
          is_read: true,
          is_starred: false,
          has_attachments: false,
          attachments: [],
        },
      },
      { status: 201 },
    )
  }),

  // Save draft
  http.post(`${API}/api/v1/email/send/draft`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    return HttpResponse.json(
      {
        message: {
          id: `em-draft-${Date.now()}`,
          subject: body.subject ?? '(Kein Betreff)',
          from: { name: 'Stefan Mueller', email: 'stefan.mueller@techvision.de' },
          to: body.to ?? [],
          date: new Date().toISOString(),
          folder_id: 'ef-drafts',
          is_read: true,
          is_starred: false,
          has_attachments: false,
          attachments: [],
        },
      },
      { status: 201 },
    )
  }),

  // List signatures
  http.get(`${API}/api/v1/email/signatures/`, () => {
    return HttpResponse.json(mockSignatures)
  }),

  // Sync status
  http.get(`${API}/api/v1/email/sync/status`, () => {
    return HttpResponse.json({
      status: 'synced',
      last_sync_at: new Date().toISOString(),
      messages_synced: 1992,
    })
  }),

  // Email links for a message (cross-module links)
  http.get(`${API}/api/v1/email/links/message/:id`, () => {
    return HttpResponse.json({ links: [] })
  }),
]
