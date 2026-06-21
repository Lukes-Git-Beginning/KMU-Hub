import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS } from '../data/shared-ids'
import { daysAgo, hoursAgo } from '../data/date-helpers'
import { mockActivities } from '../data/contacts'

const API = API_BASE_URL

// ============================================================================
// Types
// ============================================================================

type AgentDialerStatus = 'available' | 'on_call' | 'wrap_up' | 'break' | 'offline'

interface CampaignContact {
  id: string
  campaign_id: string
  contact_id: string
  position: number
  status: number // 1=pending, 2=in_progress, 3=completed, 4=skipped, 5=callback
  outcome_id: string | null
  notes: string | null
  callback_at: string | null
  last_called_at: string | null
  call_count: number
  contact_name: string
  contact_phone: string
  contact_company: string | null
}

interface Campaign {
  id: string
  tenant_id: string
  name: string
  description: string | null
  status: number // 1=draft, 2=active, 3=paused, 4=completed, 5=archived
  mode: number // 1=preview, 2=power, 3=predictive
  settings: string | null
  created_by: string
  assigned_agent_ids: string[]
  contact_count: number
  completed_count: number
  created_at: string
  updated_at: string
  started_at: string | null
  completed_at: string | null
}

interface CallOutcome {
  id: string
  tenant_id: string
  label: string
  color: string
  is_positive: boolean
  is_callback: boolean
  is_appointment: boolean
  sort_order: number
  is_active: boolean
}

// ============================================================================
// Mutable state
// ============================================================================

let agentStatus: AgentDialerStatus = 'available'
let activeCampaignId: string | null = IDS.dialer.kampagneKaltAkquise
let activeSessionId: string | null = null

const campaigns: Campaign[] = [
  {
    id: IDS.dialer.kampagneKaltAkquise,
    tenant_id: 'tenant-001',
    name: 'Kalt-Akquise Kampagne Q2/2026',
    description: 'Systematische Neukundengewinnung im DACH-Mittelstandssegment.',
    status: 2,
    mode: 1,
    settings: null,
    created_by: IDS.users.thomas,
    assigned_agent_ids: [IDS.users.thomas, IDS.users.laura, IDS.users.david],
    contact_count: 8,
    completed_count: 3,
    created_at: daysAgo(14) + 'T09:00:00Z',
    updated_at: daysAgo(1) + 'T14:30:00Z',
    started_at: daysAgo(7) + 'T08:00:00Z',
    completed_at: null,
  },
  {
    id: IDS.dialer.kampagneBestandskunden,
    tenant_id: 'tenant-001',
    name: 'Bestandskunden Follow-Up',
    description: 'Reaktivierung inaktiver Kunden — letzte Kontaktaufnahme > 6 Monate.',
    status: 1,
    mode: 1,
    settings: null,
    created_by: IDS.users.stefan,
    assigned_agent_ids: [],
    contact_count: 0,
    completed_count: 0,
    created_at: daysAgo(2) + 'T10:00:00Z',
    updated_at: daysAgo(2) + 'T10:00:00Z',
    started_at: null,
    completed_at: null,
  },
]

const campaignContacts: CampaignContact[] = [
  // 3 completed
  {
    id: IDS.dialer.cc001, campaign_id: IDS.dialer.kampagneKaltAkquise,
    contact_id: IDS.contacts.mueller, position: 1, status: 3,
    outcome_id: IDS.dialer.outcomeErreicht, notes: 'Interesse an CRM-Modul, Termin folgt.',
    callback_at: null, last_called_at: daysAgo(5) + 'T10:15:00Z', call_count: 1,
    contact_name: 'Hans Müller', contact_phone: '+49 89 4523 1101',
    contact_company: 'Gruber Maschinenbau AG',
  },
  {
    id: IDS.dialer.cc002, campaign_id: IDS.dialer.kampagneKaltAkquise,
    contact_id: IDS.contacts.weber, position: 2, status: 3,
    outcome_id: IDS.dialer.outcomeNichtErreicht, notes: 'Mailbox, 2x versucht.',
    callback_at: null, last_called_at: daysAgo(4) + 'T11:30:00Z', call_count: 2,
    contact_name: 'Claudia Weber', contact_phone: '+49 89 4523 1115',
    contact_company: 'Gruber Maschinenbau AG',
  },
  {
    id: IDS.dialer.cc003, campaign_id: IDS.dialer.kampagneKaltAkquise,
    contact_id: IDS.contacts.gruber, position: 3, status: 3,
    outcome_id: IDS.dialer.outcomeTermin, notes: 'Demo am Freitag vereinbart.',
    callback_at: null, last_called_at: daysAgo(3) + 'T14:00:00Z', call_count: 1,
    contact_name: 'Friedrich Gruber', contact_phone: '+49 89 4523 1100',
    contact_company: 'Gruber Maschinenbau AG',
  },
  // 5 pending
  {
    id: IDS.dialer.cc004, campaign_id: IDS.dialer.kampagneKaltAkquise,
    contact_id: IDS.contacts.schneider, position: 4, status: 1,
    outcome_id: null, notes: null, callback_at: null, last_called_at: null, call_count: 0,
    contact_name: 'Lukas Schneider', contact_phone: '+41 44 678 90 12',
    contact_company: 'Helvetia Software GmbH',
  },
  {
    id: IDS.dialer.cc005, campaign_id: IDS.dialer.kampagneKaltAkquise,
    contact_id: IDS.contacts.huber, position: 5, status: 1,
    outcome_id: null, notes: null, callback_at: null, last_called_at: null, call_count: 0,
    contact_name: 'Andrea Huber', contact_phone: '+41 44 678 90 20',
    contact_company: 'Helvetia Software GmbH',
  },
  {
    id: IDS.dialer.cc006, campaign_id: IDS.dialer.kampagneKaltAkquise,
    contact_id: IDS.contacts.bauer, position: 6, status: 1,
    outcome_id: null, notes: null, callback_at: null, last_called_at: null, call_count: 0,
    contact_name: 'Karl Bauer', contact_phone: '+43 1 512 78 10',
    contact_company: 'Alpen Logistik GmbH',
  },
  {
    id: IDS.dialer.cc007, campaign_id: IDS.dialer.kampagneKaltAkquise,
    contact_id: IDS.contacts.wagner, position: 7, status: 1,
    outcome_id: null, notes: null, callback_at: null, last_called_at: null, call_count: 0,
    contact_name: 'Sabine Wagner', contact_phone: '+43 1 512 78 22',
    contact_company: 'Alpen Logistik GmbH',
  },
  {
    id: IDS.dialer.cc008, campaign_id: IDS.dialer.kampagneKaltAkquise,
    contact_id: IDS.contacts.berger, position: 8, status: 1,
    outcome_id: null, notes: null, callback_at: null, last_called_at: null, call_count: 0,
    contact_name: 'Martin Berger', contact_phone: '+49 211 876 54 10',
    contact_company: 'Rhein Consulting Partners',
  },
]

const callOutcomes: CallOutcome[] = [
  { id: IDS.dialer.outcomeErreicht, tenant_id: 'tenant-001', label: 'Erreicht', color: '#22c55e', is_positive: true, is_callback: false, is_appointment: false, sort_order: 0, is_active: true },
  { id: IDS.dialer.outcomeNichtErreicht, tenant_id: 'tenant-001', label: 'Nicht erreicht', color: '#ef4444', is_positive: false, is_callback: false, is_appointment: false, sort_order: 1, is_active: true },
  { id: IDS.dialer.outcomeWiedervorlage, tenant_id: 'tenant-001', label: 'Wiedervorlage', color: '#f59e0b', is_positive: false, is_callback: true, is_appointment: false, sort_order: 2, is_active: true },
  { id: IDS.dialer.outcomeTermin, tenant_id: 'tenant-001', label: 'Termin vereinbart', color: '#3b82f6', is_positive: true, is_callback: false, is_appointment: true, sort_order: 3, is_active: true },
]

const callSessions: Array<{
  id: string
  campaign_contact_id: string
  call_session_id: string | null
  agent_id: string
  outcome_id: string | null
  duration_seconds: number | null
  notes: string | null
  next_action: string | null
  appointment_id: string | null
  created_at: string
  wrap_up_completed_at: string | null
}> = []

// ============================================================================
// Helpers
// ============================================================================

function agentStatusToInt(s: AgentDialerStatus): number {
  const map: Record<AgentDialerStatus, number> = { available: 1, on_call: 2, wrap_up: 3, break: 4, offline: 5 }
  return map[s]
}

function intToAgentStatus(n: number): AgentDialerStatus {
  const map: Record<number, AgentDialerStatus> = { 1: 'available', 2: 'on_call', 3: 'wrap_up', 4: 'break', 5: 'offline' }
  return map[n] ?? 'offline'
}

function paginate<T>(items: T[], page: number, pageSize: number): { items: T[]; total: number } {
  const start = (page - 1) * pageSize
  return { items: items.slice(start, start + pageSize), total: items.length }
}

function formatDuration(totalSeconds: number | null): string {
  const s = totalSeconds ?? 0
  const mm = Math.floor(s / 60)
  const ss = String(s % 60).padStart(2, '0')
  return `${mm}:${ss}`
}

/**
 * CRM bridge (D-1): write a "call" activity onto the contact so the dialer
 * outcome shows up in the CRM contact timeline (GET /crm/contacts/:id/timeline).
 * Mirrors the seeded activity shape in mocks/data/contacts.ts.
 */
function writeCallActivity(
  cc: CampaignContact,
  outcome: CallOutcome | undefined,
  durationSeconds: number | null,
  agentId: string,
  notes?: string | null,
): void {
  const parts = [
    `Ergebnis: ${outcome?.label ?? 'Ohne Ergebnis'}`,
    `Dauer ${formatDuration(durationSeconds)}`,
  ]
  if (notes) parts.push(notes)

  mockActivities.activities.unshift({
    id: `act-dlr-${Date.now()}-${cc.id}`,
    type: 'call' as const,
    subject: `Telefonanruf — ${cc.contact_name}`,
    description: parts.join(' · '),
    contact_id: cc.contact_id,
    user_id: agentId,
    completed: true,
    due_date: new Date().toISOString(),
    created_at: new Date().toISOString(),
  } as unknown as (typeof mockActivities.activities)[number])
}

// ============================================================================
// Handlers
// ============================================================================

export const dialerHandlers = [
  // --- Campaigns ---

  http.get(`${API}/api/v1/dialer/campaigns`, ({ request }) => {
    const url = new URL(request.url)
    const page = Number(url.searchParams.get('page') ?? 1)
    const pageSize = Number(url.searchParams.get('page_size') ?? 20)
    const statusFilter = url.searchParams.get('status_filter')

    let filtered = campaigns
    if (statusFilter) {
      const sf = Number(statusFilter)
      filtered = campaigns.filter(c => c.status === sf)
    }

    const result = paginate(filtered, page, pageSize)
    return HttpResponse.json({ campaigns: result.items, total: result.total })
  }),

  http.post(`${API}/api/v1/dialer/campaigns`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const newCampaign: Campaign = {
      id: `dlr-camp-${Date.now()}`,
      tenant_id: 'tenant-001',
      name: (body.name as string) ?? '',
      description: (body.description as string) ?? null,
      status: 1,
      mode: (body.mode as number) ?? 1,
      settings: (body.settings as string) ?? null,
      created_by: IDS.users.stefan,
      assigned_agent_ids: (body.assigned_agent_ids as string[]) ?? [],
      contact_count: 0,
      completed_count: 0,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      started_at: null,
      completed_at: null,
    }
    campaigns.push(newCampaign)
    return HttpResponse.json(newCampaign, { status: 201 })
  }),

  http.get(`${API}/api/v1/dialer/campaigns/:id`, ({ params }) => {
    const campaign = campaigns.find(c => c.id === params.id)
    if (!campaign) return HttpResponse.json({ error: 'campaign not found' }, { status: 404 })
    return HttpResponse.json(campaign)
  }),

  http.put(`${API}/api/v1/dialer/campaigns/:id`, async ({ params, request }) => {
    const campaign = campaigns.find(c => c.id === params.id)
    if (!campaign) return HttpResponse.json({ error: 'campaign not found' }, { status: 404 })
    const body = (await request.json()) as Record<string, unknown>
    if (body.name != null) campaign.name = body.name as string
    if (body.description != null) campaign.description = body.description as string
    if (body.assigned_agent_ids != null) campaign.assigned_agent_ids = body.assigned_agent_ids as string[]
    if (body.settings != null) campaign.settings = body.settings as string
    campaign.updated_at = new Date().toISOString()
    return HttpResponse.json(campaign)
  }),

  http.post(`${API}/api/v1/dialer/campaigns/:id/start`, ({ params }) => {
    const campaign = campaigns.find(c => c.id === params.id)
    if (!campaign) return HttpResponse.json({ error: 'campaign not found' }, { status: 404 })
    campaign.status = 2
    campaign.started_at = campaign.started_at ?? new Date().toISOString()
    campaign.updated_at = new Date().toISOString()
    return HttpResponse.json(campaign)
  }),

  http.post(`${API}/api/v1/dialer/campaigns/:id/pause`, ({ params }) => {
    const campaign = campaigns.find(c => c.id === params.id)
    if (!campaign) return HttpResponse.json({ error: 'campaign not found' }, { status: 404 })
    campaign.status = 3
    campaign.updated_at = new Date().toISOString()
    return HttpResponse.json(campaign)
  }),

  http.delete(`${API}/api/v1/dialer/campaigns/:id`, ({ params }) => {
    const campaign = campaigns.find(c => c.id === params.id)
    if (!campaign) return HttpResponse.json({ error: 'campaign not found' }, { status: 404 })
    campaign.status = 5
    campaign.updated_at = new Date().toISOString()
    return HttpResponse.json({ status: 'campaign archived' })
  }),

  // --- Campaign Contacts ---

  http.post(`${API}/api/v1/dialer/campaigns/:id/contacts`, async ({ params, request }) => {
    const body = (await request.json()) as { contact_ids?: string[]; filter_id?: string }
    const campaignId = params.id as string
    const campaign = campaigns.find(c => c.id === campaignId)
    if (!campaign) return HttpResponse.json({ error: 'campaign not found' }, { status: 404 })

    const contactIds = body.contact_ids ?? []
    let addedCount = 0
    let skippedCount = 0

    for (const cid of contactIds) {
      const exists = campaignContacts.some(cc => cc.campaign_id === campaignId && cc.contact_id === cid)
      if (exists) {
        skippedCount++
        continue
      }
      campaignContacts.push({
        id: `dlr-cc-${Date.now()}-${addedCount}`,
        campaign_id: campaignId,
        contact_id: cid,
        position: campaign.contact_count + addedCount + 1,
        status: 1,
        outcome_id: null,
        notes: null,
        callback_at: null,
        last_called_at: null,
        call_count: 0,
        contact_name: 'Neuer Kontakt',
        contact_phone: '+49 000 000000',
        contact_company: null,
      })
      addedCount++
    }

    campaign.contact_count += addedCount
    campaign.updated_at = new Date().toISOString()
    return HttpResponse.json({ added_count: addedCount, skipped_count: skippedCount })
  }),

  http.get(`${API}/api/v1/dialer/campaigns/:id/contacts`, ({ params, request }) => {
    const url = new URL(request.url)
    const page = Number(url.searchParams.get('page') ?? 1)
    const pageSize = Number(url.searchParams.get('page_size') ?? 20)
    const statusFilter = url.searchParams.get('status_filter')

    let contacts = campaignContacts.filter(cc => cc.campaign_id === params.id)
    if (statusFilter) {
      const sf = Number(statusFilter)
      contacts = contacts.filter(cc => cc.status === sf)
    }

    const result = paginate(contacts, page, pageSize)
    return HttpResponse.json({ contacts: result.items, total: result.total })
  }),

  http.get(`${API}/api/v1/dialer/campaigns/:id/next`, ({ params }) => {
    const next = campaignContacts.find(
      cc => cc.campaign_id === params.id && cc.status === 1
    )
    if (!next) return HttpResponse.json({ error: 'no pending contacts' }, { status: 404 })
    next.status = 2 // IN_PROGRESS
    return HttpResponse.json(next)
  }),

  http.post(`${API}/api/v1/dialer/campaigns/:id/contacts/:cid/skip`, ({ params }) => {
    const contact = campaignContacts.find(cc => cc.id === params.cid)
    if (!contact) return HttpResponse.json({ error: 'contact not found' }, { status: 404 })
    contact.status = 4 // SKIPPED
    return HttpResponse.json({ status: 'contact skipped' })
  }),

  http.post(`${API}/api/v1/dialer/campaigns/:id/contacts/:cid/requeue`, ({ params }) => {
    const contact = campaignContacts.find(cc => cc.id === params.cid)
    if (!contact) return HttpResponse.json({ error: 'contact not found' }, { status: 404 })
    contact.status = 1 // PENDING
    contact.outcome_id = null
    return HttpResponse.json({ status: 'contact requeued' })
  }),

  // --- Calls ---

  http.post(`${API}/api/v1/dialer/calls`, async ({ request }) => {
    const body = (await request.json()) as { campaign_contact_id: string; agent_id?: string; call_session_id?: string }
    agentStatus = 'on_call'

    const session = {
      id: `dlr-sess-${Date.now()}`,
      campaign_contact_id: body.campaign_contact_id,
      call_session_id: body.call_session_id ?? null,
      agent_id: body.agent_id ?? IDS.users.thomas,
      outcome_id: null,
      duration_seconds: null,
      notes: null,
      next_action: null,
      appointment_id: null,
      created_at: new Date().toISOString(),
      wrap_up_completed_at: null,
    }
    activeSessionId = session.id
    callSessions.push(session)

    // Update the campaign contact's call count
    const cc = campaignContacts.find(c => c.id === body.campaign_contact_id)
    if (cc) {
      cc.call_count++
      cc.last_called_at = new Date().toISOString()
    }

    return HttpResponse.json(session, { status: 201 })
  }),

  http.post(`${API}/api/v1/dialer/calls/:id/outcome`, async ({ params, request }) => {
    const body = (await request.json()) as {
      outcome_id: string
      notes?: string
      next_action?: string
      callback_at?: string
      appointment_id?: string
    }
    agentStatus = 'wrap_up'

    const session = callSessions.find(s => s.id === params.id)
    if (session) {
      session.outcome_id = body.outcome_id
      if (body.notes) session.notes = body.notes
      if (body.next_action) session.next_action = body.next_action
      if (body.appointment_id) session.appointment_id = body.appointment_id
      session.duration_seconds = Math.floor(Math.random() * 240) + 30

      // Mark campaign contact as completed
      const cc = campaignContacts.find(c => c.id === session.campaign_contact_id)
      if (cc) {
        const outcome = callOutcomes.find(o => o.id === body.outcome_id)
        if (outcome?.is_callback && body.callback_at) {
          cc.status = 5 // CALLBACK
          cc.callback_at = body.callback_at
        } else {
          cc.status = 3 // COMPLETED
        }
        cc.outcome_id = body.outcome_id
        if (body.notes) cc.notes = body.notes

        // CRM bridge (D-1): log the call onto the contact's timeline.
        writeCallActivity(cc, outcome, session.duration_seconds, session.agent_id, body.notes)

        // Update campaign completed_count
        const campaign = campaigns.find(camp => camp.id === cc.campaign_id)
        if (campaign) {
          campaign.completed_count = campaignContacts.filter(
            c => c.campaign_id === campaign.id && c.status === 3
          ).length
          campaign.updated_at = new Date().toISOString()
        }
      }
    }

    return HttpResponse.json(session ?? { id: params.id, outcome_id: body.outcome_id })
  }),

  http.put(`${API}/api/v1/dialer/calls/:id/notes`, async ({ params, request }) => {
    const body = (await request.json()) as { notes: string }
    const session = callSessions.find(s => s.id === params.id)
    if (session) session.notes = body.notes
    return HttpResponse.json({ status: 'notes saved' })
  }),

  http.post(`${API}/api/v1/dialer/calls/:id/complete`, () => {
    agentStatus = 'available'

    const session = callSessions.find(s => s.id === activeSessionId)
    if (session) session.wrap_up_completed_at = new Date().toISOString()
    activeSessionId = null

    return HttpResponse.json({ status: 'wrap-up completed' })
  }),

  // --- Agent Status ---

  http.get(`${API}/api/v1/dialer/agent-status`, () => {
    return HttpResponse.json({
      user_id: IDS.users.thomas,
      status: agentStatusToInt(agentStatus),
      campaign_id: activeCampaignId,
      since: hoursAgo(2),
      first_name: 'Thomas',
      last_name: 'Meier',
    })
  }),

  http.put(`${API}/api/v1/dialer/agent-status`, async ({ request }) => {
    const body = (await request.json()) as { user_id?: string; status: number; campaign_id?: string }
    agentStatus = intToAgentStatus(body.status)
    if (body.campaign_id !== undefined) activeCampaignId = body.campaign_id
    return HttpResponse.json({
      user_id: body.user_id ?? IDS.users.thomas,
      status: body.status,
      campaign_id: activeCampaignId,
      since: new Date().toISOString(),
      first_name: 'Thomas',
      last_name: 'Meier',
    })
  }),

  http.get(`${API}/api/v1/dialer/agent-status/campaign/:campaignId`, () => {
    return HttpResponse.json({
      agents: [
        { user_id: IDS.users.thomas, status: agentStatusToInt(agentStatus), campaign_id: activeCampaignId, since: hoursAgo(2), first_name: 'Thomas', last_name: 'Meier' },
        { user_id: IDS.users.laura, status: 1, campaign_id: activeCampaignId, since: hoursAgo(1), first_name: 'Sabine', last_name: 'Fischer' },
        { user_id: IDS.users.david, status: 5, campaign_id: null, since: daysAgo(1) + 'T18:00:00Z', first_name: 'Kevin', last_name: 'Baumann' },
      ],
    })
  }),

  // --- Outcomes ---

  http.get(`${API}/api/v1/dialer/outcomes`, ({ request }) => {
    const url = new URL(request.url)
    const includeInactive = url.searchParams.get('include_inactive') === 'true'
    const filtered = includeInactive ? callOutcomes : callOutcomes.filter(o => o.is_active)
    return HttpResponse.json({ outcomes: filtered })
  }),

  http.post(`${API}/api/v1/dialer/outcomes`, async ({ request }) => {
    const body = (await request.json()) as Omit<CallOutcome, 'id' | 'tenant_id' | 'is_active'>
    const outcome: CallOutcome = {
      id: `dlr-out-${Date.now()}`,
      tenant_id: 'tenant-001',
      is_active: true,
      ...body,
    }
    callOutcomes.push(outcome)
    return HttpResponse.json(outcome, { status: 201 })
  }),

  http.put(`${API}/api/v1/dialer/outcomes/:id`, async ({ params, request }) => {
    const outcome = callOutcomes.find(o => o.id === params.id)
    if (!outcome) return HttpResponse.json({ error: 'outcome not found' }, { status: 404 })
    const body = (await request.json()) as Partial<CallOutcome>
    Object.assign(outcome, body)
    return HttpResponse.json(outcome)
  }),

  http.delete(`${API}/api/v1/dialer/outcomes/:id`, ({ params }) => {
    const outcome = callOutcomes.find(o => o.id === params.id)
    if (!outcome) return HttpResponse.json({ error: 'outcome not found' }, { status: 404 })
    outcome.is_active = false
    return HttpResponse.json({ status: 'outcome deleted' })
  }),

  // --- Dashboards ---

  http.get(`${API}/api/v1/dialer/campaigns/:id/dashboard`, ({ params }) => {
    const contacts = campaignContacts.filter(c => c.campaign_id === params.id)
    const completed = contacts.filter(c => c.status === 3)
    const pending = contacts.filter(c => c.status === 1)
    const skipped = contacts.filter(c => c.status === 4)
    const callbacks = contacts.filter(c => c.status === 5)

    const outcomeBreakdown = callOutcomes
      .map(o => ({
        outcome_id: o.id,
        outcome_label: o.label,
        outcome_color: o.color,
        count: completed.filter(c => c.outcome_id === o.id).length,
      }))
      .filter(o => o.count > 0)

    return HttpResponse.json({
      campaign_id: params.id,
      total_contacts: contacts.length,
      completed_contacts: completed.length,
      pending_contacts: pending.length,
      skipped_contacts: skipped.length,
      callback_contacts: callbacks.length,
      total_calls: completed.length + skipped.length,
      positive_calls: completed.filter(c => {
        const o = callOutcomes.find(o => o.id === c.outcome_id)
        return o?.is_positive
      }).length,
      average_duration_seconds: 187.4,
      calls_per_hour: 12.3,
      outcome_breakdown: outcomeBreakdown,
    })
  }),

  http.get(`${API}/api/v1/dialer/dashboard/agent`, () => {
    const todaySessions = callSessions.filter(s => {
      const d = new Date(s.created_at)
      const today = new Date()
      return d.toDateString() === today.toDateString()
    })

    const outcomeBreakdown = callOutcomes
      .map(o => ({
        outcome_id: o.id,
        outcome_label: o.label,
        outcome_color: o.color,
        count: todaySessions.filter(s => s.outcome_id === o.id).length,
      }))
      .filter(o => o.count > 0)

    // If no sessions today, show some default data
    const totalCalls = todaySessions.length || 7
    const breakdown = outcomeBreakdown.length > 0 ? outcomeBreakdown : [
      { outcome_id: IDS.dialer.outcomeErreicht, outcome_label: 'Erreicht', outcome_color: '#22c55e', count: 3 },
      { outcome_id: IDS.dialer.outcomeNichtErreicht, outcome_label: 'Nicht erreicht', outcome_color: '#ef4444', count: 2 },
      { outcome_id: IDS.dialer.outcomeTermin, outcome_label: 'Termin vereinbart', outcome_color: '#3b82f6', count: 2 },
    ]

    const activeCamp = campaigns.find(c => c.id === activeCampaignId)

    return HttpResponse.json({
      agent_id: IDS.users.thomas,
      total_calls_today: totalCalls,
      average_duration_seconds: 203.1,
      calls_per_hour: 11.2,
      outcome_breakdown: breakdown,
      active_campaign_id: activeCampaignId,
      active_campaign_name: activeCamp?.name ?? null,
    })
  }),
]
