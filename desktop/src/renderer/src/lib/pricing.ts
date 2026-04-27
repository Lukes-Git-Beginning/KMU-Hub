/**
 * Pricing library — single source of truth fuer Modul-Preise und Volume-Rabatte.
 * Spiegel zu .knowledge/pricing.md / docs/PRICING.md.
 *
 * Use:
 *   import { calculateMonthly, MODULE_PRICES, volumeDiscount } from '@/lib/pricing'
 */

/** Modul-IDs entsprechen den nav-items.ts IDs (kein Drift). */
export type ModuleId =
  | 'crm'
  | 'tasks'
  | 'finance' // Display-Name "Buchhaltung"
  | 'calendar'
  | 'documents'
  | 'chat'
  | 'meetings'
  | 'mail'
  | 'dialer'
  | 'team'
  | 'zeiterfassung'
  | 'schichten'
  | 'projects'
  | 'inventar'
  | 'einkauf'
  | 'helpdesk'
  | 'fuhrpark'
  | 'vertraege'
  | 'produktion'
  | 'berichte'
  | 'formulare'
  | 'wiki'
  | 'rapporte'
  | 'vermietung'

/** EUR pro User pro Monat. Quelle: .knowledge/pricing.md */
export const MODULE_PRICES: Record<ModuleId, number> = {
  // Kern
  crm: 6,
  tasks: 3,
  finance: 6, // Buchhaltung
  calendar: 2,
  documents: 2,
  // Kommunikation
  chat: 4,
  meetings: 4,
  mail: 3,
  dialer: 5,
  // Team
  team: 3,
  zeiterfassung: 3,
  schichten: 4,
  // Projekte / Industry
  projects: 5,
  inventar: 5,
  einkauf: 5,
  helpdesk: 5,
  fuhrpark: 5,
  vertraege: 5,
  produktion: 7,
  vermietung: 5,
  // Tools
  berichte: 3,
  formulare: 2,
  wiki: 2,
  rapporte: 3,
}

/**
 * Volume-Rabatt-Stufen (User-Anzahl im Tenant).
 * Quelle: .knowledge/pricing.md
 */
export interface VolumeTier {
  /** Min User-Count fuer diese Stufe (inklusiv). */
  minSeats: number
  /** Rabatt als Dezimal (0.05 = 5%). */
  discount: number
  /** Label fuer UI. */
  label: string
}

export const VOLUME_TIERS: VolumeTier[] = [
  { minSeats: 250, discount: 0.25, label: '250+ User' },
  { minSeats: 100, discount: 0.2, label: '100–249 User' },
  { minSeats: 50, discount: 0.15, label: '50–99 User' },
  { minSeats: 25, discount: 0.1, label: '25–49 User' },
  { minSeats: 10, discount: 0.05, label: '10–24 User' },
  { minSeats: 1, discount: 0, label: '1–9 User' },
]

export function volumeDiscount(seats: number): VolumeTier {
  return VOLUME_TIERS.find((t) => seats >= t.minSeats) ?? VOLUME_TIERS[VOLUME_TIERS.length - 1]
}

/** Support-Tier-Aufschlaege (EUR / Monat / Tenant, flat). */
export type SupportTier = 'standard' | 'priority' | 'enterprise'

export const SUPPORT_TIER_PRICES: Record<SupportTier, number> = {
  standard: 0,
  priority: 99,
  enterprise: 299,
}

export interface ModuleAssignment {
  /** Modul-ID. */
  moduleId: ModuleId
  /** Wieviele User dieses Modul nutzen (kann 0 sein). */
  assignedSeats: number
}

export interface BillingSummary {
  /** Bruttosumme (Module x User) vor Rabatt. */
  modulesGross: number
  /** Volume-Rabatt (negativ). */
  volumeDiscountAmount: number
  /** Support-Tier-Aufschlag. */
  supportFee: number
  /** Gesamtkosten / Monat netto. */
  monthlyTotal: number
  /** Aktiver Volume-Tier fuer Anzeige. */
  tier: VolumeTier
  /** Pro Modul aufgeschluesselt. */
  perModule: { moduleId: ModuleId; seats: number; pricePerSeat: number; subtotal: number }[]
}

/**
 * Berechnet die monatlichen Kosten basierend auf Modul-Zuweisungen + Gesamt-User-Count.
 *
 * @param assignments Pro Modul: wieviele User nutzen es. Module mit 0 seats werden ignoriert.
 * @param totalSeats Gesamt-User im Tenant (fuer Volume-Tier).
 * @param supportTier Support-Stufe (default: 'standard').
 */
export function calculateMonthly(
  assignments: ModuleAssignment[],
  totalSeats: number,
  supportTier: SupportTier = 'standard',
): BillingSummary {
  const tier = volumeDiscount(totalSeats)

  const perModule = assignments
    .filter((a) => a.assignedSeats > 0)
    .map((a) => {
      const pricePerSeat = MODULE_PRICES[a.moduleId]
      const subtotal = pricePerSeat * a.assignedSeats
      return {
        moduleId: a.moduleId,
        seats: a.assignedSeats,
        pricePerSeat,
        subtotal,
      }
    })

  const modulesGross = perModule.reduce((sum, m) => sum + m.subtotal, 0)
  const volumeDiscountAmount = -(modulesGross * tier.discount)
  const supportFee = SUPPORT_TIER_PRICES[supportTier]
  const monthlyTotal = modulesGross + volumeDiscountAmount + supportFee

  return {
    modulesGross,
    volumeDiscountAmount,
    supportFee,
    monthlyTotal,
    tier,
    perModule,
  }
}

/** Format helper. */
export function formatEUR(amount: number): string {
  return new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR' }).format(amount)
}
