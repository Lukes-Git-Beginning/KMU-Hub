import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import {
  mockEmailAccount,
  mockEmailFolders,
  mockEmailMessages,
  mockEmailMessagesById,
  mockSignatures,
} from '../data/emails'
import { IDS } from '../data/shared-ids'
import { daysAgo, hoursAgo } from '../data/date-helpers'

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
          from: { name: 'Stefan Vogel', email: 'stefan.vogel@techvision.de' },
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
          from: { name: 'Stefan Vogel', email: 'stefan.vogel@techvision.de' },
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

  // Contact emails — get emails linked to a specific contact
  // emailLinkApi.getContactEmails → GET /api/v1/email/links/contact/:id
  http.get(`${API}/api/v1/email/links/contact/:contactId`, ({ params }) => {
    const contactId = params.contactId as string

    // Return contact-specific demo emails for contacts that have mock data.
    // For the primary demo contact (mueller) return 3 plausible entries;
    // for all others return 2 generic ones so the section is never empty.
    const isMueller = contactId === IDS.contacts.mueller

    const demoMessages = isMueller
      ? [
          {
            id: `ce-${contactId}-1`,
            subject: 'Angebot: IT-Infrastruktur Modernisierung',
            from: { name: 'Stefan Vogel', email: 'stefan.vogel@techvision.de' },
            to: [{ name: 'Hans Müller', email: 'h.mueller@techvision.de' }],
            date: daysAgo(2),
            is_read: true,
            is_starred: false,
            has_attachments: true,
            folder_id: IDS.emailFolders.sent,
            thread_id: `th-ce-${contactId}-1`,
          },
          {
            id: `ce-${contactId}-2`,
            subject: 'RE: Terminbestätigung Onboarding',
            from: { name: 'Hans Müller', email: 'h.mueller@techvision.de' },
            to: [{ name: 'Stefan Vogel', email: 'stefan.vogel@techvision.de' }],
            date: daysAgo(5),
            is_read: true,
            is_starred: false,
            has_attachments: false,
            folder_id: IDS.emailFolders.inbox,
            thread_id: `th-ce-${contactId}-2`,
          },
          {
            id: `ce-${contactId}-3`,
            subject: 'Willkommen bei TechVision — Erste Schritte',
            from: { name: 'Stefan Vogel', email: 'stefan.vogel@techvision.de' },
            to: [{ name: 'Hans Müller', email: 'h.mueller@techvision.de' }],
            date: hoursAgo(72),
            is_read: true,
            is_starred: true,
            has_attachments: false,
            folder_id: IDS.emailFolders.sent,
            thread_id: `th-ce-${contactId}-3`,
          },
        ]
      : [
          {
            id: `ce-${contactId}-1`,
            subject: 'Erste Kontaktaufnahme',
            from: { name: 'Stefan Vogel', email: 'stefan.vogel@techvision.de' },
            to: [{ name: 'Kontakt', email: 'kontakt@firma.de' }],
            date: daysAgo(7),
            is_read: true,
            is_starred: false,
            has_attachments: false,
            folder_id: IDS.emailFolders.sent,
            thread_id: `th-ce-${contactId}-1`,
          },
          {
            id: `ce-${contactId}-2`,
            subject: 'Unterlagen wie besprochen',
            from: { name: 'Stefan Vogel', email: 'stefan.vogel@techvision.de' },
            to: [{ name: 'Kontakt', email: 'kontakt@firma.de' }],
            date: daysAgo(14),
            is_read: true,
            is_starred: false,
            has_attachments: true,
            folder_id: IDS.emailFolders.sent,
            thread_id: `th-ce-${contactId}-2`,
          },
        ]

    return HttpResponse.json({ messages: demoMessages, total: demoMessages.length })
  }),
]
