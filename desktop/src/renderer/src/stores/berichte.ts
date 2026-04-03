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

export interface KPIDrilldownRow {
  date: string
  label: string
  amount: string
  status: string
}

export interface ComparisonBar {
  label: string
  current: number
  previous: number
}

export interface DATEVRow {
  position: string
  label: string
  currentMonth: number
  previousMonth: number
  yearToDate: number
}

interface BerichteStore {
  kpis: KPI[]
  chartData: ChartBar[]
  modules: ModuleOption[]
  savedReports: SavedReport[]
  scheduledReports: ScheduledReport[]
  kpiDrilldownData: Record<string, KPIDrilldownRow[]>
  comparisonChartData: ComparisonBar[]
  datevBWA: DATEVRow[]
  datevSuSa: DATEVRow[]
}

const MOCK_KPIS: KPI[] = [
  { id: 'kpi-1', label: 'Umsatz (Monat)', value: '284.350', unit: 'EUR', changePercent: 12.4, moduleId: 'finanzen' },
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
  { id: 'finanzen', name: 'Finanzen' },
  { id: 'crm', name: 'CRM / Kontakte' },
  { id: 'helpdesk', name: 'Helpdesk' },
  { id: 'inventar', name: 'Inventar' },
  { id: 'produktion', name: 'Produktion' },
  { id: 'einkauf', name: 'Einkauf' },
  { id: 'fuhrpark', name: 'Fuhrpark' },
  { id: 'schichten', name: 'Schichtplanung' },
]

const MOCK_SAVED_REPORTS: SavedReport[] = [
  { id: 'rpt-1', name: 'Monatlicher Umsatzbericht', description: 'Umsatz nach Kunde, Produkt und Region aufgeschlüsselt. Inkl. Vorjahresvergleich.', module: 'Finanzen', createdBy: 'Karin Pfister', createdAt: '2026-01-05T09:00:00' },
  { id: 'rpt-2', name: 'Offene Posten Übersicht', description: 'Alle offenen Debitoren und Kreditoren mit Fälligkeitsdatum und Mahnstufe.', module: 'Finanzen', createdBy: 'Karin Pfister', createdAt: '2026-01-10T14:30:00' },
  { id: 'rpt-3', name: 'Helpdesk SLA-Report', description: 'SLA-Einhaltung nach Priorität und Kategorie. Durchschnittliche Reaktions- und Lösungszeiten.', module: 'Helpdesk', createdBy: 'Marco Hartmann', createdAt: '2026-02-01T11:00:00' },
]

const MOCK_SCHEDULED_REPORTS: ScheduledReport[] = [
  { id: 'sched-1', reportId: 'rpt-1', name: 'Monatlicher Umsatzbericht', schedule: 'monthly', recipients: ['geschaeftsleitung@firma.de', 'finanzen@firma.de'], lastRun: '2026-02-01T06:00:00', active: true },
  { id: 'sched-2', reportId: 'rpt-3', name: 'Helpdesk SLA-Report', schedule: 'weekly', recipients: ['it-leitung@firma.de', 'support@firma.de'], lastRun: '2026-02-10T06:00:00', active: true },
]

const MOCK_DRILLDOWN: Record<string, KPIDrilldownRow[]> = {
  'kpi-1': [
    { date: '2026-02-18', label: 'Weber GmbH', amount: '42.500 EUR', status: 'Bezahlt' },
    { date: '2026-02-15', label: 'Müller AG', amount: '28.900 EUR', status: 'Bezahlt' },
    { date: '2026-02-12', label: 'Schmidt KG', amount: '65.200 EUR', status: 'Offen' },
    { date: '2026-02-08', label: 'Fischer OHG', amount: '31.750 EUR', status: 'Bezahlt' },
    { date: '2026-02-03', label: 'Bauer & Co.', amount: '116.000 EUR', status: 'Teilzahlung' },
  ],
  'kpi-2': [
    { date: '2026-02-20', label: 'Projekt Alpha', amount: '12 Aufträge', status: 'In Arbeit' },
    { date: '2026-02-18', label: 'Projekt Beta', amount: '8 Aufträge', status: 'In Arbeit' },
    { date: '2026-02-15', label: 'Wartung Q1', amount: '15 Aufträge', status: 'Geplant' },
    { date: '2026-02-10', label: 'Notfall-Support', amount: '7 Aufträge', status: 'Dringend' },
    { date: '2026-02-05', label: 'Kleinaufträge', amount: '5 Aufträge', status: 'In Arbeit' },
  ],
  'kpi-3': [
    { date: '2026-02-20', label: 'Support-Bewertungen', amount: '4.8 / 5.0', status: '23 Bewertungen' },
    { date: '2026-02-15', label: 'Projektabschluss-Umfrage', amount: '4.5 / 5.0', status: '8 Bewertungen' },
    { date: '2026-02-10', label: 'NPS Score', amount: '+42', status: '15 Antworten' },
    { date: '2026-02-05', label: 'Onboarding-Feedback', amount: '4.7 / 5.0', status: '5 Bewertungen' },
  ],
  'kpi-4': [
    { date: '2026-02-20', label: 'Kritisch', amount: '0.8 Std', status: '14 Tickets' },
    { date: '2026-02-20', label: 'Hoch', amount: '1.5 Std', status: '38 Tickets' },
    { date: '2026-02-20', label: 'Mittel', amount: '3.2 Std', status: '62 Tickets' },
    { date: '2026-02-20', label: 'Niedrig', amount: '5.1 Std', status: '27 Tickets' },
  ],
  'kpi-5': [
    { date: '2026-02-20', label: 'Rohmaterial', amount: '12.3x', status: 'Über Ziel' },
    { date: '2026-02-20', label: 'Halbfertig', amount: '6.8x', status: 'Im Ziel' },
    { date: '2026-02-20', label: 'Fertigware', amount: '9.1x', status: 'Im Ziel' },
    { date: '2026-02-20', label: 'Ersatzteile', amount: '4.2x', status: 'Unter Ziel' },
  ],
  'kpi-6': [
    { date: '2026-02-20', label: 'Linie A', amount: '1.2%', status: 'Im Ziel' },
    { date: '2026-02-20', label: 'Linie B', amount: '3.5%', status: 'Über Ziel' },
    { date: '2026-02-20', label: 'Linie C', amount: '1.8%', status: 'Im Ziel' },
    { date: '2026-02-20', label: 'Nacharbeit', amount: '0.9%', status: 'Unter Ziel' },
  ],
}

const MOCK_COMPARISON: ComparisonBar[] = [
  { label: 'Sep', current: 210, previous: 195 },
  { label: 'Okt', current: 245, previous: 220 },
  { label: 'Nov', current: 228, previous: 240 },
  { label: 'Dez', current: 260, previous: 235 },
  { label: 'Jan', current: 275, previous: 250 },
  { label: 'Feb', current: 284, previous: 258 },
]

const MOCK_BWA: DATEVRow[] = [
  { position: '1', label: 'Umsatzerloese', currentMonth: 284350, previousMonth: 275100, yearToDate: 559450 },
  { position: '2', label: 'Bestandsveränderung', currentMonth: -2100, previousMonth: 1500, yearToDate: -600 },
  { position: '3', label: 'Gesamtleistung', currentMonth: 282250, previousMonth: 276600, yearToDate: 558850 },
  { position: '4', label: 'Materialaufwand', currentMonth: -85200, previousMonth: -82400, yearToDate: -167600 },
  { position: '5', label: 'Rohertrag', currentMonth: 197050, previousMonth: 194200, yearToDate: 391250 },
  { position: '6', label: 'Personalkosten', currentMonth: -112000, previousMonth: -112000, yearToDate: -224000 },
  { position: '7', label: 'Raumkosten', currentMonth: -8500, previousMonth: -8500, yearToDate: -17000 },
  { position: '8', label: 'Sonstige betriebl. Aufwendungen', currentMonth: -24300, previousMonth: -22800, yearToDate: -47100 },
  { position: '9', label: 'Betriebsergebnis (EBIT)', currentMonth: 52250, previousMonth: 50900, yearToDate: 103150 },
  { position: '10', label: 'Zinsergebnis', currentMonth: -1200, previousMonth: -1200, yearToDate: -2400 },
  { position: '11', label: 'Ergebnis vor Steuern', currentMonth: 51050, previousMonth: 49700, yearToDate: 100750 },
]

const MOCK_SUSA: DATEVRow[] = [
  { position: '0800', label: 'Bank', currentMonth: 145200, previousMonth: 132800, yearToDate: 145200 },
  { position: '1000', label: 'Kasse', currentMonth: 2350, previousMonth: 1800, yearToDate: 2350 },
  { position: '1200', label: 'Forderungen aLuL', currentMonth: 87500, previousMonth: 92300, yearToDate: 87500 },
  { position: '1600', label: 'Verbindlichkeiten aLuL', currentMonth: -45600, previousMonth: -48200, yearToDate: -45600 },
  { position: '1800', label: 'USt-Verbindlichkeiten', currentMonth: -12400, previousMonth: -11800, yearToDate: -12400 },
  { position: '4400', label: 'Erloese 19%', currentMonth: 238950, previousMonth: 231200, yearToDate: 470150 },
  { position: '4300', label: 'Erloese 7%', currentMonth: 45400, previousMonth: 43900, yearToDate: 89300 },
  { position: '6300', label: 'Loehne und Gehälter', currentMonth: -95000, previousMonth: -95000, yearToDate: -190000 },
  { position: '6400', label: 'Sozialversicherung', currentMonth: -17000, previousMonth: -17000, yearToDate: -34000 },
]

export const useBerichteStore = create<BerichteStore>()(
  persist(
    () => ({
      kpis: MOCK_KPIS,
      chartData: MOCK_CHART_DATA,
      modules: MOCK_MODULES,
      savedReports: MOCK_SAVED_REPORTS,
      scheduledReports: MOCK_SCHEDULED_REPORTS,
      kpiDrilldownData: MOCK_DRILLDOWN,
      comparisonChartData: MOCK_COMPARISON,
      datevBWA: MOCK_BWA,
      datevSuSa: MOCK_SUSA,
    }),
    { name: 'kmuhub-berichte' },
  ),
)
