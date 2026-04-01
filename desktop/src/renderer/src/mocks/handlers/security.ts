import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS } from '../data/shared-ids'
import { daysAgo, hoursAgo, minutesAgo } from '../data/date-helpers'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Audit log entries (50)
// ---------------------------------------------------------------------------

const auditLogs = [
  { id: 'aud-001', action: 'login', user_id: IDS.users.stefan, user_name: 'Stefan Vogel', ip: '192.168.1.100', details: 'Desktop App Login', created_at: minutesAgo(5) },
  { id: 'aud-002', action: 'contact_created', user_id: IDS.users.thomas, user_name: 'Thomas Meier', ip: '192.168.1.103', details: 'Kontakt: Maria Schwarz erstellt', created_at: minutesAgo(12) },
  { id: 'aud-003', action: 'deal_updated', user_id: IDS.users.thomas, user_name: 'Thomas Meier', ip: '192.168.1.103', details: 'Deal "CRM Lizenz" nach Verhandlung verschoben', created_at: minutesAgo(25) },
  { id: 'aud-004', action: 'document_uploaded', user_id: IDS.users.lena, user_name: 'Lena Braun', ip: '192.168.1.109', details: 'Flyer_Messe_2026.pdf hochgeladen', created_at: minutesAgo(40) },
  { id: 'aud-005', action: 'settings_changed', user_id: IDS.users.stefan, user_name: 'Stefan Vogel', ip: '192.168.1.100', details: 'CORS-Einstellungen aktualisiert', created_at: hoursAgo(1) },
  { id: 'aud-006', action: 'login', user_id: IDS.users.markus, user_name: 'Markus Weber', ip: '192.168.1.102', details: 'Desktop App Login', created_at: hoursAgo(1) },
  { id: 'aud-007', action: 'user_role_changed', user_id: IDS.users.stefan, user_name: 'Stefan Vogel', ip: '192.168.1.100', details: 'Felix Krause: Rolle auf "Senior Developer" geaendert', created_at: hoursAgo(2) },
  { id: 'aud-008', action: 'login', user_id: IDS.users.julia, user_name: 'Julia Hofmann', ip: '192.168.1.112', details: 'Desktop App Login', created_at: hoursAgo(2) },
  { id: 'aud-009', action: 'deal_created', user_id: IDS.users.thomas, user_name: 'Thomas Meier', ip: '192.168.1.103', details: 'Deal "KI Beratung" erstellt', created_at: hoursAgo(3) },
  { id: 'aud-010', action: 'invoice_sent', user_id: IDS.users.michael, user_name: 'Petra Zimmermann', ip: '192.168.1.113', details: 'Rechnung RE-2026-080 versendet', created_at: hoursAgo(3) },
  { id: 'aud-011', action: 'document_deleted', user_id: IDS.users.felix, user_name: 'Felix Krause', ip: '192.168.1.105', details: 'Entwurf_alt.pdf gelöscht', created_at: hoursAgo(4) },
  { id: 'aud-012', action: 'login', user_id: IDS.users.nina, user_name: 'Elena Schuster', ip: '192.168.1.115', details: 'Desktop App Login', created_at: hoursAgo(4) },
  { id: 'aud-013', action: 'contact_updated', user_id: IDS.users.laura, user_name: 'Sabine Fischer', ip: '192.168.1.110', details: 'Kontakt: Hans Müller — E-Mail aktualisiert', created_at: hoursAgo(5) },
  { id: 'aud-014', action: 'project_created', user_id: IDS.users.elena, user_name: 'Sarah Beck', ip: '192.168.1.118', details: 'Projekt "Datenschutz Audit" erstellt', created_at: hoursAgo(5) },
  { id: 'aud-015', action: 'login_failed', user_id: null, user_name: 'unbekannt', ip: '203.0.113.42', details: 'Fehlgeschlagener Login: admin@techvision.de', created_at: hoursAgo(6) },
  { id: 'aud-016', action: 'password_changed', user_id: IDS.users.lena, user_name: 'Lena Braun', ip: '192.168.1.109', details: 'Passwort geaendert', created_at: hoursAgo(6) },
  { id: 'aud-017', action: 'login', user_id: IDS.users.thomas, user_name: 'Thomas Meier', ip: '192.168.1.103', details: 'Desktop App Login', created_at: hoursAgo(7) },
  { id: 'aud-018', action: 'deal_updated', user_id: IDS.users.laura, user_name: 'Sabine Fischer', ip: '192.168.1.110', details: 'Deal "Web Redesign" Betrag aktualisiert', created_at: hoursAgo(8) },
  { id: 'aud-019', action: 'settings_changed', user_id: IDS.users.markus, user_name: 'Markus Weber', ip: '192.168.1.102', details: 'Rate Limiting konfiguriert: 100 req/min', created_at: hoursAgo(9) },
  { id: 'aud-020', action: 'document_uploaded', user_id: IDS.users.sophie, user_name: 'Sophie Lang', ip: '192.168.1.108', details: 'Vertrag_Vorlage.docx hochgeladen', created_at: hoursAgo(10) },
  { id: 'aud-021', action: 'login', user_id: IDS.users.felix, user_name: 'Felix Krause', ip: '192.168.1.105', details: 'Desktop App Login', created_at: hoursAgo(12) },
  { id: 'aud-022', action: 'contact_created', user_id: IDS.users.laura, user_name: 'Sabine Fischer', ip: '192.168.1.110', details: 'Kontakt: Lisa Winkler erstellt', created_at: hoursAgo(14) },
  { id: 'aud-023', action: 'invoice_created', user_id: IDS.users.michael, user_name: 'Petra Zimmermann', ip: '192.168.1.113', details: 'Rechnung RE-2026-080 erstellt', created_at: daysAgo(1) },
  { id: 'aud-024', action: 'login', user_id: IDS.users.lena, user_name: 'Lena Braun', ip: '192.168.1.109', details: 'Desktop App Login', created_at: daysAgo(1) },
  { id: 'aud-025', action: 'deal_stage_changed', user_id: IDS.users.thomas, user_name: 'Thomas Meier', ip: '192.168.1.103', details: 'Deal "ERP Migration" -> Proposal', created_at: daysAgo(1) },
  { id: 'aud-026', action: 'login', user_id: IDS.users.sophie, user_name: 'Sophie Lang', ip: '192.168.1.108', details: 'Desktop App Login', created_at: daysAgo(1) },
  { id: 'aud-027', action: 'document_uploaded', user_id: IDS.users.markus, user_name: 'Markus Weber', ip: '192.168.1.102', details: 'API_Dokumentation.docx hochgeladen', created_at: daysAgo(1) },
  { id: 'aud-028', action: 'settings_changed', user_id: IDS.users.stefan, user_name: 'Stefan Vogel', ip: '192.168.1.100', details: 'Backup-Zeitplan geaendert: 02:00 UTC', created_at: daysAgo(1) },
  { id: 'aud-029', action: 'login_failed', user_id: null, user_name: 'unbekannt', ip: '198.51.100.17', details: 'Fehlgeschlagener Login: test@example.com', created_at: daysAgo(1) },
  { id: 'aud-030', action: 'login', user_id: IDS.users.michael, user_name: 'Petra Zimmermann', ip: '192.168.1.113', details: 'Desktop App Login', created_at: daysAgo(2) },
  { id: 'aud-031', action: 'contact_updated', user_id: IDS.users.thomas, user_name: 'Thomas Meier', ip: '192.168.1.103', details: 'Kontakt: Werner Koch — Telefon aktualisiert', created_at: daysAgo(2) },
  { id: 'aud-032', action: 'deal_created', user_id: IDS.users.laura, user_name: 'Sabine Fischer', ip: '192.168.1.110', details: 'Deal "Netzwerk Upgrade" erstellt', created_at: daysAgo(2) },
  { id: 'aud-033', action: 'login', user_id: IDS.users.elena, user_name: 'Sarah Beck', ip: '192.168.1.118', details: 'Desktop App Login', created_at: daysAgo(2) },
  { id: 'aud-034', action: 'document_uploaded', user_id: IDS.users.nina, user_name: 'Elena Schuster', ip: '192.168.1.115', details: 'Onboarding_Checkliste.pdf hochgeladen', created_at: daysAgo(2) },
  { id: 'aud-035', action: 'invoice_paid', user_id: IDS.users.michael, user_name: 'Petra Zimmermann', ip: '192.168.1.113', details: 'Rechnung RE-2026-077 als bezahlt markiert', created_at: daysAgo(3) },
  { id: 'aud-036', action: 'login', user_id: IDS.users.jan, user_name: 'Jonas Schmitt', ip: '192.168.1.117', details: 'Desktop App Login', created_at: daysAgo(3) },
  { id: 'aud-037', action: 'contact_deleted', user_id: IDS.users.thomas, user_name: 'Thomas Meier', ip: '192.168.1.103', details: 'Kontakt: Duplikat "Test Kontakt" gelöscht', created_at: daysAgo(3) },
  { id: 'aud-038', action: 'deal_updated', user_id: IDS.users.thomas, user_name: 'Thomas Meier', ip: '192.168.1.103', details: 'Deal "Support Vertrag" Wert angepasst', created_at: daysAgo(3) },
  { id: 'aud-039', action: 'login', user_id: IDS.users.david, user_name: 'Kevin Baumann', ip: '192.168.1.111', details: 'Desktop App Login', created_at: daysAgo(4) },
  { id: 'aud-040', action: 'settings_changed', user_id: IDS.users.markus, user_name: 'Markus Weber', ip: '192.168.1.102', details: 'E-Mail-SMTP konfiguriert', created_at: daysAgo(4) },
  { id: 'aud-041', action: 'document_uploaded', user_id: IDS.users.thomas, user_name: 'Thomas Meier', ip: '192.168.1.103', details: 'NDA_Rhein_Consulting.pdf hochgeladen', created_at: daysAgo(4) },
  { id: 'aud-042', action: 'login', user_id: IDS.users.christian, user_name: 'Martin Wolf', ip: '192.168.1.116', details: 'Desktop App Login', created_at: daysAgo(4) },
  { id: 'aud-043', action: 'contact_created', user_id: IDS.users.thomas, user_name: 'Thomas Meier', ip: '192.168.1.103', details: 'Kontakt: Franz Egger erstellt', created_at: daysAgo(5) },
  { id: 'aud-044', action: 'login_failed', user_id: null, user_name: 'unbekannt', ip: '203.0.113.88', details: 'Fehlgeschlagener Login: admin@test.com (3 Versuche)', created_at: daysAgo(5) },
  { id: 'aud-045', action: 'deal_stage_changed', user_id: IDS.users.laura, user_name: 'Sabine Fischer', ip: '192.168.1.110', details: 'Deal "Hosting Vertrag" -> Won', created_at: daysAgo(5) },
  { id: 'aud-046', action: 'login', user_id: IDS.users.anna, user_name: 'Julia Hofmann', ip: '192.168.1.112', details: 'Desktop App Login', created_at: daysAgo(6) },
  { id: 'aud-047', action: 'invoice_sent', user_id: IDS.users.michael, user_name: 'Petra Zimmermann', ip: '192.168.1.113', details: 'Rechnung RE-2026-079 versendet', created_at: daysAgo(6) },
  { id: 'aud-048', action: 'document_uploaded', user_id: IDS.users.felix, user_name: 'Felix Krause', ip: '192.168.1.105', details: 'Testbericht_v2.pdf hochgeladen', created_at: daysAgo(6) },
  { id: 'aud-049', action: 'project_updated', user_id: IDS.users.elena, user_name: 'Sarah Beck', ip: '192.168.1.118', details: 'Projekt "Hub V2" Status aktualisiert', created_at: daysAgo(7) },
  { id: 'aud-050', action: 'login', user_id: IDS.users.stefan, user_name: 'Stefan Vogel', ip: '192.168.1.100', details: 'Desktop App Login', created_at: daysAgo(7) },
]

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
  // Audit logs (paginated)
  http.get(`${API}/api/v1/security/audit`, ({ request }) => {
    const url = new URL(request.url)
    const page = parseInt(url.searchParams.get('page') || '1')
    const perPage = parseInt(url.searchParams.get('per_page') || '20')
    const action = url.searchParams.get('action')
    const userId = url.searchParams.get('user_id')

    let filtered = [...auditLogs]
    if (action) {
      filtered = filtered.filter((l) => l.action === action)
    }
    if (userId) {
      filtered = filtered.filter((l) => l.user_id === userId)
    }

    const start = (page - 1) * perPage
    const paginated = filtered.slice(start, start + perPage)

    return HttpResponse.json({ logs: paginated, total: filtered.length, page, per_page: perPage })
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
