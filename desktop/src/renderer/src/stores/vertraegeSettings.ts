import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { ContractType } from '@/stores/vertraege'

/**
 * Tenant-wide vertraege (Verträge) settings — administered by a module lead
 * via the "Für alle" section of the vertraege settings panel. Mock-first:
 * persisted locally until the backend persists tenant settings for the
 * vertraege module (tenant_settings, module_id='vertraege') — tracked in
 * backend-handover-luke.md. Enabled types, defaults and reminder presets
 * already apply for real inside the module (new-contract dialog).
 */

export const ALL_CONTRACT_TYPES: ContractType[] = [
  'mietvertrag',
  'liefervertrag',
  'servicevertrag',
  'arbeitsvertrag',
  'lizenz',
  'versicherung',
]

interface VertraegeSettingsState {
  /** Contract types offered in the new-contract dialog (tenant taxonomy). */
  enabledTypes: ContractType[]
  /** Default duration for new contracts, in months. */
  defaultDurationMonths: number
  /** Default notice period for new contracts, in days. */
  defaultNoticePeriodDays: number
  /** Default renewal mode for new contracts. */
  defaultRenewal: 'auto' | 'manual'
  /** Number-range format, e.g. "V-{JAHR}-{NR}". Tokens: {JAHR}, {MONAT}, {NR}. */
  numberFormat: string
  /** Running counter for the {NR} token, advanced on each auto-numbered contract. */
  numberCounter: number
  /** Tenant-wide reminder pre-selection (days before end) for new contracts. */
  tenantReminderDays: number[]

  toggleType: (type: ContractType) => void
  setDefaultDurationMonths: (months: number) => void
  setDefaultNoticePeriodDays: (days: number) => void
  setDefaultRenewal: (r: 'auto' | 'manual') => void
  setNumberFormat: (f: string) => void
  toggleTenantReminderDay: (days: number) => void
  /** Next contract number from format + counter, without advancing the counter. */
  peekContractNumber: () => string
  /** Advance the {NR} counter (call after a contract used the auto number). */
  bumpNumberCounter: () => void
}

/** Expand the number-range tokens against the current date + counter. */
export function formatContractNumber(format: string, counter: number): string {
  const now = new Date()
  return format
    .replace(/\{JAHR\}/g, String(now.getFullYear()))
    .replace(/\{MONAT\}/g, String(now.getMonth() + 1).padStart(2, '0'))
    .replace(/\{NR\}/g, String(counter).padStart(3, '0'))
}

export const useVertraegeSettingsStore = create<VertraegeSettingsState>()(
  persist(
    (set, get) => ({
      enabledTypes: [...ALL_CONTRACT_TYPES],
      defaultDurationMonths: 12,
      defaultNoticePeriodDays: 30,
      defaultRenewal: 'auto',
      numberFormat: 'V-{JAHR}-{NR}',
      numberCounter: 1,
      tenantReminderDays: [30],

      toggleType: (type) =>
        set((s) => {
          const enabled = s.enabledTypes.includes(type)
          // Never disable the last remaining type — the dialog needs one option.
          if (enabled && s.enabledTypes.length === 1) return s
          return {
            enabledTypes: enabled
              ? s.enabledTypes.filter((t) => t !== type)
              : [...s.enabledTypes, type],
          }
        }),
      setDefaultDurationMonths: (defaultDurationMonths) => set({ defaultDurationMonths }),
      setDefaultNoticePeriodDays: (defaultNoticePeriodDays) => set({ defaultNoticePeriodDays }),
      setDefaultRenewal: (defaultRenewal) => set({ defaultRenewal }),
      setNumberFormat: (numberFormat) => set({ numberFormat }),
      peekContractNumber: () => formatContractNumber(get().numberFormat, get().numberCounter),
      bumpNumberCounter: () => set((s) => ({ numberCounter: s.numberCounter + 1 })),
      toggleTenantReminderDay: (days) =>
        set((s) => ({
          tenantReminderDays: s.tenantReminderDays.includes(days)
            ? s.tenantReminderDays.filter((d) => d !== days)
            : [...s.tenantReminderDays, days].sort((a, b) => a - b),
        })),
    }),
    { name: 'cosmi-vertraege-settings' },
  ),
)
