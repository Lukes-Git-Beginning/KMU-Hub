import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS } from '../data/shared-ids'
import { hoursAgo, daysAgo, minutesAgo } from '../data/date-helpers'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Unified inbox messages matching InboxMessage interface
// ---------------------------------------------------------------------------

const inboxMessages = [
  {
    id: 'msg-001',
    user_id: IDS.users.thomas,
    channel: 'email' as const,
    source_id: 'email-001',
    sender_name: 'Peter Gruber',
    sender_id: IDS.contacts.gruber,
    sender_email: 'p.gruber@gruber-maschinenbau.de',
    subject: 'Anfrage: CRM-Demo für unser Team',
    preview: 'Mittwoch 14:00 wäre perfekt. Mein CTO Herr Wagner wird auch dabei sein. Bitte schicken Sie den Einladungslink.',
    is_read: false,
    is_starred: true,
    is_archived: false,
    assigned_to: IDS.users.thomas,
    tags: ['Demo', 'Enterprise'],
    deep_link: '/crm/contacts/' + IDS.contacts.gruber,
    crm_contact_id: IDS.contacts.gruber,
    received_at: hoursAgo(1),
    created_at: daysAgo(2),
    updated_at: hoursAgo(1),
  },
  {
    id: 'msg-002',
    user_id: IDS.users.markus,
    channel: 'chat' as const,
    source_id: 'chat-001',
    sender_name: 'Anna Schneider',
    sender_id: IDS.contacts.schneider,
    sender_email: 'a.schneider@helvetia-software.ch',
    subject: 'Technische Frage: API Rate Limiting',
    preview: 'Ja, das wäre super. Nächste Woche wäre ideal.',
    is_read: false,
    is_starred: false,
    is_archived: false,
    assigned_to: IDS.users.markus,
    tags: ['Support', 'Technik'],
    deep_link: '/crm/contacts/' + IDS.contacts.schneider,
    crm_contact_id: IDS.contacts.schneider,
    received_at: hoursAgo(3),
    created_at: daysAgo(1),
    updated_at: hoursAgo(3),
  },
  {
    id: 'msg-003',
    user_id: IDS.users.stefan,
    channel: 'notification' as const,
    source_id: 'form-001',
    sender_name: 'Michael Brunner',
    sender_email: 'm.brunner@donau-pharma.at',
    subject: 'Kontaktformular: Preisanfrage',
    preview: 'Wir suchen eine Business-Lösung für 120 Mitarbeiter. Bitte um Preisangebot für Enterprise inkl. DSGVO-Compliance-Modul.',
    is_read: false,
    is_starred: false,
    is_archived: false,
    tags: ['Lead'],
    deep_link: '/kommunikation/msg-003',
    received_at: hoursAgo(5),
    created_at: hoursAgo(5),
    updated_at: hoursAgo(5),
  },
  {
    id: 'msg-004',
    user_id: IDS.users.thomas,
    channel: 'email' as const,
    source_id: 'email-002',
    sender_name: 'Maria Huber',
    sender_id: IDS.contacts.huber,
    sender_email: 'm.huber@bavaria-elektro.de',
    subject: 'Re: Wartungsvertrag Verlängerung',
    preview: 'Bei 2 Jahren können wir 10% anbieten. Ich sende Ihnen morgen das aktualisierte Angebot.',
    is_read: true,
    is_starred: false,
    is_archived: false,
    assigned_to: IDS.users.thomas,
    tags: ['Vertrag'],
    deep_link: '/crm/contacts/' + IDS.contacts.huber,
    crm_contact_id: IDS.contacts.huber,
    received_at: daysAgo(1),
    created_at: daysAgo(5),
    updated_at: daysAgo(1),
  },
  {
    id: 'msg-005',
    user_id: IDS.users.julia,
    channel: 'email' as const,
    source_id: 'email-003',
    sender_name: 'Christian Berger',
    sender_id: IDS.contacts.berger,
    sender_email: 'c.berger@rhein-consulting.de',
    subject: 'Feedback: Sehr zufrieden mit dem Onboarding',
    preview: 'Das Onboarding war exzellent. Unser Team ist innerhalb von 2 Tagen produktiv gewesen.',
    is_read: true,
    is_starred: false,
    is_archived: true,
    assigned_to: IDS.users.julia,
    tags: ['Feedback'],
    deep_link: '/crm/contacts/' + IDS.contacts.berger,
    crm_contact_id: IDS.contacts.berger,
    received_at: daysAgo(3),
    created_at: daysAgo(4),
    updated_at: daysAgo(3),
  },
  {
    id: 'msg-006',
    user_id: IDS.users.thomas,
    channel: 'email' as const,
    source_id: 'email-004',
    sender_name: 'Werner Koch',
    sender_email: 'w.koch@alpine-bau.at',
    subject: 'Angebot für Bauprojekt Innsbruck',
    preview: 'Könnten Sie uns ein Angebot für die CRM-Anbindung an unser ERP-System schicken?',
    is_read: false,
    is_starred: false,
    is_archived: false,
    assigned_to: IDS.users.thomas,
    tags: ['Angebot'],
    deep_link: '/kommunikation/msg-006',
    received_at: minutesAgo(45),
    created_at: minutesAgo(45),
    updated_at: minutesAgo(45),
  },
  {
    id: 'msg-007',
    user_id: IDS.users.stefan,
    channel: 'notification' as const,
    source_id: 'sys-001',
    sender_name: 'System',
    subject: 'Backup erfolgreich abgeschlossen',
    preview: 'Das tägliche Backup wurde um 02:00 UTC erfolgreich durchgeführt. Größe: 2.4 GB.',
    is_read: true,
    is_starred: false,
    is_archived: false,
    tags: ['System'],
    deep_link: '/einstellungen/backups',
    received_at: hoursAgo(8),
    created_at: hoursAgo(8),
    updated_at: hoursAgo(8),
  },
  {
    id: 'msg-008',
    user_id: IDS.users.lena,
    channel: 'chat' as const,
    source_id: 'chat-002',
    sender_name: 'Sophie Lang',
    sender_id: IDS.users.sophie,
    sender_email: 'sophie.lang@techvision.de',
    subject: 'Design Review: Neue Landing Page',
    preview: 'Kannst du dir mal die neuen Mockups anschauen? Ich hab sie in den Marketing-Ordner gelegt.',
    is_read: true,
    is_starred: true,
    is_archived: false,
    assigned_to: IDS.users.lena,
    tags: ['Design', 'Intern'],
    deep_link: '/kommunikation/msg-008',
    received_at: hoursAgo(2),
    created_at: hoursAgo(2),
    updated_at: hoursAgo(2),
  },
]

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

export const inboxHandlers = [
  // List messages (matches InboxMessageList response)
  http.get(`${API}/api/v1/inbox/messages`, ({ request }) => {
    const url = new URL(request.url)
    const channel = url.searchParams.get('channel')
    const isRead = url.searchParams.get('is_read')
    const isStarred = url.searchParams.get('is_starred')
    const isArchived = url.searchParams.get('is_archived')
    const search = url.searchParams.get('search')

    let filtered = [...inboxMessages]

    if (channel) {
      filtered = filtered.filter((m) => m.channel === channel)
    }
    if (isRead !== null) {
      filtered = filtered.filter((m) => m.is_read === (isRead === 'true'))
    }
    if (isStarred === 'true') {
      filtered = filtered.filter((m) => m.is_starred)
    }
    if (isArchived !== null) {
      filtered = filtered.filter((m) => m.is_archived === (isArchived === 'true'))
    }
    if (search) {
      const q = search.toLowerCase()
      filtered = filtered.filter(
        (m) =>
          m.subject.toLowerCase().includes(q) ||
          m.sender_name.toLowerCase().includes(q) ||
          m.preview.toLowerCase().includes(q),
      )
    }

    return HttpResponse.json({
      messages: filtered,
      total_count: filtered.length,
    })
  }),

  // Message detail
  http.get(`${API}/api/v1/inbox/messages/unread-count`, () => {
    const unreadEmail = inboxMessages.filter((m) => !m.is_read && m.channel === 'email').length
    const unreadChat = inboxMessages.filter((m) => !m.is_read && m.channel === 'chat').length
    const unreadNotification = inboxMessages.filter((m) => !m.is_read && m.channel === 'notification').length

    return HttpResponse.json({
      counts: [
        { channel: 'email', count: unreadEmail },
        { channel: 'chat', count: unreadChat },
        { channel: 'notification', count: unreadNotification },
      ],
      total: unreadEmail + unreadChat + unreadNotification,
    })
  }),

  // Single message
  http.get(`${API}/api/v1/inbox/messages/:id`, ({ params }) => {
    const msg = inboxMessages.find((m) => m.id === params.id)
    if (!msg) {
      return HttpResponse.json({ error: 'Message not found' }, { status: 404 })
    }
    return HttpResponse.json(msg)
  }),

  // Mark read
  http.post(`${API}/api/v1/inbox/messages/:id/read`, () => {
    return new HttpResponse(null, { status: 204 })
  }),

  // Mark unread
  http.post(`${API}/api/v1/inbox/messages/:id/unread`, () => {
    return new HttpResponse(null, { status: 204 })
  }),

  // Toggle star
  http.post(`${API}/api/v1/inbox/messages/:id/star`, () => {
    return new HttpResponse(null, { status: 204 })
  }),

  // Archive
  http.post(`${API}/api/v1/inbox/messages/:id/archive`, () => {
    return new HttpResponse(null, { status: 204 })
  }),

  // Reply
  http.post(`${API}/api/v1/inbox/messages/:id/reply`, () => {
    return new HttpResponse(null, { status: 204 })
  }),

  // Assign
  http.post(`${API}/api/v1/inbox/messages/:id/assign`, () => {
    return new HttpResponse(null, { status: 204 })
  }),

  // Bulk read
  http.post(`${API}/api/v1/inbox/messages/bulk/read`, () => {
    return new HttpResponse(null, { status: 204 })
  }),

  // Bulk archive
  http.post(`${API}/api/v1/inbox/messages/bulk/archive`, () => {
    return new HttpResponse(null, { status: 204 })
  }),
]
