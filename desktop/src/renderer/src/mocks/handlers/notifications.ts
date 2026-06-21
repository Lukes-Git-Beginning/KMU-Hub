import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS } from '../data/shared-ids'
import { minutesAgo, hoursAgo, daysAgo } from '../data/date-helpers'
import type { QuietHours, DNDStatus, MutedResource } from '@/api/notification-client'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Notifications (13 recent)
//
// Schema mirrors `components['schemas']['Notification']` so the Center + Bell
// render unread state (is_read), priority colour (priority), the module badge
// (module_id) and the "Öffnen" deep-link (deep_link → a real app route).
// 6 of the 13 are unread → Bell badge + unread tab are alive on first load.
// ---------------------------------------------------------------------------

interface SeedNotification {
  id: string
  event_type_key: string
  module_id?: string
  priority: 'low' | 'normal' | 'high' | 'urgent'
  title: string
  body: string
  is_read: boolean
  read_at?: string
  actor_id: string | null
  actor_name: string
  resource_id: string | null
  deep_link?: string
  created_at: string
  /** When > 1, the center shows a "+N weitere" badge (grouped notifications). */
  group_count?: number
}

const notifications: SeedNotification[] = [
  {
    id: 'notif-001',
    event_type_key: 'deal_created',
    module_id: 'contacts',
    priority: 'normal',
    title: 'Neuer Deal erstellt',
    body: 'Thomas Meier hat den Deal "KI Beratung" (CHF 45\'000) erstellt.',
    is_read: false,
    actor_id: IDS.users.thomas,
    actor_name: 'Thomas Meier',
    resource_id: IDS.deals.kiBeratung,
    deep_link: '/pipeline',
    created_at: minutesAgo(15),
  },
  {
    id: 'notif-002',
    event_type_key: 'task_due',
    module_id: 'tasks',
    priority: 'high',
    title: 'Aufgabe fällig morgen',
    body: 'Sprint Review vorbereiten — fällig am morgigen Tag.',
    is_read: false,
    actor_id: null,
    actor_name: 'System',
    resource_id: 'task-001',
    deep_link: '/work/my-tasks',
    created_at: minutesAgo(45),
  },
  {
    id: 'notif-003',
    event_type_key: 'chat_mention',
    module_id: 'chat',
    priority: 'normal',
    title: 'Nachricht in #entwicklung',
    body: 'Markus Weber hat dich in #entwicklung erwähnt: "@Stefan schau dir mal den PR an"',
    is_read: false,
    actor_id: IDS.users.markus,
    actor_name: 'Markus Weber',
    resource_id: IDS.channels.entwicklung,
    deep_link: '/kommunikation',
    created_at: hoursAgo(1),
    group_count: 3,
  },
  {
    id: 'notif-004',
    event_type_key: 'leave_request',
    module_id: 'team',
    priority: 'normal',
    title: 'Urlaubsantrag eingegangen',
    body: 'Felix Krause beantragt Urlaub vom 09.04. bis 16.04.2026 (6 Tage).',
    is_read: true,
    read_at: hoursAgo(1),
    actor_id: IDS.users.felix,
    actor_name: 'Felix Krause',
    resource_id: 'lr-001',
    deep_link: '/team',
    created_at: hoursAgo(2),
  },
  {
    id: 'notif-005',
    event_type_key: 'document_shared',
    module_id: 'documents',
    priority: 'low',
    title: 'Dokument geteilt',
    body: 'Lena Braun hat "Brandbook_2026.pdf" mit dir geteilt.',
    is_read: true,
    read_at: hoursAgo(3),
    actor_id: IDS.users.lena,
    actor_name: 'Lena Braun',
    resource_id: 'file-010',
    deep_link: '/dokumente',
    created_at: hoursAgo(4),
  },
  {
    id: 'notif-006',
    event_type_key: 'deal_stage_changed',
    module_id: 'contacts',
    priority: 'high',
    title: 'Deal-Phase geändert',
    body: 'Deal "ERP Migration" wurde nach "Proposal" verschoben.',
    is_read: false,
    actor_id: IDS.users.thomas,
    actor_name: 'Thomas Meier',
    resource_id: IDS.deals.erpMigration,
    deep_link: '/pipeline',
    created_at: hoursAgo(8),
  },
  {
    id: 'notif-007',
    event_type_key: 'invoice_paid',
    module_id: 'finance',
    priority: 'normal',
    title: 'Rechnung bezahlt',
    body: 'Rechnung RE-2026-077 (EUR 12\'500) wurde als bezahlt markiert.',
    is_read: true,
    read_at: hoursAgo(20),
    actor_id: IDS.users.michael,
    actor_name: 'Petra Zimmermann',
    resource_id: IDS.invoices.inv007,
    deep_link: '/finanzen',
    created_at: daysAgo(1),
  },
  {
    id: 'notif-008',
    event_type_key: 'project_update',
    module_id: 'projects',
    priority: 'low',
    title: 'Projekt-Update',
    body: 'Sarah Beck hat das Projekt "Hub V2" aktualisiert — neuer Meilenstein hinzugefügt.',
    is_read: true,
    read_at: hoursAgo(22),
    actor_id: IDS.users.elena,
    actor_name: 'Sarah Beck',
    resource_id: IDS.projects.hubV2,
    deep_link: '/work/projects',
    created_at: daysAgo(1),
  },
  {
    id: 'notif-009',
    event_type_key: 'security_alert',
    module_id: 'settings',
    priority: 'urgent',
    title: 'Sicherheitswarnung',
    body: '3 fehlgeschlagene Login-Versuche von IP 203.0.113.88 blockiert.',
    is_read: false,
    actor_id: null,
    actor_name: 'System',
    resource_id: null,
    deep_link: '/admin/security',
    created_at: daysAgo(2),
  },
  {
    id: 'notif-010',
    event_type_key: 'welcome',
    module_id: undefined,
    priority: 'low',
    title: 'Willkommen bei Cosmi!',
    body: 'Ihr Arbeitsbereich ist eingerichtet. Erkunden Sie die Module im Seitenmenue.',
    is_read: true,
    read_at: daysAgo(89),
    actor_id: null,
    actor_name: 'System',
    resource_id: null,
    deep_link: '/',
    created_at: daysAgo(90),
  },
  // Fristen-Reminder (V-3) — gespiegelt zu den auslaufenden Seed-Verträgen
  // (v-3/v-5/v-11). Die Vertrag-Detailseite zeigt dieselben Reminder; der
  // zustand-Toast surfacet sie zusätzlich live beim Öffnen des Moduls.
  {
    id: 'notif-contract-v-3',
    event_type_key: 'contract_expiry',
    module_id: 'vertraege',
    priority: 'high',
    title: 'Vertrag läuft bald ab',
    body: 'Microsoft 365 Business (Microsoft Ireland Operations Ltd.) — Ablauf in 18 Tagen',
    is_read: false,
    actor_id: null,
    actor_name: 'System',
    resource_id: 'v-3',
    deep_link: '/vertraege',
    created_at: hoursAgo(3),
  },
  {
    id: 'notif-contract-v-5',
    event_type_key: 'contract_expiry',
    module_id: 'vertraege',
    priority: 'normal',
    title: 'Vertrag läuft bald ab',
    body: 'Müller Metallbau Rahmenvertrag (Müller Metallbau GmbH) — Ablauf in 47 Tagen',
    is_read: true,
    read_at: hoursAgo(12),
    actor_id: null,
    actor_name: 'System',
    resource_id: 'v-5',
    deep_link: '/vertraege',
    created_at: daysAgo(1),
  },
  {
    id: 'notif-contract-v-11',
    event_type_key: 'contract_expiry',
    module_id: 'vertraege',
    priority: 'low',
    title: 'Vertrag läuft bald ab',
    body: 'Lagerraum Augsburg (Immo-Invest Augsburg GmbH) — Ablauf in 82 Tagen',
    is_read: true,
    read_at: daysAgo(1),
    actor_id: null,
    actor_name: 'System',
    resource_id: 'v-11',
    deep_link: '/vertraege',
    created_at: daysAgo(2),
  },
]

// ---------------------------------------------------------------------------
// Live arrivals — notifications that "land" a few seconds into the session so
// the toast pipeline + bell badge demonstrate real-time delivery (analogous to
// the chat module). `flushArrivals` is called by every list/unread-count read
// and moves any due arrival into the live list, stamped with the current time.
// ---------------------------------------------------------------------------

const SERVER_BOOT = Date.now()

interface PendingArrival extends SeedNotification {
  reveal_after_ms: number
}

const pendingArrivals: PendingArrival[] = [
  {
    id: 'notif-live-1',
    event_type_key: 'chat_mention',
    module_id: 'chat',
    priority: 'high',
    title: 'Neue Erwähnung in #vertrieb',
    body: 'Julia Fischer hat dich erwähnt: "@Stefan kannst du das Angebot freigeben?"',
    is_read: false,
    actor_id: IDS.users.julia,
    actor_name: 'Julia Fischer',
    resource_id: IDS.channels.vertrieb,
    deep_link: '/kommunikation',
    created_at: minutesAgo(0),
    reveal_after_ms: 9000,
  },
  {
    id: 'notif-live-2',
    event_type_key: 'task_assigned',
    module_id: 'tasks',
    priority: 'normal',
    title: 'Aufgabe zugewiesen',
    body: 'Markus Weber hat dir "Q3-Reporting finalisieren" zugewiesen.',
    is_read: false,
    actor_id: IDS.users.markus,
    actor_name: 'Markus Weber',
    resource_id: 'task-042',
    deep_link: '/work/my-tasks',
    created_at: minutesAgo(0),
    reveal_after_ms: 18000,
  },
]

function flushArrivals(): void {
  const elapsed = Date.now() - SERVER_BOOT
  for (let i = pendingArrivals.length - 1; i >= 0; i--) {
    if (pendingArrivals[i].reveal_after_ms <= elapsed) {
      const { reveal_after_ms: _drop, ...notif } = pendingArrivals[i]
      void _drop
      notif.created_at = new Date().toISOString()
      notifications.unshift(notif)
      pendingArrivals.splice(i, 1)
    }
  }
}

// ---------------------------------------------------------------------------
// Notification preferences — per-event-type in-app/desktop toggles.
//
// Returned as an array of `NotificationPreference` (matching the hook + the
// Center's PreferencesPanel, which does `preferences.find(p => p.event_type_key)`).
// PUT upserts into this state → checkboxes survive a refetch within the session.
// ---------------------------------------------------------------------------

interface NotificationPreference {
  id: string
  event_type_key: string
  module_id?: string
  in_app: boolean
  desktop_push: boolean
  sound?: string
}

const preferencesState: NotificationPreference[] = [
  { id: 'pref-1', event_type_key: 'document_shared', module_id: 'documents', in_app: true, desktop_push: false },
  { id: 'pref-2', event_type_key: 'invoice_overdue', module_id: 'finance', in_app: true, desktop_push: true },
]
let prefIdCounter = 3

// Event types grouped by module — the PreferencesPanel groups by `module_id`
// and renders `display_name`/`description` + in-app/desktop checkboxes.
const eventTypes = [
  { id: 'et-01', module_id: 'contacts', event_key: 'deal_created', display_name: 'Deal erstellt', description: 'Wenn ein neuer Deal angelegt wird', default_in_app: true, default_desktop_push: true },
  { id: 'et-02', module_id: 'contacts', event_key: 'deal_stage_changed', display_name: 'Deal-Phase geändert', description: 'Wenn ein Deal die Phase wechselt', default_in_app: true, default_desktop_push: true },
  { id: 'et-03', module_id: 'contacts', event_key: 'deal_won', display_name: 'Deal gewonnen', description: 'Wenn ein Deal als gewonnen markiert wird', default_in_app: true, default_desktop_push: true },
  { id: 'et-04', module_id: 'tasks', event_key: 'task_due', display_name: 'Aufgabe fällig', description: 'Erinnerung an fällige Aufgaben', default_in_app: true, default_desktop_push: true },
  { id: 'et-05', module_id: 'tasks', event_key: 'task_assigned', display_name: 'Aufgabe zugewiesen', description: 'Wenn dir eine Aufgabe zugewiesen wird', default_in_app: true, default_desktop_push: false },
  { id: 'et-06', module_id: 'chat', event_key: 'chat_mention', display_name: 'Erwähnung im Chat', description: 'Wenn dich jemand im Chat erwähnt', default_in_app: true, default_desktop_push: true },
  { id: 'et-07', module_id: 'chat', event_key: 'chat_dm', display_name: 'Direktnachricht', description: 'Neue Direktnachricht', default_in_app: true, default_desktop_push: true },
  { id: 'et-08', module_id: 'documents', event_key: 'document_shared', display_name: 'Dokument geteilt', description: 'Wenn ein Dokument mit dir geteilt wird', default_in_app: true, default_desktop_push: false },
  { id: 'et-09', module_id: 'finance', event_key: 'invoice_paid', display_name: 'Rechnung bezahlt', description: 'Wenn eine Rechnung als bezahlt markiert wird', default_in_app: true, default_desktop_push: false },
  { id: 'et-10', module_id: 'finance', event_key: 'invoice_overdue', display_name: 'Rechnung überfällig', description: 'Wenn eine Rechnung überfällig ist', default_in_app: true, default_desktop_push: true },
  { id: 'et-11', module_id: 'team', event_key: 'leave_request', display_name: 'Urlaubsantrag', description: 'Neuer Urlaubsantrag im Team', default_in_app: true, default_desktop_push: false },
  { id: 'et-12', module_id: 'vertraege', event_key: 'contract_expiry', display_name: 'Vertrag läuft ab', description: 'Wenn ein Vertrag bald ausläuft', default_in_app: true, default_desktop_push: true },
  { id: 'et-13', module_id: 'projects', event_key: 'project_update', display_name: 'Projekt-Update', description: 'Updates zu deinen Projekten', default_in_app: true, default_desktop_push: false },
  { id: 'et-14', module_id: 'system', event_key: 'security_alert', display_name: 'Sicherheitswarnung', description: 'Sicherheitsrelevante Ereignisse', default_in_app: true, default_desktop_push: true },
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

// A few muted resources seeded so the "Stummgeschaltet"-Liste is non-empty and
// the unmute flow is demoable. resource_label is the human-readable name shown
// in settings; resource_type is localized there.
const mutesState: MutedResource[] = [
  {
    id: 'mute-seed-1',
    resource_type: 'channel',
    resource_id: IDS.channels.random,
    resource_label: '#random',
    muted_at: daysAgo(4),
  },
  {
    id: 'mute-seed-2',
    resource_type: 'conversation',
    resource_id: IDS.dms.stefanThomas,
    resource_label: 'Thomas Meier',
    muted_at: daysAgo(1),
  },
  {
    id: 'mute-seed-3',
    resource_type: 'thread',
    resource_id: 'thread-erp-angebot',
    resource_label: 'ERP Migration — Angebot',
    muted_at: hoursAgo(6),
  },
]
let muteIdCounter = 4

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

export const notificationHandlers = [
  // List notifications (neueste zuerst). Supports is_read + module_id filters
  // (sent by the hook) plus legacy `unread=true` and page/page_size paging.
  http.get(`${API}/api/v1/notifications`, ({ request }) => {
    flushArrivals()
    const url = new URL(request.url)
    const isReadParam = url.searchParams.get('is_read')
    const moduleId = url.searchParams.get('module_id')
    const unreadOnly = url.searchParams.get('unread') === 'true'

    let list = notifications.slice()
    if (moduleId) list = list.filter((n) => n.module_id === moduleId)
    if (isReadParam === 'true') list = list.filter((n) => n.is_read)
    else if (isReadParam === 'false') list = list.filter((n) => !n.is_read)
    if (unreadOnly) list = list.filter((n) => !n.is_read)

    list.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
    const total = list.length

    const page = Number(url.searchParams.get('page') ?? '1')
    const pageSize = Number(url.searchParams.get('page_size') ?? '0')
    const paged = pageSize > 0 ? list.slice((page - 1) * pageSize, page * pageSize) : list

    return HttpResponse.json({ notifications: paged, total })
  }),

  // Unread count
  http.get(`${API}/api/v1/notifications/unread-count`, () => {
    flushArrivals()
    const count = notifications.filter((n) => !n.is_read).length
    return HttpResponse.json({ count })
  }),

  // Mark single as read (stateful — survives refetch within the session)
  http.post(`${API}/api/v1/notifications/:id/read`, ({ params }) => {
    const notif = notifications.find((n) => n.id === params.id)
    if (!notif) {
      return HttpResponse.json({ error: 'Notification not found' }, { status: 404 })
    }
    notif.is_read = true
    notif.read_at = new Date().toISOString()
    return HttpResponse.json(notif)
  }),

  // Mark all as read (stateful), optionally filtered by module
  http.post(`${API}/api/v1/notifications/read-all`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { module_id?: string }
    const target = body.module_id
      ? notifications.filter((n) => n.module_id === body.module_id && !n.is_read)
      : notifications.filter((n) => !n.is_read)
    const now = new Date().toISOString()
    target.forEach((n) => {
      n.is_read = true
      n.read_at = now
    })
    return HttpResponse.json({ marked_count: target.length })
  }),

  // Notification preferences — list (per-event in-app/desktop toggles)
  http.get(`${API}/api/v1/notifications/preferences`, ({ request }) => {
    const moduleId = new URL(request.url).searchParams.get('module_id')
    const prefs = moduleId
      ? preferencesState.filter((p) => p.module_id === moduleId)
      : preferencesState
    return HttpResponse.json({ preferences: prefs })
  }),

  // Notification preferences — update (stateful upsert by event_type_key)
  http.put(`${API}/api/v1/notifications/preferences`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as {
      event_type_key?: string
      module_id?: string
      in_app?: boolean
      desktop_push?: boolean
      sound?: string
    }
    const key = body.event_type_key
    let pref = preferencesState.find((p) => p.event_type_key === key)
    if (pref) {
      if (body.in_app !== undefined) pref.in_app = body.in_app
      if (body.desktop_push !== undefined) pref.desktop_push = body.desktop_push
      if (body.sound !== undefined) pref.sound = body.sound
    } else {
      pref = {
        id: `pref-${prefIdCounter++}`,
        event_type_key: key ?? '',
        module_id: body.module_id,
        in_app: body.in_app ?? true,
        desktop_push: body.desktop_push ?? true,
        sound: body.sound,
      }
      preferencesState.push(pref)
    }
    return HttpResponse.json(pref)
  }),

  // Event types
  http.get(`${API}/api/v1/notifications/event-types`, ({ request }) => {
    const moduleId = new URL(request.url).searchParams.get('module_id')
    const types = moduleId ? eventTypes.filter((e) => e.module_id === moduleId) : eventTypes
    return HttpResponse.json({ event_types: types })
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
