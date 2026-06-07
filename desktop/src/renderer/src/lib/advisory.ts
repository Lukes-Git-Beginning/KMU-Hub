/**
 * Advisory protocol domain model ("Beratungsprotokoll").
 *
 * Legally-grounded field model for financial advisory documentation in DE
 * (FinVermV §16/§18, WpHG, VVG/IDD, ESG per EU-DelegVO 2021/1253).
 * See `.planning/kontakte-p8-beratungsprotokoll-spec.md` for the field spec.
 *
 * Enums expose stable string values + i18n label keys so they stay translatable
 * and reusable across the CRM (risk class also surfaces in CRM settings).
 */

// ---------------------------------------------------------------------------
// Shared option helper
// ---------------------------------------------------------------------------

export interface EnumOption<T extends string> {
  value: T
  /** i18n key for the human-readable label. */
  labelKey: string
}

// ---------------------------------------------------------------------------
// Risk class — PRIIP SRI 1-7 (Summary Risk Indicator)
// ---------------------------------------------------------------------------

export type SriClass = 1 | 2 | 3 | 4 | 5 | 6 | 7

/** SRI 1-7 with practice mapping + semantic colour token for badges. */
export const SRI_CLASSES: { value: SriClass; labelKey: string; color: string }[] = [
  { value: 1, labelKey: 'advisory.sri.1', color: '#16a34a' },
  { value: 2, labelKey: 'advisory.sri.2', color: '#65a30d' },
  { value: 3, labelKey: 'advisory.sri.3', color: '#ca8a04' },
  { value: 4, labelKey: 'advisory.sri.4', color: '#d97706' },
  { value: 5, labelKey: 'advisory.sri.5', color: '#ea580c' },
  { value: 6, labelKey: 'advisory.sri.6', color: '#dc2626' },
  { value: 7, labelKey: 'advisory.sri.7', color: '#b91c1c' },
]

export function sriColor(value: SriClass): string {
  return SRI_CLASSES.find((c) => c.value === value)?.color ?? '#6b7280'
}

// ---------------------------------------------------------------------------
// Section 1 — Gesprächskopf
// ---------------------------------------------------------------------------

export type AdvisoryLocation = 'office' | 'phone' | 'video' | 'onsite'
export const ADVISORY_LOCATIONS: EnumOption<AdvisoryLocation>[] = [
  { value: 'office', labelKey: 'advisory.location.office' },
  { value: 'phone', labelKey: 'advisory.location.phone' },
  { value: 'video', labelKey: 'advisory.location.video' },
  { value: 'onsite', labelKey: 'advisory.location.onsite' },
]

export type AdvisoryOccasion = 'initial' | 'followup' | 'event'
export const ADVISORY_OCCASIONS: EnumOption<AdvisoryOccasion>[] = [
  { value: 'initial', labelKey: 'advisory.occasion.initial' },
  { value: 'followup', labelKey: 'advisory.occasion.followup' },
  { value: 'event', labelKey: 'advisory.occasion.event' },
]

export type CustomerCategory = 'private' | 'professional'
export const CUSTOMER_CATEGORIES: EnumOption<CustomerCategory>[] = [
  { value: 'private', labelKey: 'advisory.customerCategory.private' },
  { value: 'professional', labelKey: 'advisory.customerCategory.professional' },
]

// ---------------------------------------------------------------------------
// Section 3 — Kenntnisse & Erfahrungen
// ---------------------------------------------------------------------------

export type AssetClass = 'stocks' | 'funds' | 'etf' | 'bonds' | 'derivatives' | 'realestate' | 'crypto'
export const ASSET_CLASSES: EnumOption<AssetClass>[] = [
  { value: 'stocks', labelKey: 'advisory.assetClass.stocks' },
  { value: 'funds', labelKey: 'advisory.assetClass.funds' },
  { value: 'etf', labelKey: 'advisory.assetClass.etf' },
  { value: 'bonds', labelKey: 'advisory.assetClass.bonds' },
  { value: 'derivatives', labelKey: 'advisory.assetClass.derivatives' },
  { value: 'realestate', labelKey: 'advisory.assetClass.realestate' },
  { value: 'crypto', labelKey: 'advisory.assetClass.crypto' },
]

// ---------------------------------------------------------------------------
// Section 5 — Anlageziele & Risikoprofil
// ---------------------------------------------------------------------------

export type InvestmentPurpose = 'retirement' | 'liquidity' | 'growth' | 'saving' | 'speculation'
export const INVESTMENT_PURPOSES: EnumOption<InvestmentPurpose>[] = [
  { value: 'retirement', labelKey: 'advisory.purpose.retirement' },
  { value: 'liquidity', labelKey: 'advisory.purpose.liquidity' },
  { value: 'growth', labelKey: 'advisory.purpose.growth' },
  { value: 'saving', labelKey: 'advisory.purpose.saving' },
  { value: 'speculation', labelKey: 'advisory.purpose.speculation' },
]

export type InvestmentHorizon = 'lt1' | '1to3' | '3to5' | '5to10' | 'gt10'
export const INVESTMENT_HORIZONS: EnumOption<InvestmentHorizon>[] = [
  { value: 'lt1', labelKey: 'advisory.horizon.lt1' },
  { value: '1to3', labelKey: 'advisory.horizon.1to3' },
  { value: '3to5', labelKey: 'advisory.horizon.3to5' },
  { value: '5to10', labelKey: 'advisory.horizon.5to10' },
  { value: 'gt10', labelKey: 'advisory.horizon.gt10' },
]

// ---------------------------------------------------------------------------
// Section 8 — Abschluss & Compliance
// ---------------------------------------------------------------------------

export type DeliveryForm = 'paper' | 'email' | 'portal'
export const DELIVERY_FORMS: EnumOption<DeliveryForm>[] = [
  { value: 'paper', labelKey: 'advisory.deliveryForm.paper' },
  { value: 'email', labelKey: 'advisory.deliveryForm.email' },
  { value: 'portal', labelKey: 'advisory.deliveryForm.portal' },
]

/** Standardised warning checklist (IDD / FinVermV mandatory disclosures). */
export type AdvisoryWarning = 'risk' | 'costs' | 'conflicts' | 'pastPerformance' | 'liquidity'
export const ADVISORY_WARNINGS: EnumOption<AdvisoryWarning>[] = [
  { value: 'risk', labelKey: 'advisory.warning.risk' },
  { value: 'costs', labelKey: 'advisory.warning.costs' },
  { value: 'conflicts', labelKey: 'advisory.warning.conflicts' },
  { value: 'pastPerformance', labelKey: 'advisory.warning.pastPerformance' },
  { value: 'liquidity', labelKey: 'advisory.warning.liquidity' },
]

// ---------------------------------------------------------------------------
// Product line (Section 6)
// ---------------------------------------------------------------------------

export interface AdvisoryProduct {
  id: string
  name: string
  isin?: string
  category?: string
  riskClass: SriClass
  opportunities?: string
  risks: string
  costsOneTime?: number
  costsRunning?: number
  recommended: boolean
}

// ---------------------------------------------------------------------------
// Full protocol
// ---------------------------------------------------------------------------

export type AdvisoryProtocolStatus = 'draft' | 'finalized'

export interface AdvisoryProtocol {
  id: string
  contactId: string
  status: AdvisoryProtocolStatus
  createdAt: string
  updatedAt: string

  // 1 — Gesprächskopf
  date: string
  timeFrom: string
  timeTo: string
  location: AdvisoryLocation
  advisor: string
  occasion: AdvisoryOccasion
  occasionNote: string
  customerCategory: CustomerCategory

  // 2 — Kunde & Profil
  birthDate: string
  maritalStatus: string
  taxStatus: string

  // 3 — Kenntnisse & Erfahrungen
  knownAssetClasses: AssetClass[]
  pastTransactions: string
  financialEducation: string
  professionalExperience: string
  selfAssessment: number | null

  // 4 — Finanzielle Situation
  monthlyNetIncome: number | null
  recurringLiabilities: number | null
  liquidAssets: number | null
  currentInvestments: number | null
  realEstate: string
  existingInsurance: string
  maxLossCapacityAbs: number | null
  maxLossCapacityPct: number | null

  // 5 — Anlageziele & Risikoprofil
  investmentPurpose: InvestmentPurpose[]
  horizon: InvestmentHorizon | ''
  riskTolerance: string
  riskCapacity: string
  riskClass: SriClass
  esgPreference: boolean
  esgDetails: string
  oneTimeAmount: number | null
  monthlySavings: number | null

  // 6 — Besprochene Produkte
  products: AdvisoryProduct[]

  // 7 — Empfehlung & Geeignetheit
  recommendationSummary: string
  suitabilityReasoning: string
  goalReference: string
  alternatives: string
  notRecommended: string

  // 8 — Abschluss & Compliance
  mainConcerns: string
  warningsGiven: AdvisoryWarning[]
  documentDelivered: boolean
  documentDeliveredDate: string
  deliveryForm: DeliveryForm
  advisorSignature: string
  customerConfirmation: boolean
  documentWaiver: boolean
  followupDate: string
  internalNotes: string
}

/** Computed duration in minutes from timeFrom/timeTo (HH:MM). */
export function protocolDurationMin(timeFrom: string, timeTo: string): number | null {
  const parse = (s: string) => {
    const m = /^(\d{1,2}):(\d{2})$/.exec(s.trim())
    if (!m) return null
    return Number(m[1]) * 60 + Number(m[2])
  }
  const from = parse(timeFrom)
  const to = parse(timeTo)
  if (from == null || to == null) return null
  const diff = to - from
  return diff >= 0 ? diff : null
}

/** Factory for a blank draft protocol bound to a contact. */
export function emptyProtocol(contactId: string, id: string, advisor = ''): AdvisoryProtocol {
  return {
    id,
    contactId,
    status: 'draft',
    createdAt: '',
    updatedAt: '',
    date: '',
    timeFrom: '',
    timeTo: '',
    location: 'office',
    advisor,
    occasion: 'initial',
    occasionNote: '',
    customerCategory: 'private',
    birthDate: '',
    maritalStatus: '',
    taxStatus: '',
    knownAssetClasses: [],
    pastTransactions: '',
    financialEducation: '',
    professionalExperience: '',
    selfAssessment: null,
    monthlyNetIncome: null,
    recurringLiabilities: null,
    liquidAssets: null,
    currentInvestments: null,
    realEstate: '',
    existingInsurance: '',
    maxLossCapacityAbs: null,
    maxLossCapacityPct: null,
    investmentPurpose: [],
    horizon: '',
    riskTolerance: '',
    riskCapacity: '',
    riskClass: 3,
    esgPreference: false,
    esgDetails: '',
    oneTimeAmount: null,
    monthlySavings: null,
    products: [],
    recommendationSummary: '',
    suitabilityReasoning: '',
    goalReference: '',
    alternatives: '',
    notRecommended: '',
    mainConcerns: '',
    warningsGiven: [],
    documentDelivered: false,
    documentDeliveredDate: '',
    deliveryForm: 'email',
    advisorSignature: '',
    customerConfirmation: false,
    documentWaiver: false,
    followupDate: '',
    internalNotes: '',
  }
}
