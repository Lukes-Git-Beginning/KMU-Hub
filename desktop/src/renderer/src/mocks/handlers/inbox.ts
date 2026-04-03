import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS } from '../data/shared-ids'
import { hoursAgo, daysAgo } from '../data/date-helpers'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Unified inbox conversations (5 — mix of email, chat, form)
// ---------------------------------------------------------------------------

const conversations = [
  {
    id: 'conv-001',
    source: 'email',
    subject: 'Anfrage: CRM-Demo für unser Team',
    status: 'open',
    priority: 'high',
    contact_id: IDS.contacts.gruber,
    contact_name: 'Peter Gruber',
    contact_email: 'p.gruber@gruber-maschinenbau.de',
    assigned_to: IDS.users.thomas,
    assigned_to_name: 'Thomas Meier',
    last_message_at: hoursAgo(1),
    message_count: 3,
    unread: true,
    tags: ['Demo', 'Enterprise'],
    created_at: daysAgo(2),
  },
  {
    id: 'conv-002',
    source: 'chat',
    subject: 'Technische Frage: API Rate Limiting',
    status: 'open',
    priority: 'normal',
    contact_id: IDS.contacts.schneider,
    contact_name: 'Anna Schneider',
    contact_email: 'a.schneider@helvetia-software.ch',
    assigned_to: IDS.users.markus,
    assigned_to_name: 'Markus Weber',
    last_message_at: hoursAgo(3),
    message_count: 5,
    unread: true,
    tags: ['Support', 'Technik'],
    created_at: daysAgo(1),
  },
  {
    id: 'conv-003',
    source: 'form',
    subject: 'Kontaktformular: Preisanfrage',
    status: 'open',
    priority: 'normal',
    contact_id: null,
    contact_name: 'Michael Brunner',
    contact_email: 'm.brunner@donau-pharma.at',
    assigned_to: null,
    assigned_to_name: null,
    last_message_at: hoursAgo(5),
    message_count: 1,
    unread: true,
    tags: ['Lead'],
    created_at: hoursAgo(5),
  },
  {
    id: 'conv-004',
    source: 'email',
    subject: 'Re: Wartungsvertrag Verlaengerung',
    status: 'pending',
    priority: 'normal',
    contact_id: IDS.contacts.huber,
    contact_name: 'Maria Huber',
    contact_email: 'm.huber@bavaria-elektro.de',
    assigned_to: IDS.users.thomas,
    assigned_to_name: 'Thomas Meier',
    last_message_at: daysAgo(1),
    message_count: 4,
    unread: false,
    tags: ['Vertrag'],
    created_at: daysAgo(5),
  },
  {
    id: 'conv-005',
    source: 'email',
    subject: 'Feedback: Sehr zufrieden mit dem Onboarding',
    status: 'closed',
    priority: 'low',
    contact_id: IDS.contacts.berger,
    contact_name: 'Christian Berger',
    contact_email: 'c.berger@rhein-consulting.de',
    assigned_to: IDS.users.julia,
    assigned_to_name: 'Julia Hofmann',
    last_message_at: daysAgo(3),
    message_count: 2,
    unread: false,
    tags: ['Feedback'],
    created_at: daysAgo(4),
  },
]

// ---------------------------------------------------------------------------
// Messages per conversation
// ---------------------------------------------------------------------------

const messages: Record<string, Array<{
  id: string
  conversation_id: string
  direction: 'inbound' | 'outbound'
  sender_name: string
  sender_email: string
  body: string
  created_at: string
}>> = {
  'conv-001': [
    { id: 'msg-001', conversation_id: 'conv-001', direction: 'inbound', sender_name: 'Peter Gruber', sender_email: 'p.gruber@gruber-maschinenbau.de', body: 'Guten Tag, wir sind ein Maschinenbau-Unternehmen mit 45 Mitarbeitern und suchen eine CRM-Lösung. Könnten wir eine Demo vereinbaren?', created_at: daysAgo(2) },
    { id: 'msg-002', conversation_id: 'conv-001', direction: 'outbound', sender_name: 'Thomas Meier', sender_email: 'thomas.meier@techvision.de', body: 'Sehr geehrter Herr Gruber, vielen Dank für Ihr Interesse! Gerne vereinbare ich eine Demo. Passt Ihnen nächste Woche Dienstag oder Mittwoch?', created_at: daysAgo(1) },
    { id: 'msg-003', conversation_id: 'conv-001', direction: 'inbound', sender_name: 'Peter Gruber', sender_email: 'p.gruber@gruber-maschinenbau.de', body: 'Mittwoch 14:00 wäre perfekt. Mein CTO Herr Wagner wird auch dabei sein. Bitte schicken Sie den Einladungslink.', created_at: hoursAgo(1) },
  ],
  'conv-002': [
    { id: 'msg-004', conversation_id: 'conv-002', direction: 'inbound', sender_name: 'Anna Schneider', sender_email: 'a.schneider@helvetia-software.ch', body: 'Hallo, wir nutzen die API und stoßen an Rate Limits. Welche Limits gelten für den Enterprise-Plan?', created_at: daysAgo(1) },
    { id: 'msg-005', conversation_id: 'conv-002', direction: 'outbound', sender_name: 'Markus Weber', sender_email: 'markus.weber@techvision.de', body: 'Hallo Frau Schneider, Enterprise hat 1000 req/min. Welche Endpoints nutzen Sie hauptsächlich?', created_at: daysAgo(1) },
    { id: 'msg-006', conversation_id: 'conv-002', direction: 'inbound', sender_name: 'Anna Schneider', sender_email: 'a.schneider@helvetia-software.ch', body: 'Hauptsächlich /contacts und /deals. Wir synchronisieren ca. alle 5 Minuten.', created_at: hoursAgo(6) },
    { id: 'msg-007', conversation_id: 'conv-002', direction: 'outbound', sender_name: 'Markus Weber', sender_email: 'markus.weber@techvision.de', body: 'Empfehlung: Nutzen Sie Webhooks statt Polling. Wir können auch Bulk-Endpoints anbieten. Soll ich ein kurzes Architektur-Gespräch anbieten?', created_at: hoursAgo(4) },
    { id: 'msg-008', conversation_id: 'conv-002', direction: 'inbound', sender_name: 'Anna Schneider', sender_email: 'a.schneider@helvetia-software.ch', body: 'Ja, das wäre super. Nächste Woche wäre ideal.', created_at: hoursAgo(3) },
  ],
  'conv-003': [
    { id: 'msg-009', conversation_id: 'conv-003', direction: 'inbound', sender_name: 'Michael Brunner', sender_email: 'm.brunner@donau-pharma.at', body: 'Über das Kontaktformular: Wir suchen eine Business-Lösung für 120 Mitarbeiter. Bitte um Preisangebot für Enterprise inkl. DSGVO-Compliance-Modul.', created_at: hoursAgo(5) },
  ],
  'conv-004': [
    { id: 'msg-010', conversation_id: 'conv-004', direction: 'inbound', sender_name: 'Maria Huber', sender_email: 'm.huber@bavaria-elektro.de', body: 'Hallo Herr Meier, unser Wartungsvertrag läuft nächsten Monat aus. Wie sind die Konditionen für eine Verlängerung?', created_at: daysAgo(5) },
    { id: 'msg-011', conversation_id: 'conv-004', direction: 'outbound', sender_name: 'Thomas Meier', sender_email: 'thomas.meier@techvision.de', body: 'Guten Tag Frau Huber, ich bereite ein Angebot vor. Möchten Sie den gleichen Umfang oder erweiterte Leistungen?', created_at: daysAgo(4) },
    { id: 'msg-012', conversation_id: 'conv-004', direction: 'inbound', sender_name: 'Maria Huber', sender_email: 'm.huber@bavaria-elektro.de', body: 'Gleicher Umfang reicht. Gibt es einen Rabatt bei 2-Jahres-Bindung?', created_at: daysAgo(2) },
    { id: 'msg-013', conversation_id: 'conv-004', direction: 'outbound', sender_name: 'Thomas Meier', sender_email: 'thomas.meier@techvision.de', body: 'Bei 2 Jahren können wir 10% anbieten. Ich sende Ihnen morgen das aktualisierte Angebot.', created_at: daysAgo(1) },
  ],
  'conv-005': [
    { id: 'msg-014', conversation_id: 'conv-005', direction: 'inbound', sender_name: 'Christian Berger', sender_email: 'c.berger@rhein-consulting.de', body: 'Das Onboarding war exzellent. Unser Team ist innerhalb von 2 Tagen produktiv gewesen. Besonderer Dank an Ihren Support.', created_at: daysAgo(4) },
    { id: 'msg-015', conversation_id: 'conv-005', direction: 'outbound', sender_name: 'Julia Hofmann', sender_email: 'julia.hofmann@techvision.de', body: 'Vielen Dank für das tolle Feedback, Herr Berger! Das freut unser ganzes Team. Bei Fragen sind wir jederzeit erreichbar.', created_at: daysAgo(3) },
  ],
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

export const inboxHandlers = [
  // List conversations
  http.get(`${API}/api/v1/inbox/conversations`, ({ request }) => {
    const url = new URL(request.url)
    const status = url.searchParams.get('status')
    const source = url.searchParams.get('source')

    let filtered = [...conversations]
    if (status) {
      filtered = filtered.filter((c) => c.status === status)
    }
    if (source) {
      filtered = filtered.filter((c) => c.source === source)
    }

    return HttpResponse.json({ conversations: filtered, total: filtered.length })
  }),

  // Conversation detail
  http.get(`${API}/api/v1/inbox/conversations/:id`, ({ params }) => {
    const conv = conversations.find((c) => c.id === params.id)
    if (!conv) {
      return HttpResponse.json({ error: 'Conversation not found' }, { status: 404 })
    }
    return HttpResponse.json({ conversation: conv })
  }),

  // Conversation messages
  http.get(`${API}/api/v1/inbox/conversations/:id/messages`, ({ params }) => {
    const convMessages = messages[params.id as string] || []
    return HttpResponse.json({ messages: convMessages, total: convMessages.length })
  }),
]
