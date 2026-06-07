import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type {
  Gender,
  MaritalStatus,
  TaxClass,
  Confession,
  SvStatus,
  EmploymentType,
  PayType,
} from '@/lib/payroll-enums'

/**
 * Payroll master data ("Lohn-Stammdaten") per employee — overlay store.
 *
 * Digital "Personalstammblatt": all data DATEV-Lohn / the payroll office needs
 * for SV-Anmeldung + correct payroll. Feeds the Lohnvorbereitung (PayrollPrepPanel).
 *
 * MOCK-FIRST overlay keyed by employeeId. These fields belong on EmployeeProfile;
 * Luke extends hr-types + API + DSGVO visibility — see backend-gaps.md. Until then
 * this client overlay is the single source. Shape is deliberately flat so it maps
 * 1:1 onto the future EmployeeProfile columns.
 */

export interface PayrollMasterData {
  // 1. Persönliche Daten (steuerrelevant)
  birthDate?: string
  birthPlace?: string
  gender?: Gender
  nationality?: string
  maritalStatus?: MaritalStatus

  // 2. Steuer
  taxId?: string // IdNr, 11-stellig
  taxClass?: TaxClass
  childAllowances?: number // Kinderfreibeträge (Komma erlaubt)
  confession?: Confession

  // 3. Sozialversicherung
  svNumber?: string // 12-stellig
  healthInsurance?: string // Krankenkasse Name
  healthInsuranceNr?: string // Betriebsnummer
  parentProperty?: boolean // Elterneigenschaft → PV-Zuschlag
  svStatus?: SvStatus

  // 4. Beschäftigung
  weeklyHours?: number
  employmentType?: EmploymentType
  jobKey?: string // Tätigkeitsschlüssel, 9-stellig
  endDate?: string // Befristung / Austritt (optional)

  // 5. Bezüge & Bank
  payType?: PayType
  monthlySalary?: number // bei Festgehalt
  hourlyWage?: number // bei Stundenlohn
  specialPayments?: string // 13. Gehalt, Urlaubsgeld (Freitext)
  payrollGroupId?: string // → payrollSettings.groups
  iban?: string
  bic?: string
  accountHolder?: string // falls abweichend
}

/** Required fields for SV-Anmeldung + payroll handover (Pflichtfelder). */
const REQUIRED_FIELDS: (keyof PayrollMasterData)[] = [
  'birthDate',
  'gender',
  'nationality',
  'maritalStatus',
  'taxId',
  'svNumber',
  'healthInsurance',
  'weeklyHours',
  'employmentType',
  'jobKey',
  'payType',
  'iban',
]

function hasValue(v: unknown): boolean {
  if (v === undefined || v === null) return false
  if (typeof v === 'string') return v.trim().length > 0
  if (typeof v === 'number') return !Number.isNaN(v)
  return true
}

/** Pure completeness check usable outside the store (e.g. in PayrollPrepPanel). */
export function isPayrollComplete(data: PayrollMasterData | undefined): boolean {
  return missingRequiredFields(data).length === 0
}

/** Names of missing required fields (for warnings before export). */
export function missingRequiredFields(data: PayrollMasterData | undefined): (keyof PayrollMasterData)[] {
  if (!data) return [...REQUIRED_FIELDS]
  const missing = REQUIRED_FIELDS.filter((f) => !hasValue(data[f]))
  // Pay amount: at least one of monthlySalary / hourlyWage must be present.
  if (!hasValue(data.monthlySalary) && !hasValue(data.hourlyWage)) {
    missing.push(data.payType === 'hourly' ? 'hourlyWage' : 'monthlySalary')
  }
  return missing
}

interface PayrollMasterDataState {
  data: Record<string, PayrollMasterData>
  get: (employeeId: string) => PayrollMasterData | undefined
  set: (employeeId: string, patch: Partial<PayrollMasterData>) => void
  clear: (employeeId: string) => void
}

export const usePayrollMasterDataStore = create<PayrollMasterDataState>()(
  persist(
    (set, getState) => ({
      data: {},
      get: (employeeId) => getState().data[employeeId],
      set: (employeeId, patch) =>
        set((s) => ({
          data: {
            ...s.data,
            [employeeId]: { ...s.data[employeeId], ...patch },
          },
        })),
      clear: (employeeId) =>
        set((s) => {
          const next = { ...s.data }
          delete next[employeeId]
          return { data: next }
        }),
    }),
    { name: 'cosmi-payroll-masterdata' },
  ),
)
