import { Contact } from 'lucide-react'
import type { ReportSource } from './types'
import { floatBetween, generateRows, intBetween, isoDaysAgo, pick } from './sample-utils'

const STAGE = [
  { value: 'lead', label: 'Lead' },
  { value: 'qualified', label: 'Qualifiziert' },
  { value: 'proposal', label: 'Angebot' },
  { value: 'negotiation', label: 'Verhandlung' },
  { value: 'won', label: 'Gewonnen' },
  { value: 'lost', label: 'Verloren' },
]
const INDUSTRY = [
  { value: 'maschinenbau', label: 'Maschinenbau' },
  { value: 'fintech', label: 'Fintech' },
  { value: 'design', label: 'Design' },
  { value: 'medien', label: 'Medien' },
  { value: 'pharma', label: 'Pharma' },
  { value: 'handwerk', label: 'Handwerk' },
]
const COUNTRY = [
  { value: 'DE', label: 'Deutschland' },
  { value: 'CH', label: 'Schweiz' },
  { value: 'AT', label: 'Österreich' },
]
const COMPANY_SIZE = [
  { value: 'small', label: 'Klein (1–20)' },
  { value: 'medium', label: 'Mittel (21–100)' },
  { value: 'large', label: 'Groß (100+)' },
]
const SOURCE = [
  { value: 'website', label: 'Website' },
  { value: 'referral', label: 'Empfehlung' },
  { value: 'fair', label: 'Messe' },
  { value: 'coldcall', label: 'Kaltakquise' },
  { value: 'campaign', label: 'Kampagne' },
]
const OWNERS = ['Stefan Vogel', 'Lena Hofer', 'Marco Reis', 'Julia Brandt', 'Tom Keller']

export const kontakteSource: ReportSource = {
  id: 'kontakte',
  module: 'crm',
  labelKey: 'berichte.sources.kontakte.label',
  label: 'Kontakte & Deals',
  icon: Contact,
  description: 'Pipeline, Leads und Abschlüsse',
  defaultViz: 'bar',
  fields: [
    { key: 'created_at', labelKey: 'berichte.fields.kontakte.created_at', label: 'Erstellt am', dataType: 'date', role: 'dimension' },
    { key: 'close_date', labelKey: 'berichte.fields.kontakte.close_date', label: 'Abschlussdatum', dataType: 'date', role: 'dimension' },
    { key: 'stage', labelKey: 'berichte.fields.kontakte.stage', label: 'Pipeline-Stufe', dataType: 'enum', role: 'dimension', enumValues: STAGE },
    { key: 'industry', labelKey: 'berichte.fields.kontakte.industry', label: 'Branche', dataType: 'enum', role: 'dimension', enumValues: INDUSTRY },
    { key: 'company_size', labelKey: 'berichte.fields.kontakte.company_size', label: 'Firmengröße', dataType: 'enum', role: 'dimension', enumValues: COMPANY_SIZE },
    { key: 'country', labelKey: 'berichte.fields.kontakte.country', label: 'Land', dataType: 'enum', role: 'dimension', enumValues: COUNTRY },
    { key: 'source', labelKey: 'berichte.fields.kontakte.source', label: 'Lead-Quelle', dataType: 'enum', role: 'dimension', enumValues: SOURCE },
    { key: 'owner', labelKey: 'berichte.fields.kontakte.owner', label: 'Verantwortlich', dataType: 'string', role: 'dimension' },
    { key: 'value', labelKey: 'berichte.fields.kontakte.value', label: 'Deal-Wert', dataType: 'number', role: 'measure', format: 'currency', defaultAgg: 'sum' },
    { key: 'weighted_value', labelKey: 'berichte.fields.kontakte.weighted_value', label: 'Gewichteter Wert', dataType: 'number', role: 'measure', format: 'currency', defaultAgg: 'sum' },
    { key: 'probability', labelKey: 'berichte.fields.kontakte.probability', label: 'Wahrscheinlichkeit', dataType: 'number', role: 'measure', format: 'percent', defaultAgg: 'avg' },
    { key: 'activity_count', labelKey: 'berichte.fields.kontakte.activity_count', label: 'Aktivitäten', dataType: 'number', role: 'measure', format: 'number', defaultAgg: 'sum' },
  ],
  sampleRows: (count = 120) =>
    generateRows(202, count, (r) => {
      const stage = pick(STAGE, r())
      const prob =
        stage.value === 'won' ? 100 : stage.value === 'lost' ? 0 : intBetween(r(), 10, 90)
      const value = floatBetween(r(), 1500, 95000, 0)
      const createdDaysAgo = intBetween(r(), 0, 364)
      const closed = stage.value === 'won' || stage.value === 'lost'
      return {
        created_at: isoDaysAgo(createdDaysAgo),
        close_date: closed ? isoDaysAgo(intBetween(r(), 0, Math.max(1, createdDaysAgo))) : '',
        stage: stage.value,
        industry: pick(INDUSTRY, r()).value,
        company_size: pick(COMPANY_SIZE, r()).value,
        country: pick(COUNTRY, r()).value,
        source: pick(SOURCE, r()).value,
        owner: pick(OWNERS, r()),
        value,
        weighted_value: Math.round((value * prob) / 100),
        probability: prob,
        activity_count: intBetween(r(), 0, 24),
      }
    }),
}
