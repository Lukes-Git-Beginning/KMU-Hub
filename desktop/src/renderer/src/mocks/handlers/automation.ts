import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS } from '../data/shared-ids'
import { daysAgo, hoursAgo } from '../data/date-helpers'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Workflow data
// ---------------------------------------------------------------------------

const workflows = [
  {
    id: IDS.workflows.leadScoring,
    name: 'Lead Scoring',
    description: 'Automatische Bewertung neuer Leads basierend auf Unternehmensdaten und Interaktionen',
    status: 'active',
    trigger: { type: 'event', event: 'contact.created' },
    actions: [
      { type: 'enrich', config: { provider: 'clearbit' } },
      { type: 'score', config: { model: 'lead_default' } },
      { type: 'assign', config: { rule: 'round_robin', team: 'sales' } },
    ],
    execution_count: 145,
    last_executed_at: hoursAgo(2),
    created_by: IDS.users.thomas,
    created_at: daysAgo(45),
    updated_at: daysAgo(3),
  },
  {
    id: IDS.workflows.onboardingEmail,
    name: 'Onboarding E-Mail Serie',
    description: 'Willkommens-E-Mails für neue Kunden: Tag 0, Tag 3, Tag 7',
    status: 'active',
    trigger: { type: 'event', event: 'deal.won' },
    actions: [
      { type: 'email', config: { template: 'welcome', delay: 0 } },
      { type: 'email', config: { template: 'getting_started', delay: 259200 } },
      { type: 'email', config: { template: 'tips_tricks', delay: 604800 } },
    ],
    execution_count: 38,
    last_executed_at: daysAgo(1),
    created_by: IDS.users.julia,
    created_at: daysAgo(60),
    updated_at: daysAgo(10),
  },
  {
    id: IDS.workflows.ticketEskalation,
    name: 'Ticket Eskalation',
    description: 'Eskalation nach 24h ohne Antwort — benachrichtigt Teamleiter',
    status: 'active',
    trigger: { type: 'schedule', cron: '0 */1 * * *' },
    actions: [
      { type: 'query', config: { filter: 'tickets.unanswered_24h' } },
      { type: 'notify', config: { channel: 'slack', target: 'support_lead' } },
      { type: 'update', config: { field: 'priority', value: 'high' } },
    ],
    execution_count: 312,
    last_executed_at: hoursAgo(1),
    created_by: IDS.users.max,
    created_at: daysAgo(30),
    updated_at: daysAgo(5),
  },
  {
    id: IDS.workflows.rechnungsMahnung,
    name: 'Rechnungs-Mahnung',
    description: 'Automatische Zahlungserinnerung bei überfälligen Rechnungen',
    status: 'paused',
    trigger: { type: 'schedule', cron: '0 9 * * 1-5' },
    actions: [
      { type: 'query', config: { filter: 'invoices.overdue' } },
      { type: 'email', config: { template: 'payment_reminder' } },
      { type: 'log', config: { category: 'finance' } },
    ],
    execution_count: 67,
    last_executed_at: daysAgo(14),
    created_by: IDS.users.michael,
    created_at: daysAgo(90),
    updated_at: daysAgo(14),
  },
  {
    id: IDS.workflows.dailyReport,
    name: 'Taeglicher Report',
    description: 'Zusammenfassung aller Aktivitaeten per E-Mail an Geschaeftsführung',
    status: 'active',
    trigger: { type: 'schedule', cron: '0 18 * * 1-5' },
    actions: [
      { type: 'aggregate', config: { sources: ['deals', 'tasks', 'tickets'] } },
      { type: 'email', config: { template: 'daily_summary', recipients: ['management'] } },
    ],
    execution_count: 210,
    last_executed_at: hoursAgo(4),
    created_by: IDS.users.stefan,
    created_at: daysAgo(120),
    updated_at: daysAgo(2),
  },
  {
    id: IDS.workflows.backupCheck,
    name: 'Backup Prüfung',
    description: 'Taegliche Backup-Verifizierung — Alarm bei Fehler',
    status: 'active',
    trigger: { type: 'schedule', cron: '0 3 * * *' },
    actions: [
      { type: 'check', config: { target: 'backup_status' } },
      { type: 'notify', config: { channel: 'slack', target: 'ops', on_failure_only: true } },
    ],
    execution_count: 365,
    last_executed_at: hoursAgo(8),
    created_by: IDS.users.markus,
    created_at: daysAgo(180),
    updated_at: daysAgo(30),
  },
  {
    id: IDS.workflows.vertragsErinnerung,
    name: 'Vertrags-Erinnerung',
    description: '30 Tage vor Vertragsablauf automatisch benachrichtigen',
    status: 'draft',
    trigger: { type: 'schedule', cron: '0 9 * * 1' },
    actions: [
      { type: 'query', config: { filter: 'contracts.expiring_30d' } },
      { type: 'notify', config: { channel: 'email', target: 'account_owner' } },
    ],
    execution_count: 0,
    last_executed_at: null,
    created_by: IDS.users.thomas,
    created_at: daysAgo(5),
    updated_at: daysAgo(5),
  },
  {
    id: IDS.workflows.feedbackSammlung,
    name: 'Feedback Sammlung',
    description: 'NPS-Umfrage nach Projektabschluss automatisch versenden',
    status: 'draft',
    trigger: { type: 'event', event: 'project.completed' },
    actions: [
      { type: 'delay', config: { duration: 172800 } },
      { type: 'email', config: { template: 'nps_survey' } },
    ],
    execution_count: 0,
    last_executed_at: null,
    created_by: IDS.users.julia,
    created_at: daysAgo(3),
    updated_at: daysAgo(3),
  },
]

// ---------------------------------------------------------------------------
// Execution log entries
// ---------------------------------------------------------------------------

const executions = [
  { id: 'exec-001', workflow_id: IDS.workflows.leadScoring, status: 'success', started_at: hoursAgo(2), finished_at: hoursAgo(2), duration_ms: 340, input: { contact_id: IDS.contacts.schwarz } },
  { id: 'exec-002', workflow_id: IDS.workflows.ticketEskalation, status: 'success', started_at: hoursAgo(1), finished_at: hoursAgo(1), duration_ms: 1200, input: { tickets_found: 2 } },
  { id: 'exec-003', workflow_id: IDS.workflows.dailyReport, status: 'success', started_at: hoursAgo(4), finished_at: hoursAgo(4), duration_ms: 2800, input: {} },
  { id: 'exec-004', workflow_id: IDS.workflows.leadScoring, status: 'success', started_at: hoursAgo(6), finished_at: hoursAgo(6), duration_ms: 280, input: { contact_id: IDS.contacts.winkler } },
  { id: 'exec-005', workflow_id: IDS.workflows.backupCheck, status: 'success', started_at: hoursAgo(8), finished_at: hoursAgo(8), duration_ms: 5200, input: {} },
  { id: 'exec-006', workflow_id: IDS.workflows.onboardingEmail, status: 'success', started_at: daysAgo(1), finished_at: daysAgo(1), duration_ms: 920, input: { deal_id: IDS.deals.crmLizenz } },
  { id: 'exec-007', workflow_id: IDS.workflows.ticketEskalation, status: 'failure', started_at: daysAgo(2), finished_at: daysAgo(2), duration_ms: 150, input: {}, error: 'Slack API timeout' },
]

// ---------------------------------------------------------------------------
// Trigger & action type catalogs
// ---------------------------------------------------------------------------

const triggerTypes = [
  { type: 'event', label: 'Event-basiert', events: ['contact.created', 'contact.updated', 'deal.created', 'deal.won', 'deal.lost', 'project.completed', 'ticket.created', 'invoice.overdue'] },
  { type: 'schedule', label: 'Zeitgesteuert', description: 'Cron-basierter Zeitplan' },
  { type: 'webhook', label: 'Webhook', description: 'Externer HTTP-Trigger' },
  { type: 'manual', label: 'Manuell', description: 'Per Klick auslösen' },
]

const actionTypes = [
  { type: 'email', label: 'E-Mail senden', icon: 'mail' },
  { type: 'notify', label: 'Benachrichtigung', icon: 'bell' },
  { type: 'update', label: 'Datensatz aktualisieren', icon: 'edit' },
  { type: 'assign', label: 'Zuweisen', icon: 'user-plus' },
  { type: 'query', label: 'Daten abfragen', icon: 'search' },
  { type: 'delay', label: 'Verzoegerung', icon: 'clock' },
  { type: 'score', label: 'Scoring', icon: 'bar-chart' },
  { type: 'enrich', label: 'Daten anreichern', icon: 'database' },
  { type: 'aggregate', label: 'Aggregieren', icon: 'layers' },
  { type: 'check', label: 'Status prüfen', icon: 'check-circle' },
  { type: 'log', label: 'Protokollieren', icon: 'file-text' },
  { type: 'webhook', label: 'Webhook aufrufen', icon: 'globe' },
]

const templates = [
  { id: 'tpl-001', name: 'Lead Qualifizierung', description: 'Neue Leads automatisch bewerten und zuweisen', category: 'sales' },
  { id: 'tpl-002', name: 'Willkommens-Serie', description: 'E-Mail-Sequenz für neue Kunden', category: 'marketing' },
  { id: 'tpl-003', name: 'Ticket SLA Warnung', description: 'Eskalation bei SLA-Verletzung', category: 'support' },
  { id: 'tpl-004', name: 'Rechnungserinnerung', description: 'Automatische Mahnung bei überfälligen Rechnungen', category: 'finance' },
  { id: 'tpl-005', name: 'Geburtstags-Gruss', description: 'Automatische Geburtstagsnachricht an Kontakte', category: 'crm' },
]

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

export const automationHandlers = [
  // List workflows
  http.get(`${API}/api/v1/automations`, ({ request }) => {
    const url = new URL(request.url)
    const status = url.searchParams.get('status')
    let filtered = [...workflows]
    if (status) {
      filtered = filtered.filter((w) => w.status === status)
    }
    return HttpResponse.json({ workflows: filtered, total: filtered.length })
  }),

  // Workflow detail
  http.get(`${API}/api/v1/automations/:id`, ({ params }) => {
    const wf = workflows.find((w) => w.id === params.id)
    if (!wf) {
      return HttpResponse.json({ error: 'Workflow not found' }, { status: 404 })
    }
    return HttpResponse.json({ workflow: wf })
  }),

  // Create workflow
  http.post(`${API}/api/v1/automations`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const newWf = {
      id: `wf-${Date.now()}`,
      name: body.name || 'Neuer Workflow',
      description: body.description || '',
      status: 'draft',
      trigger: body.trigger || { type: 'manual' },
      actions: body.actions || [],
      execution_count: 0,
      last_executed_at: null,
      created_by: IDS.users.stefan,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    return HttpResponse.json({ workflow: newWf }, { status: 201 })
  }),

  // Update workflow
  http.put(`${API}/api/v1/automations/:id`, async ({ params, request }) => {
    const existing = workflows.find((w) => w.id === params.id)
    if (!existing) {
      return HttpResponse.json({ error: 'Workflow not found' }, { status: 404 })
    }
    const body = (await request.json()) as Record<string, unknown>
    const updated = { ...existing, ...body, updated_at: new Date().toISOString() }
    return HttpResponse.json({ workflow: updated })
  }),

  // Delete workflow
  http.delete(`${API}/api/v1/automations/:id`, ({ params }) => {
    const exists = workflows.some((w) => w.id === params.id)
    if (!exists) {
      return HttpResponse.json({ error: 'Workflow not found' }, { status: 404 })
    }
    return new HttpResponse(null, { status: 204 })
  }),

  // Execution log for a workflow
  http.get(`${API}/api/v1/automations/:id/executions`, ({ params }) => {
    const wfExecutions = executions.filter((e) => e.workflow_id === params.id)
    return HttpResponse.json({ executions: wfExecutions, total: wfExecutions.length })
  }),

  // Available trigger types
  http.get(`${API}/api/v1/automations/triggers`, () => {
    return HttpResponse.json({ triggers: triggerTypes })
  }),

  // Available action types
  http.get(`${API}/api/v1/automations/actions`, () => {
    return HttpResponse.json({ actions: actionTypes })
  }),

  // Workflow templates
  http.get(`${API}/api/v1/automations/templates`, () => {
    return HttpResponse.json({ templates })
  }),

  // Overview stats
  http.get(`${API}/api/v1/automations/stats`, () => {
    const active = workflows.filter((w) => w.status === 'active').length
    const paused = workflows.filter((w) => w.status === 'paused').length
    const draft = workflows.filter((w) => w.status === 'draft').length
    const totalExecutions = workflows.reduce((sum, w) => sum + w.execution_count, 0)
    const failedRecent = executions.filter((e) => e.status === 'failure').length
    return HttpResponse.json({
      active_workflows: active,
      paused_workflows: paused,
      draft_workflows: draft,
      total_executions: totalExecutions,
      failed_last_7d: failedRecent,
      success_rate: 95.2,
    })
  }),
]
