import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS } from '../data/shared-ids'
import { daysAgo, hoursAgo } from '../data/date-helpers'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Helpers — shape matches api/automation-types.ts exactly
// ---------------------------------------------------------------------------

const EMPTY_CONDITIONS = { mode: 'simple' as const, simple: null, expression: '' }

/** Wrap a loose action list into ActionConfig[] (adds the on_error default). */
function actionList(actions: { type: string; config: Record<string, unknown> }[]) {
  return actions.map((a) => ({ type: a.type, config: a.config, on_error: 'continue' as const }))
}

// ---------------------------------------------------------------------------
// Automations — stateful (create/delete/enable/disable persist for the session)
// matches the `Automation` type so the list/table/detail actually render.
// ---------------------------------------------------------------------------

let automations = [
  {
    id: IDS.workflows.leadScoring,
    name: 'Lead Scoring',
    description: 'Automatische Bewertung neuer Leads basierend auf Unternehmensdaten und Interaktionen',
    scope: 'organization',
    owner_id: IDS.users.thomas,
    trigger_type: 'event',
    trigger_config: { event_type: 'contact.created' },
    conditions: EMPTY_CONDITIONS,
    actions: actionList([
      { type: 'enrich', config: { provider: 'clearbit' } },
      { type: 'score', config: { model: 'lead_default' } },
      { type: 'assign', config: { rule: 'round_robin', team: 'sales' } },
    ]),
    is_active: true,
    max_steps: 10,
    template_id: null,
    last_triggered_at: hoursAgo(2),
    created_at: daysAgo(45),
    updated_at: daysAgo(3),
  },
  {
    id: IDS.workflows.onboardingEmail,
    name: 'Onboarding E-Mail Serie',
    description: 'Willkommens-E-Mails für neue Kunden: Tag 0, Tag 3, Tag 7',
    scope: 'team',
    owner_id: IDS.users.julia,
    trigger_type: 'event',
    trigger_config: { event_type: 'deal.won' },
    conditions: EMPTY_CONDITIONS,
    actions: actionList([
      { type: 'email', config: { template: 'welcome', delay: 0 } },
      { type: 'email', config: { template: 'getting_started', delay: 259200 } },
      { type: 'email', config: { template: 'tips_tricks', delay: 604800 } },
    ]),
    is_active: true,
    max_steps: 10,
    template_id: null,
    last_triggered_at: daysAgo(1),
    created_at: daysAgo(60),
    updated_at: daysAgo(10),
  },
  {
    id: IDS.workflows.ticketEskalation,
    name: 'Ticket Eskalation',
    description: 'Eskalation nach 24h ohne Antwort — benachrichtigt Teamleiter',
    scope: 'organization',
    owner_id: IDS.users.max,
    trigger_type: 'schedule',
    trigger_config: { cron: '0 */1 * * *' },
    conditions: EMPTY_CONDITIONS,
    actions: actionList([
      { type: 'query', config: { filter: 'tickets.unanswered_24h' } },
      { type: 'notify', config: { channel: 'slack', target: 'support_lead' } },
      { type: 'update', config: { field: 'priority', value: 'high' } },
    ]),
    is_active: true,
    max_steps: 10,
    template_id: null,
    last_triggered_at: hoursAgo(1),
    created_at: daysAgo(30),
    updated_at: daysAgo(5),
  },
  {
    id: IDS.workflows.rechnungsMahnung,
    name: 'Rechnungs-Mahnung',
    description: 'Automatische Zahlungserinnerung bei überfälligen Rechnungen',
    scope: 'organization',
    owner_id: IDS.users.michael,
    trigger_type: 'schedule',
    trigger_config: { cron: '0 9 * * 1-5' },
    conditions: EMPTY_CONDITIONS,
    actions: actionList([
      { type: 'query', config: { filter: 'invoices.overdue' } },
      { type: 'email', config: { template: 'payment_reminder' } },
      { type: 'log', config: { category: 'finance' } },
    ]),
    is_active: false,
    max_steps: 10,
    template_id: null,
    last_triggered_at: daysAgo(14),
    created_at: daysAgo(90),
    updated_at: daysAgo(14),
  },
  {
    id: IDS.workflows.dailyReport,
    name: 'Täglicher Report',
    description: 'Zusammenfassung aller Aktivitäten per E-Mail an die Geschäftsführung',
    scope: 'organization',
    owner_id: IDS.users.stefan,
    trigger_type: 'schedule',
    trigger_config: { cron: '0 18 * * 1-5' },
    conditions: EMPTY_CONDITIONS,
    actions: actionList([
      { type: 'aggregate', config: { sources: ['deals', 'tasks', 'tickets'] } },
      { type: 'email', config: { template: 'daily_summary', recipients: ['management'] } },
    ]),
    is_active: true,
    max_steps: 10,
    template_id: null,
    last_triggered_at: hoursAgo(4),
    created_at: daysAgo(120),
    updated_at: daysAgo(2),
  },
  {
    id: IDS.workflows.backupCheck,
    name: 'Backup Prüfung',
    description: 'Tägliche Backup-Verifizierung — Alarm bei Fehler',
    scope: 'organization',
    owner_id: IDS.users.markus,
    trigger_type: 'schedule',
    trigger_config: { cron: '0 3 * * *' },
    conditions: EMPTY_CONDITIONS,
    actions: actionList([
      { type: 'check', config: { target: 'backup_status' } },
      { type: 'notify', config: { channel: 'slack', target: 'ops', on_failure_only: true } },
    ]),
    is_active: true,
    max_steps: 10,
    template_id: null,
    last_triggered_at: hoursAgo(8),
    created_at: daysAgo(180),
    updated_at: daysAgo(30),
  },
  {
    id: IDS.workflows.vertragsErinnerung,
    name: 'Vertrags-Erinnerung',
    description: '30 Tage vor Vertragsablauf automatisch benachrichtigen',
    scope: 'team',
    owner_id: IDS.users.thomas,
    trigger_type: 'schedule',
    trigger_config: { cron: '0 9 * * 1' },
    conditions: EMPTY_CONDITIONS,
    actions: actionList([
      { type: 'query', config: { filter: 'contracts.expiring_30d' } },
      { type: 'notify', config: { channel: 'email', target: 'account_owner' } },
    ]),
    is_active: false,
    max_steps: 10,
    template_id: null,
    last_triggered_at: null,
    created_at: daysAgo(5),
    updated_at: daysAgo(5),
  },
  {
    id: IDS.workflows.feedbackSammlung,
    name: 'Feedback Sammlung',
    description: 'NPS-Umfrage nach Projektabschluss automatisch versenden',
    scope: 'personal',
    owner_id: IDS.users.julia,
    trigger_type: 'event',
    trigger_config: { event_type: 'project.completed' },
    conditions: EMPTY_CONDITIONS,
    actions: actionList([
      { type: 'delay', config: { duration: 172800 } },
      { type: 'email', config: { template: 'nps_survey' } },
    ]),
    is_active: false,
    max_steps: 10,
    template_id: null,
    last_triggered_at: null,
    created_at: daysAgo(3),
    updated_at: daysAgo(3),
  },
]

// ---------------------------------------------------------------------------
// Execution log — matches `AutomationExecution` (steps, condition_result, …)
// ---------------------------------------------------------------------------

const step = (action_type: string, duration_ms: number, error = '') => ({
  action_type,
  input: {},
  output: error ? {} : { ok: true },
  error,
  duration_ms,
})

const executions = [
  { id: 'exec-001', automation_id: IDS.workflows.leadScoring, chain_id: 'chain-001', trigger_event: { contact_id: IDS.contacts.schwarz }, condition_result: true, status: 'completed', steps: [step('enrich', 120), step('score', 90), step('assign', 130)], error_message: '', started_at: hoursAgo(2), completed_at: hoursAgo(2), duration_ms: 340 },
  { id: 'exec-002', automation_id: IDS.workflows.ticketEskalation, chain_id: 'chain-002', trigger_event: { tickets_found: 2 }, condition_result: true, status: 'completed', steps: [step('query', 400), step('notify', 500), step('update', 300)], error_message: '', started_at: hoursAgo(1), completed_at: hoursAgo(1), duration_ms: 1200 },
  { id: 'exec-003', automation_id: IDS.workflows.dailyReport, chain_id: 'chain-003', trigger_event: {}, condition_result: true, status: 'completed', steps: [step('aggregate', 1800), step('email', 1000)], error_message: '', started_at: hoursAgo(4), completed_at: hoursAgo(4), duration_ms: 2800 },
  { id: 'exec-004', automation_id: IDS.workflows.leadScoring, chain_id: 'chain-004', trigger_event: { contact_id: IDS.contacts.winkler }, condition_result: true, status: 'completed', steps: [step('enrich', 100), step('score', 80), step('assign', 100)], error_message: '', started_at: hoursAgo(6), completed_at: hoursAgo(6), duration_ms: 280 },
  { id: 'exec-005', automation_id: IDS.workflows.backupCheck, chain_id: 'chain-005', trigger_event: {}, condition_result: true, status: 'completed', steps: [step('check', 5000), step('notify', 200)], error_message: '', started_at: hoursAgo(8), completed_at: hoursAgo(8), duration_ms: 5200 },
  { id: 'exec-006', automation_id: IDS.workflows.onboardingEmail, chain_id: 'chain-006', trigger_event: { deal_id: IDS.deals.crmLizenz }, condition_result: true, status: 'completed', steps: [step('email', 920)], error_message: '', started_at: daysAgo(1), completed_at: daysAgo(1), duration_ms: 920 },
  { id: 'exec-007', automation_id: IDS.workflows.ticketEskalation, chain_id: 'chain-007', trigger_event: {}, condition_result: true, status: 'failed', steps: [step('query', 100), step('notify', 50, 'Slack API timeout')], error_message: 'Slack API timeout', started_at: daysAgo(2), completed_at: daysAgo(2), duration_ms: 150 },
  { id: 'exec-008', automation_id: IDS.workflows.dailyReport, chain_id: 'chain-008', trigger_event: {}, condition_result: false, status: 'skipped', steps: [], error_message: '', started_at: daysAgo(2), completed_at: daysAgo(2), duration_ms: 20 },
]

// ---------------------------------------------------------------------------
// Trigger & action catalogs — matches `TriggerDefinition` / `ActionDefinition`
// ---------------------------------------------------------------------------

const triggerDefinitions = [
  { type: 'event', module: 'kontakte', name: 'Kontakt erstellt', description: 'Löst aus, wenn ein neuer Kontakt angelegt wird', event_type: 'contact.created', fields: [{ key: 'company', label: 'Firma', type: 'string', operators: ['equals', 'contains'] }, { key: 'tags', label: 'Tags', type: 'string', operators: ['contains'] }], config_params: [] },
  { type: 'event', module: 'kontakte', name: 'Deal gewonnen', description: 'Löst aus, wenn ein Deal als gewonnen markiert wird', event_type: 'deal.won', fields: [{ key: 'value', label: 'Wert', type: 'number', operators: ['gt', 'lt', 'equals'] }, { key: 'stage', label: 'Phase', type: 'string', operators: ['equals'] }], config_params: [] },
  { type: 'event', module: 'finanzen', name: 'Rechnung überfällig', description: 'Löst aus, wenn eine Rechnung das Fälligkeitsdatum überschreitet', event_type: 'invoice.overdue', fields: [{ key: 'amount', label: 'Betrag', type: 'number', operators: ['gt', 'lt'] }, { key: 'days_overdue', label: 'Tage überfällig', type: 'number', operators: ['gt'] }], config_params: [] },
  { type: 'event', module: 'helpdesk', name: 'Ticket erstellt', description: 'Löst aus, wenn ein neues Ticket eingeht', event_type: 'ticket.created', fields: [{ key: 'priority', label: 'Priorität', type: 'string', operators: ['equals'] }, { key: 'category', label: 'Kategorie', type: 'string', operators: ['equals'] }], config_params: [] },
  { type: 'event', module: 'work', name: 'Projekt abgeschlossen', description: 'Löst aus, wenn ein Projekt abgeschlossen wird', event_type: 'project.completed', fields: [{ key: 'project_type', label: 'Projekttyp', type: 'string', operators: ['equals'] }], config_params: [] },
  { type: 'schedule', module: 'system', name: 'Zeitgesteuert', description: 'Cron-basierter Zeitplan', event_type: 'schedule', fields: [], config_params: [{ key: 'cron', label: 'Cron-Ausdruck', type: 'string', required: true, default_value: '0 9 * * 1-5' }] },
  { type: 'webhook', module: 'system', name: 'Webhook', description: 'Externer HTTP-Trigger', event_type: 'webhook', fields: [], config_params: [{ key: 'secret', label: 'Signatur-Secret', type: 'string', required: false, default_value: '' }] },
  { type: 'manual', module: 'system', name: 'Manuell', description: 'Per Klick auslösen', event_type: 'manual', fields: [], config_params: [] },
]

const actionDefinitions = [
  { type: 'email', module: 'kommunikation', name: 'E-Mail senden', description: 'Versendet eine E-Mail aus einer Vorlage', params: [{ key: 'template', label: 'Vorlage', type: 'string', required: true, default_value: '' }, { key: 'delay', label: 'Verzögerung (Sek.)', type: 'number', required: false, default_value: 0 }], output_fields: [{ key: 'message_id', label: 'Nachrichten-ID', type: 'string' }] },
  { type: 'notify', module: 'kommunikation', name: 'Benachrichtigung', description: 'Sendet eine interne Benachrichtigung', params: [{ key: 'channel', label: 'Kanal', type: 'string', required: true, default_value: 'email' }, { key: 'target', label: 'Empfänger', type: 'string', required: true, default_value: '' }], output_fields: [] },
  { type: 'update', module: 'system', name: 'Datensatz aktualisieren', description: 'Setzt ein Feld auf einen Wert', params: [{ key: 'field', label: 'Feld', type: 'string', required: true, default_value: '' }, { key: 'value', label: 'Wert', type: 'string', required: true, default_value: '' }], output_fields: [] },
  { type: 'assign', module: 'kontakte', name: 'Zuweisen', description: 'Weist einen Datensatz einem Bearbeiter zu', params: [{ key: 'rule', label: 'Regel', type: 'string', required: true, default_value: 'round_robin' }, { key: 'team', label: 'Team', type: 'string', required: false, default_value: '' }], output_fields: [{ key: 'assignee_id', label: 'Bearbeiter', type: 'string' }] },
  { type: 'query', module: 'system', name: 'Daten abfragen', description: 'Liest Datensätze über einen Filter', params: [{ key: 'filter', label: 'Filter', type: 'string', required: true, default_value: '' }], output_fields: [{ key: 'count', label: 'Treffer', type: 'number' }] },
  { type: 'delay', module: 'system', name: 'Verzögerung', description: 'Wartet eine definierte Zeit', params: [{ key: 'duration', label: 'Dauer (Sek.)', type: 'number', required: true, default_value: 3600 }], output_fields: [] },
  { type: 'score', module: 'kontakte', name: 'Scoring', description: 'Berechnet einen Score', params: [{ key: 'model', label: 'Modell', type: 'string', required: true, default_value: 'lead_default' }], output_fields: [{ key: 'score', label: 'Score', type: 'number' }] },
  { type: 'enrich', module: 'kontakte', name: 'Daten anreichern', description: 'Reichert einen Datensatz über einen Anbieter an', params: [{ key: 'provider', label: 'Anbieter', type: 'string', required: true, default_value: 'clearbit' }], output_fields: [] },
  { type: 'aggregate', module: 'system', name: 'Aggregieren', description: 'Fasst Daten mehrerer Quellen zusammen', params: [{ key: 'sources', label: 'Quellen', type: 'string', required: true, default_value: '' }], output_fields: [] },
  { type: 'check', module: 'system', name: 'Status prüfen', description: 'Prüft einen Systemstatus', params: [{ key: 'target', label: 'Ziel', type: 'string', required: true, default_value: '' }], output_fields: [{ key: 'ok', label: 'OK', type: 'boolean' }] },
  { type: 'log', module: 'system', name: 'Protokollieren', description: 'Schreibt einen Protokolleintrag', params: [{ key: 'category', label: 'Kategorie', type: 'string', required: false, default_value: '' }], output_fields: [] },
  { type: 'webhook', module: 'system', name: 'Webhook aufrufen', description: 'Ruft einen externen HTTP-Endpunkt auf', params: [{ key: 'url', label: 'URL', type: 'string', required: true, default_value: '' }], output_fields: [{ key: 'status', label: 'HTTP-Status', type: 'number' }] },
]

const templates = [
  { id: 'tpl-001', name: 'Lead Qualifizierung', description: 'Neue Leads automatisch bewerten und zuweisen', category: 'vertrieb', complexity: 'mittel', trigger_type: 'event', trigger_config: { event_type: 'contact.created' }, conditions: EMPTY_CONDITIONS, actions: actionList([{ type: 'score', config: { model: 'lead_default' } }, { type: 'assign', config: { rule: 'round_robin' } }]) },
  { id: 'tpl-002', name: 'Willkommens-Serie', description: 'E-Mail-Sequenz für neue Kunden', category: 'kommunikation', complexity: 'einfach', trigger_type: 'event', trigger_config: { event_type: 'deal.won' }, conditions: EMPTY_CONDITIONS, actions: actionList([{ type: 'email', config: { template: 'welcome' } }]) },
  { id: 'tpl-003', name: 'Ticket SLA Warnung', description: 'Eskalation bei SLA-Verletzung', category: 'kommunikation', complexity: 'fortgeschritten', trigger_type: 'schedule', trigger_config: { cron: '0 */1 * * *' }, conditions: EMPTY_CONDITIONS, actions: actionList([{ type: 'query', config: { filter: 'tickets.sla_breach' } }, { type: 'notify', config: { channel: 'slack' } }]) },
  { id: 'tpl-004', name: 'Rechnungserinnerung', description: 'Automatische Mahnung bei überfälligen Rechnungen', category: 'finanzen', complexity: 'mittel', trigger_type: 'schedule', trigger_config: { cron: '0 9 * * 1-5' }, conditions: EMPTY_CONDITIONS, actions: actionList([{ type: 'query', config: { filter: 'invoices.overdue' } }, { type: 'email', config: { template: 'payment_reminder' } }]) },
  { id: 'tpl-005', name: 'Geburtstags-Gruß', description: 'Automatische Geburtstagsnachricht an Kontakte', category: 'personal', complexity: 'einfach', trigger_type: 'schedule', trigger_config: { cron: '0 8 * * *' }, conditions: EMPTY_CONDITIONS, actions: actionList([{ type: 'email', config: { template: 'birthday' } }]) },
]

// ---------------------------------------------------------------------------
// Handlers — literal sub-paths MUST precede the `:id` route (first-match-wins)
// ---------------------------------------------------------------------------

export const automationHandlers = [
  // List
  http.get(`${API}/api/v1/automations`, ({ request }) => {
    const url = new URL(request.url)
    const isActive = url.searchParams.get('is_active')
    const scope = url.searchParams.get('scope')
    const triggerType = url.searchParams.get('trigger_type')
    let filtered = [...automations]
    if (isActive !== null) filtered = filtered.filter((a) => a.is_active === (isActive === 'true'))
    if (scope) filtered = filtered.filter((a) => a.scope === scope)
    if (triggerType) filtered = filtered.filter((a) => a.trigger_type === triggerType)
    return HttpResponse.json({ automations: filtered, total: filtered.length })
  }),

  // Stats
  http.get(`${API}/api/v1/automations/stats`, () => {
    const total = automations.length
    const active = automations.filter((a) => a.is_active).length
    const totalExecutions = executions.length
    const successful = executions.filter((e) => e.status === 'completed').length
    const failed = executions.filter((e) => e.status === 'failed').length
    // success_rate is a fraction (0–1); the StatsBar renders it as `rate * 100`%.
    const rate = totalExecutions > 0 ? successful / totalExecutions : 0
    return HttpResponse.json({
      total,
      active,
      inactive: total - active,
      total_executions: totalExecutions,
      successful,
      failed,
      success_rate: rate,
    })
  }),

  // Trigger catalog
  http.get(`${API}/api/v1/automations/triggers`, () => {
    return HttpResponse.json({ triggers: triggerDefinitions })
  }),

  // Action catalog
  http.get(`${API}/api/v1/automations/actions`, () => {
    return HttpResponse.json({ actions: actionDefinitions })
  }),

  // Templates
  http.get(`${API}/api/v1/automations/templates`, ({ request }) => {
    const category = new URL(request.url).searchParams.get('category')
    const list = category ? templates.filter((t) => t.category === category) : templates
    return HttpResponse.json({ templates: list })
  }),

  // Single execution (literal `/executions/:id` before `/:id/...`)
  http.get(`${API}/api/v1/automations/executions/:executionId`, ({ params }) => {
    const exec = executions.find((e) => e.id === params.executionId)
    if (!exec) return HttpResponse.json({ error: 'Execution not found' }, { status: 404 })
    return HttpResponse.json(exec)
  }),

  // Test a condition
  http.post(`${API}/api/v1/automations/test-condition`, () => {
    return HttpResponse.json({ matches: true, error: '' })
  }),

  // Dry run
  http.post(`${API}/api/v1/automations/dry-run`, async ({ request }) => {
    const body = (await request.json()) as { automation_id?: string }
    const auto = automations.find((a) => a.id === body.automation_id)
    const steps = (auto?.actions ?? [{ type: 'email' }]).map((a) => step(a.type, 120))
    return HttpResponse.json({ condition_result: true, steps, would_execute: true })
  }),

  // Create from template
  http.post(`${API}/api/v1/automations/templates/:templateId/create`, async ({ params, request }) => {
    const tpl = templates.find((t) => t.id === params.templateId)
    if (!tpl) return HttpResponse.json({ error: 'Template not found' }, { status: 404 })
    const body = (await request.json()) as { name?: string }
    const now = new Date().toISOString()
    const created = {
      id: `auto-${Date.now()}`,
      name: body.name || tpl.name,
      description: tpl.description,
      scope: 'personal' as const,
      owner_id: IDS.users.stefan,
      trigger_type: tpl.trigger_type,
      trigger_config: tpl.trigger_config,
      conditions: tpl.conditions,
      actions: tpl.actions,
      is_active: false,
      max_steps: 10,
      template_id: tpl.id,
      last_triggered_at: null,
      created_at: now,
      updated_at: now,
    }
    automations = [created, ...automations]
    return HttpResponse.json(created, { status: 201 })
  }),

  // Enable / Disable
  http.post(`${API}/api/v1/automations/:id/enable`, ({ params }) => {
    const auto = automations.find((a) => a.id === params.id)
    if (!auto) return HttpResponse.json({ error: 'Not found' }, { status: 404 })
    auto.is_active = true
    auto.updated_at = new Date().toISOString()
    return HttpResponse.json(auto)
  }),
  http.post(`${API}/api/v1/automations/:id/disable`, ({ params }) => {
    const auto = automations.find((a) => a.id === params.id)
    if (!auto) return HttpResponse.json({ error: 'Not found' }, { status: 404 })
    auto.is_active = false
    auto.updated_at = new Date().toISOString()
    return HttpResponse.json(auto)
  }),

  // Executions for one automation
  http.get(`${API}/api/v1/automations/:id/executions`, ({ params, request }) => {
    const status = new URL(request.url).searchParams.get('status')
    let list = executions.filter((e) => e.automation_id === params.id)
    if (status) list = list.filter((e) => e.status === status)
    return HttpResponse.json({ executions: list, total: list.length })
  }),

  // Detail (raw object) — keep AFTER the literal sub-paths
  http.get(`${API}/api/v1/automations/:id`, ({ params }) => {
    const auto = automations.find((a) => a.id === params.id)
    if (!auto) return HttpResponse.json({ error: 'Not found' }, { status: 404 })
    return HttpResponse.json(auto)
  }),

  // Create
  http.post(`${API}/api/v1/automations`, async ({ request }) => {
    const body = (await request.json()) as Record<string, unknown>
    const now = new Date().toISOString()
    const created = {
      id: `auto-${Date.now()}`,
      name: (body.name as string) || 'Neue Automatisierung',
      description: (body.description as string) || '',
      scope: (body.scope as 'personal' | 'team' | 'organization') || 'personal',
      owner_id: IDS.users.stefan,
      trigger_type: (body.trigger_type as string) || 'manual',
      trigger_config: (body.trigger_config as Record<string, unknown>) || {},
      conditions: (body.conditions as typeof EMPTY_CONDITIONS) || EMPTY_CONDITIONS,
      actions: (body.actions as { type: string; config: Record<string, unknown>; on_error: 'continue' | 'abort' }[]) || [],
      is_active: false,
      max_steps: (body.max_steps as number) || 10,
      template_id: null,
      last_triggered_at: null,
      created_at: now,
      updated_at: now,
    }
    automations = [created, ...automations]
    return HttpResponse.json(created, { status: 201 })
  }),

  // Update
  http.put(`${API}/api/v1/automations/:id`, async ({ params, request }) => {
    const auto = automations.find((a) => a.id === params.id)
    if (!auto) return HttpResponse.json({ error: 'Not found' }, { status: 404 })
    const body = (await request.json()) as Record<string, unknown>
    Object.assign(auto, body, { updated_at: new Date().toISOString() })
    return HttpResponse.json(auto)
  }),

  // Delete
  http.delete(`${API}/api/v1/automations/:id`, ({ params }) => {
    const exists = automations.some((a) => a.id === params.id)
    if (!exists) return HttpResponse.json({ error: 'Not found' }, { status: 404 })
    automations = automations.filter((a) => a.id !== params.id)
    return new HttpResponse(null, { status: 204 })
  }),
]
