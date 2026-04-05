import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS } from '../data/shared-ids'
import { daysAgo, hoursAgo, minutesAgo } from '../data/date-helpers'

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

const passwordPolicy = {
  min_length: 12,
  require_uppercase: true,
  require_lowercase: true,
  require_numbers: true,
  require_special: true,
  max_age_days: 90,
  prevent_reuse: 5,
  lockout_attempts: 5,
  lockout_duration_minutes: 30,
}

// ---------------------------------------------------------------------------
// IP rules
// ---------------------------------------------------------------------------

const ipRules = [
  { id: 'ipr-001', type: 'allow', cidr: '192.168.1.0/24', description: 'Büro-Netzwerk München', created_at: daysAgo(90) },
  { id: 'ipr-002', type: 'allow', cidr: '10.0.0.0/8', description: 'VPN', created_at: daysAgo(90) },
  { id: 'ipr-003', type: 'block', cidr: '203.0.113.0/24', description: 'Bekannte Angreifer-Range', created_at: daysAgo(5) },
]

// ---------------------------------------------------------------------------
// 2FA policies
// ---------------------------------------------------------------------------

const twofaPolicies = [
  { id: '2fa-001', name: 'Admin-Accounts', enforce: true, methods: ['totp', 'webauthn'], roles: ['admin'], created_at: daysAgo(60) },
  { id: '2fa-002', name: 'Alle Mitarbeiter', enforce: false, methods: ['totp'], roles: ['admin', 'manager', 'member'], created_at: daysAgo(30) },
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

    let filtered = [...auditLogs]
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
    return HttpResponse.json({
      secrets: [
        { key: 'SMTP_PASSWORD', last_rotated: daysAgo(15), created_at: daysAgo(90) },
        { key: 'LIVEKIT_API_SECRET', last_rotated: daysAgo(30), created_at: daysAgo(120) },
        { key: 'JWT_SECRET', last_rotated: daysAgo(7), created_at: daysAgo(180) },
        { key: 'DATABASE_PASSWORD', last_rotated: daysAgo(45), created_at: daysAgo(180) },
        { key: 'REDIS_PASSWORD', last_rotated: daysAgo(45), created_at: daysAgo(180) },
      ],
    })
  }),

  // Password policy
  http.get(`${API}/api/v1/security/password/policy`, () => {
    return HttpResponse.json({ policy: passwordPolicy })
  }),

  // IP rules
  http.get(`${API}/api/v1/security/ip-rules`, () => {
    return HttpResponse.json({ rules: ipRules, total: ipRules.length })
  }),

  // GDPR exports — empty
  http.get(`${API}/api/v1/security/gdpr/exports`, () => {
    return HttpResponse.json({ exports: [], total: 0 })
  }),

  // 2FA policies
  http.get(`${API}/api/v1/auth/2fa/policies`, () => {
    return HttpResponse.json({ policies: twofaPolicies })
  }),

  // Validate password — always valid in demo
  http.post(`${API}/api/v1/security/password/validate`, () => {
    return HttpResponse.json({ valid: true, score: 4, feedback: [] })
  }),
]
