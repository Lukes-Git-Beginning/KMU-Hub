import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS } from '../data/shared-ids'
import { minutesAgo, hoursAgo, daysAgo } from '../data/date-helpers'
import type { QuietHours, DNDStatus, MutedResource } from '@/api/notification-client'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Notifications (10 recent)
// ---------------------------------------------------------------------------

const notifications = [
  {
    id: 'notif-001',
    type: 'deal_created',
    title: 'Neuer Deal erstellt',
    body: 'Thomas Meier hat den Deal "KI Beratung" (CHF 45\'000) erstellt.',
    read: false,
    actor_id: IDS.users.thomas,
    actor_name: 'Thomas Meier',
    resource_type: 'deal',
    resource_id: IDS.deals.kiBeratung,
    created_at: minutesAgo(15),
  },
  {
    id: 'notif-002',
    type: 'task_due',
    title: 'Task fällig morgen',
    body: 'Sprint Review vorbereiten — fällig am morgigen Tag.',
    read: false,
    actor_id: null,
    actor_name: 'System',
    resource_type: 'task',
    resource_id: 'task-001',
    created_at: minutesAgo(45),
  },
  {
    id: 'notif-003',
    type: 'chat_mention',
    title: 'Nachricht in #entwicklung',
    body: 'Markus Weber hat dich in #entwicklung erwähnt: "@Stefan schau dir mal den PR an"',
    read: false,
    actor_id: IDS.users.markus,
    actor_name: 'Markus Weber',
    resource_type: 'channel',
    resource_id: IDS.channels.entwicklung,
    created_at: hoursAgo(1),
  },
  {
    id: 'notif-004',
    type: 'leave_request',
    title: 'Urlaubsantrag eingegangen',
    body: 'Felix Krause beantragt Urlaub vom 09.04. bis 16.04.2026 (6 Tage).',
    read: false,
    actor_id: IDS.users.felix,
    actor_name: 'Felix Krause',
    resource_type: 'leave_request',
    resource_id: 'lr-001',
    created_at: hoursAgo(2),
  },
  {
    id: 'notif-005',
    type: 'document_shared',
    title: 'Dokument geteilt',
    body: 'Lena Braun hat "Brandbook_2026.pdf" mit dir geteilt.',
    read: true,
    actor_id: IDS.users.lena,
    actor_name: 'Lena Braun',
    resource_type: 'document',
    resource_id: 'file-010',
    created_at: hoursAgo(4),
  },
  {
    id: 'notif-006',
    type: 'deal_stage_changed',
    title: 'Deal-Phase geändert',
    body: 'Deal "ERP Migration" wurde nach "Proposal" verschoben.',
    read: true,
    actor_id: IDS.users.thomas,
    actor_name: 'Thomas Meier',
    resource_type: 'deal',
    resource_id: IDS.deals.erpMigration,
    created_at: hoursAgo(8),
  },
  {
    id: 'notif-007',
    type: 'invoice_paid',
    title: 'Rechnung bezahlt',
    body: 'Rechnung RE-2026-077 (EUR 12\'500) wurde als bezahlt markiert.',
    read: true,
    actor_id: IDS.users.michael,
    actor_name: 'Petra Zimmermann',
    resource_type: 'invoice',
    resource_id: IDS.invoices.inv007,
    created_at: daysAgo(1),
  },
  {
    id: 'notif-008',
    type: 'project_update',
    title: 'Projekt-Update',
    body: 'Sarah Beck hat das Projekt "Hub V2" aktualisiert — neuer Meilenstein hinzugefügt.',
    read: true,
    actor_id: IDS.users.elena,
    actor_name: 'Sarah Beck',
    resource_type: 'project',
    resource_id: IDS.projects.hubV2,
    created_at: daysAgo(1),
  },
  {
    id: 'notif-009',
    type: 'security_alert',
    title: 'Sicherheitswarnung',
    body: '3 fehlgeschlagene Login-Versuche von IP 203.0.113.88 blockiert.',
    read: true,
    actor_id: null,
    actor_name: 'System',
    resource_type: 'security',
    resource_id: null,
    created_at: daysAgo(2),
  },
  {
    id: 'notif-010',
    type: 'welcome',
    title: 'Willkommen bei Cosmi!',
    body: 'Ihr Arbeitsbereich ist eingerichtet. Erkunden Sie die Module im Seitenmenue.',
    read: true,
    actor_id: null,
    actor_name: 'System',
    resource_type: null,
    resource_id: null,
    created_at: daysAgo(90),
  },
  // Fristen-Reminder (V-3) — gespiegelt zu den auslaufenden Seed-Verträgen
  // (v-3/v-5/v-11). Die Vertrag-Detailseite zeigt dieselben Reminder; der
  // zustand-Toast surfacet sie zusätzlich live beim Öffnen des Moduls.
  {
    id: 'notif-contract-v-3',
    type: 'contract_expiry',
    title: 'Vertrag läuft bald ab',
    body: 'Microsoft 365 Business (Microsoft Ireland Operations Ltd.) — Ablauf in 18 Tagen',
    read: false,
    actor_id: null,
    actor_name: 'System',
    resource_type: 'contract',
    resource_id: 'v-3',
    created_at: hoursAgo(3),
  },
  {
    id: 'notif-contract-v-5',
    type: 'contract_expiry',
    title: 'Vertrag läuft bald ab',
    body: 'Müller Metallbau Rahmenvertrag (Müller Metallbau GmbH) — Ablauf in 47 Tagen',
    read: false,
    actor_id: null,
    actor_name: 'System',
    resource_type: 'contract',
    resource_id: 'v-5',
    created_at: daysAgo(1),
  },
  {
    id: 'notif-contract-v-11',
    type: 'contract_expiry',
    title: 'Vertrag läuft bald ab',
    body: 'Lagerraum Augsburg (Immo-Invest Augsburg GmbH) — Ablauf in 82 Tagen',
    read: false,
    actor_id: null,
    actor_name: 'System',
    resource_type: 'contract',
    resource_id: 'v-11',
    created_at: daysAgo(2),
  },
]

// ---------------------------------------------------------------------------
// Notification preferences
// ---------------------------------------------------------------------------

const preferences = {
  channels: {
    in_app: true,
    email: true,
    desktop: true,
    mobile: false,
  },
  categories: {
    deals: { in_app: true, email: true, desktop: true },
    tasks: { in_app: true, email: false, desktop: true },
    chat: { in_app: true, email: false, desktop: true },
    documents: { in_app: true, email: false, desktop: false },
    finance: { in_app: true, email: true, desktop: false },
    security: { in_app: true, email: true, desktop: true },
    hr: { in_app: true, email: true, desktop: false },
    system: { in_app: true, email: true, desktop: true },
  },
}

const eventTypes = [
  { type: 'deal_created', label: 'Deal erstellt', category: 'deals' },
  { type: 'deal_stage_changed', label: 'Deal-Phase geändert', category: 'deals' },
  { type: 'deal_won', label: 'Deal gewonnen', category: 'deals' },
  { type: 'deal_lost', label: 'Deal verloren', category: 'deals' },
  { type: 'task_due', label: 'Task fällig', category: 'tasks' },
  { type: 'task_assigned', label: 'Task zugewiesen', category: 'tasks' },
  { type: 'chat_mention', label: 'Erwähnung im Chat', category: 'chat' },
  { type: 'chat_dm', label: 'Direktnachricht', category: 'chat' },
  { type: 'document_shared', label: 'Dokument geteilt', category: 'documents' },
  { type: 'invoice_paid', label: 'Rechnung bezahlt', category: 'finance' },
  { type: 'invoice_overdue', label: 'Rechnung überfällig', category: 'finance' },
  { type: 'leave_request', label: 'Urlaubsantrag', category: 'hr' },
  { type: 'security_alert', label: 'Sicherheitswarnung', category: 'security' },
  { type: 'project_update', label: 'Projekt-Update', category: 'tasks' },
]

// ---------------------------------------------------------------------------
// Mutable state — quiet hours, DND, mutes
// ---------------------------------------------------------------------------

let quietHoursState: QuietHours = {
  start_time: '22:00',
  end_time: '07:00',
  days: [0, 1, 2, 3, 4, 5, 6],
  timezone: 'Europe/Berlin',
  is_active: true,
}

let dndState: DNDStatus = {
  is_active: false,
}

const mutesState: MutedResource[] = []
let muteIdCounter = 1

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

export const notificationHandlers = [
  // List notifications (neueste zuerst)
  http.get(`${API}/api/v1/notifications`, ({ request }) => {
    const url = new URL(request.url)
    const unreadOnly = url.searchParams.get('unread') === 'true'
    const filtered = (unreadOnly ? notifications.filter((n) => !n.read) : notifications)
      .slice()
      .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
    return HttpResponse.json({ notifications: filtered, total: filtered.length })
  }),

  // Unread count
  http.get(`${API}/api/v1/notifications/unread-count`, () => {
    const count = notifications.filter((n) => !n.read).length
    return HttpResponse.json({ count })
  }),

  // Mark single as read
  http.post(`${API}/api/v1/notifications/:id/read`, ({ params }) => {
    const notif = notifications.find((n) => n.id === params.id)
    if (!notif) {
      return HttpResponse.json({ error: 'Notification not found' }, { status: 404 })
    }
    return HttpResponse.json({ success: true })
  }),

  // Mark all as read
  http.post(`${API}/api/v1/notifications/read-all`, () => {
    return HttpResponse.json({ success: true, updated: notifications.filter((n) => !n.read).length })
  }),

  // Notification preferences
  http.get(`${API}/api/v1/notifications/preferences`, () => {
    return HttpResponse.json({ preferences })
  }),

  // Event types
  http.get(`${API}/api/v1/notifications/event-types`, () => {
    return HttpResponse.json({ event_types: eventTypes })
  }),

  // Quiet hours — get
  http.get(`${API}/api/v1/notifications/quiet-hours`, () => {
    return HttpResponse.json({ quiet_hours: quietHoursState })
  }),

  // Quiet hours — update (stateful)
  http.put(`${API}/api/v1/notifications/quiet-hours`, async ({ request }) => {
    const patch = (await request.json().catch(() => ({}))) as Partial<QuietHours>
    quietHoursState = { ...quietHoursState, ...patch }
    return HttpResponse.json({ quiet_hours: quietHoursState })
  }),

  // DND — get status
  http.get(`${API}/api/v1/notifications/dnd`, () => {
    return HttpResponse.json(dndState)
  }),

  // DND — enable (stateful)
  http.post(`${API}/api/v1/notifications/dnd`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { expires_at?: string }
    dndState = { is_active: true, expires_at: body.expires_at }
    return HttpResponse.json(dndState)
  }),

  // DND — disable (stateful)
  http.delete(`${API}/api/v1/notifications/dnd`, () => {
    dndState = { is_active: false }
    return HttpResponse.json({})
  }),

  // Muted resources — list (stateful)
  http.get(`${API}/api/v1/notifications/mutes`, () => {
    return HttpResponse.json({ mutes: mutesState })
  }),

  // Muted resources — add (stateful)
  http.post(`${API}/api/v1/notifications/mutes`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as {
      resource_type: string
      resource_id: string
      resource_label?: string
    }
    const mute: MutedResource = {
      id: `mute-${muteIdCounter++}`,
      resource_type: body.resource_type,
      resource_id: body.resource_id,
      resource_label: body.resource_label,
      muted_at: new Date().toISOString(),
    }
    mutesState.push(mute)
    return HttpResponse.json({ mute })
  }),

  // Muted resources — remove (stateful)
  http.delete(`${API}/api/v1/notifications/mutes/:id`, ({ params }) => {
    const idx = mutesState.findIndex((m) => m.id === params.id)
    if (idx !== -1) mutesState.splice(idx, 1)
    return HttpResponse.json({})
  }),
]
