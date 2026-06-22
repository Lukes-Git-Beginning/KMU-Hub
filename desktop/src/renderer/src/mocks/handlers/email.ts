import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import {
  getAccounts,
  getFolders,
  getMessage,
  getThread,
  listMessages,
  setRead,
  toggleStar,
  moveToFolder,
  deleteMessage,
  appendMessage,
  getSignatures,
  getLabels,
  createLabel,
  updateLabel,
  deleteLabel,
  setMessageLabels,
  getRules,
  createRule,
  deleteRule,
  applyRules,
  type ListMessagesExtra,
} from '../data/email-store'
import type { EmailRuleInfo } from '@/api/email-types'
import { IDS } from '../data/shared-ids'
import { daysAgo, hoursAgo, now } from '../data/date-helpers'

const API = API_BASE_URL

function parseListParams(url: URL): ListMessagesExtra {
  const p = url.searchParams
  const n = (k: string) => (p.get(k) ? Number(p.get(k)) : undefined)
  return {
    folder_id: p.get('folder_id') ?? '',
    account_id: p.get('account_id') ?? undefined,
    view: (p.get('view') as ListMessagesExtra['view']) ?? undefined,
    filter: (p.get('filter') as ListMessagesExtra['filter']) ?? undefined,
    label_id: p.get('label_id') ?? undefined,
    search: p.get('search') ?? undefined,
    sort_by: p.get('sort_by') ?? undefined,
    sort_desc: p.get('sort_desc') ? p.get('sort_desc') === 'true' : undefined,
    page: n('page'),
    per_page: n('per_page'),
  }
}

export const emailHandlers = [
  // Account list (multi-account switcher / unified inbox)
  http.get(`${API}/api/v1/email/accounts/list`, () => {
    return HttpResponse.json({ accounts: getAccounts() })
  }),

  // Accounts: the page reads `.account`; the multi-account list lives under `.accounts`.
  http.get(`${API}/api/v1/email/accounts/`, () => {
    const accounts = getAccounts()
    return HttpResponse.json({ account: accounts[0], accounts })
  }),

  // Folders for an account
  http.get(`${API}/api/v1/email/folders/`, ({ request }) => {
    const accountId = new URL(request.url).searchParams.get('account_id') ?? undefined
    return HttpResponse.json({ folders: getFolders(accountId) })
  }),

  // Thread — must precede the `:id` route below
  http.get(`${API}/api/v1/email/messages/thread/:threadId`, ({ params }) => {
    return HttpResponse.json({ messages: getThread(params.threadId as string) })
  }),

  // List messages (folder / unified / filtered / searched / sorted / paginated)
  http.get(`${API}/api/v1/email/messages/`, ({ request }) => {
    return HttpResponse.json(listMessages(parseListParams(new URL(request.url))))
  }),

  // Single message
  http.get(`${API}/api/v1/email/messages/:id`, ({ params }) => {
    const msg = getMessage(params.id as string)
    if (!msg) return HttpResponse.json({ error: 'not found' }, { status: 404 })
    return HttpResponse.json({ message: msg })
  }),

  // Mark read / unread
  http.post(`${API}/api/v1/email/messages/:id/read`, ({ params }) => {
    setRead(params.id as string, true)
    return HttpResponse.json({ success: true })
  }),
  http.post(`${API}/api/v1/email/messages/:id/unread`, ({ params }) => {
    setRead(params.id as string, false)
    return HttpResponse.json({ success: true })
  }),

  // Toggle star
  http.post(`${API}/api/v1/email/messages/:id/star`, ({ params }) => {
    const isStarred = toggleStar(params.id as string)
    return HttpResponse.json({ is_starred: isStarred })
  }),

  // Move to folder
  http.post(`${API}/api/v1/email/messages/:id/move`, async ({ params, request }) => {
    const body = (await request.json().catch(() => ({}))) as { target_folder_id?: string }
    if (body.target_folder_id) moveToFolder(params.id as string, body.target_folder_id)
    return HttpResponse.json({ success: true })
  }),

  // Delete (soft → trash, hard if already trash)
  http.delete(`${API}/api/v1/email/messages/:id`, ({ params }) => {
    deleteMessage(params.id as string)
    return HttpResponse.json({ success: true })
  }),

  // Send
  http.post(`${API}/api/v1/email/send/`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const msg = appendMessage({
      subject: (body.subject as string) ?? '',
      to: (body.to as never) ?? [],
      cc: (body.cc as never) ?? [],
      bcc: (body.bcc as never) ?? [],
      body_html: (body.body_html as string) ?? '',
      body_text: (body.body_text as string) ?? '',
      folderId: IDS.emailFolders.sent,
    })
    return HttpResponse.json({ message: msg }, { status: 201 })
  }),

  // Save draft
  http.post(`${API}/api/v1/email/send/draft`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const msg = appendMessage({
      subject: (body.subject as string) ?? '(Kein Betreff)',
      to: (body.to as never) ?? [],
      cc: (body.cc as never) ?? [],
      bcc: (body.bcc as never) ?? [],
      body_html: (body.body_html as string) ?? '',
      body_text: (body.body_text as string) ?? '',
      folderId: IDS.emailFolders.drafts,
      is_draft: true,
    })
    return HttpResponse.json({ message: msg }, { status: 201 })
  }),

  // Reply — inherit thread + RE: subject + recipient from the original message
  http.post(`${API}/api/v1/email/send/reply`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const original = getMessage((body.original_message_id as string) ?? '')
    const subject = original
      ? original.subject.replace(/^(RE|AW):\s*/i, 'RE: ').startsWith('RE:')
        ? original.subject
        : `RE: ${original.subject}`
      : 'RE:'
    const to = original ? [original.from] : []
    const msg = appendMessage({
      subject,
      to,
      body_html: (body.body_html as string) ?? '',
      body_text: (body.body_text as string) ?? '',
      folderId: IDS.emailFolders.sent,
      thread_id: original?.thread_id,
      in_reply_to: original?.message_id_header,
    })
    return HttpResponse.json({ message: msg }, { status: 201 })
  }),

  // Forward — inherit thread + FW: subject
  http.post(`${API}/api/v1/email/send/forward`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const original = getMessage((body.original_message_id as string) ?? '')
    const subject = original
      ? original.subject.startsWith('FW:')
        ? original.subject
        : `FW: ${original.subject}`
      : 'FW:'
    const msg = appendMessage({
      subject,
      to: (body.to as never) ?? [],
      body_html: (body.body_html as string) ?? '',
      body_text: (body.body_text as string) ?? '',
      folderId: IDS.emailFolders.sent,
      thread_id: original?.thread_id,
    })
    return HttpResponse.json({ message: msg }, { status: 201 })
  }),

  // ── Labels ──
  http.get(`${API}/api/v1/email/labels`, () => {
    return HttpResponse.json({ labels: getLabels() })
  }),
  http.post(`${API}/api/v1/email/labels`, async ({ request }) => {
    const body = (await request.json()) as { name?: string; color?: string }
    const label = createLabel(body.name ?? 'Label', body.color ?? 'slate')
    return HttpResponse.json({ label }, { status: 201 })
  }),
  http.patch(`${API}/api/v1/email/labels/:id`, async ({ params, request }) => {
    const body = (await request.json()) as { name?: string; color?: string }
    updateLabel(params.id as string, body)
    return HttpResponse.json({ success: true })
  }),
  http.delete(`${API}/api/v1/email/labels/:id`, ({ params }) => {
    deleteLabel(params.id as string)
    return HttpResponse.json({ success: true })
  }),

  // Assign labels to a message
  http.post(`${API}/api/v1/email/messages/:id/labels`, async ({ params, request }) => {
    const body = (await request.json()) as { label_ids?: string[] }
    const msg = setMessageLabels(params.id as string, body.label_ids ?? [])
    return HttpResponse.json({ message: msg })
  }),

  // ── Rules ──
  http.get(`${API}/api/v1/email/rules`, () => {
    return HttpResponse.json({ rules: getRules() })
  }),
  http.post(`${API}/api/v1/email/rules`, async ({ request }) => {
    const body = (await request.json()) as Omit<EmailRuleInfo, 'id'>
    const rule = createRule(body)
    return HttpResponse.json({ rule }, { status: 201 })
  }),
  http.delete(`${API}/api/v1/email/rules/:id`, ({ params }) => {
    deleteRule(params.id as string)
    return HttpResponse.json({ success: true })
  }),
  http.post(`${API}/api/v1/email/rules/apply`, () => {
    return HttpResponse.json({ affected: applyRules() })
  }),

  // Signatures
  http.get(`${API}/api/v1/email/signatures/`, () => {
    return HttpResponse.json({ signatures: getSignatures() })
  }),

  // Sync status / trigger
  http.get(`${API}/api/v1/email/sync/status`, () => {
    return HttpResponse.json({ status: 'idle', last_sync_at: now(), error_message: '' })
  }),
  http.post(`${API}/api/v1/email/sync/trigger`, () => {
    return HttpResponse.json({ status: 'started' })
  }),

  // CRM links for a message (cross-module links — populated by the CRM store)
  http.get(`${API}/api/v1/email/links/message/:id`, () => {
    return HttpResponse.json({ links: [] })
  }),

  // Contact emails — emails linked to a specific contact
  http.get(`${API}/api/v1/email/links/contact/:contactId`, ({ params }) => {
    const contactId = params.contactId as string
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
