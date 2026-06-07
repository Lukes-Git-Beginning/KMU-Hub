import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * DATEV-Lohn / payroll-export settings (tenant scope, HR-Leiter/admin).
 *
 * Cosmi does the "vorbereitende Lohnabrechnung" and hands master + movement data
 * to DATEV (LODAS or Lohn und Gehalt). These settings configure that handover.
 * MOCK-FIRST. Backend: tenant_settings (module_id='team', key='payroll.*') +
 * actual DATEV file generation / Datenservice — see backend-gaps.md + team-datev-lohn-spec.md.
 */

export type PayrollTarget = 'lodas' | 'lug'
export type PayrollTransfer = 'file' | 'service'

/** Map a Cosmi wage category to a DATEV Lohnart number. */
export interface WageTypeMapping {
  id: string
  /** Human label of the Cosmi category (e.g. "Überstunden"). */
  label: string
  /** DATEV Lohnart-Nr (string, leading zeros allowed). */
  datevNr: string
}

/** Map an absence type to a DATEV Abwesenheitsschlüssel; export toggle. */
export interface AbsenceMapping {
  id: string
  label: string
  datevKey: string
  /** Whether this absence is included in the export. */
  exported: boolean
}

export interface PayrollGroup {
  id: string
  name: string
}

interface PayrollSettingsState {
  beraterNr: string
  mandantNr: string
  target: PayrollTarget
  transfer: PayrollTransfer
  wageTypes: WageTypeMapping[]
  absences: AbsenceMapping[]
  groups: PayrollGroup[]

  setConnection: (patch: Partial<Pick<PayrollSettingsState, 'beraterNr' | 'mandantNr' | 'target' | 'transfer'>>) => void
  setWageType: (id: string, datevNr: string) => void
  setAbsence: (id: string, patch: Partial<Pick<AbsenceMapping, 'datevKey' | 'exported'>>) => void
  addGroup: (name: string) => void
  removeGroup: (id: string) => void
}

export const usePayrollSettingsStore = create<PayrollSettingsState>()(
  persist(
    (set) => ({
      beraterNr: '',
      mandantNr: '',
      target: 'lug',
      transfer: 'file',
      // Seeds: typical wage categories with DATEV default suggestions.
      wageTypes: [
        { id: 'fixed', label: 'Festgehalt', datevNr: '0100' },
        { id: 'hourly', label: 'Stundenlohn', datevNr: '0200' },
        { id: 'overtime', label: 'Überstunden', datevNr: '0210' },
        { id: 'bonus', label: 'Bonus / Prämie', datevNr: '0400' },
        { id: 'benefit', label: 'Sachbezug', datevNr: '0500' },
      ],
      absences: [
        { id: 'vacation', label: 'Urlaub', datevKey: 'U', exported: true },
        { id: 'sick', label: 'Krankheit', datevKey: 'K', exported: true },
        { id: 'unpaid', label: 'Unbezahlt', datevKey: 'UB', exported: true },
      ],
      groups: [
        { id: 'salaried', name: 'Festangestellte' },
        { id: 'hourly', name: 'Stundenlöhner' },
      ],

      setConnection: (patch) => set((s) => ({ ...s, ...patch })),
      setWageType: (id, datevNr) =>
        set((s) => ({ wageTypes: s.wageTypes.map((w) => (w.id === id ? { ...w, datevNr } : w)) })),
      setAbsence: (id, patch) =>
        set((s) => ({ absences: s.absences.map((a) => (a.id === id ? { ...a, ...patch } : a)) })),
      addGroup: (name) =>
        set((s) => ({ groups: [...s.groups, { id: crypto.randomUUID(), name }] })),
      removeGroup: (id) => set((s) => ({ groups: s.groups.filter((g) => g.id !== id) })),
    }),
    { name: 'cosmi-payroll-settings' },
  ),
)
