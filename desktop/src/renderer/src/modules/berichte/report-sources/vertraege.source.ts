import { FileSignature } from 'lucide-react'
import type { ReportSource } from './types'
import { floatBetween, generateRows, intBetween, isoDaysAgo, pick } from './sample-utils'

const TYPE = [
  { value: 'service', label: 'Dienstleistung' },
  { value: 'rental', label: 'Miete' },
  { value: 'employment', label: 'Anstellung' },
  { value: 'nda', label: 'Geheimhaltung' },
  { value: 'supply', label: 'Liefervertrag' },
  { value: 'license', label: 'Lizenz' },
]
const STATUS = [
  { value: 'active', label: 'Aktiv' },
  { value: 'draft', label: 'Entwurf' },
  { value: 'expired', label: 'Abgelaufen' },
  { value: 'terminated', label: 'Gekündigt' },
]
const RENEW = [
  { value: 'yes', label: 'Automatisch' },
  { value: 'no', label: 'Manuell' },
]
const PARTNERS = [
  'Müller Maschinenbau GmbH',
  'Fintech Solutions AG',
  'Pharma Vitalis GmbH',
  'Greentech Energie GmbH',
  'Logistik Rhein-Main',
  'Bau & Plan AG',
  'TeleNet Services GmbH',
  'Office Vermietung Süd',
]
const OWNERS = ['Lena Hofer', 'Marco Reis', 'Sara Brandt', 'Jonas Vogel', 'Mara Lang']

export const vertraegeSource: ReportSource = {
  id: 'vertraege',
  module: 'vertraege',
  labelKey: 'berichte.sources.vertraege.label',
  label: 'Verträge',
  icon: FileSignature,
  description: 'Vertragsbestand, Werte und Laufzeiten',
  defaultViz: 'bar',
  fields: [
    { key: 'contract_type', labelKey: 'berichte.fields.vertraege.contract_type', label: 'Vertragsart', dataType: 'enum', role: 'dimension', enumValues: TYPE },
    { key: 'status', labelKey: 'berichte.fields.vertraege.status', label: 'Status', dataType: 'enum', role: 'dimension', enumValues: STATUS },
    { key: 'partner', labelKey: 'berichte.fields.vertraege.partner', label: 'Vertragspartner', dataType: 'string', role: 'dimension' },
    { key: 'owner', labelKey: 'berichte.fields.vertraege.owner', label: 'Verantwortlich', dataType: 'string', role: 'dimension' },
    { key: 'auto_renew', labelKey: 'berichte.fields.vertraege.auto_renew', label: 'Verlängerung', dataType: 'enum', role: 'dimension', enumValues: RENEW },
    { key: 'start_date', labelKey: 'berichte.fields.vertraege.start_date', label: 'Vertragsbeginn', dataType: 'date', role: 'dimension' },
    { key: 'end_date', labelKey: 'berichte.fields.vertraege.end_date', label: 'Vertragsende', dataType: 'date', role: 'dimension' },
    { key: 'contract_value', labelKey: 'berichte.fields.vertraege.contract_value', label: 'Vertragswert', dataType: 'number', role: 'measure', format: 'currency', defaultAgg: 'sum' },
    { key: 'term_months', labelKey: 'berichte.fields.vertraege.term_months', label: 'Laufzeit (Monate)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'avg' },
    { key: 'notice_period_days', labelKey: 'berichte.fields.vertraege.notice_period_days', label: 'Kündigungsfrist (Tage)', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'avg' },
  ],
  sampleRows: (count = 90) =>
    generateRows(630, count, (r) => {
      const startDaysAgo = intBetween(r(), 30, 1460)
      const term = intBetween(r(), 6, 60)
      return {
        contract_type: pick(TYPE, r()).value,
        status: pick(STATUS, r()).value,
        partner: pick(PARTNERS, r()),
        owner: pick(OWNERS, r()),
        auto_renew: pick(RENEW, r()).value,
        start_date: isoDaysAgo(startDaysAgo),
        end_date: isoDaysAgo(startDaysAgo - term * 30),
        contract_value: floatBetween(r(), 1200, 240000, 2),
        term_months: term,
        notice_period_days: pick([30, 60, 90], r()),
      }
    }),
}
