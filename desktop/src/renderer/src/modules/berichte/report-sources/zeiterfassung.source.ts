import { Timer } from 'lucide-react'
import type { ReportSource } from './types'
import { generateRows, intBetween, isoDaysAgo, pick } from './sample-utils'

const DEPARTMENT = [
  { value: 'vertrieb', label: 'Vertrieb' },
  { value: 'technik', label: 'Technik' },
  { value: 'verwaltung', label: 'Verwaltung' },
  { value: 'marketing', label: 'Marketing' },
  { value: 'produktion', label: 'Produktion' },
  { value: 'support', label: 'Support' },
]
const ACTIVITY = [
  { value: 'development', label: 'Entwicklung' },
  { value: 'meeting', label: 'Besprechung' },
  { value: 'support', label: 'Support' },
  { value: 'travel', label: 'Reise' },
  { value: 'admin', label: 'Verwaltung' },
  { value: 'sales', label: 'Akquise' },
]
const STATUS = [
  { value: 'approved', label: 'Genehmigt' },
  { value: 'pending', label: 'Ausstehend' },
  { value: 'draft', label: 'Entwurf' },
]
const BILLABLE = [
  { value: 'yes', label: 'Abrechenbar' },
  { value: 'no', label: 'Nicht abrechenbar' },
]
const EMPLOYEES = [
  'Lena Hofer',
  'Marco Reis',
  'Tom Keller',
  'Nina Wolf',
  'Sara Brandt',
  'Jonas Vogel',
  'Mara Lang',
  'Felix Arnold',
]
const CUSTOMERS = [
  'Müller Maschinenbau GmbH',
  'Fintech Solutions AG',
  'Atelier Nord Design',
  'Pharma Vitalis GmbH',
  'Bau & Plan AG',
  'Greentech Energie GmbH',
]

export const zeiterfassungSource: ReportSource = {
  id: 'zeiterfassung',
  module: 'zeiterfassung',
  labelKey: 'berichte.sources.zeiterfassung.label',
  label: 'Zeiterfassung',
  icon: Timer,
  description: 'Arbeitszeit, Auslastung und Überstunden',
  defaultViz: 'bar',
  fields: [
    { key: 'entry_date', labelKey: 'berichte.fields.zeiterfassung.entry_date', label: 'Datum', dataType: 'date', role: 'dimension' },
    { key: 'department', labelKey: 'berichte.fields.zeiterfassung.department', label: 'Abteilung', dataType: 'enum', role: 'dimension', enumValues: DEPARTMENT },
    { key: 'employee', labelKey: 'berichte.fields.zeiterfassung.employee', label: 'Mitarbeiter', dataType: 'string', role: 'dimension' },
    { key: 'customer', labelKey: 'berichte.fields.zeiterfassung.customer', label: 'Kunde', dataType: 'string', role: 'dimension' },
    { key: 'activity', labelKey: 'berichte.fields.zeiterfassung.activity', label: 'Tätigkeit', dataType: 'enum', role: 'dimension', enumValues: ACTIVITY },
    { key: 'status', labelKey: 'berichte.fields.zeiterfassung.status', label: 'Status', dataType: 'enum', role: 'dimension', enumValues: STATUS },
    { key: 'billable', labelKey: 'berichte.fields.zeiterfassung.billable', label: 'Abrechenbarkeit', dataType: 'enum', role: 'dimension', enumValues: BILLABLE },
    { key: 'total_minutes', labelKey: 'berichte.fields.zeiterfassung.total_minutes', label: 'Erfasste Zeit (Min)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'sum' },
    { key: 'net_work_minutes', labelKey: 'berichte.fields.zeiterfassung.net_work_minutes', label: 'Nettoarbeitszeit (Min)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'sum' },
    { key: 'break_minutes', labelKey: 'berichte.fields.zeiterfassung.break_minutes', label: 'Pausenzeit (Min)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'sum' },
    { key: 'overtime_minutes', labelKey: 'berichte.fields.zeiterfassung.overtime_minutes', label: 'Überstunden (Min)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'sum' },
  ],
  sampleRows: (count = 200) =>
    generateRows(620, count, (r) => {
      const total = intBetween(r(), 120, 600)
      const breakMin = intBetween(r(), 0, 60)
      const net = Math.max(0, total - breakMin)
      const overtime = net > 480 ? net - 480 : 0
      return {
        entry_date: isoDaysAgo(intBetween(r(), 0, 120)),
        department: pick(DEPARTMENT, r()).value,
        employee: pick(EMPLOYEES, r()),
        customer: pick(CUSTOMERS, r()),
        activity: pick(ACTIVITY, r()).value,
        status: pick(STATUS, r()).value,
        billable: pick(BILLABLE, r()).value,
        total_minutes: total,
        net_work_minutes: net,
        break_minutes: breakMin,
        overtime_minutes: overtime,
      }
    }),
}
