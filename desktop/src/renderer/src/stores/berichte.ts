import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface KPI {
  id: string
  label: string
  value: string
  unit: string
  changePercent: number
  moduleId: string
}

export interface ChartBar {
  label: string
  value: number
}

export interface ModuleOption {
  id: string
  name: string
}

export interface SavedReport {
  id: string
  name: string
  description: string
  module: string
  createdBy: string
  createdAt: string
}

export interface ScheduledReport {
  id: string
  reportId: string
  name: string
  schedule: 'daily' | 'weekly' | 'monthly' | 'quarterly'
  recipients: string[]
  lastRun: string
  active: boolean
}

interface BerichteStore {
  kpis: KPI[]
  chartData: ChartBar[]
  modules: ModuleOption[]
  savedReports: SavedReport[]
  scheduledReports: ScheduledReport[]
}

const MOCK_KPIS: KPI[] = [
  { id: 'kpi-1', label: 'Umsatz (Monat)', value: '284\'350', unit: 'CHF', changePercent: 12.4, moduleId: 'buchhaltung' },
  { id: 'kpi-2', label: 'Offene Aufträge', value: '47', unit: '', changePercent: -3.2, moduleId: 'aufträge' },
  { id: 'kpi-3', label: 'Kundenzufriedenheit', value: '4.6', unit: '/ 5.0', changePercent: 0.2, moduleId: 'crm' },
  { id: 'kpi-4', label: 'Durchschn. Reaktionszeit', value: '2.4', unit: 'Std.', changePercent: -18.0, moduleId: 'helpdesk' },
  { id: 'kpi-5', label: 'Lagerumschlag', value: '8.3', unit: 'x / Jahr', changePercent: 1.1, moduleId: 'inventar' },
  { id: 'kpi-6', label: 'Ausschussquote', value: '2.1', unit: '%', changePercent: 0.4, moduleId: 'produktion' },
]

const MOCK_CHART_DATA: ChartBar[] = [
  { label: 'Sep', value: 210 },
  { label: 'Okt', value: 245 },
  { label: 'Nov', value: 228 },
  { label: 'Dez', value: 260 },
  { label: 'Jan', value: 275 },
  { label: 'Feb', value: 284 },
]

const MOCK_MODULES: ModuleOption[] = [
  { id: 'buchhaltung', name: 'Buchhaltung' },
  { id: 'crm', name: 'CRM / Kontakte' },
  { id: 'helpdesk', name: 'Helpdesk' },
  { id: 'inventar', name: 'Inventar' },
  { id: 'produktion', name: 'Produktion' },
  { id: 'einkauf', name: 'Einkauf' },
  { id: 'fuhrpark', name: 'Fuhrpark' },
  { id: 'schichten', name: 'Schichtplanung' },
]

const MOCK_SAVED_REPORTS: SavedReport[] = [
  { id: 'rpt-1', name: 'Monatlicher Umsatzbericht', description: 'Umsatz nach Kunde, Produkt und Region aufgeschlüsselt. Inkl. Vorjahresvergleich.', module: 'Buchhaltung', createdBy: 'Karin Pfister', createdAt: '2026-01-05T09:00:00' },
  { id: 'rpt-2', name: 'Offene Posten Übersicht', description: 'Alle offenen Debitoren und Kreditoren mit Fälligkeitsdatum und Mahnstufe.', module: 'Buchhaltung', createdBy: 'Karin Pfister', createdAt: '2026-01-10T14:30:00' },
  { id: 'rpt-3', name: 'Helpdesk SLA-Report', description: 'SLA-Einhaltung nach Priorität und Kategorie. Durchschnittliche Reaktions- und Lösungszeiten.', module: 'Helpdesk', createdBy: 'Marco Hartmann', createdAt: '2026-02-01T11:00:00' },
]

const MOCK_SCHEDULED_REPORTS: ScheduledReport[] = [
  { id: 'sched-1', reportId: 'rpt-1', name: 'Monatlicher Umsatzbericht', schedule: 'monthly', recipients: ['geschaeftsleitung@firma.ch', 'buchhaltung@firma.ch'], lastRun: '2026-02-01T06:00:00', active: true },
  { id: 'sched-2', reportId: 'rpt-3', name: 'Helpdesk SLA-Report', schedule: 'weekly', recipients: ['it-leitung@firma.ch', 'support@firma.ch'], lastRun: '2026-02-10T06:00:00', active: true },
]

export const useBerichteStore = create<BerichteStore>()(
  persist(
    () => ({
      kpis: MOCK_KPIS,
      chartData: MOCK_CHART_DATA,
      modules: MOCK_MODULES,
      savedReports: MOCK_SAVED_REPORTS,
      scheduledReports: MOCK_SCHEDULED_REPORTS,
    }),
    { name: 'kmuhub-berichte' },
  ),
)
