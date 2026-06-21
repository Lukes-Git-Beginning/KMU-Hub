import { Users } from 'lucide-react'
import type { ReportSource } from './types'
import { generateRows, intBetween, isoDaysAgo, pick } from './sample-utils'

const DEPARTMENT = [
  { value: 'vertrieb', label: 'Vertrieb' },
  { value: 'technik', label: 'Technik' },
  { value: 'verwaltung', label: 'Verwaltung' },
  { value: 'marketing', label: 'Marketing' },
  { value: 'produktion', label: 'Produktion' },
  { value: 'support', label: 'Support' },
  { value: 'geschaeftsfuehrung', label: 'Geschäftsführung' },
]
const CONTRACT = [
  { value: 'full', label: 'Vollzeit' },
  { value: 'part', label: 'Teilzeit' },
  { value: 'mini', label: 'Minijob' },
  { value: 'working_student', label: 'Werkstudent' },
]
const STATUS = [
  { value: 'active', label: 'Aktiv' },
  { value: 'probation', label: 'Probezeit' },
  { value: 'parental_leave', label: 'Elternzeit' },
  { value: 'left', label: 'Ausgeschieden' },
]
const LOCATION = [
  { value: 'muenchen', label: 'München' },
  { value: 'berlin', label: 'Berlin' },
  { value: 'hamburg', label: 'Hamburg' },
  { value: 'remote', label: 'Remote' },
]

export const hrSource: ReportSource = {
  id: 'hr',
  module: 'hr',
  labelKey: 'berichte.sources.hr.label',
  label: 'Mitarbeiter',
  icon: Users,
  description: 'Personalstand, Verträge und Abwesenheiten',
  defaultViz: 'bar',
  fields: [
    { key: 'department', labelKey: 'berichte.fields.hr.department', label: 'Abteilung', dataType: 'enum', role: 'dimension', enumValues: DEPARTMENT },
    { key: 'position', labelKey: 'berichte.fields.hr.position', label: 'Position', dataType: 'string', role: 'dimension' },
    { key: 'contract_type', labelKey: 'berichte.fields.hr.contract_type', label: 'Vertragsart', dataType: 'enum', role: 'dimension', enumValues: CONTRACT },
    { key: 'employment_status', labelKey: 'berichte.fields.hr.employment_status', label: 'Beschäftigungsstatus', dataType: 'enum', role: 'dimension', enumValues: STATUS },
    { key: 'location', labelKey: 'berichte.fields.hr.location', label: 'Standort', dataType: 'enum', role: 'dimension', enumValues: LOCATION },
    { key: 'start_date', labelKey: 'berichte.fields.hr.start_date', label: 'Eintrittsdatum', dataType: 'date', role: 'dimension' },
    { key: 'weekly_hours', labelKey: 'berichte.fields.hr.weekly_hours', label: 'Wochenstunden', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'avg' },
    { key: 'tenure_months', labelKey: 'berichte.fields.hr.tenure_months', label: 'Betriebszugehörigkeit (Monate)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'avg' },
    { key: 'leave_total', labelKey: 'berichte.fields.hr.leave_total', label: 'Urlaubsanspruch (Tage)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'sum' },
    { key: 'leave_used', labelKey: 'berichte.fields.hr.leave_used', label: 'Urlaub genommen (Tage)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'sum' },
    { key: 'leave_remaining', labelKey: 'berichte.fields.hr.leave_remaining', label: 'Resturlaub (Tage)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'sum' },
    { key: 'absence_days', labelKey: 'berichte.fields.hr.absence_days', label: 'Krankheitstage', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'sum' },
  ],
  sampleRows: (count = 48) =>
    generateRows(610, count, (r) => {
      const contract = pick(CONTRACT, r())
      const weekly =
        contract.value === 'full'
          ? 40
          : contract.value === 'part'
            ? intBetween(r(), 18, 30)
            : contract.value === 'mini'
              ? 10
              : intBetween(r(), 12, 20)
      const leaveTotal = contract.value === 'full' ? 30 : intBetween(r(), 18, 28)
      const leaveUsed = intBetween(r(), 0, leaveTotal)
      return {
        department: pick(DEPARTMENT, r()).value,
        position: pick(
          ['Sachbearbeiter:in', 'Teamleitung', 'Spezialist:in', 'Junior', 'Senior', 'Werkstudent:in'],
          r(),
        ),
        contract_type: contract.value,
        employment_status: pick(STATUS, r()).value,
        location: pick(LOCATION, r()).value,
        start_date: isoDaysAgo(intBetween(r(), 30, 3650)),
        weekly_hours: weekly,
        tenure_months: intBetween(r(), 1, 180),
        leave_total: leaveTotal,
        leave_used: leaveUsed,
        leave_remaining: leaveTotal - leaveUsed,
        absence_days: intBetween(r(), 0, 14),
      }
    }),
}
