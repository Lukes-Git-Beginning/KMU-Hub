import { create } from 'zustand'
import { persist } from 'zustand/middleware'

/**
 * Tenant-wide work (Projekte/Aufgaben) settings — administered by a module lead
 * via the "Für alle" section of the work settings panel. Mock-first: persisted
 * locally until the backend provides label / custom-field / template APIs.
 * Backend gaps tracked in .planning/backend-gaps.md.
 */

export interface WorkLabel {
  id: string
  name: string
  color: string
}

export type WorkFieldType = 'text' | 'number' | 'date' | 'dropdown' | 'checkbox'

export interface WorkCustomField {
  id: string
  name: string
  type: WorkFieldType
  required: boolean
  options?: string[]
  sortOrder: number
}

export interface WorkDefaultStatus {
  id: string
  name: string
  color: string
}

const uid = (prefix: string) => `${prefix}-${Date.now()}-${Math.floor(Math.random() * 1e4)}`

interface WorkSettingsState {
  /** Label taxonomy — assignable to tasks (P3 builds the chip UI on this). */
  labels: WorkLabel[]
  /** Custom-field definitions applied to tasks. */
  customFields: WorkCustomField[]
  /** Default status set seeded into new projects. */
  defaultStatuses: WorkDefaultStatus[]
  /** Whether time entries are billable by default. */
  billableByDefault: boolean
  /** Default hourly rate used for hours→invoice (EUR). */
  defaultHourlyRate: number

  addLabel: (name: string, color: string) => void
  updateLabel: (id: string, patch: Partial<Omit<WorkLabel, 'id'>>) => void
  removeLabel: (id: string) => void

  addCustomField: (field: Omit<WorkCustomField, 'id' | 'sortOrder'>) => void
  updateCustomField: (id: string, patch: Partial<Omit<WorkCustomField, 'id'>>) => void
  removeCustomField: (id: string) => void

  addDefaultStatus: (name: string, color: string) => void
  updateDefaultStatus: (id: string, patch: Partial<Omit<WorkDefaultStatus, 'id'>>) => void
  removeDefaultStatus: (id: string) => void

  setBillableByDefault: (b: boolean) => void
  setDefaultHourlyRate: (rate: number) => void
}

export const useWorkSettingsStore = create<WorkSettingsState>()(
  persist(
    (set) => ({
      labels: [
        { id: 'wl-bug', name: 'Bug', color: '#ef4444' },
        { id: 'wl-feature', name: 'Feature', color: '#3b82f6' },
        { id: 'wl-design', name: 'Design', color: '#a78bfa' },
        { id: 'wl-blocked', name: 'Blockiert', color: '#f59e0b' },
      ],
      customFields: [
        { id: 'wcf-effort', name: 'Aufwand', type: 'dropdown', required: false, options: ['S', 'M', 'L', 'XL'], sortOrder: 0 },
        { id: 'wcf-sprint', name: 'Sprint', type: 'text', required: false, sortOrder: 1 },
      ],
      defaultStatuses: [
        { id: 'wds-1', name: 'Offen', color: '#94a3b8' },
        { id: 'wds-2', name: 'In Bearbeitung', color: '#3b82f6' },
        { id: 'wds-3', name: 'Review', color: '#a78bfa' },
        { id: 'wds-4', name: 'Erledigt', color: '#22c55e' },
      ],
      billableByDefault: true,
      defaultHourlyRate: 120,

      addLabel: (name, color) =>
        set((s) => ({ labels: [...s.labels, { id: uid('wl'), name, color }] })),
      updateLabel: (id, patch) =>
        set((s) => ({ labels: s.labels.map((l) => (l.id === id ? { ...l, ...patch } : l)) })),
      removeLabel: (id) => set((s) => ({ labels: s.labels.filter((l) => l.id !== id) })),

      addCustomField: (field) =>
        set((s) => ({
          customFields: [...s.customFields, { ...field, id: uid('wcf'), sortOrder: s.customFields.length }],
        })),
      updateCustomField: (id, patch) =>
        set((s) => ({ customFields: s.customFields.map((f) => (f.id === id ? { ...f, ...patch } : f)) })),
      removeCustomField: (id) =>
        set((s) => ({ customFields: s.customFields.filter((f) => f.id !== id) })),

      addDefaultStatus: (name, color) =>
        set((s) => ({ defaultStatuses: [...s.defaultStatuses, { id: uid('wds'), name, color }] })),
      updateDefaultStatus: (id, patch) =>
        set((s) => ({ defaultStatuses: s.defaultStatuses.map((d) => (d.id === id ? { ...d, ...patch } : d)) })),
      removeDefaultStatus: (id) =>
        set((s) => ({ defaultStatuses: s.defaultStatuses.filter((d) => d.id !== id) })),

      setBillableByDefault: (billableByDefault) => set({ billableByDefault }),
      setDefaultHourlyRate: (defaultHourlyRate) => set({ defaultHourlyRate }),
    }),
    { name: 'cosmi-work-settings' },
  ),
)
