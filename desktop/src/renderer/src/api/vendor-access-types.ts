/**
 * Vendor Access (RBAC R-5 B) — Typen für den GDAP-light-v3-Anbieter-Zugang.
 *
 * Zentria beantragt zeitlich befristeten Zugang; der Kunde genehmigt/lehnt ab
 * oder schlägt einen anderen Starttermin vor.
 */

// ---------------------------------------------------------------------------
// Bereichs-Katalog
// ---------------------------------------------------------------------------

export interface VendorAccessArea {
  /** Technische ID (für scope-Arrays in Requests). */
  id: string
  /** i18n-Key für den Anzeigenamen. */
  labelKey: string
  /** Cosmi-Module, die dieser Bereich umfasst (informativ). */
  modules: string[]
  /** Sensibel → Warnhinweis + Extra-Checkbox bei Genehmigung. */
  sensitive: boolean
}

export const VENDOR_ACCESS_AREAS: VendorAccessArea[] = [
  {
    id: 'crm',
    labelKey: 'rbac.vendorAccess.area.crm',
    modules: ['crm', 'kommunikation', 'mail'],
    sensitive: false,
  },
  {
    id: 'finance',
    labelKey: 'rbac.vendorAccess.area.finance',
    modules: ['finance', 'berichte'],
    sensitive: false,
  },
  {
    id: 'documents',
    labelKey: 'rbac.vendorAccess.area.documents',
    modules: ['documents', 'wiki', 'formulare'],
    sensitive: false,
  },
  {
    id: 'calendar',
    labelKey: 'rbac.vendorAccess.area.calendar',
    modules: ['kalender', 'zeiterfassung'],
    sensitive: false,
  },
  {
    id: 'industry',
    labelKey: 'rbac.vendorAccess.area.industry',
    modules: ['inventar', 'einkauf', 'produktion', 'schichten', 'fuhrpark', 'vermietung', 'rapporte', 'dialer', 'vertraege', 'helpdesk'],
    sensitive: false,
  },
  {
    id: 'admin',
    labelKey: 'rbac.vendorAccess.area.admin',
    modules: ['admin', 'security', 'settings', 'automatisierung', 'integrationen'],
    sensitive: false,
  },
  {
    id: 'team',
    labelKey: 'rbac.vendorAccess.area.team',
    modules: ['team'],
    sensitive: false,
  },
  {
    id: 'hr_data',
    labelKey: 'rbac.vendorAccess.area.hr_data',
    modules: ['team'],
    sensitive: true,
  },
  {
    id: 'salary',
    labelKey: 'rbac.vendorAccess.area.salary',
    modules: ['team', 'finance'],
    sensitive: true,
  },
]

/** Default-Preset „Setup-Standard": alle non-sensitiven Bereiche. */
export const SETUP_STANDARD_SCOPE: string[] = VENDOR_ACCESS_AREAS
  .filter((a) => !a.sensitive)
  .map((a) => a.id)

// ---------------------------------------------------------------------------
// Status
// ---------------------------------------------------------------------------

export type VendorAccessStatus =
  | 'pending'
  | 'counter_proposed'
  | 'active'
  | 'declined'
  | 'expired'
  | 'revoked'
  | 'completed'

// ---------------------------------------------------------------------------
// Request-Entität
// ---------------------------------------------------------------------------

export interface VendorAccessAgent {
  name: string
}

export interface VendorAccessRequest {
  id: string
  /** Anlass / Kurztitel der Anfrage. */
  reason: string
  /** Ausführliche Arbeitsbeschreibung. */
  description: string
  /** Optionale Ticket-Referenz (z. B. „Support-Ticket #4711"). */
  ticket_ref?: string
  /** Benannte Zentria-Mitarbeiter. */
  agents: VendorAccessAgent[]
  /** Gewählte Area-IDs. */
  scope: string[]
  /** Geplanter Start (ISO-Date-String). */
  requested_start: string
  /** Dauer in Tagen (max 30). */
  duration_days: number
  /** Berechnetes Ablaufdatum (ISO-Date-String). */
  expires_at: string
  status: VendorAccessStatus
  /** Vom Kunden vorgeschlagener alternativer Start (counter_proposed). */
  counter_proposed_start?: string
  approved_at?: string
  approved_by?: string
  /** Kundenseitige Bestätigung des sensitiven Datenzugriffs. */
  sensitive_ack?: boolean
  revoked_at?: string
  revoked_by?: string
  completed_at?: string
  created_at: string
}

// ---------------------------------------------------------------------------
// API-Payloads
// ---------------------------------------------------------------------------

export interface ApproveVendorAccessInput {
  sensitive_ack?: boolean
}

export interface CounterProposeVendorAccessInput {
  proposed_start: string
}

export interface VendorAccessListResponse {
  requests: VendorAccessRequest[]
}
