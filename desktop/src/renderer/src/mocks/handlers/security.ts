import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS } from '../data/shared-ids'
import { daysAgo, hoursAgo, minutesAgo, daysFromNow } from '../data/date-helpers'
import { listLiveAuditEvents } from '../data/audit-events'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Audit log entries (50)
// ---------------------------------------------------------------------------

// Raw audit data — transformed below to match AuditEntry interface
const rawAuditLogs = [
  { action: 'login', user_id: IDS.users.stefan, user_name: 'Stefan Vogel', ip: '192.168.1.100', target: 'Desktop App', target_type: 'session', ts: minutesAgo(5) },
  { action: 'contact_created', user_id: IDS.users.thomas, user_name: 'Thomas Meier', ip: '192.168.1.103', target: 'Maria Schwarz', target_type: 'contact', ts: minutesAgo(12) },
  { action: 'deal_updated', user_id: IDS.users.thomas, user_name: 'Thomas Meier', ip: '192.168.1.103', target: 'CRM Lizenz', target_type: 'deal', ts: minutesAgo(25) },
  { action: 'document_uploaded', user_id: IDS.users.lena, user_name: 'Lena Braun', ip: '192.168.1.109', target: 'Flyer_Messe_2026.pdf', target_type: 'document', ts: minutesAgo(40) },
  { action: 'settings_changed', user_id: IDS.users.stefan, user_name: 'Stefan Vogel', ip: '192.168.1.100', target: 'CORS-Einstellungen', target_type: 'settings', ts: hoursAgo(1) },
  { action: 'login', user_id: IDS.users.markus, user_name: 'Markus Weber', ip: '192.168.1.102', target: 'Desktop App', target_type: 'session', ts: hoursAgo(1) },
  { action: 'user_role_changed', user_id: IDS.users.stefan, user_name: 'Stefan Vogel', ip: '192.168.1.100', target: 'Felix Krause', target_type: 'user', ts: hoursAgo(2) },
  { action: 'login', user_id: IDS.users.julia, user_name: 'Julia Hofmann', ip: '192.168.1.112', target: 'Desktop App', target_type: 'session', ts: hoursAgo(2) },
  { action: 'deal_created', user_id: IDS.users.thomas, user_name: 'Thomas Meier', ip: '192.168.1.103', target: 'KI Beratung', target_type: 'deal', ts: hoursAgo(3) },
  { action: 'invoice_sent', user_id: IDS.users.michael, user_name: 'Petra Zimmermann', ip: '192.168.1.113', target: 'RE-2026-080', target_type: 'invoice', ts: hoursAgo(3) },
  { action: 'document_deleted', user_id: IDS.users.felix, user_name: 'Felix Krause', ip: '192.168.1.105', target: 'Entwurf_alt.pdf', target_type: 'document', ts: hoursAgo(4) },
  { action: 'login', user_id: IDS.users.nina, user_name: 'Elena Schuster', ip: '192.168.1.115', target: 'Desktop App', target_type: 'session', ts: hoursAgo(4) },
  { action: 'contact_updated', user_id: IDS.users.laura, user_name: 'Sabine Fischer', ip: '192.168.1.110', target: 'Hans Müller', target_type: 'contact', ts: hoursAgo(5) },
  { action: 'project_created', user_id: IDS.users.elena, user_name: 'Sarah Beck', ip: '192.168.1.118', target: 'Datenschutz Audit', target_type: 'project', ts: hoursAgo(5) },
  { action: 'login_failed', user_id: '', user_name: 'unbekannt', ip: '203.0.113.42', target: 'admin@techvision.de', target_type: 'session', ts: hoursAgo(6), result: 'failure' as const },
  { action: 'password_changed', user_id: IDS.users.lena, user_name: 'Lena Braun', ip: '192.168.1.109', target: 'Eigenes Passwort', target_type: 'user', ts: hoursAgo(6) },
  { action: 'login', user_id: IDS.users.thomas, user_name: 'Thomas Meier', ip: '192.168.1.103', target: 'Desktop App', target_type: 'session', ts: hoursAgo(7) },
  { action: 'deal_updated', user_id: IDS.users.laura, user_name: 'Sabine Fischer', ip: '192.168.1.110', target: 'Web Redesign', target_type: 'deal', ts: hoursAgo(8) },
  { action: 'settings_changed', user_id: IDS.users.markus, user_name: 'Markus Weber', ip: '192.168.1.102', target: 'Rate Limiting', target_type: 'settings', ts: hoursAgo(9) },
  { action: 'document_uploaded', user_id: IDS.users.sophie, user_name: 'Sophie Lang', ip: '192.168.1.108', target: 'Vertrag_Vorlage.docx', target_type: 'document', ts: hoursAgo(10) },
  { action: 'login', user_id: IDS.users.felix, user_name: 'Felix Krause', ip: '192.168.1.105', target: 'Desktop App', target_type: 'session', ts: hoursAgo(12) },
  { action: 'contact_created', user_id: IDS.users.laura, user_name: 'Sabine Fischer', ip: '192.168.1.110', target: 'Lisa Winkler', target_type: 'contact', ts: hoursAgo(14) },
  { action: 'invoice_created', user_id: IDS.users.michael, user_name: 'Petra Zimmermann', ip: '192.168.1.113', target: 'RE-2026-080', target_type: 'invoice', ts: daysAgo(1) },
  { action: 'login', user_id: IDS.users.lena, user_name: 'Lena Braun', ip: '192.168.1.109', target: 'Desktop App', target_type: 'session', ts: daysAgo(1) },
  { action: 'deal_stage_changed', user_id: IDS.users.thomas, user_name: 'Thomas Meier', ip: '192.168.1.103', target: 'ERP Migration', target_type: 'deal', ts: daysAgo(1) },
  { action: 'login', user_id: IDS.users.sophie, user_name: 'Sophie Lang', ip: '192.168.1.108', target: 'Desktop App', target_type: 'session', ts: daysAgo(1) },
  { action: 'document_uploaded', user_id: IDS.users.markus, user_name: 'Markus Weber', ip: '192.168.1.102', target: 'API_Dokumentation.docx', target_type: 'document', ts: daysAgo(1) },
  { action: 'settings_changed', user_id: IDS.users.stefan, user_name: 'Stefan Vogel', ip: '192.168.1.100', target: 'Backup-Zeitplan', target_type: 'settings', ts: daysAgo(1) },
  { action: 'login_failed', user_id: '', user_name: 'unbekannt', ip: '198.51.100.17', target: 'test@example.com', target_type: 'session', ts: daysAgo(1), result: 'failure' as const },
  { action: 'login', user_id: IDS.users.michael, user_name: 'Petra Zimmermann', ip: '192.168.1.113', target: 'Desktop App', target_type: 'session', ts: daysAgo(2) },
  { action: 'contact_updated', user_id: IDS.users.thomas, user_name: 'Thomas Meier', ip: '192.168.1.103', target: 'Werner Koch', target_type: 'contact', ts: daysAgo(2) },
  { action: 'deal_created', user_id: IDS.users.laura, user_name: 'Sabine Fischer', ip: '192.168.1.110', target: 'Netzwerk Upgrade', target_type: 'deal', ts: daysAgo(2) },
  { action: 'login', user_id: IDS.users.elena, user_name: 'Sarah Beck', ip: '192.168.1.118', target: 'Desktop App', target_type: 'session', ts: daysAgo(2) },
  { action: 'document_uploaded', user_id: IDS.users.nina, user_name: 'Elena Schuster', ip: '192.168.1.115', target: 'Onboarding_Checkliste.pdf', target_type: 'document', ts: daysAgo(2) },
  { action: 'invoice_paid', user_id: IDS.users.michael, user_name: 'Petra Zimmermann', ip: '192.168.1.113', target: 'RE-2026-077', target_type: 'invoice', ts: daysAgo(3) },
  { action: 'login', user_id: IDS.users.jan, user_name: 'Jonas Schmitt', ip: '192.168.1.117', target: 'Desktop App', target_type: 'session', ts: daysAgo(3) },
  { action: 'contact_deleted', user_id: IDS.users.thomas, user_name: 'Thomas Meier', ip: '192.168.1.103', target: 'Test Kontakt', target_type: 'contact', ts: daysAgo(3) },
  { action: 'deal_updated', user_id: IDS.users.thomas, user_name: 'Thomas Meier', ip: '192.168.1.103', target: 'Support Vertrag', target_type: 'deal', ts: daysAgo(3) },
  { action: 'login', user_id: IDS.users.david, user_name: 'Kevin Baumann', ip: '192.168.1.111', target: 'Desktop App', target_type: 'session', ts: daysAgo(4) },
  { action: 'settings_changed', user_id: IDS.users.markus, user_name: 'Markus Weber', ip: '192.168.1.102', target: 'E-Mail-SMTP', target_type: 'settings', ts: daysAgo(4) },
  { action: 'document_uploaded', user_id: IDS.users.thomas, user_name: 'Thomas Meier', ip: '192.168.1.103', target: 'NDA_Rhein_Consulting.pdf', target_type: 'document', ts: daysAgo(4) },
  { action: 'login', user_id: IDS.users.christian, user_name: 'Martin Wolf', ip: '192.168.1.116', target: 'Desktop App', target_type: 'session', ts: daysAgo(4) },
  { action: 'contact_created', user_id: IDS.users.thomas, user_name: 'Thomas Meier', ip: '192.168.1.103', target: 'Franz Egger', target_type: 'contact', ts: daysAgo(5) },
  { action: 'login_failed', user_id: '', user_name: 'unbekannt', ip: '203.0.113.88', target: 'admin@test.com', target_type: 'session', ts: daysAgo(5), result: 'failure' as const },
  { action: 'deal_stage_changed', user_id: IDS.users.laura, user_name: 'Sabine Fischer', ip: '192.168.1.110', target: 'Hosting Vertrag', target_type: 'deal', ts: daysAgo(5) },
  { action: 'login', user_id: IDS.users.anna, user_name: 'Julia Hofmann', ip: '192.168.1.112', target: 'Desktop App', target_type: 'session', ts: daysAgo(6) },
  { action: 'invoice_sent', user_id: IDS.users.michael, user_name: 'Petra Zimmermann', ip: '192.168.1.113', target: 'RE-2026-079', target_type: 'invoice', ts: daysAgo(6) },
  { action: 'document_uploaded', user_id: IDS.users.felix, user_name: 'Felix Krause', ip: '192.168.1.105', target: 'Testbericht_v2.pdf', target_type: 'document', ts: daysAgo(6) },
  { action: 'project_updated', user_id: IDS.users.elena, user_name: 'Sarah Beck', ip: '192.168.1.118', target: 'Hub V2', target_type: 'project', ts: daysAgo(7) },
  { action: 'login', user_id: IDS.users.stefan, user_name: 'Stefan Vogel', ip: '192.168.1.100', target: 'Desktop App', target_type: 'session', ts: daysAgo(7) },
]

// Transform to AuditEntry format
const auditLogs = rawAuditLogs.map((raw, i) => ({
  id: `aud-${String(i + 1).padStart(3, '0')}`,
  sequence_num: i + 1,
  timestamp: raw.ts,
  user_id: raw.user_id,
  user_name: raw.user_name,
  action: raw.action,
  target: raw.target,
  target_type: raw.target_type,
  details: {},
  ip_address: raw.ip,
  user_agent: 'Cosmi Desktop/1.0',
  result: (raw.result ?? 'success') as 'success' | 'failure',
}))

// ---------------------------------------------------------------------------
// Password policy
// ---------------------------------------------------------------------------

// Field shape matches PasswordPolicy in security-types.ts (BE wire: encoding/json
// over protobuf, snake_case). The Go gateway wraps it as { policy: {...} }.
// `let` so the PUT handler can mutate it (stateful demo, survives navigation).
let passwordPolicy = {
  id: 'pwp-001',
  min_length: 12,
  min_entropy: 60,
  max_age_days: 90,
  prevent_reuse_count: 5,
  require_uppercase: true,
  require_lowercase: true,
  require_digit: true,
  require_special: true,
}

// ---------------------------------------------------------------------------
// IP rules — field shape matches IPAccessRule (ip_cidr, rule_type, created_by)
// ---------------------------------------------------------------------------

let ipRules = [
  { id: 'ipr-001', ip_cidr: '192.168.1.0/24', rule_type: 'allow', description: 'Büro-Netzwerk München', created_by: IDS.users.stefan, created_at: daysAgo(90) },
  { id: 'ipr-002', ip_cidr: '10.0.0.0/8', rule_type: 'allow', description: 'VPN-Range', created_by: IDS.users.stefan, created_at: daysAgo(90) },
  { id: 'ipr-003', ip_cidr: '203.0.113.0/24', rule_type: 'block', description: 'Bekannte Angreifer-Range', created_by: IDS.users.markus, created_at: daysAgo(5) },
]

// Vault secrets — `let` for stateful set/delete via the write handlers below.
let vaultSecrets = [
  { id: 'vlt-001', key_name: 'SMTP_PASSWORD', description: 'E-Mail-Versand (Postfach noreply@)', key_version: 4, created_by: IDS.users.stefan, created_at: daysAgo(90), updated_at: daysAgo(15) },
  { id: 'vlt-002', key_name: 'LIVEKIT_API_SECRET', description: 'Video-Meetings (LiveKit)', key_version: 2, created_by: IDS.users.stefan, created_at: daysAgo(120), updated_at: daysAgo(30) },
  { id: 'vlt-003', key_name: 'JWT_SECRET', description: 'Session-Token-Signatur', key_version: 7, created_by: IDS.users.markus, created_at: daysAgo(180), updated_at: daysAgo(7) },
  { id: 'vlt-004', key_name: 'DATABASE_PASSWORD', description: 'PostgreSQL Primär-DB', key_version: 3, created_by: IDS.users.markus, created_at: daysAgo(180), updated_at: daysAgo(45) },
  { id: 'vlt-005', key_name: 'REDIS_PASSWORD', description: 'Cache (Redis)', key_version: 3, created_by: IDS.users.markus, created_at: daysAgo(180), updated_at: daysAgo(45) },
]

// ---------------------------------------------------------------------------
// Active sessions — field shape matches UserSession in security-types.ts
// ---------------------------------------------------------------------------

let sessions = [
  { id: 'ses-001', user_id: IDS.users.stefan, device_name: 'MacBook Pro 16"', device_type: 'desktop', ip_address: '192.168.1.100', location: 'München, DE', user_agent: 'Cosmi Desktop/1.0 (macOS 15)', is_current: true, last_active_at: minutesAgo(2), created_at: hoursAgo(3) },
  { id: 'ses-002', user_id: IDS.users.stefan, device_name: 'iPhone 16 Pro', device_type: 'mobile', ip_address: '84.142.55.21', location: 'München, DE', user_agent: 'Cosmi Mobile/1.0 (iOS 18)', is_current: false, last_active_at: hoursAgo(5), created_at: daysAgo(2) },
  { id: 'ses-003', user_id: IDS.users.stefan, device_name: 'iPad Air', device_type: 'tablet', ip_address: '192.168.1.118', location: 'München, DE', user_agent: 'Cosmi Mobile/1.0 (iPadOS 18)', is_current: false, last_active_at: daysAgo(1), created_at: daysAgo(6) },
]

// All-sessions view (admin) adds other users' sessions
let allSessions = [
  ...sessions,
  { id: 'ses-101', user_id: IDS.users.thomas, device_name: 'ThinkPad X1', device_type: 'desktop', ip_address: '192.168.1.103', location: 'München, DE', user_agent: 'Cosmi Desktop/1.0 (Windows 11)', is_current: false, last_active_at: minutesAgo(18), created_at: hoursAgo(7) },
  { id: 'ses-102', user_id: IDS.users.lena, device_name: 'Galaxy S24', device_type: 'mobile', ip_address: '91.64.12.7', location: 'Augsburg, DE', user_agent: 'Cosmi Mobile/1.0 (Android 15)', is_current: false, last_active_at: hoursAgo(2), created_at: daysAgo(1) },
]

// Employee id → display name (mirrors the demo audit/user data) so admin views
// can resolve user_id to a readable name (the real BE joins this server-side).
const EMPLOYEE_NAMES: Record<string, string> = {
  [IDS.users.stefan]: 'Stefan Vogel',
  [IDS.users.markus]: 'Markus Weber',
  [IDS.users.thomas]: 'Thomas Meier',
  [IDS.users.lena]: 'Lena Braun',
  [IDS.users.felix]: 'Felix Krause',
  [IDS.users.laura]: 'Sabine Fischer',
  [IDS.users.michael]: 'Petra Zimmermann',
}

// ---------------------------------------------------------------------------
// GDPR export requests — field shape matches GDPRExportRequest; BE wraps as
// { export_requests: [...] }. `user_name`/`reviewer_name` are demo conveniences
// (the real BE joins them server-side; the type carries them as optional).
// ---------------------------------------------------------------------------

const gdprExports = [
  { id: 'gex-001', user_id: IDS.users.lena, user_name: EMPLOYEE_NAMES[IDS.users.lena], status: 'ready', requested_at: daysAgo(3), reviewed_by: IDS.users.stefan, reviewer_name: EMPLOYEE_NAMES[IDS.users.stefan], reviewed_at: daysAgo(2), review_note: 'Auskunftsersuchen genehmigt', download_token: 'tok-lena-001', download_expires_at: daysFromNow(27) },
  { id: 'gex-002', user_id: IDS.users.thomas, user_name: EMPLOYEE_NAMES[IDS.users.thomas], status: 'pending', requested_at: hoursAgo(6), reviewed_by: null, reviewer_name: null, reviewed_at: null, review_note: null, download_token: null, download_expires_at: null },
  { id: 'gex-003', user_id: IDS.users.felix, user_name: EMPLOYEE_NAMES[IDS.users.felix], status: 'denied', requested_at: daysAgo(10), reviewed_by: IDS.users.markus, reviewer_name: EMPLOYEE_NAMES[IDS.users.markus], reviewed_at: daysAgo(9), review_note: 'Identität nicht zweifelsfrei bestätigt', download_token: null, download_expires_at: null },
]

// ---------------------------------------------------------------------------
// DSAR (Art. 15) cross-module search — shape matches BackendPerson in
// DSARSearchPage (id, name, email, company, avatar, modules[{module,columns,records}]).
// `records` keys are the `columns` entries.
// ---------------------------------------------------------------------------

const dsarSubjects = [
  {
    id: IDS.users.lena, name: 'Lena Braun', email: 'lena.braun@techvision.de', company: 'TechVision GmbH', avatar: 'LB',
    modules: [
      { module: 'CRM Kontakte', columns: ['Name', 'E-Mail', 'Telefon'], records: [{ Name: 'Lena Braun', 'E-Mail': 'lena.braun@techvision.de', Telefon: '+49 89 1234567' }] },
      { module: 'Kalender', columns: ['Titel', 'Datum'], records: [{ Titel: 'Onboarding Lena', Datum: daysAgo(30) }, { Titel: 'Review Q2', Datum: daysAgo(12) }, { Titel: 'Projektplanung', Datum: daysAgo(2) }] },
      { module: 'Formulare', columns: ['Formular', 'Eingereicht am'], records: [{ Formular: 'Mitarbeiter-Feedback', 'Eingereicht am': daysAgo(20) }, { Formular: 'NPS-Umfrage', 'Eingereicht am': daysAgo(5) }] },
      { module: 'E-Mails', columns: ['Betreff', 'Gesendet'], records: [{ Betreff: 'Willkommen im Team', Gesendet: daysAgo(31) }, { Betreff: 'Zugangsdaten', Gesendet: daysAgo(30) }] },
      { module: 'Einwilligungen', columns: ['Zweck', 'Status', 'Erteilt am'], records: [{ Zweck: 'Marketing-Newsletter', Status: 'Erteilt', 'Erteilt am': daysAgo(60) }, { Zweck: 'Datenverarbeitung', Status: 'Erteilt', 'Erteilt am': daysAgo(60) }] },
    ],
  },
  {
    id: IDS.users.thomas, name: 'Thomas Meier', email: 'thomas.meier@techvision.de', company: 'TechVision GmbH', avatar: 'TM',
    modules: [
      { module: 'CRM Kontakte', columns: ['Name', 'E-Mail', 'Telefon'], records: [{ Name: 'Thomas Meier', 'E-Mail': 'thomas.meier@techvision.de', Telefon: '+49 89 7654321' }] },
      { module: 'CRM Deals', columns: ['Deal', 'Wert', 'Phase'], records: [{ Deal: 'CRM Lizenz', Wert: '24.000 €', Phase: 'Gewonnen' }, { Deal: 'KI Beratung', Wert: '12.000 €', Phase: 'Verhandlung' }] },
      { module: 'Anrufe', columns: ['Kontakt', 'Dauer', 'Datum'], records: [{ Kontakt: 'Maria Schwarz', Dauer: '12:34', Datum: daysAgo(4) }, { Kontakt: 'Hans Müller', Dauer: '03:10', Datum: daysAgo(1) }] },
      { module: 'Kalender', columns: ['Titel', 'Datum'], records: [{ Titel: 'Vertriebsmeeting', Datum: daysAgo(7) }] },
    ],
  },
]

// ---------------------------------------------------------------------------
// 2FA policies — field shape matches TwoFactorPolicy in security-types.ts
// (role_name, enforced, grace_period_days); BE wraps as { policies: [...] }.
// ---------------------------------------------------------------------------

const twofaPolicies = [
  { role_name: 'admin', enforced: true, grace_period_days: 0 },
  { role_name: 'manager', enforced: false, grace_period_days: 14 },
  { role_name: 'member', enforced: false, grace_period_days: 14 },
]

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

export const securityHandlers = [
  // Audit logs (paginated via offset/limit matching AuditFilter)
  http.get(`${API}/api/v1/security/audit`, ({ request }) => {
    const url = new URL(request.url)
    const offset = parseInt(url.searchParams.get('offset') || '0')
    const limit = parseInt(url.searchParams.get('limit') || '20')
    const action = url.searchParams.get('action')
    const userId = url.searchParams.get('user_id')
    const result = url.searchParams.get('result')

    // Live R-5 events (session mutations) are newer than every seed → prepend.
    let filtered = [...listLiveAuditEvents(), ...auditLogs]
    if (action) {
      filtered = filtered.filter((e) => e.action === action)
    }
    if (userId) {
      filtered = filtered.filter((e) => e.user_id === userId)
    }
    if (result) {
      filtered = filtered.filter((e) => e.result === result)
    }

    const paginated = filtered.slice(offset, offset + limit)

    return HttpResponse.json({ entries: paginated, total: filtered.length })
  }),

  // Vault secrets list
  http.get(`${API}/api/v1/security/vault`, () => {
    return HttpResponse.json({ secrets: vaultSecrets })
  }),

  // Password policy
  http.get(`${API}/api/v1/security/password/policy`, () => {
    return HttpResponse.json({ policy: passwordPolicy })
  }),

  // IP rules
  http.get(`${API}/api/v1/security/ip-rules`, () => {
    return HttpResponse.json({ rules: ipRules, total: ipRules.length })
  }),

  // GDPR export requests — BE wraps as { export_requests: [...] }
  http.get(`${API}/api/v1/security/gdpr/exports`, () => {
    return HttpResponse.json({ export_requests: gdprExports })
  }),

  // Active sessions (own)
  http.get(`${API}/api/v1/auth/sessions`, () => {
    return HttpResponse.json({ sessions })
  }),

  // All sessions (admin view) — BE requires user_id, lists sessions for one user
  http.get(`${API}/api/v1/auth/sessions/all`, ({ request }) => {
    const userId = new URL(request.url).searchParams.get('user_id')
    if (!userId) {
      return HttpResponse.json({ error: 'user_id query parameter is required' }, { status: 400 })
    }
    const filtered = allSessions.filter((s) => s.user_id === userId)
    return HttpResponse.json({ sessions: filtered, total: filtered.length })
  }),

  // DSAR (Art. 15) cross-module search
  http.get(`${API}/api/v1/security/dsar/search`, ({ request }) => {
    const url = new URL(request.url)
    const q = (url.searchParams.get('q') || '').toLowerCase().trim()
    const results = q
      ? dsarSubjects.filter(
          (s) => s.name.toLowerCase().includes(q) || s.email.toLowerCase().includes(q),
        )
      : []
    return HttpResponse.json({ results })
  }),

  // 2FA policies
  http.get(`${API}/api/v1/auth/2fa/policy`, () => {
    return HttpResponse.json({ policies: twofaPolicies })
  }),

  // Validate password — always valid in demo
  http.post(`${API}/api/v1/security/password/validate`, () => {
    return HttpResponse.json({ valid: true, score: 4, feedback: [] })
  }),

  // ── Stateful write handlers (in-memory; survives navigation, resets on reload) ──

  // Reveal a secret value (mutation; never cached). Demo value only.
  http.get(`${API}/api/v1/security/vault/:keyName`, ({ params }) => {
    const keyName = String(params.keyName)
    return HttpResponse.json({ key_name: keyName, value: `demo-secret-${keyName.toLowerCase()}-9f3a21` })
  }),

  // Create or update a secret
  http.put(`${API}/api/v1/security/vault/:keyName`, async ({ params, request }) => {
    const keyName = String(params.keyName)
    const body = (await request.json().catch(() => ({}))) as { value?: string; description?: string }
    const nowIso = new Date().toISOString()
    const existing = vaultSecrets.find((s) => s.key_name === keyName)
    if (existing) {
      existing.key_version += 1
      existing.updated_at = nowIso
      if (body.description !== undefined) existing.description = body.description
      return HttpResponse.json({ secret: existing })
    }
    const created = { id: `vlt-${Date.now()}`, key_name: keyName, description: body.description ?? '', key_version: 1, created_by: IDS.users.stefan, created_at: nowIso, updated_at: nowIso }
    vaultSecrets = [created, ...vaultSecrets]
    return HttpResponse.json({ secret: created })
  }),

  // Delete a secret
  http.delete(`${API}/api/v1/security/vault/:keyName`, ({ params }) => {
    const keyName = String(params.keyName)
    vaultSecrets = vaultSecrets.filter((s) => s.key_name !== keyName)
    return HttpResponse.json({ status: 'secret deleted' })
  }),

  // Update password policy
  http.put(`${API}/api/v1/security/password/policy`, async ({ request }) => {
    const patch = (await request.json().catch(() => ({}))) as Partial<typeof passwordPolicy>
    passwordPolicy = { ...passwordPolicy, ...patch, id: passwordPolicy.id }
    return HttpResponse.json({ policy: passwordPolicy })
  }),

  // Create IP rule
  http.post(`${API}/api/v1/security/ip-rules`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { ip_cidr?: string; rule_type?: string; description?: string }
    const created = { id: `ipr-${Date.now()}`, ip_cidr: body.ip_cidr ?? '', rule_type: body.rule_type ?? 'allow', description: body.description ?? '', created_by: IDS.users.stefan, created_at: new Date().toISOString() }
    ipRules = [...ipRules, created]
    return HttpResponse.json({ rule: created })
  }),

  // Delete IP rule
  http.delete(`${API}/api/v1/security/ip-rules/:id`, ({ params }) => {
    const id = String(params.id)
    ipRules = ipRules.filter((r) => r.id !== id)
    return HttpResponse.json({ status: 'ip rule deleted' })
  }),

  // Export audit log (CSV / JSON blob)
  http.get(`${API}/api/v1/security/audit/export`, ({ request }) => {
    const format = new URL(request.url).searchParams.get('format') || 'csv'
    const exportLogs = [...listLiveAuditEvents(), ...auditLogs]
    if (format === 'json') {
      return new HttpResponse(JSON.stringify(exportLogs, null, 2), {
        headers: { 'Content-Type': 'application/json' },
      })
    }
    const header = 'timestamp,user,action,target,target_type,ip_address,result'
    const rows = exportLogs.map(
      (e) => `${e.timestamp},"${e.user_name}",${e.action},"${e.target}",${e.target_type},${e.ip_address},${e.result}`,
    )
    return new HttpResponse([header, ...rows].join('\n'), {
      headers: { 'Content-Type': 'text/csv' },
    })
  }),

  // Verify audit chain integrity
  http.post(`${API}/api/v1/security/audit/verify`, () => {
    return HttpResponse.json({ valid: true, entries_checked: auditLogs.length, first_invalid_sequence: 0 })
  }),

  // Terminate a single session
  http.delete(`${API}/api/v1/auth/sessions/:id`, ({ params }) => {
    const id = String(params.id)
    sessions = sessions.filter((s) => s.id !== id)
    allSessions = allSessions.filter((s) => s.id !== id)
    return HttpResponse.json({ status: 'session terminated' })
  }),

  // Terminate all other sessions (keep the current one)
  http.delete(`${API}/api/v1/auth/sessions`, () => {
    const removed = sessions.filter((s) => !s.is_current).length
    sessions = sessions.filter((s) => s.is_current)
    allSessions = allSessions.filter((s) => s.is_current || s.user_id !== IDS.users.stefan)
    return HttpResponse.json({ terminated_count: removed })
  }),

  // Approve a GDPR export request → status ready, issue download token (Art. 12: 30-day window)
  http.post(`${API}/api/v1/security/gdpr/export/:id/approve`, ({ params }) => {
    const id = String(params.id)
    const exp = gdprExports.find((e) => e.id === id)
    if (exp) {
      exp.status = 'ready'
      exp.reviewed_by = IDS.users.stefan
      exp.reviewer_name = EMPLOYEE_NAMES[IDS.users.stefan]
      exp.reviewed_at = new Date().toISOString()
      exp.review_note = 'Auskunftsersuchen genehmigt'
      exp.download_token = `tok-${id}-${Date.now()}`
      exp.download_expires_at = daysFromNow(30)
    }
    return HttpResponse.json({ export_request: exp })
  }),

  // Deny a GDPR export request
  http.post(`${API}/api/v1/security/gdpr/export/:id/deny`, async ({ params, request }) => {
    const id = String(params.id)
    const body = (await request.json().catch(() => ({}))) as { note?: string }
    const exp = gdprExports.find((e) => e.id === id)
    if (exp) {
      exp.status = 'denied'
      exp.reviewed_by = IDS.users.stefan
      exp.reviewer_name = EMPLOYEE_NAMES[IDS.users.stefan]
      exp.reviewed_at = new Date().toISOString()
      exp.review_note = body.note ?? 'Abgelehnt'
    }
    return HttpResponse.json({ export_request: exp })
  }),

  // Erasure preview (Art. 17): affected modules per subject. Modules under a
  // legal retention duty default to 'retain' (blocked, not deleted).
  http.post(`${API}/api/v1/security/gdpr/erasure/preview`, () => {
    const modules = [
      { module_name: 'CRM Kontakte', record_count: 47, action: 'anonymize' },
      { module_name: 'Chat-Nachrichten', record_count: 1283, action: 'delete' },
      { module_name: 'Kalender-Termine', record_count: 156, action: 'delete' },
      { module_name: 'Aufgaben & Projekte', record_count: 89, action: 'anonymize' },
      { module_name: 'Helpdesk-Tickets', record_count: 34, action: 'anonymize' },
      { module_name: 'Dokumente', record_count: 23, action: 'delete' },
      { module_name: 'E-Mails', record_count: 412, action: 'delete' },
      { module_name: 'Formulare', record_count: 8, action: 'anonymize' },
      { module_name: 'Audit-Log', record_count: 342, action: 'retain' },
      { module_name: 'Rechnungen', record_count: 12, action: 'retain' },
    ]
    const total_records = modules.reduce((s, m) => s + m.record_count, 0)
    return HttpResponse.json({ modules, total_records })
  }),

  // Erasure execute (Art. 17): anonymize/delete per chosen action, return receipt
  http.post(`${API}/api/v1/security/gdpr/erasure/execute`, async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as { user_id?: string }
    const suffix = (body.user_id ?? 'user').slice(-4)
    return HttpResponse.json({
      anonymized_label: `anonymisiert-${suffix}`,
      confirmation_hash: `sha256:${Date.now().toString(16)}`,
      executed_at: new Date().toISOString(),
    })
  }),

  // Download a ready export artefact (JSON stub blob)
  http.get(`${API}/api/v1/security/gdpr/export/:token/download`, ({ params }) => {
    const token = String(params.token)
    const exp = gdprExports.find((e) => e.download_token === token)
    const payload = {
      generated_at: new Date().toISOString(),
      subject: exp ? EMPLOYEE_NAMES[exp.user_id] ?? exp.user_id : token,
      note: 'Demo-Datenexport (Art. 15/20 DSGVO) — Platzhalter-Artefakt.',
      modules: ['CRM', 'Kalender', 'E-Mails', 'Formulare', 'Einwilligungen'],
    }
    return new HttpResponse(JSON.stringify(payload, null, 2), {
      headers: { 'Content-Type': 'application/json' },
    })
  }),
]
