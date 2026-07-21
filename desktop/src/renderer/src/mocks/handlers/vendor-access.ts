/**
 * MSW handler — Vendor Access / GDAP-light v3 (RBAC R-5 B).
 *
 * Stateful in-memory registry (VENDOR_ACCESS_MAP). Alle Status-Transitionen
 * schreiben einen writeAuditEvent.
 *
 * Endpoints:
 *   GET  /api/v1/vendor-access
 *   POST /api/v1/vendor-access/:id/approve    { sensitive_ack? }
 *   POST /api/v1/vendor-access/:id/decline
 *   POST /api/v1/vendor-access/:id/counter-propose  { proposed_start }
 *   POST /api/v1/vendor-access/:id/revoke
 */
import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { getDemoSessionUserId } from '../data/rbac'
import { USER_DISPLAY_NAMES } from '../data/shared-ids'
import { writeAuditEvent } from '../data/audit-events'
import { daysAgo, daysFromNow } from '../data/date-helpers'
import {
  VENDOR_ACCESS_AREAS,
  SETUP_STANDARD_SCOPE,
} from '@/api/vendor-access-types'
import type { VendorAccessRequest } from '@/api/vendor-access-types'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Berechnet expires_at aus Start + Dauer. */
function calcExpiry(start: string, durationDays: number): string {
  const d = new Date(start)
  d.setDate(d.getDate() + durationDays)
  return d.toISOString().split('T')[0]
}

/** Prüft ob irgendein Area im Scope sensitive ist. */
function hasSensitiveScope(scope: string[]): boolean {
  return scope.some((id) => VENDOR_ACCESS_AREAS.find((a) => a.id === id)?.sensitive === true)
}

// ---------------------------------------------------------------------------
// Stateful registry
// ---------------------------------------------------------------------------

const VENDOR_ACCESS_MAP = new Map<string, VendorAccessRequest>()

// ---------------------------------------------------------------------------
// Seeds
// ---------------------------------------------------------------------------

const seeds: VendorAccessRequest[] = [
  // (a) Pending „Ersteinrichtung" — Setup-Standard-Scope, 14 Tage, Start in 3 Tagen
  {
    id: 'va-seed-001',
    reason: 'Ersteinrichtung nach Prozessanalyse',
    description:
      'Konfiguration der Module gemäß den Ergebnissen der Onsite-Prozessanalyse vom 15.07.2026. Einrichtung von Workflows, Benutzerrollen, Dokumentenvorlagen und Integrationen.',
    ticket_ref: undefined,
    agents: [{ name: 'Luke S.' }, { name: 'Darien R.' }],
    scope: SETUP_STANDARD_SCOPE,
    requested_start: daysFromNow(3),
    duration_days: 14,
    expires_at: calcExpiry(daysFromNow(3), 14),
    status: 'pending',
    created_at: daysAgo(1),
  },
  // (b) Pending „Fehlerbehebung Lohnabrechnungs-Export" — mit Ticket, sensitiver Scope
  {
    id: 'va-seed-002',
    reason: 'Fehlerbehebung Lohnabrechnungs-Export',
    description:
      'Analyse und Behebung des gemeldeten Fehlers beim DATEV-Export der Lohnabrechnungen. Betroffen sind die Monate Mai und Juni 2026.',
    ticket_ref: 'Support-Ticket #4711',
    agents: [{ name: 'Darien R.' }],
    scope: ['finance', 'salary'],
    requested_start: daysFromNow(1),
    duration_days: 3,
    expires_at: calcExpiry(daysFromNow(1), 3),
    status: 'pending',
    created_at: daysAgo(0),
  },
  // (c) Active „Modul-Einrichtung Einkauf" — 7 Tage, 4 Tage Rest
  {
    id: 'va-seed-003',
    reason: 'Modul-Einrichtung Einkauf',
    description:
      'Einrichtung des Einkaufsmoduls: Lieferantenstamm importieren, Genehmigungsworkflow konfigurieren, Bestellnummernkreis einrichten.',
    ticket_ref: undefined,
    agents: [{ name: 'Luke S.' }],
    scope: ['crm', 'documents', 'industry', 'admin'],
    requested_start: daysAgo(3),
    duration_days: 7,
    expires_at: calcExpiry(daysAgo(3), 7),
    status: 'active',
    approved_at: daysAgo(3),
    approved_by: 'Stefan Vogel',
    created_at: daysAgo(4),
  },
  // (d) Verlauf: expired
  {
    id: 'va-seed-004',
    reason: 'Initiales Onboarding Setup',
    description: 'Grundkonfiguration nach Vertragsabschluss.',
    ticket_ref: undefined,
    agents: [{ name: 'Luke S.' }, { name: 'Darien R.' }],
    scope: SETUP_STANDARD_SCOPE,
    requested_start: daysAgo(45),
    duration_days: 14,
    expires_at: calcExpiry(daysAgo(45), 14),
    status: 'expired',
    approved_at: daysAgo(45),
    approved_by: 'Stefan Vogel',
    created_at: daysAgo(46),
  },
  // (d) Verlauf: revoked
  {
    id: 'va-seed-005',
    reason: 'Notfall-Fehleranalyse Datenbank',
    description: 'Analyse eines gemeldeten Performance-Problems.',
    ticket_ref: 'Support-Ticket #3290',
    agents: [{ name: 'Luke S.' }],
    scope: ['admin', 'documents'],
    requested_start: daysAgo(20),
    duration_days: 7,
    expires_at: calcExpiry(daysAgo(20), 7),
    status: 'revoked',
    approved_at: daysAgo(20),
    approved_by: 'Stefan Vogel',
    revoked_at: daysAgo(18),
    revoked_by: 'Stefan Vogel',
    created_at: daysAgo(21),
  },
  // (d) Verlauf: completed
  {
    id: 'va-seed-006',
    reason: 'CRM-Datenmigration',
    description: 'Übernahme der Kontakt- und Auftragsdaten aus dem Altsystem.',
    ticket_ref: undefined,
    agents: [{ name: 'Darien R.' }],
    scope: ['crm', 'documents'],
    requested_start: daysAgo(60),
    duration_days: 5,
    expires_at: calcExpiry(daysAgo(60), 5),
    status: 'completed',
    approved_at: daysAgo(60),
    approved_by: 'Stefan Vogel',
    completed_at: daysAgo(56),
    created_at: daysAgo(61),
  },
]

for (const s of seeds) {
  VENDOR_ACCESS_MAP.set(s.id, s)
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

export const vendorAccessHandlers = [
  // GET /api/v1/vendor-access
  http.get(`${API}/api/v1/vendor-access`, () => {
    const requests = Array.from(VENDOR_ACCESS_MAP.values()).sort(
      (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
    )
    return HttpResponse.json({ requests })
  }),

  // POST /api/v1/vendor-access/:id/approve
  http.post(`${API}/api/v1/vendor-access/:id/approve`, async ({ params, request }) => {
    const req = VENDOR_ACCESS_MAP.get(String(params.id))
    if (!req) return HttpResponse.json({ error: 'Not found' }, { status: 404 })
    if (req.status !== 'pending' && req.status !== 'counter_proposed') {
      return HttpResponse.json({ error: 'Invalid status transition' }, { status: 409 })
    }

    const body = (await request.json().catch(() => ({}))) as { sensitive_ack?: boolean }
    const isSensitive = hasSensitiveScope(req.scope)

    if (isSensitive && !body.sensitive_ack) {
      return HttpResponse.json({ error: 'sensitive_ack_required' }, { status: 422 })
    }

    const sessionUserId = getDemoSessionUserId()
    const approvedBy = USER_DISPLAY_NAMES[sessionUserId] ?? 'Admin'

    const updated: VendorAccessRequest = {
      ...req,
      status: 'active',
      approved_at: new Date().toISOString(),
      approved_by: approvedBy,
      sensitive_ack: isSensitive ? true : undefined,
    }
    VENDOR_ACCESS_MAP.set(req.id, updated)

    writeAuditEvent({
      action: 'vendor_access.approved',
      target: req.reason,
      targetType: 'vendor_access',
      newValue: { status: 'active', approved_by: approvedBy },
      extraDetails: { request_id: req.id, sensitive_ack: isSensitive },
    })

    return HttpResponse.json({ request: updated })
  }),

  // POST /api/v1/vendor-access/:id/decline
  http.post(`${API}/api/v1/vendor-access/:id/decline`, ({ params }) => {
    const req = VENDOR_ACCESS_MAP.get(String(params.id))
    if (!req) return HttpResponse.json({ error: 'Not found' }, { status: 404 })
    if (req.status !== 'pending' && req.status !== 'counter_proposed') {
      return HttpResponse.json({ error: 'Invalid status transition' }, { status: 409 })
    }

    const updated: VendorAccessRequest = { ...req, status: 'declined' }
    VENDOR_ACCESS_MAP.set(req.id, updated)

    writeAuditEvent({
      action: 'vendor_access.declined',
      target: req.reason,
      targetType: 'vendor_access',
      newValue: { status: 'declined' },
      extraDetails: { request_id: req.id },
    })

    return HttpResponse.json({ request: updated })
  }),

  // POST /api/v1/vendor-access/:id/counter-propose  { proposed_start }
  http.post(`${API}/api/v1/vendor-access/:id/counter-propose`, async ({ params, request }) => {
    const req = VENDOR_ACCESS_MAP.get(String(params.id))
    if (!req) return HttpResponse.json({ error: 'Not found' }, { status: 404 })
    if (req.status !== 'pending') {
      return HttpResponse.json({ error: 'Invalid status transition' }, { status: 409 })
    }

    const body = (await request.json()) as { proposed_start: string }
    if (!body.proposed_start) {
      return HttpResponse.json({ error: 'proposed_start required' }, { status: 400 })
    }

    const updated: VendorAccessRequest = {
      ...req,
      status: 'counter_proposed',
      counter_proposed_start: body.proposed_start,
    }
    VENDOR_ACCESS_MAP.set(req.id, updated)

    writeAuditEvent({
      action: 'vendor_access.counter_proposed',
      target: req.reason,
      targetType: 'vendor_access',
      newValue: { status: 'counter_proposed', proposed_start: body.proposed_start },
      extraDetails: { request_id: req.id },
    })

    // Zentria bestätigt simuliert nach ~15 s
    setTimeout(() => {
      const current = VENDOR_ACCESS_MAP.get(req.id)
      if (!current || current.status !== 'counter_proposed') return
      const newExpiry = calcExpiry(body.proposed_start, current.duration_days)
      const confirmed: VendorAccessRequest = {
        ...current,
        status: 'active',
        requested_start: body.proposed_start,
        expires_at: newExpiry,
        approved_at: new Date().toISOString(),
        counter_proposed_start: undefined,
      }
      VENDOR_ACCESS_MAP.set(req.id, confirmed)
      writeAuditEvent({
        action: 'vendor_access.approved',
        target: current.reason,
        targetType: 'vendor_access',
        actorName: 'Zentria (auto)',
        newValue: { status: 'active', start: body.proposed_start },
        extraDetails: { request_id: req.id, auto_confirmed: true },
      })
    }, 15000)

    return HttpResponse.json({ request: updated })
  }),

  // POST /api/v1/vendor-access/:id/revoke
  http.post(`${API}/api/v1/vendor-access/:id/revoke`, ({ params }) => {
    const req = VENDOR_ACCESS_MAP.get(String(params.id))
    if (!req) return HttpResponse.json({ error: 'Not found' }, { status: 404 })
    if (req.status !== 'active') {
      return HttpResponse.json({ error: 'Invalid status transition' }, { status: 409 })
    }

    const sessionUserId = getDemoSessionUserId()
    const revokedBy = USER_DISPLAY_NAMES[sessionUserId] ?? 'Admin'

    const updated: VendorAccessRequest = {
      ...req,
      status: 'revoked',
      revoked_at: new Date().toISOString(),
      revoked_by: revokedBy,
    }
    VENDOR_ACCESS_MAP.set(req.id, updated)

    writeAuditEvent({
      action: 'vendor_access.revoked',
      target: req.reason,
      targetType: 'vendor_access',
      newValue: { status: 'revoked', revoked_by: revokedBy },
      extraDetails: { request_id: req.id },
    })

    return HttpResponse.json({ request: updated })
  }),
]
