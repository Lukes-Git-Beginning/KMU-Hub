import { ClipboardList } from 'lucide-react'
import type { ReportSource } from './types'
import { floatBetween, generateRows, intBetween, isoDaysAgo, pick } from './sample-utils'

const STATUS = [
  { value: 'approved', label: 'Genehmigt' },
  { value: 'submitted', label: 'Eingereicht' },
  { value: 'draft', label: 'Entwurf' },
  { value: 'rejected', label: 'Abgelehnt' },
]
const WORK_TYPE = [
  { value: 'assembly', label: 'Montage' },
  { value: 'maintenance', label: 'Wartung' },
  { value: 'repair', label: 'Reparatur' },
  { value: 'installation', label: 'Installation' },
  { value: 'inspection', label: 'Inspektion' },
]
const AUTHORS = ['Tom Keller', 'Marco Reis', 'Felix Arnold', 'Jonas Vogel', 'Mara Lang']
const REVIEWERS = ['Lena Hofer', 'Sara Brandt', 'Nina Wolf']
const PROJECTS = [
  'Baustelle Nord',
  'Projekt Hafencity',
  'Sanierung Altbau West',
  'Neubau Gewerbepark',
  'Wartungsvertrag Stadtwerke',
]
const CUSTOMERS = [
  'Müller Maschinenbau GmbH',
  'Bau & Plan AG',
  'Greentech Energie GmbH',
  'Logistik Rhein-Main',
  'Pharma Vitalis GmbH',
]

export const rapporteSource: ReportSource = {
  id: 'rapporte',
  module: 'rapporte',
  labelKey: 'berichte.sources.rapporte.label',
  label: 'Arbeitsrapporte',
  icon: ClipboardList,
  description: 'Leistungsnachweise, Stunden und Material',
  defaultViz: 'bar',
  fields: [
    { key: 'report_date', labelKey: 'berichte.fields.rapporte.report_date', label: 'Rapportdatum', dataType: 'date', role: 'dimension' },
    { key: 'status', labelKey: 'berichte.fields.rapporte.status', label: 'Status', dataType: 'enum', role: 'dimension', enumValues: STATUS },
    { key: 'work_type', labelKey: 'berichte.fields.rapporte.work_type', label: 'Leistungsart', dataType: 'enum', role: 'dimension', enumValues: WORK_TYPE },
    { key: 'author', labelKey: 'berichte.fields.rapporte.author', label: 'Ersteller', dataType: 'string', role: 'dimension' },
    { key: 'reviewer', labelKey: 'berichte.fields.rapporte.reviewer', label: 'Prüfer', dataType: 'string', role: 'dimension' },
    { key: 'project', labelKey: 'berichte.fields.rapporte.project', label: 'Projekt', dataType: 'string', role: 'dimension' },
    { key: 'customer', labelKey: 'berichte.fields.rapporte.customer', label: 'Kunde', dataType: 'string', role: 'dimension' },
    { key: 'hours', labelKey: 'berichte.fields.rapporte.hours', label: 'Arbeitsstunden', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'sum' },
    { key: 'material_cost', labelKey: 'berichte.fields.rapporte.material_cost', label: 'Materialkosten', dataType: 'number', role: 'measure', format: 'currency', defaultAgg: 'sum' },
    { key: 'travel_km', labelKey: 'berichte.fields.rapporte.travel_km', label: 'Fahrtweg (km)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'sum' },
    { key: 'line_count', labelKey: 'berichte.fields.rapporte.line_count', label: 'Positionen', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'sum' },
  ],
  sampleRows: (count = 160) =>
    generateRows(660, count, (r) => ({
      report_date: isoDaysAgo(intBetween(r(), 0, 180)),
      status: pick(STATUS, r()).value,
      work_type: pick(WORK_TYPE, r()).value,
      author: pick(AUTHORS, r()),
      reviewer: pick(REVIEWERS, r()),
      project: pick(PROJECTS, r()),
      customer: pick(CUSTOMERS, r()),
      hours: floatBetween(r(), 0.5, 12, 1),
      material_cost: floatBetween(r(), 0, 4800, 2),
      travel_km: intBetween(r(), 0, 220),
      line_count: intBetween(r(), 1, 18),
    })),
}
